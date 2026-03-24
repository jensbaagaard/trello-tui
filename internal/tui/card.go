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
	cardChecklistPane
	cardActivityPane
	cardAddComment
)

type checkRef struct{ cl, it int }

type CardModel struct {
	client       *trello.Client
	card         trello.Card
	lists        []trello.List
	listIndex    int
	listName     string
	boardID      string
	boardMembers []trello.Member
	boardLabels  []trello.Label
	checklists   []trello.Checklist
	actions      []trello.Action
	mode         cardMode
	titleEdit    textinput.Model
	descEdit     textarea.Model
	dueInput     textinput.Model
	pickerFilter textinput.Model
	commentInput textarea.Model
	moveIndex    int
	memberIndex  int
	labelIndex   int
	checkItemIdx int
	activityIdx  int
	infoScroll   int
	clScroll     int
	actScroll    int
	width        int
	height       int
	statusMsg    string
	loadingCL    bool
	loadingCom   bool
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

	pf := textinput.New()
	pf.Placeholder = "type to filter..."
	pf.CharLimit = 50

	ci := textarea.New()
	ci.Placeholder = "Write a comment..."
	ci.SetWidth(60)
	ci.SetHeight(4)

	listName := ""
	if listIndex >= 0 && listIndex < len(lists) {
		listName = lists[listIndex].Name
	}
	boardID := ""
	if len(lists) > 0 {
		boardID = lists[0].IDBoard
	}

	return CardModel{
		client:       client,
		card:         card,
		lists:        lists,
		listIndex:    listIndex,
		listName:     listName,
		boardID:      boardID,
		moveIndex:    listIndex,
		titleEdit:    ti,
		descEdit:     ta,
		dueInput:     di,
		pickerFilter: pf,
		commentInput: ci,
		loadingCL:    true,
		loadingCom:   true,
	}
}

