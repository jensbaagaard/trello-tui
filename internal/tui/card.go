package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jensbaagaard/trello-tui/internal/trello"
)

type cardMode int

const (
	cardView cardMode = iota
	cardEditTitle
	cardEditDesc
	cardMoveList
)

type CardModel struct {
	client    *trello.Client
	card      trello.Card
	lists     []trello.List
	listIndex int
	listName  string
	mode      cardMode
	titleEdit textinput.Model
	descEdit  textarea.Model
	moveIndex int
	width     int
	height    int
	statusMsg string
}

func NewCardModel(client *trello.Client, card trello.Card, lists []trello.List, listIndex int) CardModel {
	ti := textinput.New()
	ti.Placeholder = "Card title"
	ti.CharLimit = 200

	ta := textarea.New()
	ta.Placeholder = "Description..."
	ta.SetWidth(60)
	ta.SetHeight(10)

	listName := ""
	if listIndex >= 0 && listIndex < len(lists) {
		listName = lists[listIndex].Name
	}

	return CardModel{
		client:    client,
		card:      card,
		lists:     lists,
		listIndex: listIndex,
		listName:  listName,
		moveIndex: listIndex,
		titleEdit: ti,
		descEdit:  ta,
	}
}

func (m CardModel) Init() tea.Cmd {
	return nil
}

func (m CardModel) Update(msg tea.Msg) (CardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.descEdit.SetWidth(msg.Width - 4)

	case CardUpdatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
			return m, nil
		}
		m.card = msg.Card
		m.statusMsg = "Card updated"
		return m, nil

	case CardMovedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error moving: %v", msg.Err)
			return m, nil
		}
		m.card = msg.Card
		m.listIndex = m.moveIndex
		if m.listIndex >= 0 && m.listIndex < len(m.lists) {
			m.listName = m.lists[m.listIndex].Name
		}
		m.statusMsg = fmt.Sprintf("Moved to %s", m.listName)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.mode == cardEditTitle {
		var cmd tea.Cmd
		m.titleEdit, cmd = m.titleEdit.Update(msg)
		return m, cmd
	}
	if m.mode == cardEditDesc {
		var cmd tea.Cmd
		m.descEdit, cmd = m.descEdit.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m CardModel) handleKey(msg tea.KeyMsg) (CardModel, tea.Cmd) {
	switch m.mode {
	case cardEditTitle:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.titleEdit.Value())
			if name != "" {
				m.card.Name = name
				m.mode = cardView
				return m, m.updateCard(map[string]string{"name": name})
			}
			m.mode = cardView
		case "esc":
			m.mode = cardView
		default:
			var cmd tea.Cmd
			m.titleEdit, cmd = m.titleEdit.Update(msg)
			return m, cmd
		}
		return m, nil

	case cardEditDesc:
		switch msg.String() {
		case "esc":
			desc := m.descEdit.Value()
			m.card.Desc = desc
			m.mode = cardView
			return m, m.updateCard(map[string]string{"desc": desc})
		default:
			var cmd tea.Cmd
			m.descEdit, cmd = m.descEdit.Update(msg)
			return m, cmd
		}

	case cardMoveList:
		switch msg.String() {
		case "j", "down":
			if m.moveIndex < len(m.lists)-1 {
				m.moveIndex++
			}
		case "k", "up":
			if m.moveIndex > 0 {
				m.moveIndex--
			}
		case "enter":
			if m.moveIndex != m.listIndex {
				return m, m.moveToList(m.moveIndex)
			}
			m.mode = cardView
		case "esc":
			m.moveIndex = m.listIndex
			m.mode = cardView
		}
		return m, nil

	default:
		switch msg.String() {
		case "t", "e":
			m.mode = cardEditTitle
			m.titleEdit.SetValue(m.card.Name)
			m.titleEdit.Focus()
			return m, textinput.Blink
		case "E":
			m.mode = cardEditDesc
			m.descEdit.SetValue(m.card.Desc)
			m.descEdit.Focus()
			return m, textarea.Blink
		case "m":
			m.mode = cardMoveList
			m.moveIndex = m.listIndex
		case ",":
			if m.listIndex > 0 {
				m.moveIndex = m.listIndex - 1
				return m, m.moveToList(m.moveIndex)
			}
		case ".":
			if m.listIndex < len(m.lists)-1 {
				m.moveIndex = m.listIndex + 1
				return m, m.moveToList(m.moveIndex)
			}
		case "<":
			if m.listIndex > 0 {
				m.moveIndex = 0
				return m, m.moveToList(0)
			}
		case ">":
			if m.listIndex < len(m.lists)-1 {
				m.moveIndex = len(m.lists) - 1
				return m, m.moveToList(len(m.lists) - 1)
			}
		case "c":
			m.statusMsg = "Use board view to archive cards"
		}
	}

	return m, nil
}

