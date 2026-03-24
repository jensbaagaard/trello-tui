package tui

import (
	"fmt"
	"strings"

	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jensbaagaard/trello-tui/internal/trello"
)

type boardListMode int

const (
	boardListNav           boardListMode = iota
	boardListCreate                      // typing a new board name
	boardListWorkspacePick               // picking a workspace for the new board
)

// boardItem implements list.Item for the bubbles/list component
type boardItem struct {
	board trello.Board
}

func (b boardItem) Title() string       { return b.board.Name }
func (b boardItem) Description() string { return b.board.ID }
func (b boardItem) FilterValue() string { return b.board.Name }

type BoardListModel struct {
	list           list.Model
	client         *trello.Client
	loading        bool
	err            error
	mode           boardListMode
	textInput      textinput.Model
	statusMsg      string
	pendingName    string                // board name waiting for workspace selection
	organizations  []trello.Organization
	orgCursor      int
	loadingOrgs    bool
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
			bkey.NewBinding(bkey.WithKeys("n"), bkey.WithHelp("n", "new board")),
			bkey.NewBinding(bkey.WithKeys("s"), bkey.WithHelp("s", "search")),
		}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys

	ti := textinput.New()
	ti.Placeholder = "Board name..."
	ti.CharLimit = 200

	return BoardListModel{
		list:      l,
		client:    client,
		loading:   true,
		textInput: ti,
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

func (m BoardListModel) fetchOrganizations() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		orgs, err := client.GetOrganizations()
		return OrganizationsFetchedMsg{Organizations: orgs, Err: err}
	}
}

func (m BoardListModel) createBoard(name, idOrganization string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		board, err := client.CreateBoard(name, idOrganization)
		return BoardCreatedMsg{Board: board, Err: err}
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

	case OrganizationsFetchedMsg:
		m.loadingOrgs = false
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error fetching workspaces: %v", msg.Err)
			m.mode = boardListNav
			return m, nil
		}
		m.organizations = msg.Organizations
		m.orgCursor = 0
		return m, nil

	case BoardCreatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Error creating board: %v", msg.Err)
			return m, nil
		}
		m.list.InsertItem(0, boardItem{board: msg.Board})
		m.statusMsg = "Board created"
		return m, nil

	case tea.KeyMsg:
		if m.mode == boardListCreate {
			switch msg.String() {
			case "enter":
				name := strings.TrimSpace(m.textInput.Value())
				m.textInput.SetValue("")
				m.textInput.Blur()
				if name == "" {
					m.mode = boardListNav
					return m, nil
				}
				m.pendingName = name
				m.mode = boardListWorkspacePick
				m.loadingOrgs = true
				m.orgCursor = 0
				return m, m.fetchOrganizations()
			case "esc":
				m.mode = boardListNav
				m.textInput.SetValue("")
				m.textInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}
		if m.mode == boardListWorkspacePick {
			switch msg.String() {
			case "j", "down":
				if m.orgCursor < len(m.organizations)-1 {
					m.orgCursor++
				}
			case "k", "up":
				if m.orgCursor > 0 {
					m.orgCursor--
				}
			case "enter":
				if len(m.organizations) > 0 && m.orgCursor < len(m.organizations) {
					orgID := m.organizations[m.orgCursor].ID
					name := m.pendingName
					m.pendingName = ""
					m.mode = boardListNav
					return m, m.createBoard(name, orgID)
				}
			case "esc":
				m.mode = boardListNav
				m.pendingName = ""
			}
			return m, nil
		}
		if msg.String() == "n" && !m.IsFiltering() {
			m.mode = boardListCreate
			m.statusMsg = ""
			m.textInput.SetValue("")
			m.textInput.Focus()
			return m, textinput.Blink
		}
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
	if m.mode == boardListCreate {
		header := titleStyle.Render("Trello Boards") + "\n\n"
		sT := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		return header + sT.Render("New board") + "\n\n" +
			"Name: " + m.textInput.View() + "\n\n" +
			helpStyle.Render("enter:next  esc:cancel")
	}
	if m.mode == boardListWorkspacePick {
		header := titleStyle.Render("Trello Boards") + "\n\n"
		sT := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
		var b strings.Builder
		b.WriteString(header + sT.Render("Pick workspace for: "+m.pendingName) + "\n\n")
		if m.loadingOrgs {
			b.WriteString(helpStyle.Render("Loading workspaces..."))
		} else if len(m.organizations) == 0 {
			b.WriteString(helpStyle.Render("No workspaces found.") + "\n\n" +
				helpStyle.Render("esc:cancel"))
		} else {
			for i, org := range m.organizations {
				cursor := "  "
				s := lipgloss.NewStyle()
				if i == m.orgCursor {
					cursor = "▸ "
					s = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
				}
				b.WriteString(cursor + s.Render(org.DisplayName) + "\n")
			}
			b.WriteString("\n" + helpStyle.Render("j/k:navigate  enter:select  esc:cancel"))
		}
		return b.String()
	}
	view := m.list.View()
	if m.statusMsg != "" {
		view += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render(m.statusMsg)
	}
	return view
}

func (m BoardListModel) IsFiltering() bool {
	return m.list.FilterState() == list.Filtering
}

func (m BoardListModel) IsCreating() bool {
	return m.mode == boardListCreate || m.mode == boardListWorkspacePick
}

func (m BoardListModel) SelectedBoard() *trello.Board {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	bi := item.(boardItem)
	return &bi.board
}
