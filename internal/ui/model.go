package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"

	"github.com/jteer/prwatch/internal/config"
	gh "github.com/jteer/prwatch/internal/github"
	"github.com/jteer/prwatch/internal/logger"
	"github.com/jteer/prwatch/internal/ui/layout"
	"github.com/jteer/prwatch/internal/ui/layout/twopane"
)

// -- messages -----------------------------------------------------------------

type configSavedMsg struct{}
type configSaveErrMsg struct{ err error }
type configSignOutMsg struct{}

// prBatchMsg carries the first page from all repos (one HTTP round-trip).
type prBatchMsg struct {
	gen     uint64
	prs     []gh.PR
	cursors map[int]string // repoIdx -> cursor for repos with more pages
	err     error
}

// prPageMsg carries a follow-up page for a single repo.
type prPageMsg struct {
	gen     uint64
	prs     []gh.PR
	repoIdx int
	next    string // cursor for next page; empty = done
	err     error
}

type tickMsg time.Time

// -- sort / scope -------------------------------------------------------------

var sortFields = []string{"age", "status", "repo"}
var scopeFields = []string{"ALL", "mine", "review"}

// -- model --------------------------------------------------------------------

type AppModel struct {
	cfg      *config.Config
	client   *gh.Client
	tokenSrc config.TokenSource

	allPRs      []gh.PR
	filteredPRs []gh.PR
	selected    int
	lastUpdated time.Time
	loading     bool
	inFlight    int    // pending fetch commands
	fetchGen    uint64 // incremented on each refresh; stale msgs are dropped
	err         error

	filterMode  bool
	filterInput textinput.Model
	sortIdx     int
	scopeIdx    int
	focusPane   int

	showHelp     bool
	showConfig   bool
	configTab    configTab
	configRepoSel int
	pausedRepos  map[string]bool // full "owner/name" of paused repos
	configPath   string
	configSaved  bool

	layouts   []layout.Layout
	layoutIdx int

	width  int
	height int
}

func New(cfg *config.Config, client *gh.Client, tokenSrc config.TokenSource, configPath string) AppModel {
	fi := textinput.New()
	fi.Placeholder = "repo · title · author"

	return AppModel{
		cfg:         cfg,
		client:      client,
		tokenSrc:    tokenSrc,
		filterInput: fi,
		loading:     true,
		fetchGen:    1, // init here so Init() cmd and model agree on gen
		inFlight:    1,
		pausedRepos: make(map[string]bool),
		layouts:     []layout.Layout{twopane.New()},
		configPath:  configPath,
	}
}

func (m AppModel) Init() tea.Cmd {
	// fetchGen and inFlight already set in New(); Init has a value receiver
	// so mutations here would be lost — read model state, don't write it.
	return tea.Batch(
		fetchBatchCmd(m.client, m.fetchGen, m.activeRepoIdxs()),
		tickCmd(m.cfg.Interval()),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case prBatchMsg:
		if msg.gen != m.fetchGen {
			return m, nil // stale
		}
		m.inFlight-- // batch done
		if msg.err != nil {
			m.err = msg.err
			logger.Logf("[github] batch error: %v", msg.err)
			if m.inFlight == 0 {
				m.loading = false
			}
			return m, nil
		}
		m.allPRs = append(m.allPRs, msg.prs...)
		m.lastUpdated = time.Now()
		m.err = nil
		m.applyFilterSort()

		// Fire a follow-up command for each repo with more pages.
		cmds := make([]tea.Cmd, 0, len(msg.cursors))
		for repoIdx, cursor := range msg.cursors {
			m.inFlight++
			cmds = append(cmds, fetchPageCmd(m.client, m.fetchGen, repoIdx, cursor))
		}
		if m.inFlight == 0 {
			m.loading = false
		}
		logger.Logf("[github] batch: +%d PRs, %d repos paging", len(msg.prs), len(msg.cursors))
		return m, tea.Batch(cmds...)

	case prPageMsg:
		if msg.gen != m.fetchGen {
			return m, nil // stale
		}
		if msg.err != nil {
			logger.Logf("[github] page error repo %d: %v", msg.repoIdx, msg.err)
			m.inFlight--
			if m.inFlight == 0 {
				m.loading = false
			}
			return m, nil
		}
		m.allPRs = append(m.allPRs, msg.prs...)
		m.applyFilterSort()
		logger.Logf("[github] page repo %d: +%d PRs", msg.repoIdx, len(msg.prs))

		if msg.next != "" {
			// Same inFlight count: old command done, new one started.
			return m, fetchPageCmd(m.client, m.fetchGen, msg.repoIdx, msg.next)
		}
		m.inFlight--
		if m.inFlight == 0 {
			m.loading = false
		}
		return m, nil

	case tickMsg:
		m.allPRs = nil
		m.loading = true
		m.fetchGen++
		m.inFlight = 1
		return m, tea.Batch(
			fetchBatchCmd(m.client, m.fetchGen, m.activeRepoIdxs()),
			tickCmd(m.cfg.Interval()),
		)

	case configSavedMsg:
		m.configSaved = false
		return m, nil

	case configSaveErrMsg:
		m.configSaved = false
		logger.Logf("[config] save error: %v", msg.err)
		return m, nil

	case configSignOutMsg:
		m.tokenSrc = config.SourceNone
		return m, nil

	case tea.KeyPressMsg:
		if m.showConfig {
			return m.updateConfigMode(msg)
		}
		if m.filterMode {
			return m.updateFilter(msg)
		}
		return m.updateKeys(msg)
	}

	return m, nil
}