func (m CardModel) moveToList(targetIndex int) tea.Cmd {
	client := m.client
	cardID := m.card.ID
	fromListID := m.card.IDList
	toListID := m.lists[targetIndex].ID
	return func() tea.Msg {
		card, err := client.MoveCard(cardID, toListID)
		return CardMovedMsg{Card: card, FromListID: fromListID, ToListID: toListID, Err: err}
	}
}

func (m CardModel) updateCard(fields map[string]string) tea.Cmd {
	client := m.client
	cardID := m.card.ID
	return func() tea.Msg {
		card, err := client.UpdateCard(cardID, fields)
		return CardUpdatedMsg{Card: card, Err: err}
	}
}

func (m CardModel) View() string {
	contentWidth := m.width - 8
	if contentWidth < 40 {
		contentWidth = 40
	}
	if contentWidth > 100 {
		contentWidth = 100
	}

	var b strings.Builder

	// Header: list name as breadcrumb
	b.WriteString(helpStyle.Render("in " + m.listName))
	b.WriteString("\n\n")

	// Card title
	cardTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(contentWidth)
	b.WriteString(cardTitleStyle.Render(m.card.Name))
	b.WriteString("\n")

	// Labels
	if len(m.card.Labels) > 0 {
		b.WriteString("\n")
		var labels []string
		for _, l := range m.card.Labels {
			name := l.Name
			if name == "" {
				name = l.Color
			}
			labels = append(labels, labelColor(l.Color).Render(name))
		}
		b.WriteString(strings.Join(labels, "  "))
		b.WriteString("\n")
	}

	// Due date
	if m.card.Due != "" {
		label, style := formatDue(m.card.Due, m.card.DueComplete)
		if label != "" {
			b.WriteString("\n")
			b.WriteString(style.Render("Due: " + label))
			b.WriteString("\n")
		}
	}

	// Members
	if len(m.card.Members) > 0 {
		b.WriteString("\n")
		var names []string
		for _, member := range m.card.Members {
			names = append(names, member.FullName)
		}
		b.WriteString(helpStyle.Render("Members: " + strings.Join(names, ", ")))
		b.WriteString("\n")
	}

	// URL
	if m.card.ShortURL != "" {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(m.card.ShortURL))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	divider := helpStyle.Render(strings.Repeat("─", contentWidth))

	switch m.mode {
	case cardEditDesc:
		b.WriteString("Description (esc to save):\n\n")
		b.WriteString(m.descEdit.View())
	case cardEditTitle:
		b.WriteString("Edit title (enter to save, esc to cancel):\n\n")
		b.WriteString(m.titleEdit.View())
	case cardMoveList:
		sectionTitle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		b.WriteString(sectionTitle.Render("Move to list"))
		b.WriteString("\n" + divider + "\n\n")
		for i, l := range m.lists {
			cursor := "  "
			style := lipgloss.NewStyle()
			if i == m.moveIndex {
				cursor = "▸ "
				style = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
			}
			suffix := ""
			if i == m.listIndex {
				suffix = helpStyle.Render("  (current)")
			}
			b.WriteString(cursor + style.Render(l.Name) + suffix + "\n")
		}
	default:
		sectionTitle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		b.WriteString(sectionTitle.Render("Description"))
		b.WriteString("\n" + divider + "\n\n")
		desc := m.card.Desc
		if desc == "" {
			desc = helpStyle.Render("(no description)")
		}
		descStyle := lipgloss.NewStyle().Width(contentWidth)
		b.WriteString(descStyle.Render(desc))
	}

	b.WriteString("\n\n")

	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
		b.WriteString(statusStyle.Render(m.statusMsg) + "\n\n")
	}

	if m.mode == cardView {
		b.WriteString(helpStyle.Render("t:edit title  E:edit desc  m:move to list  ,/.:move left/right  </>:first/last  esc:back"))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
