package tui

import (
	"fmt"
	"strings"
	"time"

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
	cardAddMember
	cardAddLabel
	cardSetDue
)

type CardModel struct {
	client       *trello.Client
	card         trello.Card
	lists        []trello.List
	listIndex    int
	listName     string
	boardID      string
	boardMembers []trello.Member
	boardLabels  []trello.Label
	mode         cardMode
	titleEdit    textinput.Model
	descEdit     textarea.Model
	dueInput     textinput.Model
	moveIndex    int
	memberIndex  int
	labelIndex   int
	width        int
	height       int
	statusMsg    string
}

func NewCardModel(client *trello.Client, card trello.Card, lists []trello.List, listIndex int) CardModel {
	ti := textinput.New()
	ti.Placeholder = "Card title"
	ti.CharLimit = 200

	ta := textarea.New()
	ta.Placeholder = "Description..."
	ta.SetWidth(60)
	ta.SetHeight(10)

	di := textinput.New()
	di.Placeholder = "YYYY-MM-DD (empty to clear)"
	di.CharLimit = 20

	listName := ""
	if listIndex >= 0 && listIndex < len(lists) {
		listName = lists[listIndex].Name
	}

	boardID := ""
	if len(lists) > 0 {
		boardID = lists[0].IDBoard
	}

	return CardModel{
		client:    client,
		card:      card,
		lists:     lists,
		listIndex: listIndex,
		listName:  listName,
		boardID:   boardID,
		moveIndex: listIndex,
		titleEdit: ti,
		descEdit:  ta,
		dueInput:  di,
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
		// Preserve members/labels from in-memory state since UpdateCard response may omit them
		members := m.card.Members
		labels := m.card.Labels
		m.card = msg.Card
		if len(m.card.Members) == 0 {
			m.card.Members = members
		}
		if len(m.card.Labels) == 0 {
			m.card.Labels = labels
		}
		m.statusMsg = "Card updated"
		return m, nil

	case CardMovedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error moving: %v", msg.Err)
			return m, nil
		}
		members := m.card.Members
		labels := m.card.Labels
		m.card = msg.Card
		if len(m.card.Members) == 0 {
			m.card.Members = members
		}
		if len(m.card.Labels) == 0 {
			m.card.Labels = labels
		}
		m.listIndex = m.moveIndex
		if m.listIndex >= 0 && m.listIndex < len(m.lists) {
			m.listName = m.lists[m.listIndex].Name
		}
		m.statusMsg = fmt.Sprintf("Moved to %s", m.listName)
		return m, nil

	case BoardMembersFetchedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error fetching members: %v", msg.Err)
			m.mode = cardView
			return m, nil
		}
		m.boardMembers = msg.Members
		return m, nil

	case BoardLabelsFetchedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error fetching labels: %v", msg.Err)
			m.mode = cardView
			return m, nil
		}
		m.boardLabels = msg.Labels
		return m, nil

	case MemberToggledMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
		}
		return m, nil

	case LabelToggledMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
		}
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
	if m.mode == cardSetDue {
		var cmd tea.Cmd
		m.dueInput, cmd = m.dueInput.Update(msg)
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

	case cardAddMember:
		switch msg.String() {
		case "j", "down":
			if m.memberIndex < len(m.boardMembers)-1 {
				m.memberIndex++
			}
		case "k", "up":
			if m.memberIndex > 0 {
				m.memberIndex--
			}
		case "enter", " ":
			if len(m.boardMembers) > 0 && m.memberIndex < len(m.boardMembers) {
				member := m.boardMembers[m.memberIndex]
				client := m.client
				cardID := m.card.ID
				memberID := member.ID
				for i, cm := range m.card.Members {
					if cm.ID == memberID {
						m.card.Members = append(m.card.Members[:i], m.card.Members[i+1:]...)
						return m, func() tea.Msg {
							return MemberToggledMsg{Err: client.RemoveMemberFromCard(cardID, memberID)}
						}
					}
				}
				m.card.Members = append(m.card.Members, member)
				return m, func() tea.Msg {
					return MemberToggledMsg{Err: client.AddMemberToCard(cardID, memberID)}
				}
			}
		case "esc":
			m.mode = cardView
		}
		return m, nil

	case cardAddLabel:
		switch msg.String() {
		case "j", "down":
			if m.labelIndex < len(m.boardLabels)-1 {
				m.labelIndex++
			}
		case "k", "up":
			if m.labelIndex > 0 {
				m.labelIndex--
			}
		case "enter", " ":
			if len(m.boardLabels) > 0 && m.labelIndex < len(m.boardLabels) {
				label := m.boardLabels[m.labelIndex]
				client := m.client
				cardID := m.card.ID
				labelID := label.ID
				for i, cl := range m.card.Labels {
					if cl.ID == labelID {
						m.card.Labels = append(m.card.Labels[:i], m.card.Labels[i+1:]...)
						return m, func() tea.Msg {
							return LabelToggledMsg{Err: client.RemoveLabelFromCard(cardID, labelID)}
						}
					}
				}
				m.card.Labels = append(m.card.Labels, label)
				return m, func() tea.Msg {
					return LabelToggledMsg{Err: client.AddLabelToCard(cardID, labelID)}
				}
			}
		case "esc":
			m.mode = cardView
		}
		return m, nil

	case cardSetDue:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.dueInput.Value())
			m.mode = cardView
			if val == "" {
				m.card.Due = ""
				m.card.DueComplete = false
				return m, m.updateCard(map[string]string{"due": ""})
			}
			t, err := time.Parse("2006-01-02", val)
			if err != nil {
				m.statusMsg = "Invalid date (use YYYY-MM-DD)"
				return m, nil
			}
			dueStr := t.UTC().Format(time.RFC3339)
			m.card.Due = dueStr
			return m, m.updateCard(map[string]string{"due": dueStr})
		case "esc":
			m.mode = cardView
		default:
			var cmd tea.Cmd
			m.dueInput, cmd = m.dueInput.Update(msg)
			return m, cmd
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
		case "a":
			m.mode = cardAddMember
			m.memberIndex = 0
			if len(m.boardMembers) == 0 {
				return m, m.fetchBoardMembers()
			}
		case "l":
			m.mode = cardAddLabel
			m.labelIndex = 0
			if len(m.boardLabels) == 0 {
				return m, m.fetchBoardLabels()
			}
		case "d":
			m.mode = cardSetDue
			currentVal := ""
			if m.card.Due != "" {
				for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z"} {
					if t, err := time.Parse(layout, m.card.Due); err == nil {
						currentVal = t.Format("2006-01-02")
						break
					}
				}
			}
			m.dueInput.SetValue(currentVal)
			m.dueInput.Focus()
			return m, textinput.Blink
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

