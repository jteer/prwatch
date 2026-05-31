package styles

import lip "charm.land/lipgloss/v2"

// Terminal dark palette — true-colour hex, degrades to nearest 256-colour ANSI.
var (
	ColorBg       = lip.Color("#12151d") // main + detail background (unified)
	ColorDetailBg = lip.Color("#12151d") // kept as alias; same as ColorBg
	ColorHeader   = lip.Color("#161a24") // 236 header row
	ColorSelected = lip.Color("#1d2535") // 237 selected row
	ColorBorder   = lip.Color("#2b3140") // 238 borders
	ColorFG       = lip.Color("#c7ccd6") // 251 default text
	ColorMeta     = lip.Color("#828aa0") // 244 dim meta text
	ColorMeta2    = lip.Color("#8b93a7") // slightly lighter meta
	ColorLink     = lip.Color("#4cb3e8") // 39  PR numbers / links
	ColorTitleBg  = lip.Color("#0b0d12") // title bar / footer bg

	ColorUrgent  = lip.Color("#e5575c") // 203 changes req / review owed
	ColorWarning = lip.Color("#e0a23a") // 214 CI fail / stale >5d
	ColorGood    = lip.Color("#57b56a") // 114 approved / ready
	ColorDim     = lip.Color("#5f6675") // 240 draft

	ColorUrgentBg  = lip.Color("#1c1418")
	ColorWarningBg = lip.Color("#161410")
	ColorGoodBg    = lip.Color("#111a13")
)

// Base styles.
var (
	TitleBar = lip.NewStyle().
			Background(ColorTitleBg).
			Foreground(ColorMeta2)

	PaneHead = lip.NewStyle().
			Background(ColorHeader).
			Foreground(ColorMeta).
			PaddingLeft(1).PaddingRight(1)

	HeaderRow = lip.NewStyle().
			Background(ColorHeader).
			Foreground(ColorMeta).
			Bold(true)

	NormalRow = lip.NewStyle().
			Background(ColorBg).
			Foreground(ColorFG)

	SelectedRow = lip.NewStyle().
			Background(ColorSelected)

	FooterBar = lip.NewStyle().
			Background(ColorTitleBg).
			Foreground(ColorMeta)

	DetailBg = lip.NewStyle().
			Background(ColorDetailBg).
			Foreground(ColorFG)

	KeyCap = lip.NewStyle().
			Background(ColorMeta).
			Foreground(ColorBg).
			Bold(true).
			PaddingLeft(1).PaddingRight(1)

	HelpCard = lip.NewStyle().
			Background(ColorDetailBg).
			Border(lip.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	// Urgency status text styles.
	StatusUrgent  = lip.NewStyle().Foreground(ColorUrgent).Bold(true)
	StatusWarning = lip.NewStyle().Foreground(ColorWarning).Bold(true)
	StatusGood    = lip.NewStyle().Foreground(ColorGood).Bold(true)
	StatusDim     = lip.NewStyle().Foreground(ColorDim)
	StatusNeutral = lip.NewStyle().Foreground(ColorLink).Bold(true)

	// CI glyph styles.
	CIPass = lip.NewStyle().Foreground(ColorGood)
	CIFail = lip.NewStyle().Foreground(ColorUrgent)
	CIPend = lip.NewStyle().Foreground(ColorWarning)

	MetaText = lip.NewStyle().Foreground(ColorMeta)
	LinkText  = lip.NewStyle().Foreground(ColorLink)
	RepoText  = lip.NewStyle().Foreground(ColorMeta2).Bold(true)
)
