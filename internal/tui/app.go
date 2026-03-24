package tui

import (
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
)

type AppModel struct {
	client       *trello.Client
	screen       screen
	boardList    BoardListModel
	board        BoardModel
	card         CardModel
	width        int
	height       int
	version      string
	updateNotice string
}

func NewAppModel(client *trello.Client, version string) AppModel {
	return AppModel{
		client:    client,
		screen:    screenBoardList,
		boardList: NewBoardListModel(client),
		version:   version,
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
			if m.screen == screenBoardList && !m.boardList.IsFiltering() {
				return m, tea.Quit
			}
		case "enter":
			if m.screen == screenBoardList && !m.boardList.IsFiltering() {
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
					fullListIndex := 0
				for i, l := range m.board.lists {
					if l.ID == card.IDList {
						fullListIndex = i
						break
					}
				}
				m.card = NewCardModel(m.client, *card, m.board.lists, fullListIndex)
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
				if m.card.mode != cardView {
					break
				}
				m.updateCardInBoard(m.card.card)
				m.board.boardLabels = m.card.boardLabels
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

var updateNoticeStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#F59E0B"))

func (m AppModel) View() string {
	var content string
	switch m.screen {
	case screenBoardList:
		content = m.boardList.View()
	case screenBoard:
		content = m.board.View()
	case screenCard:
		content = m.card.View()
	}
	if m.updateNotice != "" {
		content += "\n" + updateNoticeStyle.Render(m.updateNotice)
	}
	return content
}
