package tui

import (
	"testing"

	"github.com/jensbaagaard/trello-tui/internal/trello"
)

func newTestSearchModel() SearchModel {
	return NewSearchModel(nil)
}

func TestSearchModel_InitialState(t *testing.T) {
	m := newTestSearchModel()
	if !m.textInput.Focused() {
		t.Error("expected text input to be focused initially")
	}
	if len(m.results) != 0 {
		t.Errorf("results = %d, want 0", len(m.results))
	}
	if m.searched {
		t.Error("expected searched = false initially")
	}
}

func TestSearchModel_EnterWithEmptyQuery_NoOp(t *testing.T) {
	m := newTestSearchModel()
	m.textInput.SetValue("")

	_, cmd := m.Update(key("enter"))
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(SearchResultsMsg); ok {
			t.Error("expected no search command for empty query")
		}
	}
}

func TestSearchModel_EnterWithQuery_StartSearch(t *testing.T) {
	m := newTestSearchModel()
	m.textInput.SetValue("bug fix")

	updated, cmd := m.Update(key("enter"))
	if !updated.loading {
		t.Error("expected loading = true after enter with query")
	}
	if cmd == nil {
		t.Fatal("expected search command, got nil")
	}
}

func TestSearchModel_NavigateResults(t *testing.T) {
	m := newTestSearchModel()
	m.searched = true
	m.textInput.Blur()
	m.results = []trello.SearchCard{
		{ID: "c1", Name: "Card 1"},
		{ID: "c2", Name: "Card 2"},
		{ID: "c3", Name: "Card 3"},
	}
	m.width = 80
	m.height = 40

	updated, _ := m.handleResultKeys(key("j"))
	if updated.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after j", updated.cursor)
	}

	updated, _ = updated.handleResultKeys(key("j"))
	if updated.cursor != 2 {
		t.Errorf("cursor = %d, want 2 after j", updated.cursor)
	}

	updated, _ = updated.handleResultKeys(key("k"))
	if updated.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after k", updated.cursor)
	}
}

func TestSearchModel_CursorBounds(t *testing.T) {
	m := newTestSearchModel()
	m.searched = true
	m.textInput.Blur()
	m.results = []trello.SearchCard{
		{ID: "c1", Name: "Card 1"},
		{ID: "c2", Name: "Card 2"},
	}
	m.cursor = 0
	m.width = 80
	m.height = 40

	updated, _ := m.handleResultKeys(key("k"))
	if updated.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (can't go below 0)", updated.cursor)
	}

	m.cursor = 1
	updated, _ = m.handleResultKeys(key("j"))
	if updated.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (can't exceed result count)", updated.cursor)
	}
}

func TestSearchModel_EnterOpensCard(t *testing.T) {
	m := newTestSearchModel()
	m.searched = true
	m.textInput.Blur()
	m.results = []trello.SearchCard{
		{ID: "c1", Name: "Card 1", IDBoard: "b1"},
	}
	m.cursor = 0
	m.width = 80
	m.height = 40

	updated, cmd := m.handleResultKeys(key("enter"))
	if updated.pendingCard == nil {
		t.Error("expected pendingCard to be set")
	}
	if cmd == nil {
		t.Fatal("expected fetch lists command, got nil")
	}
}

func TestSearchModel_SlashRefocusesInput(t *testing.T) {
	m := newTestSearchModel()
	m.searched = true
	m.textInput.Blur()
	m.results = []trello.SearchCard{{ID: "c1"}}
	m.width = 80
	m.height = 40

	updated, _ := m.handleResultKeys(key("/"))
	if !updated.textInput.Focused() {
		t.Error("expected text input to be re-focused after /")
	}
}

func TestSearchModel_QuestionMarkShowsHelp(t *testing.T) {
	m := newTestSearchModel()
	m.searched = true
	m.textInput.Blur()
	m.results = []trello.SearchCard{{ID: "c1"}}
	m.width = 80
	m.height = 40

	updated, _ := m.handleResultKeys(key("?"))
	if !updated.showHelp {
		t.Error("expected showHelp = true")
	}
}

func TestSearchModel_SearchResultsMsg_ResetsState(t *testing.T) {
	m := newTestSearchModel()
	m.cursor = 5
	m.scrollTop = 3
	m.loading = true

	cards := []trello.SearchCard{
		{ID: "c1", Name: "Result"},
	}
	updated, _ := m.Update(SearchResultsMsg{Cards: cards})

	if updated.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after new results", updated.cursor)
	}
	if updated.scrollTop != 0 {
		t.Errorf("scrollTop = %d, want 0 after new results", updated.scrollTop)
	}
	if updated.loading {
		t.Error("expected loading = false after results")
	}
	if len(updated.results) != 1 {
		t.Errorf("results = %d, want 1", len(updated.results))
	}
}
