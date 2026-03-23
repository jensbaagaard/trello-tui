package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jensaagaard/trello-tui/internal/trello"
)

const minColWidth = 36

type boardMode int

const (
	boardNav boardMode = iota
	boardAddCard
	boardConfirmArchive
	boardFilter
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
}

func NewBoardModel(client *trello.Client, board trello.Board) BoardModel {
	ti := textinput.New()
	ti.Placeholder = "Card title..."
	ti.CharLimit = 200

	return BoardModel{
		client:      client,
		board:       board,
		cardsByList: make(map[string][]trello.Card),
		loading:     true,
		textInput:   ti,
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

// visibleColCount returns how many columns fit on screen at the fixed width.
func (m BoardModel) visibleColCount() int {
	if m.width <= 0 || len(m.lists) == 0 {
		return 1
	}
	n := m.width / minColWidth
	if n < 1 {
		n = 1
	}
	if n > len(m.lists) {
		n = len(m.lists)
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
	max := len(m.lists) - vis
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

	switch msg.String() {
	case "left":
		if m.activeList > 0 {
			m.activeList--
			m.scrollTop = 0
			m.clampCardCursor()
			m.ensureListVisible()
		}
	case "right":
		if m.activeList < len(m.lists)-1 {
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
		m.clampCardCursor()
		return m, nil
	case "esc":
		m.filterText = ""
		m.mode = boardNav
		m.activeCard = 0
		m.scrollTop = 0
		m.clampCardCursor()
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.filterText = m.textInput.Value()
	m.activeCard = 0
	m.scrollTop = 0
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

func (m BoardModel) moveCardLeft() (BoardModel, tea.Cmd) {
	if m.activeList == 0 {
		return m, nil
	}
	card := m.selectedCard()
	if card == nil {
		return m, nil
	}
	targetList := m.lists[m.activeList-1]

	listID := m.currentListID()
	m.removeCardFromList(listID, card.ID)

	m.cardsByList[targetList.ID] = append(m.cardsByList[targetList.ID], *card)
	m.activeList--
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

func (m BoardModel) moveCardRight() (BoardModel, tea.Cmd) {
	if m.activeList >= len(m.lists)-1 {
		return m, nil
	}
	card := m.selectedCard()
	if card == nil {
		return m, nil
	}
	targetList := m.lists[m.activeList+1]

	listID := m.currentListID()
	m.removeCardFromList(listID, card.ID)

	m.cardsByList[targetList.ID] = append(m.cardsByList[targetList.ID], *card)
	m.activeList++
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

func (m BoardModel) moveCardTo(targetIdx int) (BoardModel, tea.Cmd) {
	if targetIdx == m.activeList || targetIdx < 0 || targetIdx >= len(m.lists) {
		return m, nil
	}
	card := m.selectedCard()
	if card == nil {
		return m, nil
	}
	targetList := m.lists[targetIdx]

	listID := m.currentListID()
	m.removeCardFromList(listID, card.ID)

	m.cardsByList[targetList.ID] = append(m.cardsByList[targetList.ID], *card)
	m.activeList = targetIdx
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

func (m BoardModel) moveCardToFirst() (BoardModel, tea.Cmd) {
	return m.moveCardTo(0)
}

func (m BoardModel) moveCardToLast() (BoardModel, tea.Cmd) {
	return m.moveCardTo(len(m.lists) - 1)
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
	if m.activeList >= 0 && m.activeList < len(m.lists) {
		return m.lists[m.activeList].ID
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

	colW := m.colWidth()
	vis := m.visibleColCount()
	start := m.colOffset
	end := start + vis
	if end > len(m.lists) {
		end = len(m.lists)
	}

	columns := make([]string, 0, vis)
	for i := start; i < end; i++ {
		columns = append(columns, m.renderColumn(i, m.lists[i], colW))
	}

	board := lipgloss.JoinHorizontal(lipgloss.Top, columns...)

	var scrollHint string
	if start > 0 || end < len(m.lists) {
		scrollHint = helpStyle.Render(fmt.Sprintf(" [%d-%d of %d lists]", start+1, end, len(m.lists)))
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
		status = helpStyle.Render("←→:lists  j/k:cards  ,/.:move card  </>:move first/last  n:new  c:archive  enter:open  /:filter  r:refresh  esc:back")
	}

	header := titleStyle.Render(m.board.Name) + scrollHint
	return header + "\n" + board + "\n" + status
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

	if len(c.Members) > 0 {
		var badges []string
		for _, m := range c.Members {
			initials := memberInitials(m.FullName)
			badges = append(badges, memberColor(m.ID).Render(initials))
		}
		content += "\n\n" + strings.Join(badges, " ")
	}

	style := cardStyle
	if selected {
		style = selectedCardStyle
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
