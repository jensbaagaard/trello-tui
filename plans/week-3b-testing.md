# Week 3b: Testing — Close Critical Gaps

Priority: P0 test coverage gaps from the Test Engineer review.

---

## Current State

- 7 of 25 Go files have tests (28%)
- ~25-30% estimated line coverage
- Zero tests for: `board_keys.go`, `card_keys.go`, `*_view.go`, `search.go`, `main.go`
- Good test patterns exist in: `config_test.go`, `version_test.go`, `client_test.go`

---

## 1. Test keyboard handlers (P0 gap)

**Files to create/extend**: `internal/tui/board_keys_test.go`, `internal/tui/card_keys_test.go`

### Board keyboard tests

Focus on card movement (most critical user-facing operations):

```go
func TestBoardKeys_DotMovesCardRight(t *testing.T)
func TestBoardKeys_CommaMovesCardLeft(t *testing.T)
func TestBoardKeys_CommaAtFirstList_NoOp(t *testing.T)
func TestBoardKeys_DotAtLastList_NoOp(t *testing.T)
func TestBoardKeys_LessThanMovesToFirstList(t *testing.T)
func TestBoardKeys_GreaterThanMovesToLastList(t *testing.T)
func TestBoardKeys_N_EntersAddCardMode(t *testing.T)
func TestBoardKeys_C_EntersConfirmArchiveMode(t *testing.T)
func TestBoardKeys_Slash_EntersFilterMode(t *testing.T)
func TestBoardKeys_A_EntersArchiveMode(t *testing.T)
func TestBoardKeys_L_EntersLabelManager(t *testing.T)
func TestBoardKeys_M_EntersMemberManager(t *testing.T)
func TestBoardKeys_BigN_EntersAddListMode(t *testing.T)
func TestBoardKeys_BigR_EntersRenameListMode(t *testing.T)
func TestBoardKeys_EscFromFilter_ClearsFilter(t *testing.T)
func TestBoardKeys_EscFromNav_GoesBack(t *testing.T)
```

### Card keyboard tests

```go
func TestCardKeys_T_EntersEditTitle(t *testing.T)
func TestCardKeys_E_EntersEditDesc(t *testing.T)
func TestCardKeys_D_EntersSetDue(t *testing.T)
func TestCardKeys_M_EntersMoveList(t *testing.T)
func TestCardKeys_B_EntersMoveBoard(t *testing.T)
func TestCardKeys_A_EntersAddMember(t *testing.T)
func TestCardKeys_L_EntersAddLabel(t *testing.T)
func TestCardKeys_Tab_CyclesPanes(t *testing.T)
func TestCardKeys_EscFromEditTitle_ReturnsToView(t *testing.T)
func TestCardKeys_CommaMovesCardLeft(t *testing.T)
func TestCardKeys_DotMovesCardRight(t *testing.T)
```

### Test pattern

Use the existing `key()` helper and `newTestBoardModel()` / `newTestCardModel()`:

```go
func TestBoardKeys_DotMovesCardRight(t *testing.T) {
    m := newTestBoardModel()
    // Setup: 2 lists, card on first list, activeList=0
    m.lists = []trello.List{{ID: "l1"}, {ID: "l2"}}
    m.cardsByList = map[string][]trello.Card{
        "l1": {{ID: "c1", IDList: "l1"}},
        "l2": {},
    }
    m.activeList = 0
    m.activeCard = 0

    updated, cmd := m.handleKey(key("."))
    // Verify a move command was returned
    if cmd == nil {
        t.Fatal("expected move command, got nil")
    }
    _ = updated
}
```

---

## 2. Add client.go error path tests

**File**: `internal/trello/client_test.go`

### Tests to add

```go
func TestGetBoards_NetworkError(t *testing.T)       // server down
func TestGetBoards_MalformedJSON(t *testing.T)      // invalid response body
func TestGetBoards_ServerError(t *testing.T)        // 500 response
func TestCreateCard_ValidationError(t *testing.T)   // 400 response
func TestDownloadAttachment_WriteFails(t *testing.T) // disk full simulation
func TestSearchCards_EmptyResults(t *testing.T)      // valid but empty
```

Use `httptest.NewServer` with handlers that return error conditions.

---

## 3. Test search flow

**File to create**: `internal/tui/search_test.go`

```go
func TestSearchModel_InitialState(t *testing.T)        // input focused, no results
func TestSearchModel_EnterExecutesSearch(t *testing.T)  // sends search command
func TestSearchModel_NavigateResults(t *testing.T)      // j/k moves cursor
func TestSearchModel_EnterOpensCard(t *testing.T)       // returns open card msg
func TestSearchModel_EscGoesBack(t *testing.T)          // returns back msg
func TestSearchModel_SlashRefocusesInput(t *testing.T)  // / re-focuses input
func TestSearchModel_ScrollBounds(t *testing.T)         // cursor doesn't exceed results
```

---

## 4. Extract named constants for magic values

**Severity**: MEDIUM (AI Optimizer finding, also helps tests)
**File**: `internal/tui/styles.go` (add constants section)

### Changes

Add a constants block at the top of a new or existing file:

```go
// Layout constants
const (
    MinColumnWidth = 36
    CardBorderWidth = 2

    // Pane height percentages for card detail view
    InfoPanePercent4Pane      = 35
    ChecklistPanePercent4Pane = 25
    AttachmentPanePercent4Pane = 20

    InfoPanePercent3PaneCL  = 40
    ChecklistPanePercent3CL = 30

    InfoPanePercent3PaneAtt = 45
    AttachmentPanePercent3  = 25

    InfoPanePercent2Pane = 55

    MinInfoPaneHeight = 9
    MinSubPaneHeight  = 6
)
```

Update `card_view.go` and `board.go` to reference these constants instead of
inline numbers.

---

## Checklist

- [ ] Create `board_keys_test.go` with 15+ keyboard handler tests
- [ ] Create `card_keys_test.go` with 10+ keyboard handler tests
- [ ] Add 6+ error path tests to `client_test.go`
- [ ] Create `search_test.go` with 7+ search flow tests
- [ ] Extract layout magic values to named constants
- [ ] Update `card_view.go` to use named constants
- [ ] Update `board.go` to use `MinColumnWidth` constant (exported)
- [ ] All tests pass: `go test ./...`
