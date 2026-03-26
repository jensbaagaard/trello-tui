package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jensbaagaard/trello-tui/internal/trello"
	"github.com/jensbaagaard/trello-tui/internal/version"
)

type screen int

const (
	screenBoardList screen = iota
	screenBoard
	screenCard
	screenSearch
)

type AppModel struct {
	client           *trello.Client
	screen           screen
	boardList        BoardListModel
	board            BoardModel
	card             CardModel
	search           SearchModel
	returnToSearch   bool
	width            int
	height           int
	version          string
	updateNotice     string
	opts             Options
	pendingBoardName string
}

func NewAppModel(client *trello.Client, version string, opts Options) AppModel {
	return AppModel{
		client:           client,
		screen:           screenBoardList,
		boardList:        NewBoardListModel(client),
		version:          version,
		opts:             opts,
		pendingBoardName: opts.BoardName,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.boardList.Init(), m.checkVersion())
}

func (m AppModel) checkVersion() tea.Cmd {
	currentVersion := m.version
	return func() tea.Msg {
		latest := version.CheckLatest()
		if version.IsNewer(latest, currentVersion) {
			return VersionCheckMsg{UpdateNotice: version.FormatNotice(latest)}
		}
		return VersionCheckMsg{}
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case VersionCheckMsg:
		m.updateNotice = msg.UpdateNotice
		return m, nil

	case BoardsFetchedMsg:
		if m.pendingBoardName != "" {
			name := m.pendingBoardName
			m.pendingBoardName = ""
			// Let boardList process the message first
			m.boardList, _ = m.boardList.Update(msg)
			if msg.Err == nil {
				board := findBoardByName(msg.Boards, name)
				if board != nil {
					m.board = NewBoardModel(m.client, *board)
					m.board.autoRefreshSecs = m.opts.AutoRefreshSecs
					m.screen = screenBoard
					return m, tea.Batch(
						m.board.Init(),
						func() tea.Msg { return tea.WindowSizeMsg{Width: m.width, Height: m.height} },
					)
				}
			}
			return m, nil
		}

	case BoardCreatedMsg:
		if msg.Err == nil {
			m.boardList, _ = m.boardList.Update(msg)
			m.board = NewBoardModel(m.client, msg.Board)
			m.board.autoRefreshSecs = m.opts.AutoRefreshSecs
			m.screen = screenBoard
			return m, tea.Batch(
				m.board.Init(),
				func() tea.Msg { return tea.WindowSizeMsg{Width: m.width, Height: m.height} },
			)
		}
		// On error, let boardList show the status message
		m.boardList, _ = m.boardList.Update(msg)
		return m, nil

	case CardMovedMsg:
		// Update board state when a card is moved from the card detail view
		if msg.Err == nil && m.screen == screenCard {
			m.moveCardInBoard(msg.Card, msg.FromListID, msg.ToListID)
		}

	case CardMovedToBoardMsg:
		if msg.Err == nil && m.screen == screenCard {
			m.board.removeCardFromList(msg.FromListID, msg.Card.ID)
			m.board.statusMsg = m.card.statusMsg
			m.screen = screenBoard
			return m, nil
		}

	case SearchCardListsFetchedMsg:
		if m.screen == screenSearch {
			m.search, _ = m.search.Update(msg)
			if m.search.pendingCard != nil && m.search.pendingLists != nil {
				card := m.search.pendingCard.ToCard()
				lists := m.search.pendingLists
				listIndex := 0
				for i, l := range lists {
					if l.ID == card.IDList {
						listIndex = i
						break
					}
				}
				m.card = NewCardModel(m.client, card, lists, listIndex)
				m.card.boardName = m.search.pendingCard.Board.Name
				m.returnToSearch = true
				m.screen = screenCard
				return m, tea.Batch(
					m.card.Init(),
					func() tea.Msg {
						return tea.WindowSizeMsg{Width: m.width, Height: m.height}
					},
				)
			}
			return m, nil
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.screen == screenBoardList && !m.boardList.IsFiltering() && !m.boardList.IsCreating() {
				return m, tea.Quit
			}
		case "s":
			if m.screen == screenBoardList && !m.boardList.IsFiltering() && !m.boardList.IsCreating() {
				m.search = NewSearchModel(m.client)
				m.screen = screenSearch
				return m, tea.Batch(
					m.search.Init(),
					func() tea.Msg { return tea.WindowSizeMsg{Width: m.width, Height: m.height} },
				)
			}
		case "enter":
			if m.screen == screenBoardList && !m.boardList.IsFiltering() && !m.boardList.IsCreating() {
				board := m.boardList.SelectedBoard()
				if board != nil {
					m.board = NewBoardModel(m.client, *board)
					m.board.autoRefreshSecs = m.opts.AutoRefreshSecs
					m.screen = screenBoard
					return m, tea.Batch(
						m.board.Init(),
						func() tea.Msg { return tea.WindowSizeMsg{Width: m.width, Height: m.height} },
					)
				}
			}
			if m.screen == screenBoard && m.board.mode == boardNav {
				card := m.board.selectedCard()
				if card != nil {
					fullListIndex := 0
				for i, l := range m.board.lists {
					if l.ID == card.IDList {
						fullListIndex = i
						break
					}
				}
				m.card = NewCardModel(m.client, *card, m.board.lists, fullListIndex)
				m.card.boardName = m.board.board.Name
				m.card.boardLabels = m.board.boardLabels
					m.screen = screenCard
					return m, tea.Batch(
						m.card.Init(),
						func() tea.Msg {
							return tea.WindowSizeMsg{Width: m.width, Height: m.height}
						},
					)
				}
			}
		case "esc":
			switch m.screen {
			case screenCard:
				if m.card.showHelp || m.card.mode != cardView {
					break
				}
				if m.returnToSearch {
					m.returnToSearch = false
					m.search.pendingCard = nil
					m.search.pendingLists = nil
					m.screen = screenSearch
					return m, nil
				}
				m.updateCardInBoard(m.card.card)
				m.board.boardLabels = m.card.boardLabels
				m.screen = screenBoard
				return m, nil
			case screenSearch:
				if m.search.showHelp {
					break
				}
				if !m.search.textInput.Focused() || !m.search.searched {
					m.screen = screenBoardList
					return m, nil
				}
			case screenBoard:
				if m.board.showHelp || m.board.mode != boardNav {
					break
				}
				if m.board.filterText != "" {
					break // let board.go handle clearing the filter
				}
				m.screen = screenBoardList
				return m, nil
			}
		}
	}

	// Route to active screen
	switch m.screen {
	case screenBoardList:
		var cmd tea.Cmd
		m.boardList, cmd = m.boardList.Update(msg)
		return m, cmd
	case screenBoard:
		var cmd tea.Cmd
		m.board, cmd = m.board.Update(msg)
		return m, cmd
	case screenCard:
		var cmd tea.Cmd
		m.card, cmd = m.card.Update(msg)
		return m, cmd
	case screenSearch:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *AppModel) updateCardInBoard(card trello.Card) {
	listCards := m.board.cardsByList[card.IDList]
	for i, c := range listCards {
		if c.ID == card.ID {
			m.board.cardsByList[card.IDList][i] = card
			return
		}
	}
}

func (m *AppModel) moveCardInBoard(card trello.Card, fromListID, toListID string) {
	// Remove from old list
	oldCards := m.board.cardsByList[fromListID]
	for i, c := range oldCards {
		if c.ID == card.ID {
			m.board.cardsByList[fromListID] = append(oldCards[:i], oldCards[i+1:]...)
			break
		}
	}
	// Add to new list
	m.board.cardsByList[toListID] = append(m.board.cardsByList[toListID], card)
}

func findBoardByName(boards []trello.Board, name string) *trello.Board {
	q := strings.ToLower(name)
	var match *trello.Board
	for i, b := range boards {
		if strings.Contains(strings.ToLower(b.Name), q) {
			if match != nil {
				return nil // multiple matches
			}
			match = &boards[i]
		}
	}
	return match
}

var updateNoticeStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#F59E0B"))

func (m AppModel) View() string {
	if m.width > 0 && m.height > 0 && (m.width < minTermWidth || m.height < minTermHeight) {
		msg := fmt.Sprintf("Terminal too small (%dx%d)\nMinimum: %dx%d", m.width, m.height, minTermWidth, minTermHeight)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpStyle.Render(msg))
	}

	var content string
	switch m.screen {
	case screenBoardList:
		content = m.boardList.View()
	case screenBoard:
		content = m.board.View()
	case screenCard:
		content = m.card.View()
	case screenSearch:
		content = m.search.View()
	}
	if m.updateNotice != "" {
		content += "\n" + updateNoticeStyle.Render(m.updateNotice)
	}
	return content
}