func (m AppModel) updateKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
	case "esc":
		m.showHelp = false
	case "c":
		m.showConfig = true
		m.showHelp = false
		m.configRepoSel = 0
		logger.Logf("[ui] opening config screen")
	case "j", "down":
		m.moveDown()
	case "k", "up":
		m.moveUp()
	case "g":
		m.selected = 0
	case "G":
		if len(m.filteredPRs) > 0 {
			m.selected = len(m.filteredPRs) - 1
		}
	case "tab":
		m.focusPane = 1 - m.focusPane
	case "/":
		m.filterMode = true
		m.filterInput.SetValue("")
		cmd := m.filterInput.Focus()
		return m, cmd
	case "s":
		m.sortIdx = (m.sortIdx + 1) % len(sortFields)
		m.applyFilterSort()
	case "f":
		m.scopeIdx = (m.scopeIdx + 1) % len(scopeFields)
		m.applyFilterSort()
	case "r":
		m.allPRs = nil
		m.loading = true
		m.fetchGen++
		m.inFlight = 1
		logger.Logf("[ui] manual refresh")
		return m, fetchBatchCmd(m.client, m.fetchGen, m.activeRepoIdxs())
	case "o":
		return m, openBrowser(m.selectedURL())
	case "L":
		m.layoutIdx = (m.layoutIdx + 1) % len(m.layouts)
	}
	return m, nil
}

func (m AppModel) updateConfigMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	repos := m.client.Repos()
	switch msg.String() {
	case "esc", "q":
		m.showConfig = false
	case "tab":
		m.configTab = configTab((int(m.configTab) + 1) % len(configTabLabels))
	case "j", "down":
		if m.configTab == tabRepos && len(repos) > 0 {
			m.configRepoSel = (m.configRepoSel + 1) % len(repos)
		}
	case "k", "up":
		if m.configTab == tabRepos && len(repos) > 0 {
			m.configRepoSel = (m.configRepoSel - 1 + len(repos)) % len(repos)
		}
	case "1":
		m.configTab = tabRepos
	case "2":
		m.configTab = tabCreds
	case "space":
		if m.configTab == tabRepos && m.configRepoSel < len(repos) {
			repo := repos[m.configRepoSel]
			m.pausedRepos[repo] = !m.pausedRepos[repo]
			logger.Logf("[config] toggled %s paused=%v", repo, m.pausedRepos[repo])
		}
	case "enter":
		m.configSaved = true
		return m, saveConfigCmd(m.cfg, m.configPath)
	case "x":
		if m.configTab == tabCreds {
			return m, signOutCmd()
		}
	}
	return m, nil
}

func (m AppModel) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterMode = false
		m.filterInput.Blur()
		m.filterInput.SetValue("")
		m.applyFilterSort()
		return m, nil
	case "enter":
		m.filterMode = false
		m.filterInput.Blur()
		m.applyFilterSort()
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.applyFilterSort()
	return m, cmd
}