func (m CardModel) fetchBoardMembers() tea.Cmd {
	client := m.client
	boardID := m.boardID
	return func() tea.Msg {
		members, err := client.GetBoardMembers(boardID)
		return BoardMembersFetchedMsg{Members: members, Err: err}
	}
}

func (m CardModel) fetchBoardLabels() tea.Cmd {
	client := m.client
	boardID := m.boardID
	return func() tea.Msg {
		labels, err := client.GetBoardLabels(boardID)
		return BoardLabelsFetchedMsg{Labels: labels, Err: err}
	}
}

func (m CardModel) isOnCard(memberID string) bool {
	for _, cm := range m.card.Members {
		if cm.ID == memberID {
			return true
		}
	}
	return false
}

func (m CardModel) isLabelOnCard(labelID string) bool {
	for _, cl := range m.card.Labels {
		if cl.ID == labelID {
			return true
		}
	}
	return false
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
	sectionTitle := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)

	switch m.mode {
	case cardEditDesc:
		b.WriteString("Description (esc to save):\n\n")
		b.WriteString(m.descEdit.View())
	case cardEditTitle:
		b.WriteString("Edit title (enter to save, esc to cancel):\n\n")
		b.WriteString(m.titleEdit.View())
	case cardMoveList:
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
	case cardAddMember:
		b.WriteString(sectionTitle.Render("Add / remove member"))
		b.WriteString("\n" + divider + "\n\n")
		if len(m.boardMembers) == 0 {
			b.WriteString(helpStyle.Render("Loading...") + "\n")
		} else {
			for i, member := range m.boardMembers {
				cursor := "  "
				style := lipgloss.NewStyle()
				if i == m.memberIndex {
					cursor = "▸ "
					style = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
				}
				check := "  "
				if m.isOnCard(member.ID) {
					check = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("✓ ")
				}
				b.WriteString(cursor + check + style.Render(member.FullName) + "\n")
			}
		}
		b.WriteString("\n" + helpStyle.Render("enter/space: toggle  esc: close"))
	case cardAddLabel:
		b.WriteString(sectionTitle.Render("Add / remove label"))
		b.WriteString("\n" + divider + "\n\n")
		if len(m.boardLabels) == 0 {
			b.WriteString(helpStyle.Render("Loading...") + "\n")
		} else {
			for i, label := range m.boardLabels {
				cursor := "  "
				rowStyle := lipgloss.NewStyle()
				if i == m.labelIndex {
					cursor = "▸ "
					rowStyle = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
				}
				check := "  "
				if m.isLabelOnCard(label.ID) {
					check = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("✓ ")
				}
				name := label.Name
				if name == "" {
					name = label.Color
				}
				colorDot := labelColor(label.Color).Render("●")
				b.WriteString(cursor + check + colorDot + " " + rowStyle.Render(name) + "\n")
			}
		}
		b.WriteString("\n" + helpStyle.Render("enter/space: toggle  esc: close"))
	case cardSetDue:
		b.WriteString(sectionTitle.Render("Set due date"))
		b.WriteString("\n" + divider + "\n\n")
		b.WriteString("Date (YYYY-MM-DD, empty to clear):\n\n")
		b.WriteString(m.dueInput.View())
		b.WriteString("\n\n" + helpStyle.Render("enter: save  esc: cancel"))
	default:
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
		b.WriteString(helpStyle.Render("t:title  E:desc  m:move  a:members  l:labels  d:due  ,/.:move left/right  </>:first/last  esc:back"))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
