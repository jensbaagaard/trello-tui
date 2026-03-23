package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jensbaagaard/trello-tui/internal/config"
	"github.com/jensbaagaard/trello-tui/internal/trello"
	"github.com/jensbaagaard/trello-tui/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := trello.NewClient(cfg.APIKey, cfg.Token)
	app := tui.NewAppModel(client)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running program:", err)
		os.Exit(1)
	}
}
