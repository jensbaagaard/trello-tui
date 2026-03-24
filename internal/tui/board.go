package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m BoardModel) handleKey(msg tea.KeyMsg) (BoardModel, tea.Cmd) {
	if m.mode == boardAddCard {
		return m.handleAddCardKey(msg)
	}
	if m.mode == boardConfirmArchive {
		return m.handleConfirmArchiveKey(msg)
	}
	if m.mode == boardFilter {
		return m.handleFilterKey(msg)
	}
	if m.mode == boardLabelManager || m.mode == boardLabelCreate || m.mode == boardLabelEdit ||
		m.mode == boardLabelColorPick || m.mode == boardLabelConfirmDelete {
		return m.handleLabelManagerKey(msg)
	}

	switch msg.String() {
	case "left":
		if m.activeList > 0 {
			m.activeList--
			m.scrollTop = 0
			m.clampCardCursor()
			m.ensureListVisible()
		}
	case "right":
		if m.activeList < len(m.visibleLists())-1 {
			m.activeList++
			m.scrollTop = 0
			m.clampCardCursor()
			m.ensureListVisible()
		}
	case "j", "down":
		cards := m.currentCards()
		if m.activeCard < len(cards)-1 {
			m.activeCard++
			m.ensureCardVisible()
		}
	case "k", "up":
		if m.activeCard > 0 {
			m.activeCard--
			m.ensureCardVisible()
		}
	case "n":
		m.mode = boardAddCard
		m.textInput.Placeholder = "Card title..."
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, textinput.Blink
	case "c":
		if m.selectedCard() != nil {
			m.mode = boardConfirmArchive
		}
	case ",":
		return m.moveCardLeft()
	case ".":
		return m.moveCardRight()
	case "<":
		return m.moveCardToFirst()
	case ">":
		return m.moveCardToLast()
	case "L":
		m.mode = boardLabelManager
		m.labelCursor = 0
		m.statusMsg = ""
		if len(m.boardLabels) == 0 {
			return m, m.fetchBoardLabels()
		}
		return m, nil
	case "/":
		m.mode = boardFilter
		m.textInput.Placeholder = "title, description, member, label..."
		m.textInput.SetValue(m.filterText)
		m.textInput.Focus()
		return m, textinput.Blink
	case "esc":
		if m.filterText != "" {
			m.filterText = ""
			m.clampCardCursor()
			m.scrollTop = 0
			return m, nil
		}
	case "r":
		m.loading = true
		m.statusMsg = ""
		return m, m.fetchLists()
	}

	return m, nil
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

func measureCard(c trello.Card, width int) int {
	rendered := renderCard(c, width, false)
	return lipgloss.Height(rendered)
}

func (m BoardModel) handleAddCardKey(msg tea.KeyMsg) (BoardModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.textInput.Value())
		if name == "" {
			m.mode = boardNav
			return m, nil
		}
		m.mode = boardNav
		return m, m.createCard(name)
	case "esc":
		m.mode = boardNav
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m BoardModel) handleConfirmArchiveKey(msg tea.KeyMsg) (BoardModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		card := m.selectedCard()
		if card != nil {
			m.mode = boardNav
			return m, m.archiveCard(card.ID)
		}
		m.mode = boardNav
	case "n", "N", "esc":
		m.mode = boardNav
	}
	return m, nil
}

