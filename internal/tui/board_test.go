package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jensbaagaard/trello-tui/internal/trello"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"hello", 5, "hello"},
		{"hi", 3, "hi"},
		{"hello", 3, "hel"},
		{"hello", 0, "hello"},
		{"", 5, ""},
		{"日本語テスト", 4, "日..."},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q/%d", tt.input, tt.max), func(t *testing.T) {
			got := truncate(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestMemberInitials(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Alice Smith", "ali"},
		{"Bo", "bo"},
		{"", ""},
		{"  Jens  ", "jen"},
		{"日本語テスト", "日本語"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := memberInitials(tt.name)
			if got != tt.want {
				t.Errorf("memberInitials(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestMemberColorDeterministic(t *testing.T) {
	// Same ID should always produce the same color
	c1 := memberColor("abc123")
	c2 := memberColor("abc123")
	if c1.GetForeground() != c2.GetForeground() {
		t.Error("memberColor should be deterministic for the same ID")
	}

	// Different IDs should (very likely) produce different colors
	c3 := memberColor("xyz789")
	if c1.GetBackground() == c3.GetBackground() {
		t.Log("warning: different IDs produced same color (possible but unlikely)")
	}
}

func TestVisibleColCount(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		listCount int
		want      int
	}{
		{"zero width", 0, 3, 1},
		{"narrow terminal", 40, 5, 1},
		{"medium terminal", 100, 5, 2},
		{"wide terminal", 200, 5, 5},
		{"fewer lists than fit", 200, 2, 2},
		{"no lists", 200, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := BoardModel{width: tt.width}
			for i := 0; i < tt.listCount; i++ {
				m.lists = append(m.lists, trello.List{ID: fmt.Sprintf("l%d", i)})
			}
			got := m.visibleColCount()
			if got != tt.want {
				t.Errorf("visibleColCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestColWidth(t *testing.T) {
	tests := []struct {
		name  string
		width int
		lists int
		want  int
	}{
		{"zero width returns min", 0, 3, minColWidth},
		{"narrow returns min", 30, 3, minColWidth},
		{"fills space", 200, 5, 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := BoardModel{width: tt.width}
			for i := 0; i < tt.lists; i++ {
				m.lists = append(m.lists, trello.List{ID: fmt.Sprintf("l%d", i)})
			}
			got := m.colWidth()
			if got != tt.want {
				t.Errorf("colWidth() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestColumnHeight(t *testing.T) {
	tests := []struct {
		height int
		want   int
	}{
		{20, 18},
		{10, 8},
		{3, 6}, // minimum clamp
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("height=%d", tt.height), func(t *testing.T) {
			m := BoardModel{height: tt.height}
			got := m.columnHeight()
			if got != tt.want {
				t.Errorf("columnHeight() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCardBudget(t *testing.T) {
	tests := []struct {
		height int
		want   int
	}{
		{20, 16},
		{10, 6},
		{3, 4}, // minimum clamp
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("height=%d", tt.height), func(t *testing.T) {
			m := BoardModel{height: tt.height}
			got := m.cardBudget()
			if got != tt.want {
				t.Errorf("cardBudget() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestClampCardCursor(t *testing.T) {
	m := BoardModel{
		lists:      []trello.List{{ID: "l1"}},
		activeList: 0,
		activeCard: 5,
		cardsByList: map[string][]trello.Card{
			"l1": {{ID: "c1"}, {ID: "c2"}},
		},
	}
	m.clampCardCursor()
	if m.activeCard != 1 {
		t.Errorf("activeCard = %d, want 1", m.activeCard)
	}

	m.activeCard = -3
	m.cardsByList["l1"] = nil
	m.clampCardCursor()
	if m.activeCard != 0 {
		t.Errorf("activeCard = %d, want 0 for empty list", m.activeCard)
	}
}

func TestCurrentListID(t *testing.T) {
	m := BoardModel{
		lists:      []trello.List{{ID: "l1"}, {ID: "l2"}},
		activeList: 1,
	}
	if got := m.currentListID(); got != "l2" {
		t.Errorf("currentListID() = %q, want %q", got, "l2")
	}

	m.lists = nil
	if got := m.currentListID(); got != "" {
		t.Errorf("currentListID() = %q, want empty for no lists", got)
	}
}

func TestSelectedCard(t *testing.T) {
	m := BoardModel{
		lists:      []trello.List{{ID: "l1"}},
		activeList: 0,
		activeCard: 0,
		cardsByList: map[string][]trello.Card{
			"l1": {{ID: "c1", Name: "Card One"}},
		},
	}
	card := m.selectedCard()
	if card == nil {
		t.Fatal("expected non-nil card")
	}
	if card.ID != "c1" {
		t.Errorf("card.ID = %q, want %q", card.ID, "c1")
	}

	m.activeCard = 5
	if m.selectedCard() != nil {
		t.Error("expected nil for out-of-range cursor")
	}
}

func newTestBoardModelWithFilter() BoardModel {
	lists := []trello.List{
		{ID: "l1", Name: "To Do"},
		{ID: "l2", Name: "Doing"},
		{ID: "l3", Name: "Done"},
	}
	cardsByList := map[string][]trello.Card{
		"l1": {
			{ID: "c1", Name: "Fix login bug", IDList: "l1", Desc: "Users cannot log in",
				Members: []trello.Member{{ID: "m1", FullName: "Alice Smith", Username: "asmith"}},
				Labels:  []trello.Label{{ID: "lb1", Name: "urgent", Color: "red"}}},
			{ID: "c2", Name: "Add search feature", IDList: "l1",
				Labels: []trello.Label{{ID: "lb2", Name: "enhancement", Color: "green"}}},
			{ID: "c3", Name: "Update docs", IDList: "l1"},
		},
		"l2": {
			{ID: "c4", Name: "Deploy to staging", IDList: "l2",
				Members: []trello.Member{{ID: "m2", FullName: "Bob Jones", Username: "bjones"}}},
		},
		"l3": {},
	}
	ti := textinput.New()
	ti.Placeholder = "Card title..."
	ti.CharLimit = 200
	return BoardModel{
		board:       trello.Board{ID: "b1", Name: "Test Board"},
		lists:       lists,
		cardsByList: cardsByList,
		textInput:   ti,
		width:       200,
		height:      40,
	}
}

func newTestBoardModel() BoardModel {
	lists := []trello.List{
		{ID: "l1", Name: "To Do"},
		{ID: "l2", Name: "Doing"},
		{ID: "l3", Name: "Done"},
	}
	cardsByList := map[string][]trello.Card{
		"l1": {{ID: "c1", Name: "Card 1", IDList: "l1"}, {ID: "c2", Name: "Card 2", IDList: "l1"}},
		"l2": {{ID: "c3", Name: "Card 3", IDList: "l2"}},
		"l3": {},
	}
	ti := textinput.New()
	ti.Placeholder = "Card title..."
	ti.CharLimit = 200
	return BoardModel{
		board:       trello.Board{ID: "b1", Name: "Test Board"},
		lists:       lists,
		cardsByList: cardsByList,
		textInput:   ti,
		width:       200,
		height:      40,
	}
}

func TestUpdateListsFetched(t *testing.T) {
	m := BoardModel{
		loading:     true,
		cardsByList: make(map[string][]trello.Card),
	}

	lists := []trello.List{{ID: "l1", Name: "To Do"}}
	updated, cmd := m.Update(ListsFetchedMsg{Lists: lists})
	if len(updated.lists) != 1 {
		t.Errorf("len(lists) = %d, want 1", len(updated.lists))
	}
	// Should return a command to fetch cards
	if cmd == nil {
		t.Error("expected non-nil cmd to fetch cards")
	}
}

func TestUpdateListsFetchedError(t *testing.T) {
	m := BoardModel{loading: true, cardsByList: make(map[string][]trello.Card)}
	updated, cmd := m.Update(ListsFetchedMsg{Err: fmt.Errorf("network error")})
	if updated.loading {
		t.Error("loading should be false after error")
	}
	if updated.err == nil {
		t.Error("expected error to be set")
	}
	if cmd != nil {
		t.Error("expected nil cmd on error")
	}
}

func TestUpdateAllCardsFetched(t *testing.T) {
	m := BoardModel{
		loading:     true,
		lists:       []trello.List{{ID: "l1"}},
		cardsByList: make(map[string][]trello.Card),
	}
	cards := map[string][]trello.Card{
		"l1": {{ID: "c1", Name: "Card"}},
	}
	updated, _ := m.Update(AllCardsFetchedMsg{CardsByList: cards})
	if updated.loading {
		t.Error("loading should be false")
	}
	if len(updated.cardsByList["l1"]) != 1 {
		t.Errorf("cards in l1 = %d, want 1", len(updated.cardsByList["l1"]))
	}
}

func TestUpdateCardCreated(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 0

	card := trello.Card{ID: "c-new", Name: "New Card", IDList: "l1"}
	updated, _ := m.Update(CardCreatedMsg{Card: card})

	cards := updated.cardsByList["l1"]
	if len(cards) != 3 {
		t.Fatalf("len = %d, want 3", len(cards))
	}
	if cards[2].ID != "c-new" {
		t.Errorf("last card ID = %q, want %q", cards[2].ID, "c-new")
	}
	if updated.statusMsg != "Card created" {
		t.Errorf("statusMsg = %q, want %q", updated.statusMsg, "Card created")
	}
}

func TestUpdateCardArchived(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 0
	m.activeCard = 0

	updated, _ := m.Update(CardArchivedMsg{CardID: "c1"})
	cards := updated.cardsByList["l1"]
	if len(cards) != 1 {
		t.Fatalf("len = %d, want 1", len(cards))
	}
	if cards[0].ID != "c2" {
		t.Errorf("remaining card ID = %q, want %q", cards[0].ID, "c2")
	}
	if updated.statusMsg != "Card archived" {
		t.Errorf("statusMsg = %q, want %q", updated.statusMsg, "Card archived")
	}
}

func TestUpdateErrors(t *testing.T) {
	m := newTestBoardModel()

	tests := []struct {
		name string
		msg  tea.Msg
		want string
	}{
		{"card created error", CardCreatedMsg{Err: fmt.Errorf("fail")}, "Error creating card"},
		{"card archived error", CardArchivedMsg{Err: fmt.Errorf("fail")}, "Error archiving card"},
		{"card updated error", CardUpdatedMsg{Err: fmt.Errorf("fail")}, "Error moving card"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, _ := m.Update(tt.msg)
			if !strings.Contains(updated.statusMsg, tt.want) {
				t.Errorf("statusMsg = %q, want it to contain %q", updated.statusMsg, tt.want)
			}
		})
	}
}

func key(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func TestKeyNavigation(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 0
	m.activeCard = 0

	// j moves card down
	updated, _ := m.Update(key("j"))
	if updated.activeCard != 1 {
		t.Errorf("after j: activeCard = %d, want 1", updated.activeCard)
	}

	// k moves card up
	updated, _ = updated.Update(key("k"))
	if updated.activeCard != 0 {
		t.Errorf("after k: activeCard = %d, want 0", updated.activeCard)
	}

	// right moves to next list
	updated, _ = m.Update(key("right"))
	if updated.activeList != 1 {
		t.Errorf("after right: activeList = %d, want 1", updated.activeList)
	}

	// left moves back
	updated, _ = updated.Update(key("left"))
	if updated.activeList != 0 {
		t.Errorf("after left: activeList = %d, want 0", updated.activeList)
	}

	// left at first list doesn't go negative
	updated, _ = m.Update(key("left"))
	if updated.activeList != 0 {
		t.Errorf("left at start: activeList = %d, want 0", updated.activeList)
	}
}

func specialKey(k tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: k}
}

func TestKeyNavigationArrows(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 1
	m.activeCard = 0

	updated, _ := m.Update(specialKey(tea.KeyRight))
	if updated.activeList != 2 {
		t.Errorf("after right: activeList = %d, want 2", updated.activeList)
	}

	updated, _ = updated.Update(specialKey(tea.KeyLeft))
	if updated.activeList != 1 {
		t.Errorf("after left: activeList = %d, want 1", updated.activeList)
	}
}

func TestKeyMoveCard(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 0
	m.activeCard = 0

	// , at first list does nothing
	updated, _ := m.Update(key(","))
	if updated.activeList != 0 {
		t.Errorf("comma at first: activeList = %d, want 0", updated.activeList)
	}

	// . moves card right
	updated, cmd := m.Update(key("."))
	if updated.activeList != 1 {
		t.Errorf("after dot: activeList = %d, want 1", updated.activeList)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for API call")
	}
	// Card should now be in l2
	if len(updated.cardsByList["l1"]) != 1 {
		t.Errorf("l1 cards = %d, want 1", len(updated.cardsByList["l1"]))
	}
	if len(updated.cardsByList["l2"]) != 2 {
		t.Errorf("l2 cards = %d, want 2", len(updated.cardsByList["l2"]))
	}
}

func TestKeyArchive(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 0
	m.activeCard = 0

	// c enters confirm mode
	updated, _ := m.Update(key("c"))
	if updated.mode != boardConfirmArchive {
		t.Errorf("mode = %d, want boardConfirmArchive", updated.mode)
	}

	// n cancels
	updated, _ = updated.Update(key("n"))
	if updated.mode != boardNav {
		t.Errorf("after n: mode = %d, want boardNav", updated.mode)
	}

	// c then y confirms (returns cmd)
	updated, _ = m.Update(key("c"))
	updated, cmd := updated.Update(key("y"))
	if updated.mode != boardNav {
		t.Errorf("after y: mode = %d, want boardNav", updated.mode)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for archive API call")
	}
}

func TestKeyNewCard(t *testing.T) {
	m := newTestBoardModel()

	// n enters add mode
	updated, _ := m.Update(key("n"))
	if updated.mode != boardAddCard {
		t.Errorf("mode = %d, want boardAddCard", updated.mode)
	}

	// esc cancels
	updated, _ = updated.Update(specialKey(tea.KeyEsc))
	if updated.mode != boardNav {
		t.Errorf("after esc: mode = %d, want boardNav", updated.mode)
	}

	// n then enter with empty text cancels
	updated, _ = m.Update(key("n"))
	updated, _ = updated.Update(specialKey(tea.KeyEnter))
	if updated.mode != boardNav {
		t.Errorf("empty enter: mode = %d, want boardNav", updated.mode)
	}
}

func TestViewLoading(t *testing.T) {
	m := BoardModel{
		board:   trello.Board{Name: "Test Board"},
		loading: true,
	}
	v := m.View()
	if !strings.Contains(v, "Loading board") {
		t.Errorf("view = %q, want it to contain 'Loading board'", v)
	}
}

func TestViewError(t *testing.T) {
	m := BoardModel{
		board: trello.Board{Name: "Test Board"},
		err:   fmt.Errorf("something broke"),
	}
	v := m.View()
	if !strings.Contains(v, "something broke") {
		t.Errorf("view = %q, want it to contain error message", v)
	}
}

func TestViewNoLists(t *testing.T) {
	m := BoardModel{
		board:       trello.Board{Name: "Test Board"},
		lists:       []trello.List{},
		cardsByList: make(map[string][]trello.Card),
	}
	v := m.View()
	if !strings.Contains(v, "No lists") {
		t.Errorf("view = %q, want it to contain 'No lists'", v)
	}
}

func TestMatchesFilter(t *testing.T) {
	card := trello.Card{
		ID:   "c1",
		Name: "Fix Login Bug",
		Desc: "Users cannot authenticate",
		Members: []trello.Member{
			{ID: "m1", FullName: "Alice Smith", Username: "asmith"},
		},
		Labels: []trello.Label{
			{ID: "lb1", Name: "Urgent", Color: "red"},
		},
	}

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"empty query matches all", "", true},
		{"matches title", "login", true},
		{"matches title case-insensitive", "LOGIN", true},
		{"matches description", "authenticate", true},
		{"matches member full name", "alice", true},
		{"matches member username", "asmith", true},
		{"matches label name", "urgent", true},
		{"no match", "nonexistent", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFilter(card, tt.query)
			if got != tt.want {
				t.Errorf("matchesFilter(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestFilteredCards(t *testing.T) {
	m := newTestBoardModelWithFilter()

	// No filter returns all cards
	all := m.filteredCards("l1")
	if len(all) != 3 {
		t.Errorf("no filter: got %d cards, want 3", len(all))
	}

	// Filter returns subset
	m.filterText = "bug"
	filtered := m.filteredCards("l1")
	if len(filtered) != 1 {
		t.Errorf("filter 'bug': got %d cards, want 1", len(filtered))
	}
	if filtered[0].ID != "c1" {
		t.Errorf("filter 'bug': got card %q, want 'c1'", filtered[0].ID)
	}

	// Filter with no matches returns empty
	m.filterText = "zzzzz"
	none := m.filteredCards("l1")
	if len(none) != 0 {
		t.Errorf("filter 'zzzzz': got %d cards, want 0", len(none))
	}
}

func TestKeyFilter(t *testing.T) {
	m := newTestBoardModelWithFilter()
	m.activeList = 0
	m.activeCard = 0

	// / enters filter mode
	updated, _ := m.Update(key("/"))
	if updated.mode != boardFilter {
		t.Errorf("after /: mode = %d, want boardFilter", updated.mode)
	}

	// Type some text, then enter keeps the filter
	updated.textInput.SetValue("bug")
	updated, _ = updated.Update(specialKey(tea.KeyEnter))
	if updated.mode != boardNav {
		t.Errorf("after enter: mode = %d, want boardNav", updated.mode)
	}
	if updated.filterText != "bug" {
		t.Errorf("after enter: filterText = %q, want 'bug'", updated.filterText)
	}

	// / re-enters filter mode with current text
	updated, _ = updated.Update(key("/"))
	if updated.mode != boardFilter {
		t.Errorf("after second /: mode = %d, want boardFilter", updated.mode)
	}
	if updated.textInput.Value() != "bug" {
		t.Errorf("re-entered filter: textInput = %q, want 'bug'", updated.textInput.Value())
	}

	// esc in filter mode clears filter
	updated, _ = updated.Update(specialKey(tea.KeyEsc))
	if updated.mode != boardNav {
		t.Errorf("after esc: mode = %d, want boardNav", updated.mode)
	}
	if updated.filterText != "" {
		t.Errorf("after esc: filterText = %q, want empty", updated.filterText)
	}
}

func TestEscClearsFilterInNavMode(t *testing.T) {
	m := newTestBoardModelWithFilter()
	m.filterText = "bug"

	// First esc clears the filter
	updated, _ := m.Update(specialKey(tea.KeyEsc))
	if updated.filterText != "" {
		t.Errorf("after first esc: filterText = %q, want empty", updated.filterText)
	}
}

func TestNavigationWithFilter(t *testing.T) {
	m := newTestBoardModelWithFilter()
	m.activeList = 0
	m.activeCard = 0
	m.filterText = "feature" // matches only "Add search feature" (c2)

	cards := m.currentCards()
	if len(cards) != 1 {
		t.Fatalf("filtered cards = %d, want 1", len(cards))
	}
	if cards[0].ID != "c2" {
		t.Errorf("filtered card = %q, want 'c2'", cards[0].ID)
	}

	// j should not go beyond filtered set
	m.activeCard = 0
	updated, _ := m.Update(key("j"))
	if updated.activeCard != 0 {
		t.Errorf("j with 1 filtered card: activeCard = %d, want 0", updated.activeCard)
	}
}

func TestMoveCardWithFilter(t *testing.T) {
	m := newTestBoardModelWithFilter()
	m.activeList = 0
	m.filterText = "bug" // matches "Fix login bug" (c1)
	m.activeCard = 0     // points to c1 in filtered view

	// Move the filtered card right
	updated, cmd := m.Update(key("."))
	if cmd == nil {
		t.Error("expected non-nil cmd for API call")
	}
	// After moving c1 ("Fix login bug") to l2, only l2 has matching cards
	// for filter "bug", so visible lists = [l2] and activeList = 0
	if updated.activeList != 0 {
		t.Errorf("after move right: activeList = %d, want 0", updated.activeList)
	}

	// c1 should no longer be in l1
	for _, c := range updated.cardsByList["l1"] {
		if c.ID == "c1" {
			t.Error("c1 should have been removed from l1")
		}
	}

	// c1 should be in l2
	found := false
	for _, c := range updated.cardsByList["l2"] {
		if c.ID == "c1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("c1 should be in l2 after move")
	}
}

func TestFormatDue(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	origTimeNow := timeNow
	timeNow = func() time.Time { return now }
	defer func() { timeNow = origTimeNow }()

	tests := []struct {
		name      string
		due       string
		complete  bool
		wantLabel string
		wantStyle lipgloss.Style
	}{
		{
			name:      "overdue",
			due:       "2026-03-20T12:00:00.000Z",
			wantLabel: "⚠ 20 Mar",
			wantStyle: dueOverdueStyle,
		},
		{
			name:      "due soon (within a week)",
			due:       "2026-03-25T12:00:00.000Z",
			wantLabel: "⚠ 25 Mar",
			wantStyle: dueSoonStyle,
		},
		{
			name:      "due later (more than a week)",
			due:       "2026-04-15T12:00:00.000Z",
			wantLabel: "15 Apr",
			wantStyle: dueDefaultStyle,
		},
		{
			name:      "completed",
			due:       "2026-03-20T12:00:00.000Z",
			complete:  true,
			wantLabel: "✓ 20 Mar",
			wantStyle: dueDoneStyle,
		},
		{
			name: "invalid date",
			due:  "not-a-date",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, style := formatDue(tt.due, tt.complete)
			if label != tt.wantLabel {
				t.Errorf("label = %q, want %q", label, tt.wantLabel)
			}
			if tt.wantLabel != "" && style.GetForeground() != tt.wantStyle.GetForeground() {
				t.Errorf("style foreground mismatch")
			}
		})
	}
}

func TestMatchesFilterDueDate(t *testing.T) {
	card := trello.Card{
		ID:   "c1",
		Name: "Some card",
		Due:  "2026-03-25T12:00:00.000Z",
	}
	if !matchesFilter(card, "mar") {
		t.Error("expected due date 'mar' to match")
	}
	if !matchesFilter(card, "25 mar") {
		t.Error("expected due date '25 mar' to match")
	}
}
