package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jensaagaard/trello-tui/internal/trello"
)

func newTestCardModel() CardModel {
	lists := []trello.List{
		{ID: "l1", Name: "To Do"},
		{ID: "l2", Name: "Doing"},
		{ID: "l3", Name: "Done"},
	}
	card := trello.Card{
		ID:       "c1",
		Name:     "Test Card",
		Desc:     "A description",
		IDList:   "l1",
		Labels:   []trello.Label{{ID: "lb1", Name: "Bug", Color: "red"}},
		ShortURL: "https://trello.com/c/abc123",
		Members:  []trello.Member{{ID: "m1", FullName: "Alice Smith", Username: "alice"}},
	}
	return NewCardModel(nil, card, lists, 0)
}

func TestCardUpdateMsg(t *testing.T) {
	m := newTestCardModel()
	updatedCard := trello.Card{ID: "c1", Name: "Updated Title", Desc: "New desc", IDList: "l1"}

	updated, _ := m.Update(CardUpdatedMsg{Card: updatedCard})
	if updated.card.Name != "Updated Title" {
		t.Errorf("card.Name = %q, want %q", updated.card.Name, "Updated Title")
	}
	if updated.statusMsg != "Card updated" {
		t.Errorf("statusMsg = %q, want %q", updated.statusMsg, "Card updated")
	}
}

func TestCardMovedMsg(t *testing.T) {
	m := newTestCardModel()
	movedCard := trello.Card{ID: "c1", Name: "Test Card", IDList: "l2"}

	m.moveIndex = 1
	updated, _ := m.Update(CardMovedMsg{Card: movedCard, FromListID: "l1", ToListID: "l2"})
	if updated.card.IDList != "l2" {
		t.Errorf("card.IDList = %q, want %q", updated.card.IDList, "l2")
	}
	if updated.listIndex != 1 {
		t.Errorf("listIndex = %d, want 1", updated.listIndex)
	}
	if updated.listName != "Doing" {
		t.Errorf("listName = %q, want %q", updated.listName, "Doing")
	}
	if !strings.Contains(updated.statusMsg, "Moved to") {
		t.Errorf("statusMsg = %q, want it to contain 'Moved to'", updated.statusMsg)
	}
}

func TestCardUpdateError(t *testing.T) {
	m := newTestCardModel()

	updated, _ := m.Update(CardUpdatedMsg{Err: fmt.Errorf("update failed")})
	if !strings.Contains(updated.statusMsg, "Error") {
		t.Errorf("statusMsg = %q, want it to contain 'Error'", updated.statusMsg)
	}

	updated, _ = m.Update(CardMovedMsg{Err: fmt.Errorf("move failed")})
	if !strings.Contains(updated.statusMsg, "Error moving") {
		t.Errorf("statusMsg = %q, want it to contain 'Error moving'", updated.statusMsg)
	}
}

func TestKeyEditTitle(t *testing.T) {
	m := newTestCardModel()

	// t enters edit mode
	updated, _ := m.Update(key("t"))
	if updated.mode != cardEditTitle {
		t.Errorf("mode = %d, want cardEditTitle", updated.mode)
	}

	// esc cancels
	updated, _ = updated.Update(specialKey(tea.KeyEsc))
	if updated.mode != cardView {
		t.Errorf("after esc: mode = %d, want cardView", updated.mode)
	}

	// e also enters edit mode
	updated, _ = m.Update(key("e"))
	if updated.mode != cardEditTitle {
		t.Errorf("mode after e = %d, want cardEditTitle", updated.mode)
	}
}

func TestKeyEditDesc(t *testing.T) {
	m := newTestCardModel()

	// E enters desc edit mode
	updated, _ := m.Update(key("E"))
	if updated.mode != cardEditDesc {
		t.Errorf("mode = %d, want cardEditDesc", updated.mode)
	}
}

