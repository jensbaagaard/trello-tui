package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jensbaagaard/trello-tui/internal/trello"
)

const minColWidth = 36

type boardMode int

const (
	boardNav boardMode = iota
	boardAddCard
	boardConfirmArchive
	boardFilter
	boardLabelManager
	boardLabelCreate
	boardLabelEdit
	boardLabelColorPick
	boardLabelConfirmDelete
	boardArchive
	boardArchiveFilter
)

type BoardModel struct {
	client      *trello.Client
	board       trello.Board
	lists       []trello.List
	cardsByList map[string][]trello.Card
	activeList  int
	activeCard  int
	scrollTop   int // per-column vertical scroll offset for the active list
	colOffset   int // first visible column index (horizontal scroll)
	width       int
	height      int
	loading     bool
	err         error
	mode        boardMode
	textInput   textinput.Model
	statusMsg   string
	filterText  string

	// Label manager
	boardLabels    []trello.Label
	labelCursor    int
	labelNameInput textinput.Model
	labelColorIdx  int
	editingLabelID string

	// Archive viewer
	archivedCards      []trello.Card
	archiveCursor      int
	archiveScrollTop   int
	archiveFilterText  string
}

func NewBoardModel(client *trello.Client, board trello.Board) BoardModel {
	ti := textinput.New()
	ti.Placeholder = "Card title..."
	ti.CharLimit = 200

	li := textinput.New()
	li.Placeholder = "Label name..."
	li.CharLimit = 100

	return BoardModel{
		client:         client,
		board:          board,
		cardsByList:    make(map[string][]trello.Card),
		loading:        true,
		textInput:      ti,
		labelNameInput: li,
	}
}

func (m BoardModel) Init() tea.Cmd {
	return m.fetchLists()
}

// ── Commands ──────────────────────────────────────────────────────────────────

func (m BoardModel) fetchLists() tea.Cmd {
	client := m.client
	boardID := m.board.ID
	return func() tea.Msg {
		lists, err := client.GetLists(boardID)
		return ListsFetchedMsg{Lists: lists, Err: err}
	}
}

func (m BoardModel) fetchAllCards() tea.Cmd {
	client := m.client
	lists := m.lists
	return func() tea.Msg {
		cardsByList := make(map[string][]trello.Card)
		for _, l := range lists {
			cards, err := client.GetCards(l.ID)
			if err != nil {
				return AllCardsFetchedMsg{Err: err}
			}
			cardsByList[l.ID] = cards
		}
		return AllCardsFetchedMsg{CardsByList: cardsByList}
	}
}

func (m BoardModel) createCard(name string) tea.Cmd {
	client := m.client
	listID := m.currentListID()
	return func() tea.Msg {
		card, err := client.CreateCard(listID, name)
		return CardCreatedMsg{Card: card, Err: err}
	}
}

func (m BoardModel) archiveCard(cardID string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		err := client.ArchiveCard(cardID)
		return CardArchivedMsg{CardID: cardID, Err: err}
	}
}

func (m BoardModel) fetchBoardLabels() tea.Cmd {
	client := m.client
	boardID := m.board.ID
	return func() tea.Msg {
		labels, err := client.GetBoardLabels(boardID)
		return BoardLabelsFetchedMsg{Labels: labels, Err: err}
	}
}

func (m BoardModel) createLabel(name, color string) tea.Cmd {
	client := m.client
	boardID := m.board.ID
	return func() tea.Msg {
		label, err := client.CreateLabel(boardID, name, color)
		return LabelCreatedMsg{Label: label, Err: err}
	}
}

func (m BoardModel) updateLabel(labelID, name, color string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		label, err := client.UpdateLabel(labelID, name, color)
		return LabelUpdatedMsg{Label: label, Err: err}
	}
}

func (m BoardModel) deleteLabel(labelID string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		err := client.DeleteLabel(labelID)
		return LabelDeletedMsg{LabelID: labelID, Err: err}
	}
}

func (m BoardModel) fetchArchivedCards() tea.Cmd {
	client := m.client
	boardID := m.board.ID
	return func() tea.Msg {
		cards, err := client.GetArchivedCards(boardID)
		return ArchivedCardsFetchedMsg{Cards: cards, Err: err}
	}
}