func (m BoardModel) handleFilterKey(msg tea.KeyMsg) (BoardModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filterText = m.textInput.Value()
		m.mode = boardNav
		m.activeCard = 0
		m.scrollTop = 0
		m.clampActiveList()
		m.clampCardCursor()
		return m, nil
	case "esc":
		m.filterText = ""
		m.mode = boardNav
		m.activeCard = 0
		m.scrollTop = 0
		m.clampActiveList()
		m.clampCardCursor()
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.filterText = m.textInput.Value()
	m.activeCard = 0
	m.scrollTop = 0
	m.clampActiveList()
	m.clampCardCursor()
	return m, cmd
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

func (m BoardModel) handleLabelManagerKey(msg tea.KeyMsg) (BoardModel, tea.Cmd) {
	switch m.mode {
	case boardLabelManager:
		switch msg.String() {
		case "j", "down":
			if m.labelCursor < len(m.boardLabels)-1 {
				m.labelCursor++
			}
		case "k", "up":
			if m.labelCursor > 0 {
				m.labelCursor--
			}
		case "n":
			m.mode = boardLabelCreate
			m.labelNameInput.SetValue("")
			m.labelNameInput.Focus()
			m.editingLabelID = ""
			return m, textinput.Blink
		case "e":
			if len(m.boardLabels) > 0 && m.labelCursor < len(m.boardLabels) {
				label := m.boardLabels[m.labelCursor]
				m.mode = boardLabelEdit
				m.labelNameInput.SetValue(label.Name)
				m.labelNameInput.Focus()
				m.editingLabelID = label.ID
				for i, c := range TrelloColors {
					if c == label.Color {
						m.labelColorIdx = i
						break
					}
				}
				return m, textinput.Blink
			}
		case "d":
			if len(m.boardLabels) > 0 && m.labelCursor < len(m.boardLabels) {
				m.mode = boardLabelConfirmDelete
			}
		case "esc":
			m.mode = boardNav
			m.statusMsg = ""
		}
		return m, nil

	case boardLabelCreate, boardLabelEdit:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.labelNameInput.Value())
			m.mode = boardLabelColorPick
			m.labelNameInput.Blur()
			if m.editingLabelID == "" {
				m.labelColorIdx = 0
			}
			_ = name // stored in labelNameInput, read when color is confirmed
			return m, nil
		case "esc":
			m.mode = boardLabelManager
			return m, nil
		default:
			var cmd tea.Cmd
			m.labelNameInput, cmd = m.labelNameInput.Update(msg)
			return m, cmd
		}

	case boardLabelColorPick:
		switch msg.String() {
		case "j", "down":
			if m.labelColorIdx < len(TrelloColors)-1 {
				m.labelColorIdx++
			}
		case "k", "up":
			if m.labelColorIdx > 0 {
				m.labelColorIdx--
			}
		case "enter":
			name := strings.TrimSpace(m.labelNameInput.Value())
			color := TrelloColors[m.labelColorIdx]
			if m.editingLabelID != "" {
				return m, m.updateLabel(m.editingLabelID, name, color)
			}
			return m, m.createLabel(name, color)
		case "esc":
			if m.editingLabelID != "" {
				m.mode = boardLabelEdit
			} else {
				m.mode = boardLabelCreate
			}
			m.labelNameInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case boardLabelConfirmDelete:
		switch msg.String() {
		case "y", "Y":
			if m.labelCursor < len(m.boardLabels) {
				labelID := m.boardLabels[m.labelCursor].ID
				return m, m.deleteLabel(labelID)
			}
			m.mode = boardLabelManager
		case "n", "N", "esc":
			m.mode = boardLabelManager
		}
		return m, nil
	}
	return m, nil
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

func (m BoardModel) fullListIndex(listID string) int {
	for i, l := range m.lists {
		if l.ID == listID {
			return i
		}
	}
	return -1
}

func (m BoardModel) moveCardLeft() (BoardModel, tea.Cmd) {
	card := m.selectedCard()
	if card == nil {
		return m, nil
	}
	srcListID := m.currentListID()
	fullIdx := m.fullListIndex(srcListID)
	if fullIdx <= 0 {
		return m, nil
	}
	return m.doMoveCard(card, srcListID, fullIdx-1)
}

func (m BoardModel) moveCardRight() (BoardModel, tea.Cmd) {
	card := m.selectedCard()
	if card == nil {
		return m, nil
	}
	srcListID := m.currentListID()
	fullIdx := m.fullListIndex(srcListID)
	if fullIdx < 0 || fullIdx >= len(m.lists)-1 {
		return m, nil
	}
	return m.doMoveCard(card, srcListID, fullIdx+1)
}

func (m BoardModel) moveCardToFirst() (BoardModel, tea.Cmd) {
	card := m.selectedCard()
	if card == nil {
		return m, nil
	}
	srcListID := m.currentListID()
	fullIdx := m.fullListIndex(srcListID)
	if fullIdx <= 0 {
		return m, nil
	}
	return m.doMoveCard(card, srcListID, 0)
}

func (m BoardModel) moveCardToLast() (BoardModel, tea.Cmd) {
	card := m.selectedCard()
	if card == nil {
		return m, nil
	}
	srcListID := m.currentListID()
	fullIdx := m.fullListIndex(srcListID)
	last := len(m.lists) - 1
	if fullIdx < 0 || fullIdx >= last {
		return m, nil
	}
	return m.doMoveCard(card, srcListID, last)
}

func (m BoardModel) doMoveCard(card *trello.Card, srcListID string, targetFullIdx int) (BoardModel, tea.Cmd) {
	targetList := m.lists[targetFullIdx]
	m.removeCardFromList(srcListID, card.ID)
	m.cardsByList[targetList.ID] = append(m.cardsByList[targetList.ID], *card)

	for i, l := range m.visibleLists() {
		if l.ID == targetList.ID {
			m.activeList = i
			break
		}
	}
	m.activeCard = len(m.filteredCards(targetList.ID)) - 1
	m.scrollTop = 0
	m.ensureCardVisible()
	m.ensureListVisible()

	client := m.client
	cardID := card.ID
	targetListID := targetList.ID
	return m, func() tea.Msg {
		c, err := client.MoveCard(cardID, targetListID)
		return CardUpdatedMsg{Card: c, Err: err}
	}
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

func (m BoardModel) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}
	if m.loading {
		return fmt.Sprintf("Loading board: %s...", m.board.Name)
	}
	if len(m.lists) == 0 {
		return "No lists found on this board."
	}

	if m.mode == boardLabelManager || m.mode == boardLabelCreate || m.mode == boardLabelEdit ||
		m.mode == boardLabelColorPick || m.mode == boardLabelConfirmDelete {
		return m.renderLabelManager()
	}

	visLists := m.visibleLists()

	if m.filterText != "" && len(visLists) == 0 {
		header := titleStyle.Render(m.board.Name)
		noResults := helpStyle.Render(fmt.Sprintf("\nNo cards match \"%s\"", m.filterText))
		status := helpStyle.Render("esc:clear filter")
		return header + noResults + "\n\n" + status
	}

	colW := m.colWidth()
	vis := m.visibleColCount()
	start := m.colOffset
	end := start + vis
	if end > len(visLists) {
		end = len(visLists)
	}

	columns := make([]string, 0, vis)
	for i := start; i < end; i++ {
		columns = append(columns, m.renderColumn(i, visLists[i], colW))
	}

	board := lipgloss.JoinHorizontal(lipgloss.Top, columns...)

	var scrollHint string
	if start > 0 || end < len(visLists) {
		scrollHint = helpStyle.Render(fmt.Sprintf(" [%d-%d of %d lists]", start+1, end, len(visLists)))
	}

	var status string
	if m.mode == boardFilter {
		status = "Filter: " + m.textInput.View()
	} else if m.mode == boardAddCard {
		status = "New card: " + m.textInput.View()
	} else if m.mode == boardConfirmArchive {
		card := m.selectedCard()
		name := ""
		if card != nil {
			name = card.Name
		}
		status = errorStyle.Render(fmt.Sprintf("Archive \"%s\"? (y/n)", name))
	} else if m.statusMsg != "" {
		status = m.statusMsg
	} else if m.filterText != "" {
		status = helpStyle.Render(fmt.Sprintf("filter: %s  ←→:lists  j/k:cards  /:edit filter  esc:clear filter", m.filterText))
	} else {
		status = helpStyle.Render("←→:lists  j/k:cards  ,/.:move card  </>:move first/last  n:new  c:archive  L:labels  enter:open  /:filter  r:refresh  esc:back")
	}

	header := titleStyle.Render(m.board.Name) + scrollHint
	return header + "\n" + board + "\n" + status
}

