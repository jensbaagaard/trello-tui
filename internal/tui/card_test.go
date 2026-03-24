package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jensbaagaard/trello-tui/internal/trello"
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

func TestAttachmentsFetchedMsg(t *testing.T) {
	m := newTestCardModel()
	m.loadingAtt = true

	attachments := []trello.Attachment{
		{ID: "a1", Name: "file.pdf", Bytes: 1024, IsUpload: true},
		{ID: "a2", Name: "link", Bytes: 0, IsUpload: false},
	}

	updated, _ := m.Update(AttachmentsFetchedMsg{Attachments: attachments})
	if updated.loadingAtt {
		t.Error("loadingAtt should be false after fetch")
	}
	if len(updated.attachments) != 2 {
		t.Fatalf("len(attachments) = %d, want 2", len(updated.attachments))
	}
	if updated.attachments[0].Name != "file.pdf" {
		t.Errorf("Name = %q, want %q", updated.attachments[0].Name, "file.pdf")
	}
}

func TestAttachmentsFetchedError(t *testing.T) {
	m := newTestCardModel()
	m.loadingAtt = true

	updated, _ := m.Update(AttachmentsFetchedMsg{Err: fmt.Errorf("network error")})
	if updated.loadingAtt {
		t.Error("loadingAtt should be false after error")
	}
	if !strings.Contains(updated.statusMsg, "Error fetching attachments") {
		t.Errorf("statusMsg = %q, want it to contain 'Error fetching attachments'", updated.statusMsg)
	}
}

func TestAttachmentsPaneNavigation(t *testing.T) {
	m := newTestCardModel()
	m.loadingAtt = false
	m.attachments = []trello.Attachment{
		{ID: "a1", Name: "file1.pdf"},
		{ID: "a2", Name: "file2.png"},
		{ID: "a3", Name: "file3.txt"},
	}
	m.mode = cardAttachmentsPane

	// j moves down
	updated, _ := m.Update(key("j"))
	if updated.attachmentIdx != 1 {
		t.Errorf("after j: attachmentIdx = %d, want 1", updated.attachmentIdx)
	}

	// j again
	updated, _ = updated.Update(key("j"))
	if updated.attachmentIdx != 2 {
		t.Errorf("after j j: attachmentIdx = %d, want 2", updated.attachmentIdx)
	}

	// j at end stays
	updated, _ = updated.Update(key("j"))
	if updated.attachmentIdx != 2 {
		t.Errorf("after j at end: attachmentIdx = %d, want 2", updated.attachmentIdx)
	}

	// k moves up
	updated, _ = updated.Update(key("k"))
	if updated.attachmentIdx != 1 {
		t.Errorf("after k: attachmentIdx = %d, want 1", updated.attachmentIdx)
	}

	// esc returns to card view
	updated, _ = updated.Update(specialKey(tea.KeyEsc))
	if updated.mode != cardView {
		t.Errorf("after esc: mode = %d, want cardView", updated.mode)
	}
}

func TestAttachmentsPaneTabCycling(t *testing.T) {
	m := newTestCardModel()
	m.loadingAtt = false
	m.loadingCL = false
	m.attachments = []trello.Attachment{
		{ID: "a1", Name: "file.pdf"},
	}
	m.checklists = []trello.Checklist{
		{ID: "cl1", Name: "Checklist", CheckItems: []trello.CheckItem{{ID: "ci1", Name: "Item"}}},
	}

	// Info → tab → Checklist (has checklists)
	updated, _ := m.Update(specialKey(tea.KeyTab))
	if updated.mode != cardChecklistPane {
		t.Errorf("from info: mode = %d, want cardChecklistPane", updated.mode)
	}

	// Checklist → tab → Attachments (has attachments)
	updated, _ = updated.Update(specialKey(tea.KeyTab))
	if updated.mode != cardAttachmentsPane {
		t.Errorf("from checklist: mode = %d, want cardAttachmentsPane", updated.mode)
	}

	// Attachments → tab → Activity
	updated, _ = updated.Update(specialKey(tea.KeyTab))
	if updated.mode != cardActivityPane {
		t.Errorf("from attachments: mode = %d, want cardActivityPane", updated.mode)
	}

	// Activity → tab → Info (cardView)
	updated, _ = updated.Update(specialKey(tea.KeyTab))
	if updated.mode != cardView {
		t.Errorf("from activity: mode = %d, want cardView", updated.mode)
	}
}

func TestTabCyclingSkipsEmptyAttachments(t *testing.T) {
	m := newTestCardModel()
	m.loadingAtt = false
	m.loadingCL = false
	m.attachments = nil
	m.checklists = []trello.Checklist{
		{ID: "cl1", Name: "Checklist", CheckItems: []trello.CheckItem{{ID: "ci1", Name: "Item"}}},
	}

	// Info → tab → Checklist
	updated, _ := m.Update(specialKey(tea.KeyTab))
	if updated.mode != cardChecklistPane {
		t.Errorf("from info: mode = %d, want cardChecklistPane", updated.mode)
	}

	// Checklist → tab → Activity (skip attachments since empty)
	updated, _ = updated.Update(specialKey(tea.KeyTab))
	if updated.mode != cardActivityPane {
		t.Errorf("from checklist: mode = %d, want cardActivityPane (skip empty attachments)", updated.mode)
	}
}

func TestAttachmentsPaneRendering(t *testing.T) {
	m := newTestCardModel()
	m.width = 80
	m.height = 50
	m.loadingAtt = false
	m.loadingCL = false
	m.attachments = []trello.Attachment{
		{ID: "a1", Name: "report.pdf", Bytes: 1048576},
		{ID: "a2", Name: "image.png", Bytes: 2048},
	}

	v := m.View()
	if !strings.Contains(v, "Attachments") {
		t.Error("view should contain 'Attachments' pane title")
	}
	if !strings.Contains(v, "report.pdf") {
		t.Error("view should contain attachment name 'report.pdf'")
	}
	if !strings.Contains(v, "image.png") {
		t.Error("view should contain attachment name 'image.png'")
	}
}

func TestAttachmentOpenedError(t *testing.T) {
	m := newTestCardModel()
	updated, _ := m.Update(AttachmentOpenedMsg{Err: fmt.Errorf("open failed")})
	if !strings.Contains(updated.statusMsg, "Error opening attachment") {
		t.Errorf("statusMsg = %q, want it to contain 'Error opening attachment'", updated.statusMsg)
	}
}
