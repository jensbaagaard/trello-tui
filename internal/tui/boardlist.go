package tui

import (
	"fmt"

	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jensbaagaard/trello-tui/internal/trello"
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
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(primaryColor).BorderForeground(primaryColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(secondaryColor).BorderForeground(primaryColor)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color("#FFFFFF"))
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(dimColor)

	l := list.New(nil, delegate, 0, 0)
	l.Title = "Trello Boards"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.AdditionalShortHelpKeys = func() []bkey.Binding {
		return []bkey.Binding{
			bkey.NewBinding(bkey.WithKeys("s"), bkey.WithHelp("s", "search")),
		}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys

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

func (m BoardListModel) IsFiltering() bool {
	return m.list.FilterState() == list.Filtering
}

func (m BoardListModel) SelectedBoard() *trello.Board {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	bi := item.(boardItem)
	return &bi.board
}
