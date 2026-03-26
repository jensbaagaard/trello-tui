package tui

import (
	"testing"

	"github.com/jensbaagaard/trello-tui/internal/trello"
)

func TestBoardKeys_DotMovesCardRight(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 0
	m.activeCard = 0

	_, cmd := m.handleKey(key("."))
	if cmd == nil {
		t.Fatal("expected move command, got nil")
	}
}

func TestBoardKeys_CommaMovesCardLeft(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 1
	m.activeCard = 0

	_, cmd := m.handleKey(key(","))
	if cmd == nil {
		t.Fatal("expected move command, got nil")
	}
}

func TestBoardKeys_CommaAtFirstList_NoOp(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 0
	m.activeCard = 0

	_, cmd := m.handleKey(key(","))
	if cmd != nil {
		t.Error("expected nil cmd when moving left from first list")
	}
}

func TestBoardKeys_DotAtLastList_NoOp(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = len(m.lists) - 1
	m.activeCard = 0
	// Ensure there's a card in the last list
	m.cardsByList["l3"] = []trello.Card{{ID: "c99", IDList: "l3"}}

	_, cmd := m.handleKey(key("."))
	if cmd != nil {
		t.Error("expected nil cmd when moving right from last list")
	}
}

func TestBoardKeys_LessThanMovesToFirstList(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 1
	m.activeCard = 0

	_, cmd := m.handleKey(key("<"))
	if cmd == nil {
		t.Fatal("expected move command, got nil")
	}
}

func TestBoardKeys_GreaterThanMovesToLastList(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 0
	m.activeCard = 0

	_, cmd := m.handleKey(key(">"))
	if cmd == nil {
		t.Fatal("expected move command, got nil")
	}
}

func TestBoardKeys_N_EntersAddCardMode(t *testing.T) {
	m := newTestBoardModel()

	updated, _ := m.handleKey(key("n"))
	if updated.mode != boardAddCard {
		t.Errorf("mode = %d, want boardAddCard (%d)", updated.mode, boardAddCard)
	}
}

func TestBoardKeys_C_EntersConfirmArchiveMode(t *testing.T) {
	m := newTestBoardModel()
	m.activeList = 0
	m.activeCard = 0

	updated, _ := m.handleKey(key("c"))
	if updated.mode != boardConfirmArchive {
		t.Errorf("mode = %d, want boardConfirmArchive (%d)", updated.mode, boardConfirmArchive)
	}
}

func TestBoardKeys_Slash_EntersFilterMode(t *testing.T) {
	m := newTestBoardModel()

	updated, _ := m.handleKey(key("/"))
	if updated.mode != boardFilter {
		t.Errorf("mode = %d, want boardFilter (%d)", updated.mode, boardFilter)
	}
}

func TestBoardKeys_A_EntersArchiveMode(t *testing.T) {
	m := newTestBoardModel()

	updated, cmd := m.handleKey(key("a"))
	if updated.mode != boardArchive {
		t.Errorf("mode = %d, want boardArchive (%d)", updated.mode, boardArchive)
	}
	if cmd == nil {
		t.Error("expected fetch command for archived cards")
	}
}

func TestBoardKeys_L_EntersLabelManager(t *testing.T) {
	m := newTestBoardModel()

	updated, _ := m.handleKey(key("L"))
	if updated.mode != boardLabelManager {
		t.Errorf("mode = %d, want boardLabelManager (%d)", updated.mode, boardLabelManager)
	}
}

func TestBoardKeys_M_EntersMemberManager(t *testing.T) {
	m := newTestBoardModel()

	updated, cmd := m.handleKey(key("M"))
	if updated.mode != boardMemberManager {
		t.Errorf("mode = %d, want boardMemberManager (%d)", updated.mode, boardMemberManager)
	}
	if cmd == nil {
		t.Error("expected fetch command for members")
	}
}

func TestBoardKeys_BigN_EntersAddListMode(t *testing.T) {
	m := newTestBoardModel()

	updated, _ := m.handleKey(key("N"))
	if updated.mode != boardAddList {
		t.Errorf("mode = %d, want boardAddList (%d)", updated.mode, boardAddList)
	}
}

func TestBoardKeys_BigR_EntersRenameListMode(t *testing.T) {
	m := newTestBoardModel()

	updated, _ := m.handleKey(key("R"))
	if updated.mode != boardRenameList {
		t.Errorf("mode = %d, want boardRenameList (%d)", updated.mode, boardRenameList)
	}
}

func TestBoardKeys_BigC_EntersConfirmArchiveListMode(t *testing.T) {
	m := newTestBoardModel()

	updated, _ := m.handleKey(key("C"))
	if updated.mode != boardConfirmArchiveList {
		t.Errorf("mode = %d, want boardConfirmArchiveList (%d)", updated.mode, boardConfirmArchiveList)
	}
}

func TestBoardKeys_EscFromFilter_ClearsFilter(t *testing.T) {
	m := newTestBoardModel()
	m.filterText = "bug"

	updated, _ := m.handleKey(specialKey(0x1b)) // esc raw
	// esc in boardNav with filterText clears filter
	updated2, _ := updated.handleKey(key("esc"))
	_ = updated2 // filter handling depends on exact key type; verify via filterText
	// The handleKey for "esc" checks m.filterText
	if m.filterText != "" {
		// Re-test with the proper approach
		m2 := newTestBoardModel()
		m2.filterText = "bug"
		u, _ := m2.handleKey(key("esc"))
		if u.filterText != "" {
			t.Errorf("filterText = %q, want empty after esc", u.filterText)
		}
	}
}

func TestBoardKeys_R_Refreshes(t *testing.T) {
	m := newTestBoardModel()

	updated, cmd := m.handleKey(key("r"))
	if !updated.loading {
		t.Error("expected loading = true after refresh")
	}
	if cmd == nil {
		t.Error("expected fetch command for refresh")
	}
}

func TestBoardKeys_QuestionMark_ShowsHelp(t *testing.T) {
	m := newTestBoardModel()

	updated, _ := m.handleKey(key("?"))
	if !updated.showHelp {
		t.Error("expected showHelp = true")
	}
}

func TestBoardKeys_HelpDismissOnAnyKey(t *testing.T) {
	m := newTestBoardModel()
	m.showHelp = true

	updated, _ := m.handleKey(key("j"))
	if updated.showHelp {
		t.Error("expected showHelp = false after keypress")
	}
}
