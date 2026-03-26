# Trello TUI — Architecture Guide

## Overview

Terminal-based Trello client built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Model-Update-View pattern). All user interaction is keyboard-driven; all API calls are async via `tea.Cmd` closures.

## Screen Flow

```
BoardList ──enter──> Board ──enter──> Card (with panes)
    │                  ↑                 │
    │                  └───esc───────────┘
    │
    └──s──> Search ──enter──> Card
              ↑                 │
              └───esc───────────┘
```

## Package Structure

```
internal/
├── trello/        API client + domain types (zero TUI knowledge)
│   ├── client.go  HTTP methods (GetBoards, CreateCard, etc.)
│   └── types.go   Board, List, Card, Member, Label, etc.
├── tui/           Bubble Tea models, views, key handlers
│   ├── app.go         Root model — screen routing + navigation
│   ├── boardlist.go   Board selection screen (uses bubbles/list)
│   ├── board.go       Kanban board model + commands
│   ├── board_keys.go  Board keyboard handlers
│   ├── board_view.go  Board rendering (columns, cards, help)
│   ├── card.go        Card detail model + commands
│   ├── card_keys.go   Card keyboard handlers
│   ├── card_view.go   Card rendering (multi-pane layout)
│   ├── search.go      Global card search model
│   ├── search_view.go Search rendering
│   ├── messages.go    All typed message structs (43+)
│   ├── styles.go      Colors, lipgloss styles, label/member rendering
│   ├── help.go        Help overlay renderer
│   └── options.go     CLI option struct
├── config/        Credential loading (env vars > config file)
└── version/       GitHub release version check
```

## Key Patterns

### Message-based async I/O

All API calls return `tea.Cmd` closures that produce typed messages:

```go
func (m BoardModel) fetchLists() tea.Cmd {
    client := m.client
    boardID := m.board.ID
    return func() tea.Msg {
        lists, err := client.GetLists(boardID)
        return ListsFetchedMsg{Lists: lists, Err: err}
    }
}
```

Messages are defined in `messages.go`. Each carries both data and error.

### State machines via mode enums

Each screen has a `mode` enum that controls which keys are active and what renders:

- `boardMode` (17 states): `boardNav`, `boardAddCard`, `boardFilter`, `boardArchive`, `boardLabelManager`, etc.
- `cardMode` (20 states): `cardView`, `cardEditTitle`, `cardMoveList`, `cardChecklistPane`, etc.
- `boardListMode` (3 states): `boardListNav`, `boardListCreate`, `boardListWorkspacePick`

### File split per screen

Each screen splits into up to 3 files:
- `model.go` — struct, constructor, Update(), commands, helpers
- `_keys.go` — `handleKey(msg tea.KeyMsg)` with mode-specific switch
- `_view.go` — `View() string` rendering logic

### Card pane system

Card detail view has multiple panes rendered vertically:
1. **Info** (always) — title, description, labels, members, due date
2. **Checklist** (if any exist) — interactive items with toggle
3. **Attachments** (if any exist) — file list with open/delete
4. **Activity** (always) — interlaced comments and actions

Tab cycles focus between panes. Each pane has its own scroll position.

## Adding Features

1. **Define message type** in `messages.go`
2. **Add command function** in the relevant model (returns `tea.Cmd`)
3. **Handle message** in `Update()` — update state, return next command
4. **Add rendering** in the `_view.go` file
5. **Add keybinding** in the `_keys.go` file
6. **Add help entry** in the help section of the `_view.go` file
7. **Update README.md** keybinding table if user-facing

## Running & Testing

```bash
go build ./...       # compile
go test ./...        # run all tests
go vet ./...         # static analysis
```

Test patterns: httptest servers for API mocking, `key()` helper for simulating keypresses, `newTestBoardModel()` / `newTestCardModel()` for model construction.

## Configuration

Credentials loaded by priority:
1. Environment variables: `TRELLO_API_KEY`, `TRELLO_TOKEN`
2. Config file: `~/.config/trello-tui/config.json` (Linux) / `~/Library/Application Support/trello-tui/config.json` (macOS)

Config file permissions are auto-fixed to `0o600` if too permissive.

## CLI Flags

- `--board=<name>` — jump directly to a board (substring match)
- `--auto-refresh=<seconds>` — re-fetch board data on a timer
- `--version` — print version and exit
