# Week 2a: UX Quick Wins

Priority: UX-BUGs and accessibility issues from the TUI/UX review.

---

## 1. Document all keybindings in help screens

**Severity**: UX-BUG
**Files**: `internal/tui/card_view.go:603-638`

### Problem

Several keybindings are functional but not shown in the `?` help overlay:
- `B` (move card to another board) — works but not in card help
- `ctrl+n` (create new label from picker) — works but not in card help
- `?` itself — only hinted in status bar, not in help sections

### Changes

**card_view.go — `renderCardHelp()`**:

Add the missing entries to the help sections:

```go
{Title: "Info Pane", Entries: []helpEntry{
    // ... existing entries ...
    {"B", "Move to another board"},
}},
{Title: "Labels", Entries: []helpEntry{
    // ... existing entries ...
    {"ctrl+n", "Create new label"},
}},
```

Also add `?` / toggle help entry to every help section's footer if not present.

---

## 2. Add due date icons to board view (accessibility)

**Severity**: ACCESSIBILITY-BUG
**Files**: `internal/tui/board_view.go:396-399`

### Problem

In card detail view, due dates show icons: `✓` (done), `⚠` (overdue/soon). But on
the board's card rendering, due dates are color-only — colorblind users cannot
distinguish states.

### Changes

**board_view.go — `renderCard()` (line 396-399)**:

The `formatDue()` function already returns icon-prefixed labels (`"✓ 5 Jan"`,
`"⚠ 3 Mar"`). It's already being used here:

```go
if label, style := formatDue(c.Due, c.DueComplete); label != "" {
    parts = append(parts, style.Render(label))
}
```

Verify this renders the icons on the board. If `formatDue` strips icons for board
context, update it to include them consistently.

Also make label pills consistent — currently board uses `━━` (line 392) while card
detail uses `●` (card_view.go:270). Standardize to `●`:

```go
// board_view.go line 392 — change from:
pills = append(pills, labelColor(l.Color).Render("━━"))
// to:
pills = append(pills, labelColor(l.Color).Render("●"))
```

---

## 3. Fix search cursor preservation on return from card

**Severity**: UX-BUG
**Files**: `internal/tui/app.go:206-224`, `internal/tui/search.go`

### Problem

User searches, scrolls to result #15, opens the card. On pressing `esc` to return,
the search view resets cursor to position 0. The `cursor` and `scrollTop` fields
exist on `SearchModel` but aren't preserved across the card→search transition.

### Changes

In `app.go`, when returning from card to search, the search model is already
preserved (line 214-216):

```go
m.screen = screenSearch
return m, nil
```

Verify that `m.search.cursor` and `m.search.scrollTop` are not reset elsewhere.
If `pendingCard`/`pendingLists` cleanup resets them, fix that:

```go
m.search.pendingCard = nil
m.search.pendingLists = nil
// Do NOT reset m.search.cursor or m.search.scrollTop
```

### Test to add

- `TestSearchCursorPreservedAfterCardView` — set cursor=5, open card, return, assert cursor=5

---

## 4. Truncate long card names in search results

**Severity**: UX-BUG
**File**: `internal/tui/search_view.go:48-72`

### Problem

Card names in search results aren't truncated. A 200-character card name (the
CharLimit) consumes the entire terminal width, breaking layout.

### Changes

**search_view.go — result rendering (line 66)**:

```go
// Calculate available width for card name
maxNameW := m.width - len(prefix) - len(boardName) - len(listName) - 10
if maxNameW < 20 {
    maxNameW = 20
}
cardName := truncate(card.Name, maxNameW)
line := fmt.Sprintf("%s%s  [%s] -> %s", prefix, cardName, boardName, listName)
```

The `truncate()` helper already exists in `board_view.go:456-468`.

---

## Checklist

- [ ] Add `B`, `ctrl+n` to card help overlay
- [ ] Add `?` hint to all help section footers
- [ ] Standardize label pills to `●` across board and card views
- [ ] Verify due date icons render on board cards (accessibility)
- [ ] Preserve search cursor/scroll on return from card view
- [ ] Truncate long card names in search results
- [ ] Run `go test ./...` pass