func (m BoardModel) renderLabelManager() string {
	header := titleStyle.Render(m.board.Name+" — Labels") + "\n\n"
	var b strings.Builder

	switch m.mode {
	case boardLabelCreate, boardLabelEdit:
		action := "Create"
		if m.mode == boardLabelEdit {
			action = "Edit"
		}
		sT := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		b.WriteString(sT.Render(action+" label") + "\n\n")
		b.WriteString("Name: " + m.labelNameInput.View() + "\n\n")
		b.WriteString(helpStyle.Render("enter:pick color  esc:cancel"))

	case boardLabelColorPick:
		sT := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		name := strings.TrimSpace(m.labelNameInput.Value())
		if name == "" {
			name = "(unnamed)"
		}
		b.WriteString(sT.Render("Pick color for: "+name) + "\n\n")
		for i, c := range TrelloColors {
			cursor := "  "
			s := lipgloss.NewStyle()
			if i == m.labelColorIdx {
				cursor = "▸ "
				s = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
			}
			b.WriteString(cursor + labelColor(c).Render("● ") + s.Render(c) + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("j/k:navigate  enter:confirm  esc:back"))

	case boardLabelConfirmDelete:
		name := ""
		if m.labelCursor < len(m.boardLabels) {
			label := m.boardLabels[m.labelCursor]
			name = label.Name
			if name == "" {
				name = label.Color
			}
		}
		b.WriteString(errorStyle.Render(fmt.Sprintf("Delete label \"%s\"? This removes it from all cards. (y/n)", name)))

	default: // boardLabelManager
		if len(m.boardLabels) == 0 {
			b.WriteString(helpStyle.Render("No labels on this board."))
		} else {
			for i, label := range m.boardLabels {
				cursor := "  "
				s := lipgloss.NewStyle()
				if i == m.labelCursor {
					cursor = "▸ "
					s = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
				}
				name := label.Name
				if name == "" {
					name = "(unnamed)"
				}
				b.WriteString(cursor + labelColor(label.Color).Render("● ") + s.Render(name) + helpStyle.Render("  ["+label.Color+"]") + "\n")
			}
		}
		b.WriteString("\n")
		if m.statusMsg != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render(m.statusMsg) + "\n")
		}
		b.WriteString(helpStyle.Render("j/k:navigate  n:new  e:edit  d:delete  esc:back"))
	}

	return header + b.String()
}

