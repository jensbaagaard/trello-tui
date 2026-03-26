package tui

import (
	"testing"

	"github.com/jensbaagaard/trello-tui/internal/trello"
)

func TestCardKeys_T_EntersEditTitle(t *testing.T) {
	m := newTestCardModel()
	m.width = 100
	m.height = 40

	updated, _ := m.handleKey(key("t"))
	if updated.mode != cardEditTitle {
		t.Errorf("mode = %d, want cardEditTitle (%d)", updated.mode, cardEditTitle)
	}
}

func TestCardKeys_E_EntersEditDesc(t *testing.T) {
	m := newTestCardModel()
	m.width = 100
	m.height = 40

	updated, _ := m.handleKey(key("e"))
	if updated.mode != cardEditDesc {
		t.Errorf("mode = %d, want cardEditDesc (%d)", updated.mode, cardEditDesc)
	}
}

func TestCardKeys_D_EntersSetDue(t *testing.T) {
	m := newTestCardModel()
	m.width = 100
	m.height = 40

	updated, _ := m.handleKey(key("d"))
	if updated.mode != cardSetDue {
		t.Errorf("mode = %d, want cardSetDue (%d)", updated.mode, cardSetDue)
	}
}

func TestCardKeys_M_EntersMoveList(t *testing.T) {
	m := newTestCardModel()

	updated, _ := m.handleKey(key("m"))
	if updated.mode != cardMoveList {
		t.Errorf("mode = %d, want cardMoveList (%d)", updated.mode, cardMoveList)
	}
}

func TestCardKeys_A_EntersAddMember(t *testing.T) {
	m := newTestCardModel()
	m.boardMembers = []trello.Member{{ID: "m1", FullName: "Alice"}}

	updated, _ := m.handleKey(key("a"))
	if updated.mode != cardAddMember {
		t.Errorf("mode = %d, want cardAddMember (%d)", updated.mode, cardAddMember)
	}
}

func TestCardKeys_L_EntersAddLabel(t *testing.T) {
	m := newTestCardModel()
	m.boardLabels = []trello.Label{{ID: "lb1", Name: "Bug"}}

	updated, _ := m.handleKey(key("l"))
	if updated.mode != cardAddLabel {
		t.Errorf("mode = %d, want cardAddLabel (%d)", updated.mode, cardAddLabel)
	}
}

func TestCardKeys_Dash_EntersAddChecklist(t *testing.T) {
	m := newTestCardModel()

	updated, _ := m.handleKey(key("-"))
	if updated.mode != cardAddChecklist {
		t.Errorf("mode = %d, want cardAddChecklist (%d)", updated.mode, cardAddChecklist)
	}
}

func TestCardKeys_BigA_EntersAddAttachment(t *testing.T) {
	m := newTestCardModel()

	updated, _ := m.handleKey(key("A"))
	if updated.mode != cardAddAttachment {
		t.Errorf("mode = %d, want cardAddAttachment (%d)", updated.mode, cardAddAttachment)
	}
}

func TestCardKeys_Tab_CyclesPanes_ToChecklist(t *testing.T) {
	m := newTestCardModel()
	m.checklists = []trello.Checklist{{ID: "cl1", Name: "Todo"}}

	updated, _ := m.handleKey(key("tab"))
	if updated.mode != cardChecklistPane {
		t.Errorf("mode = %d, want cardChecklistPane (%d)", updated.mode, cardChecklistPane)
	}
}

func TestCardKeys_Tab_CyclesPanes_SkipsChecklistToAttachments(t *testing.T) {
	m := newTestCardModel()
	m.checklists = nil
	m.attachments = []trello.Attachment{{ID: "a1", Name: "file.pdf"}}

	updated, _ := m.handleKey(key("tab"))
	if updated.mode != cardAttachmentsPane {
		t.Errorf("mode = %d, want cardAttachmentsPane (%d)", updated.mode, cardAttachmentsPane)
	}
}

func TestCardKeys_Tab_CyclesPanes_ToActivityWhenEmpty(t *testing.T) {
	m := newTestCardModel()
	m.checklists = nil
	m.attachments = nil

	updated, _ := m.handleKey(key("tab"))
	if updated.mode != cardActivityPane {
		t.Errorf("mode = %d, want cardActivityPane (%d)", updated.mode, cardActivityPane)
	}
}

func TestCardKeys_EscFromEditTitle_ReturnsToView(t *testing.T) {
	m := newTestCardModel()
	m.mode = cardEditTitle

	updated, _ := m.handleKey(key("esc"))
	if updated.mode != cardView {
		t.Errorf("mode = %d, want cardView (%d)", updated.mode, cardView)
	}
}

func TestCardKeys_EscFromMoveList_ReturnsToView(t *testing.T) {
	m := newTestCardModel()
	m.mode = cardMoveList

	updated, _ := m.handleKey(key("esc"))
	if updated.mode != cardView {
		t.Errorf("mode = %d, want cardView (%d)", updated.mode, cardView)
	}
}

func TestCardKeys_CommaMovesCardLeft(t *testing.T) {
	m := newTestCardModel()
	m.listIndex = 1 // on second list

	_, cmd := m.handleKey(key(","))
	if cmd == nil {
		t.Fatal("expected move command, got nil")
	}
}

func TestCardKeys_DotMovesCardRight(t *testing.T) {
	m := newTestCardModel()
	m.listIndex = 0 // on first list, can move right

	_, cmd := m.handleKey(key("."))
	if cmd == nil {
		t.Fatal("expected move command, got nil")
	}
}

func TestCardKeys_CommaAtFirstList_NoOp(t *testing.T) {
	m := newTestCardModel()
	m.listIndex = 0

	_, cmd := m.handleKey(key(","))
	if cmd != nil {
		t.Error("expected nil cmd when at first list")
	}
}

func TestCardKeys_QuestionMark_ShowsHelp(t *testing.T) {
	m := newTestCardModel()

	updated, _ := m.handleKey(key("?"))
	if !updated.showHelp {
		t.Error("expected showHelp = true")
	}
}

func TestCardKeys_HelpDismissOnAnyKey(t *testing.T) {
	m := newTestCardModel()
	m.showHelp = true

	updated, _ := m.handleKey(key("j"))
	if updated.showHelp {
		t.Error("expected showHelp = false after keypress")
	}
}
