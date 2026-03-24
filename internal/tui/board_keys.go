package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jensbaagaard/trello-tui/internal/trello"
)

func (m BoardModel) handleKey(msg tea.KeyMsg) (BoardModel, tea.Cmd) {
	if m.mode == boardAddList || m.mode == boardRenameList {
		return m.handleListInputKey(msg)
	}
	if m.mode == boardConfirmArchiveList {
		return m.handleConfirmArchiveListKey(msg)
	}
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
	if m.mode == boardArchiveFilter {
		return m.handleArchiveFilterKey(msg)
	}
	if m.mode == boardArchive {
		return m.handleArchiveKey(msg)
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
	case "a":
		m.mode = boardArchive
		m.archiveCursor = 0
		m.statusMsg = ""
		return m, m.fetchArchivedCards()
	case "L":
		m.mode = boardLabelManager
		m.labelCursor = 0
		m.statusMsg = ""
		if len(m.boardLabels) == 0 {
			return m, m.fetchBoardLabels()
		}
		return m, nil
	case "N":
		m.mode = boardAddList
		m.textInput.Placeholder = "List name..."
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, textinput.Blink
	case "R":
		if len(m.lists) > 0 {
			m.mode = boardRenameList
			m.textInput.Placeholder = "List name..."
			vis := m.visibleLists()
			if m.activeList >= 0 && m.activeList < len(vis) {
				m.textInput.SetValue(vis[m.activeList].Name)
			}
			m.textInput.Focus()
			return m, textinput.Blink
		}
	case "C":
		if len(m.lists) > 0 {
			m.mode = boardConfirmArchiveList
		}
	case "{":
		return m.moveListLeft()
	case "}":
		return m.moveListRight()
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

func (m BoardModel) handleArchiveKey(msg tea.KeyMsg) (BoardModel, tea.Cmd) {
	filtered := m.filteredArchivedCards()
	switch msg.String() {
	case "j", "down":
		if m.archiveCursor < len(filtered)-1 {
			m.archiveCursor++
			m.ensureArchiveCursorVisible()
		}
	case "k", "up":
		if m.archiveCursor > 0 {
			m.archiveCursor--
			m.ensureArchiveCursorVisible()
		}
	case "enter", "u":
		if len(filtered) > 0 && m.archiveCursor < len(filtered) {
			card := filtered[m.archiveCursor]
			return m, m.restoreCard(card.ID)
		}
	case "/":
		m.mode = boardArchiveFilter
		m.textInput.Placeholder = "Filter archived cards..."
		m.textInput.SetValue(m.archiveFilterText)
		m.textInput.Focus()
		return m, textinput.Blink
	case "r":
		m.statusMsg = ""
		m.archiveScrollTop = 0
		m.archiveCursor = 0
		return m, m.fetchArchivedCards()
	case "esc":
		if m.archiveFilterText != "" {
			m.archiveFilterText = ""
			m.archiveCursor = 0
			m.archiveScrollTop = 0
			return m, nil
		}
		m.mode = boardNav
		m.statusMsg = ""
	}
	return m, nil
}

func (m BoardModel) handleArchiveFilterKey(msg tea.KeyMsg) (BoardModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.archiveFilterText = m.textInput.Value()
		m.mode = boardArchive
		m.archiveCursor = 0
		m.archiveScrollTop = 0
		return m, nil
	case "esc":
		m.archiveFilterText = ""
		m.mode = boardArchive
		m.archiveCursor = 0
		m.archiveScrollTop = 0
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.archiveFilterText = m.textInput.Value()
	m.archiveCursor = 0
	m.archiveScrollTop = 0
	return m, cmd
}

func (m BoardModel) handleListInputKey(msg tea.KeyMsg) (BoardModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.textInput.Value())
		if name == "" {
			m.mode = boardNav
			return m, nil
		}
		if m.mode == boardAddList {
			m.mode = boardNav
			return m, m.createList(name)
		}
		// boardRenameList
		listID := m.currentListID()
		m.mode = boardNav
		return m, m.renameList(listID, name)
	case "esc":
		m.mode = boardNav
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m BoardModel) handleConfirmArchiveListKey(msg tea.KeyMsg) (BoardModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		listID := m.currentListID()
		if listID != "" {
			m.mode = boardNav
			return m, m.archiveList(listID)
		}
		m.mode = boardNav
	case "n", "N", "esc":
		m.mode = boardNav
	}
	return m, nil
}

func (m BoardModel) moveListLeft() (BoardModel, tea.Cmd) {
	vis := m.visibleLists()
	if m.activeList <= 0 || len(vis) < 2 {
		return m, nil
	}
	listID := vis[m.activeList].ID
	fullIdx := m.fullListIndex(listID)
	if fullIdx <= 0 {
		return m, nil
	}

	m.lists[fullIdx], m.lists[fullIdx-1] = m.lists[fullIdx-1], m.lists[fullIdx]
	m.activeList--
	m.ensureListVisible()

	var newPos float64
	if fullIdx-1 == 0 {
		newPos = m.lists[1].Pos / 2
	} else {
		newPos = (m.lists[fullIdx-2].Pos + m.lists[fullIdx].Pos) / 2
	}

	return m, m.reorderList(listID, newPos)
}

func (m BoardModel) moveListRight() (BoardModel, tea.Cmd) {
	vis := m.visibleLists()
	if m.activeList >= len(vis)-1 || len(vis) < 2 {
		return m, nil
	}
	listID := vis[m.activeList].ID
	fullIdx := m.fullListIndex(listID)
	if fullIdx < 0 || fullIdx >= len(m.lists)-1 {
		return m, nil
	}

	m.lists[fullIdx], m.lists[fullIdx+1] = m.lists[fullIdx+1], m.lists[fullIdx]
	m.activeList++
	m.ensureListVisible()

	var newPos float64
	last := len(m.lists) - 1
	if fullIdx+1 == last {
		newPos = m.lists[last-1].Pos + 65536
	} else {
		newPos = (m.lists[fullIdx].Pos + m.lists[fullIdx+2].Pos) / 2
	}

	return m, m.reorderList(listID, newPos)
}

func (m *BoardModel) ensureArchiveCursorVisible() {
	visible := m.archiveVisibleCount()
	if m.archiveCursor < m.archiveScrollTop {
		m.archiveScrollTop = m.archiveCursor
	}
	if m.archiveCursor >= m.archiveScrollTop+visible {
		m.archiveScrollTop = m.archiveCursor - visible + 1
	}
}

func (m BoardModel) archiveVisibleCount() int {
	// header (2 lines) + footer (2-3 lines for status + help)
	overhead := 5
	n := m.height - overhead
	if n < 3 {
		n = 3
	}
	return n
}