func (m AppModel) View() tea.View {
	var s string
	if m.showConfig {
		s = renderConfig(m)
	} else {
		s = m.layouts[m.layoutIdx].View(m.layoutState())
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func (m *AppModel) moveDown() {
	if m.selected < len(m.filteredPRs)-1 {
		m.selected++
	}
}

func (m *AppModel) moveUp() {
	if m.selected > 0 {
		m.selected--
	}
}

func (m *AppModel) selectedURL() string {
	if len(m.filteredPRs) == 0 || m.selected >= len(m.filteredPRs) {
		return ""
	}
	return m.filteredPRs[m.selected].URL
}

func (m *AppModel) activeRepoIdxs() []int {
	count := m.client.RepoCount()
	out := make([]int, 0, count)
	for i := 0; i < count; i++ {
		if !m.pausedRepos[m.client.RepoFull(i)] {
			out = append(out, i)
		}
	}
	return out
}

func (m *AppModel) applyFilterSort() {
	query := strings.ToLower(m.filterInput.Value())
	scope := scopeFields[m.scopeIdx]

	filtered := make([]gh.PR, 0, len(m.allPRs))
	for _, p := range m.allPRs {
		if m.pausedRepos[p.Repo] {
			continue
		}
		if !matchesScope(p, scope, m.cfg.GitHub.Username) {
			continue
		}
		if query != "" && !matchesFilter(p, query) {
			continue
		}
		filtered = append(filtered, p)
	}

	sf := sortFields[m.sortIdx]
	sort.SliceStable(filtered, func(i, j int) bool {
		switch sf {
		case "status":
			return filtered[i].Urgency > filtered[j].Urgency
		case "repo":
			return filtered[i].Repo < filtered[j].Repo
		default:
			return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
		}
	})

	m.filteredPRs = filtered
	if m.selected >= len(m.filteredPRs) {
		m.selected = max(0, len(m.filteredPRs)-1)
	}
}

func matchesScope(p gh.PR, scope, username string) bool {
	switch scope {
	case "mine":
		return p.Author == username
	case "review":
		for _, r := range p.Reviewers {
			if r.Login == username && r.Decision == "PENDING" {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func matchesFilter(p gh.PR, q string) bool {
	return strings.Contains(strings.ToLower(p.Repo), q) ||
		strings.Contains(strings.ToLower(p.Title), q) ||
		strings.Contains(strings.ToLower(p.Author), q)
}

func (m AppModel) layoutState() layout.State {
	upd := "never"
	if !m.lastUpdated.IsZero() {
		upd = fmtAge(time.Since(m.lastUpdated))
	}
	if m.err != nil {
		upd = "error: " + m.err.Error()
	}

	repos := make(map[string]bool)
	for _, p := range m.filteredPRs {
		repos[p.Repo] = true
	}

	return layout.State{
		PRs:         m.filteredPRs,
		Selected:    m.selected,
		FilterQuery: m.filterInput.Value(),
		SortField:   sortFields[m.sortIdx],
		Scope:       scopeFields[m.scopeIdx],
		FocusPane:   m.focusPane,
		ShowHelp:    m.showHelp,
		Loading:     m.loading,
		LastUpdated: upd,
		RepoCount:   len(repos),
		Width:       m.width,
		Height:      m.height,
	}
}

func fmtAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// -- commands -----------------------------------------------------------------

func fetchBatchCmd(client *gh.Client, gen uint64, activeIdxs []int) tea.Cmd {
	return func() tea.Msg {
		logger.Logf("[github] fetching batch (gen=%d, repos=%d)", gen, len(activeIdxs))
		prs, cursors, err := client.FetchBatch(context.Background(), activeIdxs)
		return prBatchMsg{gen: gen, prs: prs, cursors: cursors, err: err}
	}
}

func fetchPageCmd(client *gh.Client, gen uint64, repoIdx int, cursor string) tea.Cmd {
	return func() tea.Msg {
		prs, next, err := client.FetchPage(context.Background(), repoIdx, cursor)
		return prPageMsg{gen: gen, prs: prs, repoIdx: repoIdx, next: next, err: err}
	}
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func saveConfigCmd(cfg *config.Config, path string) tea.Cmd {
	return func() tea.Msg {
		if err := cfg.Save(path); err != nil {
			logger.Logf("[config] save failed: %v", err)
			return configSaveErrMsg{err}
		}
		logger.Logf("[config] saved to %s", path)
		return tea.Tick(800*time.Millisecond, func(time.Time) tea.Msg {
			return configSavedMsg{}
		})()
	}
}

func signOutCmd() tea.Cmd {
	return func() tea.Msg {
		path, err := exec.LookPath("gh")
		if err != nil {
			return configSignOutMsg{}
		}
		_ = exec.Command(path, "auth", "logout", "--hostname", "github.com").Run()
		return configSignOutMsg{}
	}
}

func openBrowser(url string) tea.Cmd {
	if url == "" {
		return nil
	}
	return func() tea.Msg {
		logger.Logf("[ui] opening browser: %s", url)
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		default:
			cmd = exec.Command("cmd", "/c", "start", url)
		}
		if err := cmd.Start(); err != nil {
			logger.Logf("[ui] open browser error: %v", err)
		}
		return nil
	}
}
