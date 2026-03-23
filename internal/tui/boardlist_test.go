package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jensaagaard/trello-tui/internal/trello"
)

func TestBoardsFetched(t *testing.T) {
	m := NewBoardListModel(nil)
	boards := []trello.Board{
		{ID: "b1", Name: "Project Alpha"},
		{ID: "b2", Name: "Project Beta"},
	}

	updated, _ := m.Update(BoardsFetchedMsg{Boards: boards})
	if updated.loading {
		t.Error("loading should be false")
	}
	if updated.err != nil {
		t.Errorf("unexpected error: %v", updated.err)
	}

	// Verify the list was populated by checking SelectedBoard
	board := updated.SelectedBoard()
	if board == nil {
		t.Fatal("expected non-nil selected board")
	}
	if board.Name != "Project Alpha" {
		t.Errorf("board.Name = %q, want %q", board.Name, "Project Alpha")
	}
}

func TestBoardsFetchedError(t *testing.T) {
	m := NewBoardListModel(nil)
	updated, _ := m.Update(BoardsFetchedMsg{Err: fmt.Errorf("API error")})
	if updated.loading {
		t.Error("loading should be false")
	}
	if updated.err == nil {
		t.Error("expected error to be set")
	}
}

func TestBoardListViewLoading(t *testing.T) {
	m := NewBoardListModel(nil)
	v := m.View()
	if !strings.Contains(v, "Loading boards") {
		t.Errorf("view = %q, want it to contain 'Loading boards'", v)
	}
}

func TestBoardListViewError(t *testing.T) {
	m := NewBoardListModel(nil)
	m.loading = false
	m.err = fmt.Errorf("something went wrong")
	v := m.View()
	if !strings.Contains(v, "something went wrong") {
		t.Errorf("view = %q, want it to contain error message", v)
	}
}