func (m CardModel) Init() tea.Cmd {
	return tea.Batch(m.fetchChecklists(), m.fetchActions())
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m CardModel) Update(msg tea.Msg) (CardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.descEdit.SetWidth(msg.Width - 10)
		m.commentInput.SetWidth(msg.Width - 10)

	case CardUpdatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
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
		m.statusMsg = "Saved"
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

	case ChecklistsFetchedMsg:
		m.loadingCL = false
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error fetching checklists: %v", msg.Err)
			return m, nil
		}
		m.checklists = msg.Checklists
		return m, nil

	case ActionsFetchedMsg:
		m.loadingCom = false
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error fetching activity: %v", msg.Err)
			return m, nil
		}
		m.actions = msg.Actions
		return m, nil

	case CheckItemToggledMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
		}
		return m, nil

	case CommentAddedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
			return m, nil
		}
		m.actions = append([]trello.Action{msg.Action}, m.actions...)
		m.statusMsg = "Comment added"
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward ticks to active input
	switch m.mode {
	case cardEditTitle:
		var cmd tea.Cmd
		m.titleEdit, cmd = m.titleEdit.Update(msg)
		return m, cmd
	case cardEditDesc:
		var cmd tea.Cmd
		m.descEdit, cmd = m.descEdit.Update(msg)
		return m, cmd
	case cardSetDue:
		var cmd tea.Cmd
		m.dueInput, cmd = m.dueInput.Update(msg)
		return m, cmd
	case cardAddMember, cardAddLabel:
		var cmd tea.Cmd
		m.pickerFilter, cmd = m.pickerFilter.Update(msg)
		return m, cmd
	case cardAddComment:
		var cmd tea.Cmd
		m.commentInput, cmd = m.commentInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

// ── Key handling ──────────────────────────────────────────────────────────────

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
			if m.memberIndex < len(m.filteredMembers())-1 {
				m.memberIndex++
			}
		case "k", "up":
			if m.memberIndex > 0 {
				m.memberIndex--
			}
		case "enter", " ":
			filtered := m.filteredMembers()
			if len(filtered) > 0 && m.memberIndex < len(filtered) {
				member := filtered[m.memberIndex]
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
			m.pickerFilter.SetValue("")
		default:
			prev := m.pickerFilter.Value()
			var cmd tea.Cmd
			m.pickerFilter, cmd = m.pickerFilter.Update(msg)
			if m.pickerFilter.Value() != prev {
				m.memberIndex = 0
			}
			return m, cmd
		}
		return m, nil

	case cardAddLabel:
		switch msg.String() {
		case "j", "down":
			if m.labelIndex < len(m.filteredLabels())-1 {
				m.labelIndex++
			}
		case "k", "up":
			if m.labelIndex > 0 {
				m.labelIndex--
			}
		case "enter", " ":
			filtered := m.filteredLabels()
			if len(filtered) > 0 && m.labelIndex < len(filtered) {
				label := filtered[m.labelIndex]
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
			m.pickerFilter.SetValue("")
		default:
			prev := m.pickerFilter.Value()
			var cmd tea.Cmd
			m.pickerFilter, cmd = m.pickerFilter.Update(msg)
			if m.pickerFilter.Value() != prev {
				m.labelIndex = 0
			}
			return m, cmd
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

	case cardChecklistPane:
		refs := m.allCheckItemRefs()
		switch msg.String() {
		case "j", "down":
			if m.checkItemIdx < len(refs)-1 {
				m.checkItemIdx++
			}
		case "k", "up":
			if m.checkItemIdx > 0 {
				m.checkItemIdx--
			}
		case "enter", " ":
			if len(refs) > 0 && m.checkItemIdx < len(refs) {
				r := refs[m.checkItemIdx]
				item := m.checklists[r.cl].CheckItems[r.it]
				newComplete := item.State != "complete"
				if newComplete {
					m.checklists[r.cl].CheckItems[r.it].State = "complete"
				} else {
					m.checklists[r.cl].CheckItems[r.it].State = "incomplete"
				}
				client := m.client
				cardID := m.card.ID
				checkItemID := item.ID
				return m, func() tea.Msg {
					return CheckItemToggledMsg{Err: client.ToggleCheckItem(cardID, checkItemID, newComplete)}
				}
			}
		case "tab":
			m.clScroll = 0
			m.mode = cardActivityPane
		case "esc":
			m.clScroll = 0
			m.mode = cardView
		}
		return m, nil

	case cardActivityPane:
		switch msg.String() {
		case "j", "down":
			if m.activityIdx < len(m.actions)-1 {
				m.activityIdx++
			}
		case "k", "up":
			if m.activityIdx > 0 {
				m.activityIdx--
			}
		case "n":
			m.mode = cardAddComment
			m.commentInput.SetValue("")
			m.commentInput.Focus()
			return m, textarea.Blink
		case "tab":
			m.actScroll = 0
			m.mode = cardView
		case "esc":
			m.actScroll = 0
			m.mode = cardView
		}
		return m, nil

	case cardAddComment:
		switch msg.String() {
		case "ctrl+s":
			text := strings.TrimSpace(m.commentInput.Value())
			if text == "" {
				m.mode = cardActivityPane
				return m, nil
			}
			m.mode = cardActivityPane
			client := m.client
			cardID := m.card.ID
			return m, func() tea.Msg {
				action, err := client.AddComment(cardID, text)
				return CommentAddedMsg{Action: action, Err: err}
			}
		case "esc":
			m.mode = cardActivityPane
		default:
			var cmd tea.Cmd
			m.commentInput, cmd = m.commentInput.Update(msg)
			return m, cmd
		}
		return m, nil

	default: // cardView — info pane active
		switch msg.String() {
		case "j", "down":
			m.infoScroll++
		case "k", "up":
			if m.infoScroll > 0 {
				m.infoScroll--
			}
		case "tab":
			m.infoScroll = 0
			if len(m.checklists) > 0 {
				m.mode = cardChecklistPane
			} else {
				m.mode = cardActivityPane
			}
		case "t":
			m.mode = cardEditTitle
			m.titleEdit.SetValue(m.card.Name)
			m.titleEdit.Focus()
			return m, textinput.Blink
		case "e":
			m.mode = cardEditDesc
			m.descEdit.SetValue(m.card.Desc)
			// size to fill the info pane
			available := m.height
			if available < 24 {
				available = 24
			}
			showChecklist := m.loadingCL || len(m.checklists) > 0
			var pane1H int
			if showChecklist {
				pane1H = available * 40 / 100
			} else {
				pane1H = available * 55 / 100
			}
			if pane1H < 9 {
				pane1H = 9
			}
			innerW := m.width - 2 - 4 // outer padding(2) + pane borders+padding(4)
			if innerW < 20 {
				innerW = 20
			}
			innerH := pane1H - 5 // borders(2) + title+blank(2) + hint(1)
			if innerH < 3 {
				innerH = 3
			}
			m.descEdit.SetWidth(innerW)
			m.descEdit.SetHeight(innerH)
			m.descEdit.Focus()
			return m, textarea.Blink
		case "m":
			m.mode = cardMoveList
			m.moveIndex = m.listIndex
		case "a":
			m.mode = cardAddMember
			m.memberIndex = 0
			m.pickerFilter.SetValue("")
			m.pickerFilter.Focus()
			if len(m.boardMembers) == 0 {
				return m, tea.Batch(textinput.Blink, m.fetchBoardMembers())
			}
			return m, textinput.Blink
		case "l":
			m.mode = cardAddLabel
			m.labelIndex = 0
			m.pickerFilter.SetValue("")
			m.pickerFilter.Focus()
			if len(m.boardLabels) == 0 {
				return m, tea.Batch(textinput.Blink, m.fetchBoardLabels())
			}
			return m, textinput.Blink
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
		}
	}
	return m, nil
}

// ── Commands ──────────────────────────────────────────────────────────────────

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

func (m CardModel) fetchChecklists() tea.Cmd {
	client := m.client
	cardID := m.card.ID
	return func() tea.Msg {
		cl, err := client.GetChecklists(cardID)
		return ChecklistsFetchedMsg{Checklists: cl, Err: err}
	}
}

func (m CardModel) fetchActions() tea.Cmd {
	client := m.client
	cardID := m.card.ID
	return func() tea.Msg {
		actions, err := client.GetActions(cardID)
		return ActionsFetchedMsg{Actions: actions, Err: err}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (m CardModel) allCheckItemRefs() []checkRef {
	var refs []checkRef
	for ci, cl := range m.checklists {
		for ii := range cl.CheckItems {
			refs = append(refs, checkRef{ci, ii})
		}
	}
	return refs
}

func (m CardModel) filteredMembers() []trello.Member {
	q := strings.ToLower(m.pickerFilter.Value())
	if q == "" {
		return m.boardMembers
	}
	var result []trello.Member
	for _, member := range m.boardMembers {
		if strings.Contains(strings.ToLower(member.FullName), q) ||
			strings.Contains(strings.ToLower(member.Username), q) {
			result = append(result, member)
		}
	}
	return result
}

func (m CardModel) filteredLabels() []trello.Label {
	q := strings.ToLower(m.pickerFilter.Value())
	if q == "" {
		return m.boardLabels
	}
	var result []trello.Label
	for _, label := range m.boardLabels {
		if strings.Contains(strings.ToLower(label.Name), q) ||
			strings.Contains(strings.ToLower(label.Color), q) {
			result = append(result, label)
		}
	}
	return result
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

// ── View ──────────────────────────────────────────────────────────────────────

func (m CardModel) View() string {
	w := m.width - 2
	if w < 44 {
		w = 44
	}

	available := m.height - 3
	if available < 24 {
		available = 24
	}

	var statusBar string
	if m.statusMsg != "" {
		statusBar = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render(m.statusMsg)
	} else {
		statusBar = m.helpLine()
	}

	showChecklist := m.loadingCL || len(m.checklists) > 0

	// Layout: box1 \n [box2 \n] box3 \n statusBar
	// Overhead: 3 newlines + 1 status line = 4 rows with checklist, 3 rows without
	var pane1H, pane2H, pane3H int
	if showChecklist {
		pane1H = available * 40 / 100
		pane2H = available * 30 / 100
		pane3H = available - pane1H - pane2H - 4
		if pane1H < 9 {
			pane1H = 9
		}
		if pane2H < 6 {
			pane2H = 6
		}
		if pane3H < 6 {
			pane3H = 6
		}
	} else {
		pane1H = available * 55 / 100
		pane3H = available - pane1H - 3
		if pane1H < 9 {
			pane1H = 9
		}
		if pane3H < 6 {
			pane3H = 6
		}
	}

	box1 := m.renderInfoPane(w, pane1H, m.infoScroll)
	var box2 string
	if showChecklist {
		box2 = m.renderChecklistPane(w, pane2H, m.checkItemIdx)
	}
	box3 := m.renderActivityPane(w, pane3H, m.activityIdx)

	body := box1
	if box2 != "" {
		body += "\n" + box2
	}
	body += "\n" + box3 + "\n" + statusBar
	return lipgloss.NewStyle().Padding(0, 1).Render(body)
}

func (m CardModel) renderInfoPane(width, height, scroll int) string {
	active := m.mode != cardChecklistPane && m.mode != cardActivityPane && m.mode != cardAddComment

	var b strings.Builder

	switch m.mode {
	case cardEditTitle:
		b.WriteString("Edit title (enter:save  esc:cancel):\n\n")
		b.WriteString(m.titleEdit.View())

	case cardEditDesc:
		b.WriteString("Description (esc:save):\n\n")
		b.WriteString(m.descEdit.View())

	case cardMoveList:
		sT := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		b.WriteString(sT.Render("Move to list") + "\n\n")
		for i, l := range m.lists {
			cursor := "  "
			s := lipgloss.NewStyle()
			if i == m.moveIndex {
				cursor = "▸ "
				s = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
			}
			suffix := ""
			if i == m.listIndex {
				suffix = helpStyle.Render("  (current)")
			}
			b.WriteString(cursor + s.Render(l.Name) + suffix + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("j/k:navigate  enter:move  esc:cancel"))

	case cardAddMember:
		sT := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		b.WriteString(sT.Render("Add / remove member") + "\n")
		b.WriteString(m.pickerFilter.View() + "\n\n")
		if len(m.boardMembers) == 0 {
			b.WriteString(helpStyle.Render("Loading..."))
		} else {
			filtered := m.filteredMembers()
			if len(filtered) == 0 {
				b.WriteString(helpStyle.Render("No matches"))
			} else {
				for i, member := range filtered {
					cursor := "  "
					s := lipgloss.NewStyle()
					if i == m.memberIndex {
						cursor = "▸ "
						s = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
					}
					check := "  "
					if m.isOnCard(member.ID) {
						check = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("✓ ")
					}
					b.WriteString(cursor + check + s.Render(member.FullName) + "\n")
				}
			}
		}
		b.WriteString("\n" + helpStyle.Render("enter/space:toggle  esc:close"))

	case cardAddLabel:
		sT := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		b.WriteString(sT.Render("Add / remove label") + "\n")
		b.WriteString(m.pickerFilter.View() + "\n\n")
		if len(m.boardLabels) == 0 {
			b.WriteString(helpStyle.Render("Loading..."))
		} else {
			filtered := m.filteredLabels()
			if len(filtered) == 0 {
				b.WriteString(helpStyle.Render("No matches"))
			} else {
				for i, label := range filtered {
					cursor := "  "
					rs := lipgloss.NewStyle()
					if i == m.labelIndex {
						cursor = "▸ "
						rs = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
					}
					check := "  "
					if m.isLabelOnCard(label.ID) {
						check = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("✓ ")
					}
					name := label.Name
					if name == "" {
						name = label.Color
					}
					b.WriteString(cursor + check + labelColor(label.Color).Render("● ") + rs.Render(name) + "\n")
				}
			}
		}
		b.WriteString("\n" + helpStyle.Render("enter/space:toggle  esc:close"))

	case cardSetDue:
		sT := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		b.WriteString(sT.Render("Set due date") + "\n\n")
		b.WriteString("Date (YYYY-MM-DD, empty to clear):\n\n")
		b.WriteString(m.dueInput.View())
		b.WriteString("\n\n" + helpStyle.Render("enter:save  esc:cancel"))

	default:
		// breadcrumb
		b.WriteString(helpStyle.Render("in "+m.listName) + "\n\n")

		// title
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).
			Render(m.card.Name) + "\n")

		// labels
		if len(m.card.Labels) > 0 {
			var parts []string
			for _, l := range m.card.Labels {
				name := l.Name
				if name == "" {
					name = l.Color
				}
				parts = append(parts, labelColor(l.Color).Render(name))
			}
			b.WriteString("\n" + strings.Join(parts, "  ") + "\n")
		}

		// due date
		if m.card.Due != "" {
			if label, s := formatDue(m.card.Due, m.card.DueComplete); label != "" {
				b.WriteString("\n" + s.Render("Due: "+label) + "\n")
			}
		}

		// members
		if len(m.card.Members) > 0 {
			var names []string
			for _, mem := range m.card.Members {
				names = append(names, mem.FullName)
			}
			b.WriteString("\n" + helpStyle.Render("Members: "+strings.Join(names, ", ")) + "\n")
		}

		// url
		if m.card.ShortURL != "" {
			b.WriteString("\n" + helpStyle.Render(m.card.ShortURL) + "\n")
		}

		b.WriteString("\n")

		// description
		desc := m.card.Desc
		if desc == "" {
			desc = helpStyle.Render("(no description)")
		}
		b.WriteString(desc)
	}

	return paneBox("Card Info", b.String(), width, height, active, scroll)
}

func (m CardModel) renderChecklistPane(width, height, cursorIdx int) string {
	active := m.mode == cardChecklistPane
	var b strings.Builder

	if m.loadingCL {
		b.WriteString(helpStyle.Render("Loading..."))
	} else if len(m.checklists) == 0 {
		b.WriteString(helpStyle.Render("(no checklists)"))
	} else {
		refs := m.allCheckItemRefs()
		flatIdx := 0
		for _, cl := range m.checklists {
			// checklist header with progress
			done := 0
			for _, it := range cl.CheckItems {
				if it.State == "complete" {
					done++
				}
			}
			total := len(cl.CheckItems)
			clTitle := lipgloss.NewStyle().Bold(true).Render(
				fmt.Sprintf("%s (%d/%d)", cl.Name, done, total),
			)
			b.WriteString(clTitle + "\n")

			for _, item := range cl.CheckItems {
				cursor := "  "
				itemStyle := lipgloss.NewStyle()
				if active && flatIdx < len(refs) && m.checkItemIdx == flatIdx {
					cursor = "▸ "
					itemStyle = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
				}
				box := "[ ]"
				if item.State == "complete" {
					box = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("[x]")
					itemStyle = itemStyle.Foreground(dimColor)
				}
				b.WriteString(cursor + box + " " + itemStyle.Render(item.Name) + "\n")
				flatIdx++
			}
		}
	}

	title := "Checklist"
	if active {
		title += helpStyle.Render("  j/k:navigate  enter:toggle  tab:next  esc:back")
	}
	innerW := width - 4
	if innerW < 10 {
		innerW = 10
	}
	availLines := (height - 2) - 2
	if availLines < 1 {
		availLines = 1
	}
	scroll := clampScroll(cursorIdx, availLines)
	return paneBox(title, b.String(), width, height, active, scroll)
}

func (m CardModel) renderActivityPane(width, height, cursorIdx int) string {
	active := m.mode == cardActivityPane || m.mode == cardAddComment
	var b strings.Builder

	if m.mode == cardAddComment {
		b.WriteString(helpStyle.Render("New comment (ctrl+s:send  esc:cancel)") + "\n\n")
		b.WriteString(m.commentInput.View())
	} else if m.loadingCom {
		b.WriteString(helpStyle.Render("Loading..."))
	} else if len(m.actions) == 0 {
		b.WriteString(helpStyle.Render("(no activity)"))
	} else {
		for i, a := range m.actions {
			cursor := "  "
			if active && i == m.activityIdx {
				cursor = "▸ "
			}

			dateStr := ""
			for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z"} {
				if t, err := time.Parse(layout, a.Date); err == nil {
					dateStr = t.Format("2 Jan")
					break
				}
			}

			author := a.MemberCreator.FullName
			if author == "" {
				author = a.MemberCreator.Username
			}

			switch a.Type {
			case "commentCard":
				header := lipgloss.NewStyle().Bold(true).Render(author)
				if dateStr != "" {
					header += helpStyle.Render(" • " + dateStr)
				}
				b.WriteString(cursor + header + "\n")
				b.WriteString("  " + a.Data.Text + "\n\n")

			case "updateCard":
				var line string
				if a.Data.ListBefore != nil && a.Data.ListAfter != nil {
					line = fmt.Sprintf("%s moved this card from %s to %s", author, a.Data.ListBefore.Name, a.Data.ListAfter.Name)
				} else {
					line = fmt.Sprintf("%s updated this card", author)
				}
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")

			case "createCard":
				listName := ""
				if a.Data.List != nil {
					listName = a.Data.List.Name
				}
				line := fmt.Sprintf("%s added this card to %s", author, listName)
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")

			case "addMemberToCard":
				memberName := ""
				if a.Data.Member != nil {
					memberName = a.Data.Member.FullName
				}
				line := fmt.Sprintf("%s added %s to this card", author, memberName)
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")

			case "removeMemberFromCard":
				memberName := ""
				if a.Data.Member != nil {
					memberName = a.Data.Member.FullName
				}
				line := fmt.Sprintf("%s removed %s from this card", author, memberName)
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")

			case "addAttachmentToCard":
				attachName := ""
				if a.Data.Attachment != nil {
					attachName = a.Data.Attachment.Name
				}
				line := fmt.Sprintf("%s attached %s", author, attachName)
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")

			default:
				line := fmt.Sprintf("%s performed an action", author)
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")
			}
		}
	}

	title := "Activity"
	if active && m.mode != cardAddComment {
		title += helpStyle.Render("  j/k:scroll  n:comment  tab:next  esc:back")
	}
	availLines := (height - 2) - 2
	if availLines < 1 {
		availLines = 1
	}
	// each activity item is ~2-3 lines; scroll to keep cursor visible
	scroll := clampScroll(cursorIdx*3, availLines)
	return paneBox(title, b.String(), width, height, active, scroll)
}

// clampScroll returns a scroll offset that ensures the cursor line is visible.
func clampScroll(cursorLine, visibleLines int) int {
	if cursorLine < visibleLines {
		return 0
	}
	return cursorLine - visibleLines + 1
}

func (m CardModel) helpLine() string {
	switch m.mode {
	case cardChecklistPane:
		return ""
	case cardActivityPane:
		return ""
	case cardAddComment:
		return ""
	default:
		return helpStyle.Render("t:title  e:desc  m:move  a:members  l:labels  d:due  ,/.:move lr  tab:next pane  esc:back")
	}
}

// ── paneBox ───────────────────────────────────────────────────────────────────

func paneBox(title, content string, width, height int, active bool, scroll int) string {
	borderColor := dimColor
	if active {
		borderColor = primaryColor
	}
	titleColor := secondaryColor
	if !active {
		titleColor = dimColor
	}
	innerW := width - 4 // border(1 each side) + padding(1 each side)
	if innerW < 10 {
		innerW = 10
	}
	innerH := height - 2 // subtract top+bottom borders
	// title takes 1 line + 1 blank line = 2 lines
	availLines := innerH - 2
	if availLines < 1 {
		availLines = 1
	}
	content = scrollLines(content, scroll, availLines, innerW)
	titleLine := lipgloss.NewStyle().Bold(true).Foreground(titleColor).Render(title)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(innerW).
		Height(innerH).
		Render(titleLine + "\n\n" + content)
}

// scrollLines skips the first `offset` visual rows then returns up to maxLines rows,
// accounting for long lines that wrap at wrapWidth characters.
func scrollLines(content string, offset, maxLines, wrapWidth int) string {
	lines := strings.Split(content, "\n")
	// expand each logical line into visual rows
	type row struct{ text string }
	var rows []row
	for _, line := range lines {
		vw := lipgloss.Width(line)
		if wrapWidth > 0 && vw > wrapWidth {
			// approximate: split into chunks by rune count
			runes := []rune(line)
			for len(runes) > 0 {
				end := wrapWidth
				if end > len(runes) {
					end = len(runes)
				}
				rows = append(rows, row{string(runes[:end])})
				runes = runes[end:]
			}
		} else {
			rows = append(rows, row{line})
		}
	}
	if offset >= len(rows) {
		offset = len(rows) - 1
	}
	if offset < 0 {
		offset = 0
	}
	visible := rows[offset:]
	if len(visible) > maxLines {
		visible = visible[:maxLines]
	}
	out := make([]string, len(visible))
	for i, r := range visible {
		out[i] = r.text
	}
	return strings.Join(out, "\n")
}
