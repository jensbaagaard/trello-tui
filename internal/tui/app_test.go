package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jensbaagaard/trello-tui/internal/trello"
)

func newTestAppModel() AppModel {
	return AppModel{
		screen:    screenBoardList,
		boardList: NewBoardListModel(nil),
		version:   "dev",
	}
}

func TestInitialScreen(t *testing.T) {
	m := newTestAppModel()
	if m.screen != screenBoardList {
		t.Errorf("initial screen = %d, want screenBoardList", m.screen)
	}
}

func TestQuitFromBoardList(t *testing.T) {
	m := newTestAppModel()
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	// Execute the cmd to check it produces tea.QuitMsg
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestQuitNotFromBoard(t *testing.T) {
	m := newTestAppModel()
	m.screen = screenBoard
	m.board = BoardModel{
		cardsByList: make(map[string][]trello.Card),
	}
	updated, cmd := m.Update(key("q"))
	// q from board screen should NOT quit
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Error("q should not quit from board screen")
		}
	}
	_ = updated
}

func TestCtrlCQuits(t *testing.T) {
	screens := []screen{screenBoardList, screenBoard, screenCard}
	for _, s := range screens {
		m := newTestAppModel()
		m.screen = s
		if s == screenBoard {
			m.board = BoardModel{cardsByList: make(map[string][]trello.Card)}
		}
		if s == screenCard {
			m.card = CardModel{}
		}
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Fatalf("screen=%d: expected non-nil cmd for ctrl+c", s)
		}
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("screen=%d: expected tea.QuitMsg, got %T", s, msg)
		}
	}
}

func TestEscNavigation(t *testing.T) {
	// From card view → board
	m := newTestAppModel()
	m.screen = screenCard
	m.card = CardModel{mode: cardView}
	m.board = BoardModel{
		cardsByList: map[string][]trello.Card{},
	}
	updated, _ := m.Update(specialKey(tea.KeyEsc))
	app := updated.(AppModel)
	if app.screen != screenBoard {
		t.Errorf("from card: screen = %d, want screenBoard", app.screen)
	}

	// From board → board list
	m2 := newTestAppModel()
	m2.screen = screenBoard
	m2.board = BoardModel{
		mode:        boardNav,
		cardsByList: map[string][]trello.Card{},
	}
	updated2, _ := m2.Update(specialKey(tea.KeyEsc))
	app2 := updated2.(AppModel)
	if app2.screen != screenBoardList {
		t.Errorf("from board: screen = %d, want screenBoardList", app2.screen)
	}
}

func TestEscDoesNotGoBackInEditMode(t *testing.T) {
	// When card is in edit mode, esc should NOT navigate back
	m := newTestAppModel()
	m.screen = screenCard
	m.card = CardModel{mode: cardEditTitle}
	m.board = BoardModel{cardsByList: map[string][]trello.Card{}}

	updated, _ := m.Update(specialKey(tea.KeyEsc))
	app := updated.(AppModel)
	if app.screen != screenCard {
		t.Errorf("esc in edit mode should stay on card screen, got %d", app.screen)
	}
}

func TestCardMovedSyncsBoard(t *testing.T) {
	m := newTestAppModel()
	m.screen = screenCard
	m.card = CardModel{mode: cardView}
	m.board = BoardModel{
		cardsByList: map[string][]trello.Card{
			"l1": {{ID: "c1", Name: "Card 1", IDList: "l1"}},
			"l2": {},
		},
	}

	movedCard := trello.Card{ID: "c1", Name: "Card 1", IDList: "l2"}
	updated, _ := m.Update(CardMovedMsg{Card: movedCard, FromListID: "l1", ToListID: "l2"})
	app := updated.(AppModel)

	if len(app.board.cardsByList["l1"]) != 0 {
		t.Errorf("l1 should be empty, has %d cards", len(app.board.cardsByList["l1"]))
	}
	if len(app.board.cardsByList["l2"]) != 1 {
		t.Errorf("l2 should have 1 card, has %d", len(app.board.cardsByList["l2"]))
	}
}

func TestUpdateCardInBoard(t *testing.T) {
	m := newTestAppModel()
	m.screen = screenCard
	m.board = BoardModel{
		cardsByList: map[string][]trello.Card{
			"l1": {{ID: "c1", Name: "Old Name", IDList: "l1"}},
		},
	}
	m.card = CardModel{
		card: trello.Card{ID: "c1", Name: "New Name", IDList: "l1"},
		mode: cardView,
	}

	// esc from card view triggers updateCardInBoard
	updated, _ := m.Update(specialKey(tea.KeyEsc))
	app := updated.(AppModel)
	if app.board.cardsByList["l1"][0].Name != "New Name" {
		t.Errorf("card name = %q, want %q", app.board.cardsByList["l1"][0].Name, "New Name")
	}
}

func TestVersionCheckMsgSetsNotice(t *testing.T) {
	m := newTestAppModel()
	updated, _ := m.Update(VersionCheckMsg{UpdateNotice: "Update available: v0.4.0"})
	app := updated.(AppModel)
	if app.updateNotice != "Update available: v0.4.0" {
		t.Errorf("updateNotice = %q, want %q", app.updateNotice, "Update available: v0.4.0")
	}
}

func TestVersionCheckMsgEmptyNotice(t *testing.T) {
	m := newTestAppModel()
	updated, _ := m.Update(VersionCheckMsg{})
	app := updated.(AppModel)
	if app.updateNotice != "" {
		t.Errorf("updateNotice = %q, want empty", app.updateNotice)
	}
}

func TestSearchCursorPreservedAfterCardView(t *testing.T) {
	m := newTestAppModel()
	m.screen = screenCard
	m.returnToSearch = true
	m.search = SearchModel{
		results:   make([]trello.SearchCard, 20),
		cursor:    5,
		scrollTop: 3,
		searched:  true,
	}
	m.card = CardModel{mode: cardView}
	m.board = BoardModel{cardsByList: map[string][]trello.Card{}}

	// esc from card with returnToSearch should go to search and preserve cursor
	updated, _ := m.Update(specialKey(tea.KeyEsc))
	app := updated.(AppModel)
	if app.screen != screenSearch {
		t.Fatalf("screen = %d, want screenSearch", app.screen)
	}
	if app.search.cursor != 5 {
		t.Errorf("search cursor = %d, want 5", app.search.cursor)
	}
	if app.search.scrollTop != 3 {
		t.Errorf("search scrollTop = %d, want 3", app.search.scrollTop)
	}
}
