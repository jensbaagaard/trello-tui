package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jensbaagaard/trello-tui/internal/trello"
)

type screen int

const (
	screenBoardList screen = iota
	screenBoard
	screenCard
)

type AppModel struct {
	client    *trello.Client
	screen    screen
	boardList BoardListModel
	board     BoardModel
	card      CardModel
	width     int
	height    int
}

func NewAppModel(client *trello.Client) AppModel {
	return AppModel{
		client:    client,
		screen:    screenBoardList,
		boardList: NewBoardListModel(client),
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.boardList.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case CardMovedMsg:
		// Update board state when a card is moved from the card detail view
		if msg.Err == nil && m.screen == screenCard {
			m.moveCardInBoard(msg.Card, msg.FromListID, msg.ToListID)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.screen == screenBoardList {
				return m, tea.Quit
			}
		case "enter":
			if m.screen == screenBoardList {
				board := m.boardList.SelectedBoard()
				if board != nil {
					m.board = NewBoardModel(m.client, *board)
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
					m.card = NewCardModel(m.client, *card, m.board.lists, m.board.activeList)
					m.screen = screenCard
					return m, func() tea.Msg {
						return tea.WindowSizeMsg{Width: m.width, Height: m.height}
					}
				}
			}
		case "esc":
			switch m.screen {
			case screenCard:
				if m.card.mode != cardView {
					break
				}
				m.updateCardInBoard(m.card.card)
				m.screen = screenBoard
				return m, nil
			case screenBoard:
				if m.board.mode != boardNav {
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

func (m AppModel) View() string {
	switch m.screen {
	case screenBoardList:
		return m.boardList.View()
	case screenBoard:
		return m.board.View()
	case screenCard:
		return m.card.View()
	}
	return ""
}
