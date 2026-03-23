package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jensaagaard/trello-tui/internal/trello"
)

// boardItem implements list.Item for the bubbles/list component
type boardItem struct {
	board trello.Board
}

func (b boardItem) Title() string       { return b.board.Name }
func (b boardItem) Description() string { return b.board.ID }
func (b boardItem) FilterValue() string { return b.board.Name }

type BoardListModel struct {
	list    list.Model
	client  *trello.Client
	loading bool
	err     error
}

func NewBoardListModel(client *trello.Client) BoardListModel {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Trello Boards"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return BoardListModel{
		list:    l,
		client:  client,
		loading: true,
	}
}

func (m BoardListModel) Init() tea.Cmd {
	return m.fetchBoards()
}

func (m BoardListModel) fetchBoards() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		boards, err := client.GetBoards()
		return BoardsFetchedMsg{Boards: boards, Err: err}
	}
}

func (m BoardListModel) Update(msg tea.Msg) (BoardListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-2)

	case BoardsFetchedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		items := make([]list.Item, len(msg.Boards))
		for i, b := range msg.Boards {
			items[i] = boardItem{board: b}
		}
		m.list.SetItems(items)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m BoardListModel) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error loading boards: %v", m.err))
	}
	if m.loading {
		return "Loading boards..."
	}
	return m.list.View()
}

func (m BoardListModel) SelectedBoard() *trello.Board {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	bi := item.(boardItem)
	return &bi.board
}
