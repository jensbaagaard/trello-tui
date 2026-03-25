package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m SearchModel) View() string {
	if m.showHelp {
		return m.renderSearchHelp()
	}

	var b strings.Builder

	title := titleStyle.Render("Search Cards")
	b.WriteString(title)
	b.WriteString("\n")

	b.WriteString(m.textInput.View())
	b.WriteString("\n")

	if m.loading {
		b.WriteString(helpStyle.Render("Searching..."))
		b.WriteString("\n")
	} else if m.statusMsg != "" {
		b.WriteString(errorStyle.Render(m.statusMsg))
		b.WriteString("\n")
	} else if !m.searched {
		b.WriteString(helpStyle.Render("Type a query and press enter"))
		b.WriteString("\n")
	} else if len(m.results) == 0 {
		b.WriteString(helpStyle.Render("No results"))
		b.WriteString("\n")
	} else {
		visible := m.visibleResultCount()
		end := m.scrollTop + visible
		if end > len(m.results) {
			end = len(m.results)
		}

		if m.scrollTop > 0 {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d more", m.scrollTop)))
			b.WriteString("\n")
		}

		for i := m.scrollTop; i < end; i++ {
			card := m.results[i]
			prefix := "  "
			style := lipgloss.NewStyle()
			if i == m.cursor {
				prefix = "▸ "
				style = style.Foreground(primaryColor).Bold(true)
			}

			boardName := card.Board.Name
			if boardName == "" {
				boardName = card.IDBoard[:8]
			}
			listName := card.List.Name
			if listName == "" {
				listName = "?"
			}

			line := fmt.Sprintf("%s%s  [%s] → %s", prefix, card.Name, boardName, listName)
			if card.Closed {
				line += " (archived)"
			}

			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}

		remaining := len(m.results) - end
		if remaining > 0 {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  ↓ %d more", remaining)))
			b.WriteString("\n")
		}
	}

	var help string
	if !m.searched || m.textInput.Focused() {
		help = "enter:search  esc:back"
	} else {
		help = "j/k:navigate  enter:open  /:search again  esc:back"
	}
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m SearchModel) renderSearchHelp() string {
	sections := []helpSection{
		{Title: "Search Results", Entries: []helpEntry{
			{"j/k", "Move up/down"},
			{"enter", "Open card"},
			{"/", "Search again"},
			{"?", "Toggle help"},
			{"esc", "Back"},
		}},
	}
	return renderHelpOverlay("Search — Help", sections, m.width, m.height)
}
