package tui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	cardCreateLabel
	cardCreateLabelColor
	cardChecklistPane
	cardAttachmentsPane
	cardActivityPane
	cardAddComment
	cardAddChecklist
	cardAddCheckItem
	cardAddAttachment
	cardConfirmDeleteChecklist
	cardConfirmDeleteAttachment
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
	attachments  []trello.Attachment
	actions      []trello.Action
	mode         cardMode
	titleEdit    textinput.Model
	descEdit     textarea.Model
	dueInput     textinput.Model
	pickerFilter   textinput.Model
	labelNameInput textinput.Model
	labelColorIdx  int
	commentInput    textarea.Model
	checklistInput  textinput.Model
	moveIndex    int
	memberIndex  int
	labelIndex   int
	checkItemIdx  int
	attachmentIdx int
	activityIdx   int
	infoScroll    int
	clScroll      int
	attScroll     int
	actScroll     int
	width        int
	height       int
	statusMsg    string
	loadingCL    bool
	loadingAtt   bool
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

	ln := textinput.New()
	ln.Placeholder = "Label name..."
	ln.CharLimit = 100

	ci := textarea.New()
	ci.Placeholder = "Write a comment..."
	ci.SetWidth(60)
	ci.SetHeight(4)

	cli := textinput.New()
	cli.Placeholder = "Name..."
	cli.CharLimit = 200

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
		titleEdit:      ti,
		descEdit:       ta,
		dueInput:       di,
		pickerFilter:   pf,
		labelNameInput: ln,
		commentInput:    ci,
		checklistInput:  cli,
		loadingCL:    true,
		loadingAtt:   true,
		loadingCom:   true,
	}
}

func (m CardModel) Init() tea.Cmd {
	return tea.Batch(m.fetchChecklists(), m.fetchAttachments(), m.fetchActions())
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

	case LabelCreatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error creating label: %v", msg.Err)
			m.mode = cardAddLabel
			return m, nil
		}
		m.boardLabels = append(m.boardLabels, msg.Label)
		m.card.Labels = append(m.card.Labels, msg.Label)
		m.statusMsg = "Label created"
		m.mode = cardAddLabel
		m.pickerFilter.SetValue("")
		client := m.client
		cardID := m.card.ID
		labelID := msg.Label.ID
		return m, func() tea.Msg {
			return LabelToggledMsg{Err: client.AddLabelToCard(cardID, labelID)}
		}

	case ChecklistsFetchedMsg:
		m.loadingCL = false
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error fetching checklists: %v", msg.Err)
			return m, nil
		}
		m.checklists = msg.Checklists
		return m, nil

	case AttachmentsFetchedMsg:
		m.loadingAtt = false
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error fetching attachments: %v", msg.Err)
			return m, nil
		}
		m.attachments = msg.Attachments
		return m, nil

	case AttachmentOpenedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error opening attachment: %v", msg.Err)
		}
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

	case ChecklistCreatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
			return m, nil
		}
		m.checklists = append(m.checklists, msg.Checklist)
		m.mode = cardChecklistPane
		m.statusMsg = "Checklist created"
		return m, nil

	case CheckItemCreatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
			return m, nil
		}
		for i, cl := range m.checklists {
			if cl.ID == msg.ChecklistID {
				m.checklists[i].CheckItems = append(m.checklists[i].CheckItems, msg.CheckItem)
				break
			}
		}
		m.statusMsg = "Item added"
		return m, nil

	case AttachmentAddedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
			return m, nil
		}
		m.statusMsg = "Attachment added"
		return m, m.fetchAttachments()

	case ChecklistDeletedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
			return m, nil
		}
		for i, cl := range m.checklists {
			if cl.ID == msg.ChecklistID {
				m.checklists = append(m.checklists[:i], m.checklists[i+1:]...)
				break
			}
		}
		refs := m.allCheckItemRefs()
		if m.checkItemIdx >= len(refs) {
			m.checkItemIdx = len(refs) - 1
		}
		if m.checkItemIdx < 0 {
			m.checkItemIdx = 0
		}
		m.statusMsg = "Checklist deleted"
		if len(m.checklists) == 0 {
			m.mode = cardView
		}
		return m, nil

	case AttachmentDeletedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
			return m, nil
		}
		m.statusMsg = "Attachment deleted"
		return m, m.fetchAttachments()

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
	case cardCreateLabel:
		var cmd tea.Cmd
		m.labelNameInput, cmd = m.labelNameInput.Update(msg)
		return m, cmd
	case cardAddComment:
		var cmd tea.Cmd
		m.commentInput, cmd = m.commentInput.Update(msg)
		return m, cmd
	case cardAddChecklist, cardAddCheckItem, cardAddAttachment:
		var cmd tea.Cmd
		m.checklistInput, cmd = m.checklistInput.Update(msg)
		return m, cmd
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

func (m CardModel) fetchAttachments() tea.Cmd {
	client := m.client
	cardID := m.card.ID
	return func() tea.Msg {
		att, err := client.GetAttachments(cardID)
		return AttachmentsFetchedMsg{Attachments: att, Err: err}
	}
}

func (m CardModel) openAttachment(att trello.Attachment) tea.Cmd {
	client := m.client
	cardID := m.card.ID
	return func() tea.Msg {
		path, err := client.DownloadAttachment(cardID, att)
		if err != nil {
			return AttachmentOpenedMsg{Err: err}
		}
		err = exec.Command("open", path).Start()
		return AttachmentOpenedMsg{Err: err}
	}
}

func (m CardModel) deleteChecklist(checklistID string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		err := client.DeleteChecklist(checklistID)
		return ChecklistDeletedMsg{ChecklistID: checklistID, Err: err}
	}
}

func (m CardModel) deleteAttachment(attachmentID string) tea.Cmd {
	client := m.client
	cardID := m.card.ID
	return func() tea.Msg {
		err := client.DeleteAttachment(cardID, attachmentID)
		return AttachmentDeletedMsg{AttachmentID: attachmentID, Err: err}
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
