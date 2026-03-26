# Week 2b: Go Performance & Idiom Improvements

Priority: OPTIMIZATION and ANTI-PATTERN findings from the Go expert review.

---

## 1. Extract `labelColor` map to package-level variable

**Severity**: OPTIMIZATION
**File**: `internal/tui/styles.go:93-111`

### Problem

`labelColor()` recreates the color lookup map on every call. This function is called
per-label per-card per-frame — potentially hundreds of times per render.

### Changes

```go
// Package-level map (created once)
var labelColorMap = map[string]lipgloss.Color{
    "green":  lipgloss.Color("#10B981"),
    "yellow": lipgloss.Color("#F59E0B"),
    "orange": lipgloss.Color("#F97316"),
    "red":    lipgloss.Color("#EF4444"),
    "purple": lipgloss.Color("#8B5CF6"),
    "blue":   lipgloss.Color("#3B82F6"),
    "sky":    lipgloss.Color("#0EA5E9"),
    "lime":   lipgloss.Color("#84CC16"),
    "pink":   lipgloss.Color("#EC4899"),
    "black":  lipgloss.Color("#374151"),
}

var defaultLabelColor = lipgloss.Color("#6B7280")

func labelColor(color string) lipgloss.Style {
    c, ok := labelColorMap[color]
    if !ok {
        c = defaultLabelColor
    }
    return labelStyle.Foreground(c).Bold(true)
}
```

---

## 2. Use `strings.Builder` in View() hot paths

**Severity**: OPTIMIZATION
**Files**: `internal/tui/board_view.go:129-133`, `internal/tui/board_view.go:386-446`

### Problem

String concatenation with `+` in View methods creates unnecessary allocations.
View is called on every frame, making this a hot path.

### Changes

**board_view.go — `View()` status/footer area (lines 101-133)**:

Replace the chain of `if/else if` string concatenation with a `strings.Builder`:

```go
var b strings.Builder
b.WriteString(titleStyle.Render(m.board.Name))
if scrollHint != "" {
    b.WriteString(scrollHint)
}
b.WriteString("\n")
b.WriteString(board)
b.WriteString("\n")
b.WriteString(status)
return b.String()
```

**board_view.go — `renderCard()` (lines 386-446)**:

The `renderCard` function uses multiple intermediate string variables (`topLine`,
`content`, `bottomParts`). Convert to `strings.Builder` for fewer allocations.

### Scope

Only convert the most frequently called render functions. Don't touch functions
called once per screen (like help overlays).

---

## 3. Cache parsed dates in filter hot path

**Severity**: OPTIMIZATION
**File**: `internal/tui/board.go:712-740` (`matchesFilter`)

### Problem

`time.Parse()` is called for every card on every filter keystroke:

```go
if t, err := time.Parse(time.RFC3339Nano, c.Due); err == nil {
    if strings.Contains(strings.ToLower(t.Format("2 Jan 2006")), q) {
```

### Changes

Since the filter function runs on every keystroke, pre-compute the lowercase query
once (already done) but also avoid re-parsing dates. Two options:

**Option A** (simpler): Cache the formatted date string in a local map during
filter. Since `matchesFilter` is called per-card:

```go
// In the filtering loop, outside matchesFilter:
// No structural change needed — time.Parse is fast enough for ~100 cards.
// Only optimize if profiling shows this as a bottleneck.
```

**Option B** (if profiling shows need): Add a `FormattedDue string` field to the
Card struct, populated on fetch. This avoids repeated parsing.

**Recommendation**: Skip this unless profiling shows it matters. Mark as "profile first".

---

## 4. Fix unsafe type assertion in boardlist.go

**Severity**: ANTI-PATTERN
**File**: `internal/tui/boardlist.go:290`

### Problem

```go
bi := item.(boardItem)  // panics if item is nil or wrong type
```

### Changes

```go
bi, ok := item.(boardItem)
if !ok {
    return nil
}
return &bi.board
```

---

## 5. Use consistent error wrapping with `%w`

**Severity**: ANTI-PATTERN
**File**: `internal/trello/client.go:53, 81, 231`

### Problem

API errors use `%s` formatting instead of `%w` wrapping:

```go
return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
```

This breaks `errors.Is()` and `errors.As()` chains.

### Changes

If implementing the `apiError` helper from week-1a, this is already addressed.
Otherwise, wrap errors with `%w` where the body is itself an error:

```go
return fmt.Errorf("API error %d: %w", resp.StatusCode, errors.New(string(body)))
```

Or better, define a structured `APIError` type (see week 4).

---

## Checklist

- [x] Extract `labelColor` map to package level
- [x] Convert `renderCard()` and `View()` footer to `strings.Builder`
- [x] Fix unsafe type assertion in `boardlist.go`
- [x] Consistent error wrapping with `%w` — already addressed by `apiError()` helper in week-1a
- [ ] ~~Profile `matchesFilter`~~ — skipped per plan recommendation (profile first)
- [x] Run `go vet ./...` and `go test ./...` pass