func (m BoardModel) restoreCard(cardID string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		card, err := client.RestoreCard(cardID)
		return CardRestoredMsg{Card: card, Err: err}
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m BoardModel) Update(msg tea.Msg) (BoardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureListVisible()

	case ListsFetchedMsg:
		if msg.Err != nil {
			m.loading = false
			m.err = msg.Err
			return m, nil
		}
		m.lists = msg.Lists
		return m, m.fetchAllCards()

	case AllCardsFetchedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.cardsByList = msg.CardsByList
		return m, nil

	case CardCreatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error creating card: %v", msg.Err)
			return m, nil
		}
		listID := m.currentListID()
		m.cardsByList[listID] = append(m.cardsByList[listID], msg.Card)
		m.statusMsg = "Card created"
		return m, nil

	case CardArchivedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error archiving card: %v", msg.Err)
			return m, nil
		}
		listID := m.currentListID()
		cards := m.cardsByList[listID]
		for i, c := range cards {
			if c.ID == msg.CardID {
				m.cardsByList[listID] = append(cards[:i], cards[i+1:]...)
				break
			}
		}
		m.clampCardCursor()
		m.statusMsg = "Card archived"
		return m, nil

	case CardUpdatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error moving card: %v", msg.Err)
			return m, nil
		}
		m.statusMsg = "Card moved"
		return m, nil

	case BoardLabelsFetchedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error fetching labels: %v", msg.Err)
			m.mode = boardNav
			return m, nil
		}
		m.boardLabels = msg.Labels
		return m, nil

	case LabelCreatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error creating label: %v", msg.Err)
			return m, nil
		}
		m.boardLabels = append(m.boardLabels, msg.Label)
		m.labelCursor = len(m.boardLabels) - 1
		m.statusMsg = "Label created"
		m.mode = boardLabelManager
		return m, nil

	case LabelUpdatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error updating label: %v", msg.Err)
			return m, nil
		}
		for i, l := range m.boardLabels {
			if l.ID == msg.Label.ID {
				m.boardLabels[i] = msg.Label
				break
			}
		}
		m.updateLabelOnCards(msg.Label)
		m.statusMsg = "Label updated"
		m.mode = boardLabelManager
		return m, nil

	case LabelDeletedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error deleting label: %v", msg.Err)
			return m, nil
		}
		for i, l := range m.boardLabels {
			if l.ID == msg.LabelID {
				m.boardLabels = append(m.boardLabels[:i], m.boardLabels[i+1:]...)
				break
			}
		}
		m.removeLabelFromCards(msg.LabelID)
		if m.labelCursor >= len(m.boardLabels) && m.labelCursor > 0 {
			m.labelCursor--
		}
		m.statusMsg = "Label deleted"
		m.mode = boardLabelManager
		return m, nil

	case ArchivedCardsFetchedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error fetching archived cards: %v", msg.Err)
			m.mode = boardNav
			return m, nil
		}
		m.archivedCards = msg.Cards
		return m, nil

	case CardRestoredMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error restoring card: %v", msg.Err)
			return m, nil
		}
		for i, c := range m.archivedCards {
			if c.ID == msg.Card.ID {
				m.archivedCards = append(m.archivedCards[:i], m.archivedCards[i+1:]...)
				break
			}
		}
		if m.archiveCursor >= len(m.archivedCards) && m.archiveCursor > 0 {
			m.archiveCursor--
		}
		m.cardsByList[msg.Card.IDList] = append(m.cardsByList[msg.Card.IDList], msg.Card)
		listName := msg.Card.IDList
		for _, l := range m.lists {
			if l.ID == msg.Card.IDList {
				listName = l.Name
				break
			}
		}
		m.statusMsg = fmt.Sprintf("Card restored to %s", listName)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// ── Label state mutations ─────────────────────────────────────────────────────

func (m *BoardModel) updateLabelOnCards(label trello.Label) {
	for listID, cards := range m.cardsByList {
		for ci, card := range cards {
			for li, cl := range card.Labels {
				if cl.ID == label.ID {
					m.cardsByList[listID][ci].Labels[li] = label
				}
			}
		}
	}
}

func (m *BoardModel) removeLabelFromCards(labelID string) {
	for listID, cards := range m.cardsByList {
		for ci, card := range cards {
			for li, cl := range card.Labels {
				if cl.ID == labelID {
					m.cardsByList[listID][ci].Labels = append(card.Labels[:li], card.Labels[li+1:]...)
					break
				}
			}
		}
	}
}

// ── Navigation & dimension helpers ────────────────────────────────────────────

// visibleLists returns the lists to render. When a filter is active, lists with
// no matching cards are hidden.
func (m BoardModel) visibleLists() []trello.List {
	if m.filterText == "" {
		return m.lists
	}
	var result []trello.List
	for _, l := range m.lists {
		if len(m.filteredCards(l.ID)) > 0 {
			result = append(result, l)
		}
	}
	return result
}

// visibleColCount returns how many columns fit on screen at the fixed width.
func (m BoardModel) visibleColCount() int {
	vis := len(m.visibleLists())
	if m.width <= 0 || vis == 0 {
		return 1
	}
	n := m.width / minColWidth
	if n < 1 {
		n = 1
	}
	if n > vis {
		n = vis
	}
	return n
}

// colWidth returns the width to use per column, expanding to fill available space.
func (m BoardModel) colWidth() int {
	vis := m.visibleColCount()
	if m.width <= 0 {
		return minColWidth
	}
	w := m.width / vis
	if w < minColWidth {
		w = minColWidth
	}
	return w
}

