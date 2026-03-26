# Week 1b: Security — Hardening & Error Recovery UX

Priority: MEDIUM security findings + critical UX-BUG for error recovery.

---

## 1. Add HTTP client timeouts to all API calls

**Severity**: MEDIUM
**Files**: `internal/trello/client.go:20-27`

### Problem

The HTTP client is created with no timeout: `httpClient: &http.Client{}`. A slow or
unresponsive Trello API will hang the TUI indefinitely. The only place with a timeout
is `internal/version/version.go:19` (5s for version check).

### Changes

**client.go — `NewClient()`**:

```go
func NewClient(apiKey, token string) *Client {
    return &Client{
        apiKey: apiKey,
        token:  token,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                TLSClientConfig: &tls.Config{
                    MinVersion: tls.VersionTLS12,
                },
            },
        },
        baseURL: "https://api.trello.com/1",
    }
}
```

This also adds explicit TLS 1.2 minimum (another MEDIUM finding from SecOps review).

### Tests to add

- `TestNewClient_HasTimeout` — verify httpClient.Timeout is set
- `TestGet_TimesOut` — use httptest server with delayed response

---

## 2. Handle config file permission `os.Chmod` error

**Severity**: LOW
**File**: `internal/config/config.go:38`

### Problem

```go
_ = os.Chmod(configPath, 0o600)
```

The Chmod error is silently discarded. If permission fix fails (e.g., file owned by
another user), the user is never warned that their config remains world-readable.

### Changes

```go
if chmodErr := os.Chmod(configPath, 0o600); chmodErr != nil {
    fmt.Fprintf(os.Stderr, "Warning: could not fix permissions on %s: %v\n", configPath, chmodErr)
}
```

---

## 3. Add error recovery hints to all error display states

**Severity**: UX-BUG
**Files**: `internal/tui/board_view.go:45-46`, `internal/tui/search_view.go:27-29`

### Problem

When board loading fails, the user sees a bare error with no guidance:
```
Error: request failed: connection refused
```

No hint to press `r` to retry or `esc` to go back.

### Changes

**board_view.go — `View()` error display** (line 45-46):

```go
if m.err != nil {
    return errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) +
        "\n\n" + helpStyle.Render("r:retry  esc:back to boards")
}
```

**board_view.go — empty lists** (line 51-53):

```go
if len(m.lists) == 0 {
    return "No lists found on this board.\n\n" +
        helpStyle.Render("N:new list  r:refresh  esc:back")
}
```

**search_view.go — error display** (line 27-29):

```go
if m.statusMsg != "" {
    b.WriteString(errorStyle.Render(m.statusMsg))
    b.WriteString("\n")
    b.WriteString(helpStyle.Render("/:try again  esc:back"))
    b.WriteString("\n")
}
```

**search_view.go — empty results** (line 33-35):

```go
} else if len(m.results) == 0 {
    b.WriteString(helpStyle.Render("No results — try a different query"))
    b.WriteString("\n")
}
```

---

## 4. Fix archived cards showing raw list ID for deleted lists

**Severity**: UX-IMPROVEMENT
**File**: `internal/tui/board_view.go:290-296`

### Problem

When a card's original list has been deleted, the archive view shows the raw Trello
list UUID instead of a human-readable fallback.

### Changes

```go
listName := card.IDList
for _, l := range m.lists {
    if l.ID == card.IDList {
        listName = l.Name
        break
    }
}
if listName == card.IDList {
    listName = "(deleted list)"
}
```

---

## Checklist

- [ ] HTTP client timeout (30s) + TLS 1.2 minimum
- [ ] Handle `os.Chmod` error in config.go
- [ ] Error recovery hints on board error state
- [ ] Error recovery hints on empty board state
- [ ] Error recovery hints on search error/empty states
- [ ] Archived card list name fallback
- [ ] Tests for timeout behavior
- [ ] Run `go vet ./...` and `go test ./...` pass