func TestKeyMoveList(t *testing.T) {
	m := newTestCardModel()

	// m enters move mode
	updated, _ := m.Update(key("m"))
	if updated.mode != cardMoveList {
		t.Errorf("mode = %d, want cardMoveList", updated.mode)
	}
	if updated.moveIndex != 0 {
		t.Errorf("moveIndex = %d, want 0", updated.moveIndex)
	}

	// j moves down
	updated, _ = updated.Update(key("j"))
	if updated.moveIndex != 1 {
		t.Errorf("after j: moveIndex = %d, want 1", updated.moveIndex)
	}

	// k moves up
	updated, _ = updated.Update(key("k"))
	if updated.moveIndex != 0 {
		t.Errorf("after k: moveIndex = %d, want 0", updated.moveIndex)
	}

	// esc cancels and resets moveIndex
	updated, _ = m.Update(key("m"))
	updated, _ = updated.Update(key("j"))
	updated, _ = updated.Update(specialKey(tea.KeyEsc))
	if updated.mode != cardView {
		t.Errorf("after esc: mode = %d, want cardView", updated.mode)
	}
	if updated.moveIndex != 0 {
		t.Errorf("after esc: moveIndex = %d, want 0 (reset to listIndex)", updated.moveIndex)
	}

	// enter on same list returns to view without cmd
	updated, _ = m.Update(key("m"))
	updated, cmd := updated.Update(specialKey(tea.KeyEnter))
	if updated.mode != cardView {
		t.Errorf("enter same list: mode = %d, want cardView", updated.mode)
	}
	if cmd != nil {
		t.Error("expected nil cmd when selecting same list")
	}
}

func TestKeyMoveShortcuts(t *testing.T) {
	m := newTestCardModel()
	m.listIndex = 1
	m.moveIndex = 1

	// , moves left — needs a client for the API call, but we just check the moveIndex
	// We can't fully test the cmd without a client, so we check that it doesn't panic
	// when client is nil by just checking it returns
	m2 := NewCardModel(nil, m.card, m.lists, 1)

	// . from middle
	updated, _ := m2.Update(key("."))
	if updated.moveIndex != 2 {
		t.Errorf("after dot: moveIndex = %d, want 2", updated.moveIndex)
	}

	// , from middle
	m2 = NewCardModel(nil, m.card, m.lists, 1)
	updated, _ = m2.Update(key(","))
	if updated.moveIndex != 0 {
		t.Errorf("after comma: moveIndex = %d, want 0", updated.moveIndex)
	}

	// < moves to first
	m2 = NewCardModel(nil, m.card, m.lists, 1)
	updated, _ = m2.Update(key("<"))
	if updated.moveIndex != 0 {
		t.Errorf("after <: moveIndex = %d, want 0", updated.moveIndex)
	}

	// > moves to last
	m2 = NewCardModel(nil, m.card, m.lists, 1)
	updated, _ = m2.Update(key(">"))
	if updated.moveIndex != 2 {
		t.Errorf("after >: moveIndex = %d, want 2", updated.moveIndex)
	}
}

func TestCardView(t *testing.T) {
	m := newTestCardModel()
	m.width = 80
	m.height = 40

	v := m.View()
	if !strings.Contains(v, "Test Card") {
		t.Error("view should contain card title")
	}
	if !strings.Contains(v, "To Do") {
		t.Error("view should contain list name")
	}
	if !strings.Contains(v, "Bug") {
		t.Error("view should contain label name")
	}
	if !strings.Contains(v, "A description") {
		t.Error("view should contain description")
	}
	if !strings.Contains(v, "Alice Smith") {
		t.Error("view should contain member name")
	}
	if !strings.Contains(v, "https://trello.com/c/abc123") {
		t.Error("view should contain card URL")
	}
}

func TestCardViewModes(t *testing.T) {
	m := newTestCardModel()
	m.width = 80
	m.height = 40

	// Edit title mode
	m.mode = cardEditTitle
	v := m.View()
	if !strings.Contains(v, "Edit title") {
		t.Error("edit title mode should show 'Edit title'")
	}

	// Edit desc mode
	m.mode = cardEditDesc
	v = m.View()
	if !strings.Contains(v, "Description (esc to save)") {
		t.Error("edit desc mode should show save instruction")
	}

	// Move list mode
	m.mode = cardMoveList
	v = m.View()
	if !strings.Contains(v, "Move to list") {
		t.Error("move list mode should show 'Move to list'")
	}

	// View mode shows help text
	m.mode = cardView
	v = m.View()
	if !strings.Contains(v, "t:edit title") {
		t.Error("view mode should show help text")
	}
}

func TestCardViewNoDescription(t *testing.T) {
	m := newTestCardModel()
	m.card.Desc = ""
	m.width = 80
	m.height = 40

	v := m.View()
	if !strings.Contains(v, "no description") {
		t.Error("view should show 'no description' placeholder")
	}
}