func (m BoardModel) renderColumn(idx int, l trello.List, width int) string {
	isActive := idx == m.activeList
	allCards := m.cardsByList[l.ID]
	cards := m.filteredCards(l.ID)
	budget := m.cardBudget()
	colH := m.columnHeight()
	innerWidth := width - 2
	cardInner := m.cardInnerWidth()

	var titleText string
	if m.filterText != "" {
		titleText = fmt.Sprintf("%s (%d/%d)", l.Name, len(cards), len(allCards))
	} else {
		titleText = fmt.Sprintf("%s (%d)", l.Name, len(cards))
	}
	title := columnTitleStyle.Width(innerWidth).Render(titleText)

	scrollTop := 0
	if isActive {
		scrollTop = m.scrollTop
	}

	var cardViews []string
	used := 0

	if scrollTop > 0 {
		indicator := dimRender(fmt.Sprintf("  ↑ %d more", scrollTop))
		used += lipgloss.Height(indicator)
		cardViews = append(cardViews, indicator)
	}

	lastRendered := scrollTop
	for i := scrollTop; i < len(cards); i++ {
		selected := isActive && i == m.activeCard
		rendered := renderCard(cards[i], cardInner, selected)
		h := lipgloss.Height(rendered)

		remaining := budget - used
		needsDown := i < len(cards)-1
		reserve := 0
		if needsDown {
			reserve = 1
		}

		if used > 0 && h > remaining-reserve {
			break
		}
		cardViews = append(cardViews, rendered)
		used += h
		lastRendered = i + 1
	}

	if lastRendered < len(cards) {
		cardViews = append(cardViews, dimRender(fmt.Sprintf("  ↓ %d more", len(cards)-lastRendered)))
	}

	content := title + "\n" + strings.Join(cardViews, "\n")

	style := columnStyle
	if isActive {
		style = activeColumnStyle
	}

	return style.Width(innerWidth).Height(colH).Render(content)
}

