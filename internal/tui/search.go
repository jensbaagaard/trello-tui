package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jensbaagaard/trello-tui/internal/trello"
)

type SearchModel struct {
	client      *trello.Client
	textInput   textinput.Model
	results     []trello.SearchCard
	cursor      int
	scrollTop   int
	width       int
	height      int
	loading     bool
	searched    bool
	statusMsg   string
	pendingCard  *trello.SearchCard
	pendingLists []trello.List
	showHelp     bool
}

func NewSearchModel(client *trello.Client) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search cards..."
	ti.CharLimit = 200
	ti.Focus()

	return SearchModel{
		client:    client,
		textInput: ti,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) searchCards(query string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		cards, err := client.SearchCards(query, 0)
		return SearchResultsMsg{Cards: cards, Err: err}
	}
}

func (m SearchModel) fetchCardLists(boardID string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		lists, err := client.GetLists(boardID)
		return SearchCardListsFetchedMsg{Lists: lists, Err: err}
	}
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 4
		if m.textInput.Width < 20 {
			m.textInput.Width = 20
		}

	case SearchResultsMsg:
		m.loading = false
		if msg.Err != nil {
			m.statusMsg = "Search failed: " + msg.Err.Error()
			return m, nil
		}
		m.results = msg.Cards
		m.searched = true
		m.cursor = 0
		m.scrollTop = 0
		m.statusMsg = ""
		return m, nil

	case SearchCardListsFetchedMsg:
		m.loading = false
		if msg.Err != nil {
			m.statusMsg = "Failed to load lists: " + msg.Err.Error()
			m.pendingCard = nil
			return m, nil
		}
		m.pendingLists = msg.Lists
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		if !m.searched || m.textInput.Focused() {
			return m.handleInputKeys(msg)
		}
		return m.handleResultKeys(msg)
	}

	if m.textInput.Focused() {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m SearchModel) handleInputKeys(msg tea.KeyMsg) (SearchModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := m.textInput.Value()
		if query == "" {
			return m, nil
		}
		m.loading = true
		m.statusMsg = ""
		m.textInput.Blur()
		return m, m.searchCards(query)
	case "esc":
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m SearchModel) handleResultKeys(msg tea.KeyMsg) (SearchModel, tea.Cmd) {
	if m.showHelp {
		if msg.String() == "?" || msg.String() == "esc" {
			m.showHelp = false
		}
		return m, nil
	}

	switch msg.String() {
	case "?":
		m.showHelp = true
		return m, nil
	case "j", "down":
		if m.cursor < len(m.results)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
	case "enter":
		if len(m.results) > 0 {
			card := m.results[m.cursor]
			m.pendingCard = &card
			m.pendingLists = nil
			m.loading = true
			m.statusMsg = ""
			return m, m.fetchCardLists(card.IDBoard)
		}
	case "/":
		m.textInput.Focus()
		return m, textinput.Blink
	case "esc":
		return m, nil
	}
	return m, nil
}

func (m *SearchModel) ensureCursorVisible() {
	visible := m.visibleResultCount()
	if visible < 1 {
		visible = 1
	}
	if m.cursor < m.scrollTop {
		m.scrollTop = m.cursor
	}
	if m.cursor >= m.scrollTop+visible {
		m.scrollTop = m.cursor - visible + 1
	}
}

func (m SearchModel) visibleResultCount() int {
	// header(1) + input(1) + blank(1) + footer(1) = 4 lines overhead
	available := m.height - 4
	if available < 1 {
		return 1
	}
	return available
}