// ensureListVisible adjusts colOffset so the active list stays in the visible window.
func (m *BoardModel) ensureListVisible() {
	vis := m.visibleColCount()
	if m.activeList >= m.colOffset+vis {
		m.colOffset = m.activeList - vis + 1
	}
	if m.activeList < m.colOffset {
		m.colOffset = m.activeList
	}
	max := len(m.visibleLists()) - vis
	if max < 0 {
		max = 0
	}
	if m.colOffset > max {
		m.colOffset = max
	}
	if m.colOffset < 0 {
		m.colOffset = 0
	}
}

// columnHeight returns the fixed inner height for column content (title + cards).
func (m BoardModel) columnHeight() int {
	h := m.height - 2
	if h < 6 {
		h = 6
	}
	return h
}

// cardBudget returns how many terminal rows are available for cards inside a column.
func (m BoardModel) cardBudget() int {
	b := m.columnHeight() - 2
	if b < 4 {
		b = 4
	}
	return b
}

func (m *BoardModel) ensureCardVisible() {
	cards := m.currentCards()
	if len(cards) == 0 {
		m.scrollTop = 0
		return
	}
	budget := m.cardBudget()
	cardInner := m.cardInnerWidth()

	if m.activeCard < m.scrollTop {
		m.scrollTop = m.activeCard
		return
	}

	used := 0
	for i := m.scrollTop; i < len(cards); i++ {
		h := measureCard(cards[i], cardInner)
		if used+h > budget && i > m.scrollTop {
			if m.activeCard >= i {
				m.scrollTop++
				m.ensureCardVisible()
			}
			return
		}
		used += h
		if i == m.activeCard {
			return
		}
	}
}

func (m BoardModel) cardInnerWidth() int {
	colW := m.colWidth()
	inner := colW - 6 // column padding (2) + card border (2) + card padding (2)
	if inner < 10 {
		inner = 10
	}
	return inner
}

func (m BoardModel) fullListIndex(listID string) int {
	for i, l := range m.lists {
		if l.ID == listID {
			return i
		}
	}
	return -1
}

func (m *BoardModel) clampActiveList() {
	vis := m.visibleLists()
	if len(vis) == 0 {
		m.activeList = 0
		return
	}
	if m.activeList >= len(vis) {
		m.activeList = len(vis) - 1
	}
	if m.activeList < 0 {
		m.activeList = 0
	}
	m.colOffset = 0
}

func (m *BoardModel) clampCardCursor() {
	cards := m.currentCards()
	if m.activeCard >= len(cards) {
		m.activeCard = len(cards) - 1
	}
	if m.activeCard < 0 {
		m.activeCard = 0
	}
}

// ── Filter helpers ────────────────────────────────────────────────────────────

func matchesFilter(c trello.Card, query string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(c.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(c.Desc), q) {
		return true
	}
	for _, m := range c.Members {
		if strings.Contains(strings.ToLower(m.FullName), q) || strings.Contains(strings.ToLower(m.Username), q) {
			return true
		}
	}
	for _, l := range c.Labels {
		if strings.Contains(strings.ToLower(l.Name), q) {
			return true
		}
	}
	if c.Due != "" {
		if t, err := time.Parse(time.RFC3339Nano, c.Due); err == nil {
			if strings.Contains(strings.ToLower(t.Format("2 Jan 2006")), q) {
				return true
			}
		}
	}
	return false
}

func (m BoardModel) filteredCards(listID string) []trello.Card {
	cards := m.cardsByList[listID]
	if m.filterText == "" {
		return cards
	}
	var result []trello.Card
	for _, c := range cards {
		if matchesFilter(c, m.filterText) {
			result = append(result, c)
		}
	}
	return result
}

func (m BoardModel) filteredArchivedCards() []trello.Card {
	if m.archiveFilterText == "" {
		return m.archivedCards
	}
	var result []trello.Card
	for _, c := range m.archivedCards {
		if matchesFilter(c, m.archiveFilterText) {
			result = append(result, c)
		}
	}
	return result
}

func (m *BoardModel) removeCardFromList(listID, cardID string) {
	cards := m.cardsByList[listID]
	for i, c := range cards {
		if c.ID == cardID {
			m.cardsByList[listID] = append(cards[:i], cards[i+1:]...)
			return
		}
	}
}

func (m BoardModel) currentListID() string {
	vis := m.visibleLists()
	if m.activeList >= 0 && m.activeList < len(vis) {
		return vis[m.activeList].ID
	}
	return ""
}

func (m BoardModel) currentCards() []trello.Card {
	return m.filteredCards(m.currentListID())
}

func (m BoardModel) selectedCard() *trello.Card {
	cards := m.currentCards()
	if m.activeCard >= 0 && m.activeCard < len(cards) {
		return &cards[m.activeCard]
	}
	return nil
}
