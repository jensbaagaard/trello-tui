# Week 4: LLM Optimization & Code Quality Polish

Priority: Making the codebase more effective for LLM-assisted development.

---

## 1. Refactor large Update() methods into per-message handlers

**Severity**: MEDIUM (AI reviewer rated function granularity 4/10)
**Files**: `internal/tui/board.go`, `internal/tui/card.go`

### Problem

`BoardModel.Update()` and `CardModel.Update()` are 200-400+ line switch statements.
An LLM must read the entire function to understand any single message handler.

### Changes

**card.go** — Extract each message case into a named method:

```go
func (m CardModel) Update(msg tea.Msg) (CardModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        return m.handleWindowSize(msg)
    case CardUpdatedMsg:
        return m.handleCardUpdated(msg)
    case CardMovedMsg:
        return m.handleCardMoved(msg)
    case BoardMembersFetchedMsg:
        return m.handleBoardMembersFetched(msg)
    case BoardLabelsFetchedMsg:
        return m.handleBoardLabelsFetched(msg)
    // ... etc for all message types
    case tea.KeyMsg:
        return m.handleKey(msg)
    }
    return m.forwardToActiveInput(msg)
}

func (m CardModel) handleCardUpdated(msg CardUpdatedMsg) (CardModel, tea.Cmd) {
    if msg.Err != nil {
        m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)
        return m, nil
    }
    members := m.card.Members
    labels := m.card.Labels
    m.card = msg.Card
    if len(m.card.Members) == 0 {
        m.card.Members = members
    }
    if len(m.card.Labels) == 0 {
        m.card.Labels = labels
    }
    m.statusMsg = "Saved"
    return m, nil
}
```

Each handler becomes a focused 5-20 line method that's easy for an LLM to
understand and modify independently.

**board.go** — Same pattern. Extract the `Update()` switch into named handlers.
The `handleKey()` extraction is already done (in `board_keys.go`), so this
focuses on message handlers only.

### Scope

- Extract message handlers only (not the tick-forwarding logic)
- Keep the main `Update()` as a router
- Each extracted method stays in the same file (don't create new files)
- The tick-forwarding block at the bottom of `card.go:416-446` can be extracted
  to a `forwardToActiveInput(msg tea.Msg)` method

---

## 2. Define structured API error type

**Severity**: MEDIUM
**File**: `internal/trello/client.go`

### Problem

API errors are plain `fmt.Errorf` strings. Code that needs to distinguish a 401
from a 404 must parse the error string.

### Changes

```go
// APIError represents a non-OK response from the Trello API.
type APIError struct {
    StatusCode int
    Endpoint   string
    Body       string
}

func (e *APIError) Error() string {
    switch e.StatusCode {
    case 401:
        return "authentication failed — check your API key and token"
    case 403:
        return "access denied"
    case 404:
        return "resource not found"
    case 429:
        return "rate limited — please wait and try again"
    default:
        body := e.Body
        if len(body) > 200 {
            body = body[:200] + "..."
        }
        return fmt.Sprintf("API error %d: %s", e.StatusCode, body)
    }
}
```

Update `get()` and `request()` to return `&APIError{...}` instead of `fmt.Errorf`.

This also fulfills week-1a's "strip credentials from errors" if not yet done,
since the error message no longer includes raw response bodies for auth failures.

### Tests to add

- `TestAPIError_Error_401` — returns friendly auth message
- `TestAPIError_Error_TruncatesLongBody`
- `TestAPIError_CanUnwrap` — `errors.As(err, &apiErr)` works

---

## 3. Consider typed IDs (research / optional)

**Severity**: LOW (nice-to-have for type safety)
**Files**: `internal/trello/types.go`, `internal/trello/client.go`

### Problem

All IDs are `string` — easy to accidentally swap a `cardID` with a `listID`:

```go
func (c *Client) MoveCardToBoard(cardID, boardID, listID string) // three strings!
```

### Approach

This is a larger refactor. Evaluate the cost/benefit:

**Pros**: Compile-time safety, clearer function signatures, LLM generates correct arg order.
**Cons**: Verbose, breaks all callers, JSON unmarshaling needs custom types.

**Recommendation**: If time allows, start with a proof-of-concept for `CardID` and
`ListID` only. If the ergonomics are good, expand in a future sprint.

```go
type CardID string
type ListID string
type BoardID string

func (c *Client) MoveCard(cardID CardID, listID ListID) (Card, error)
```

JSON works automatically since the underlying type is `string`.

**Decision**: Mark as "future consideration" unless the team finds value now.

---

## 4. Add table-driven tests as executable documentation

**Severity**: MEDIUM
**Files**: `internal/tui/board_test.go`, `internal/tui/card_test.go`

### Problem

Existing tests are individual functions. Table-driven tests serve as documentation
of expected behavior across multiple scenarios.

### Changes

Convert filter tests to table-driven format:

```go
func TestMatchesFilter(t *testing.T) {
    tests := []struct {
        name   string
        card   trello.Card
        query  string
        want   bool
    }{
        {"matches card name", trello.Card{Name: "Fix login bug"}, "login", true},
        {"case insensitive", trello.Card{Name: "Fix LOGIN"}, "login", true},
        {"no match", trello.Card{Name: "Unrelated"}, "login", false},
        {"matches description", trello.Card{Desc: "auth flow"}, "auth", true},
        {"matches member name", trello.Card{Members: []trello.Member{{FullName: "Alice"}}}, "alice", true},
        {"matches label name", trello.Card{Labels: []trello.Label{{Name: "bug"}}}, "bug", true},
        {"empty query matches all", trello.Card{Name: "anything"}, "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := matchesFilter(tt.card, tt.query)
            if got != tt.want {
                t.Errorf("matchesFilter(%q, %q) = %v, want %v", tt.card.Name, tt.query, got, tt.want)
            }
        })
    }
}
```

Also add table-driven tests for:
- `formatDue()` — various date states (overdue, soon, far, complete)
- `truncate()` — edge cases (empty, exact, longer, Unicode)
- `memberInitials()` — various name formats

---

## Checklist

- [ ] Extract `CardModel.Update()` message handlers to named methods
- [ ] Extract `BoardModel.Update()` message handlers to named methods
- [ ] Define `APIError` struct type in client.go
- [ ] Update `get()` and `request()` to return `*APIError`
- [ ] Tests for `APIError` behavior
- [ ] Convert `matchesFilter` tests to table-driven
- [ ] Convert `formatDue` tests to table-driven
- [ ] Evaluate typed IDs — document decision
- [ ] All tests pass: `go test ./...`
- [ ] Run `go vet ./...` clean
