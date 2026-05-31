package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/jteer/prwatch/internal/config"
	gh "github.com/jteer/prwatch/internal/github"
	"github.com/jteer/prwatch/internal/logger"
	"github.com/jteer/prwatch/internal/ui"
)

func main() {
	cfgPath := flag.String("config", config.DefaultPath(), "path to config.yaml")
	logPath := flag.String("log", logger.DefaultPath(), "path to log file (empty = disable)")
	flag.Parse()

	if *logPath != "" {
		closer, err := logger.Init(*logPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "prwatch: warning: cannot open log file %s: %v\n", *logPath, err)
		} else {
			defer closer.Close()
		}
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "prwatch: config: %v\n\nCopy config.yaml.example to %s and fill in your details.\n",
			err, *cfgPath)
		logger.Logf("[main] config load error: %v", err)
		os.Exit(1)
	}

	token, tokenSrc := config.ResolveToken(cfg.GitHub.TokenEnv)
	if token == "" {
		fmt.Fprintln(os.Stderr,
			"prwatch: no GitHub token found.\n"+
				"  Option 1: run `gh auth login` (recommended)\n"+
				"  Option 2: set "+cfg.GitHub.TokenEnv+" env var")
		logger.Logf("[main] no token available (env=%s)", cfg.GitHub.TokenEnv)
		os.Exit(1)
	}
	logger.Logf("[main] token source: %s", tokenSrc)

	username := config.ResolveUsername(cfg.GitHub.Username)
	if username == "" {
		fmt.Fprintln(os.Stderr,
			"prwatch: cannot determine GitHub username.\n"+
				"  Option 1: run `gh auth login`\n"+
				"  Option 2: set github.username in config.yaml")
		logger.Logf("[main] no username available")
		os.Exit(1)
	}
	logger.Logf("[main] username: %s (configured: %v)", username, cfg.GitHub.Username != "")

	if len(cfg.Repos) == 0 {
		fmt.Fprintln(os.Stderr, "prwatch: no repos configured — add them to config.yaml")
		os.Exit(1)
	}

	logger.Logf("[main] starting — %d repos, org=%s, user=%s, token=%s",
		len(cfg.Repos), cfg.Org, username, tokenSrc)

	client := gh.NewClient(token, username, cfg.ResolvedRepos(), cfg.Team)
	model := ui.New(cfg, client, tokenSrc, *cfgPath)

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "prwatch:", err)
		logger.Logf("[main] program error: %v", err)
		os.Exit(1)
	}
}
