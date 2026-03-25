package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpEntry struct {
	Key  string
	Desc string
}

type helpSection struct {
	Title   string
	Entries []helpEntry
}

func renderHelpOverlay(title string, sections []helpSection, width, height int) string {
	maxW := width - 8
	if maxW < 30 {
		maxW = 30
	}
	if maxW > 72 {
		maxW = 72
	}

	var b strings.Builder

	heading := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(title)
	b.WriteString(heading + "\n\n")

	keyStyle := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true)

	for i, sec := range sections {
		if sec.Title != "" {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render(sec.Title) + "\n")
		}
		for _, e := range sec.Entries {
			b.WriteString("  " + keyStyle.Render(e.Key) + "  " + helpStyle.Render(e.Desc) + "\n")
		}
		if i < len(sections)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("Press ? or esc to close"))

	content := b.String()

	contentH := strings.Count(content, "\n") + 1
	boxH := contentH + 2 // borders
	if boxH > height-2 {
		boxH = height - 2
	}
	innerH := boxH - 2
	if innerH < 3 {
		innerH = 3
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(0, 1).
		Width(maxW).
		Height(innerH).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