var timeNow = time.Now

func formatDue(due string, complete bool) (string, lipgloss.Style) {
	t, err := time.Parse(time.RFC3339Nano, due)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", due)
		if err != nil {
			return "", lipgloss.NewStyle()
		}
	}

	label := t.Format("2 Jan")

	if complete {
		return "✓ " + label, dueDoneStyle
	}

	now := timeNow()
	if t.Before(now) {
		return "⚠ " + label, dueOverdueStyle
	}
	if t.Before(now.AddDate(0, 0, 7)) {
		return "⚠ " + label, dueSoonStyle
	}
	return label, dueDefaultStyle
}

func renderCard(c trello.Card, width int, selected bool) string {
	var topLine string
	var parts []string
	if len(c.Labels) > 0 {
		var pills []string
		for _, l := range c.Labels {
			pills = append(pills, labelColor(l.Color).Render("━━"))
		}
		parts = append(parts, strings.Join(pills, " "))
	}
	if c.Due != "" {
		if label, style := formatDue(c.Due, c.DueComplete); label != "" {
			parts = append(parts, style.Render(label))
		}
	}
	if len(parts) > 0 {
		topLine = strings.Join(parts, "  ") + "\n"
	}

	content := topLine + c.Name

	var bottomParts []string
	if len(c.Members) > 0 {
		var memberBadges []string
		for _, m := range c.Members {
			initials := memberInitials(m.FullName)
			memberBadges = append(memberBadges, memberColor(m.ID).Render(initials))
		}
		bottomParts = append(bottomParts, strings.Join(memberBadges, " "))
	}
	if c.Badges.CheckItems > 0 {
		checkBadge := fmt.Sprintf("☑ %d/%d", c.Badges.CheckItemsChecked, c.Badges.CheckItems)
		style := lipgloss.NewStyle().Foreground(dimColor)
		if c.Badges.CheckItemsChecked == c.Badges.CheckItems {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
		}
		bottomParts = append(bottomParts, style.Render(checkBadge))
	}
	if len(bottomParts) > 0 {
		content += "\n\n" + strings.Join(bottomParts, "  ")
	}

	style := cardStyle
	if selected {
		style = selectedCardStyle
	} else if c.Due != "" && !c.DueComplete {
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z", time.RFC3339} {
			if t, err := time.Parse(layout, c.Due); err == nil {
				now := timeNow()
				if t.Before(now) {
					style = cardStyle.BorderForeground(lipgloss.Color("#EF4444"))
				} else if t.Before(now.AddDate(0, 0, 7)) {
					style = cardStyle.BorderForeground(lipgloss.Color("#F59E0B"))
				}
				break
			}
		}
	}

	return style.Width(width).Render(content)
}

func memberInitials(name string) string {
	runes := []rune(strings.ToLower(strings.TrimSpace(name)))
	if len(runes) > 3 {
		runes = runes[:3]
	}
	return string(runes)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) > max {
		if max > 3 {
			return string(runes[:max-3]) + "..."
		}
		return string(runes[:max])
	}
	return s
}

func dimRender(s string) string {
	return lipgloss.NewStyle().Foreground(dimColor).Render(s)
}
