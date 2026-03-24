package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jensbaagaard/trello-tui/internal/config"
	"github.com/jensbaagaard/trello-tui/internal/trello"
	"github.com/jensbaagaard/trello-tui/internal/tui"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	boardName := flag.String("board", "", "open directly into a board by name")
	autoRefresh := flag.Int("auto-refresh", 0, "auto-refresh interval in seconds (0=disabled)")
	flag.Parse()

	if *showVersion {
		fmt.Println("trello-tui " + version)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := trello.NewClient(cfg.APIKey, cfg.Token)
	app := tui.NewAppModel(client, version, tui.Options{
		BoardName:       *boardName,
		AutoRefreshSecs: *autoRefresh,
	})

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running program:", err)
		os.Exit(1)
	}
}
