# Week 3a: Documentation & Code Comments

Priority: LLM-friendliness and onboarding — highest ROI from the AI Code Optimizer review.

---

## 1. Create CLAUDE.md architecture guide

**Severity**: HIGH (rated 1/10 for documentation by AI reviewer)
**File**: `CLAUDE.md` (new, project root)

### Problem

No architecture documentation exists. LLMs (and new contributors) must reverse-engineer
the entire state machine, message flow, and pane system from code.

### Content to include

```markdown
# Trello TUI — Architecture Guide

## Overview
Terminal Trello client using Bubble Tea (Model-Update-View).

## Screen Flow
BoardList -> Board -> Card (with panes)
         \-> Search -> Card

## Package Structure
- internal/trello/ — API client + domain types (zero TUI knowledge)
- internal/tui/    — Bubble Tea models, views, key handlers
- internal/config/ — Credential loading (env vars > config file)
- internal/version/ — GitHub release version check

## Key Patterns
- All I/O is async via tea.Cmd closures
- Messages carry data + error in typed structs (messages.go)
- Each screen has a mode enum controlling sub-states
- Views split into model.go / _keys.go / _view.go per screen
- Pane system: card detail has Info + optional Checklist/Attachments + Activity

## State Machines
Board modes: boardNav, boardAddCard, boardFilter, boardArchive, ...
Card modes:  cardView, cardEditTitle, cardEditDesc, cardMoveList, ...

## Adding Features
1. Define message type in messages.go
2. Add command function in the relevant model
3. Handle message in Update()
4. Add rendering in the _view.go file
5. Add keybinding in the _keys.go file

## Running Tests
go test ./...
```

---

## 2. Document all state machine modes with comments

**Severity**: HIGH
**Files**: `internal/tui/board.go:15-35`, `internal/tui/card.go:15-38`, `internal/tui/boardlist.go:24-28`

### Problem

35+ mode constants with zero documentation. An LLM reading `boardLabelColorPick`
cannot distinguish it from `boardLabelCreate` without tracing all usages.

### Changes

**board.go**:

```go
type boardMode int

const (
    boardNav                 boardMode = iota // Default: navigating lists and cards
    boardAddCard                              // Typing new card name (textInput active)
    boardConfirmArchive                       // Awaiting y/n to archive selected card
    boardFilter                               // Typing filter query (textInput active)
    boardLabelManager                         // Browsing board labels list
    boardLabelCreate                          // Typing new label name
    boardLabelEdit                            // Editing existing label name
    boardLabelColorPick                       // Selecting color for label (cursor on color list)
    boardLabelConfirmDelete                   // Awaiting y/n to delete label
    boardArchive                              // Browsing archived cards
    boardArchiveFilter                        // Typing filter in archive view
    boardAddList                              // Typing new list name
    boardRenameList                           // Typing new name for current list
    boardConfirmArchiveList                   // Awaiting y/n to archive current list
    boardMemberManager                        // Browsing board members
    boardInviteMember                         // Typing email to invite
    boardConfirmRemoveMember                  // Awaiting y/n to remove member
)
```

**card.go**:

```go
type cardMode int

const (
    cardView                    cardMode = iota // Default: viewing card with pane tabs
    cardEditTitle                               // Editing card title (textInput active)
    cardEditDesc                                // Editing description (textarea active)
    cardMoveList                                // Selecting target list with cursor
    cardAddMember                               // Toggling members on/off with filter
    cardAddLabel                                // Toggling labels on/off with filter
    cardSetDue                                  // Typing due date (textInput active)
    cardCreateLabel                             // Typing name for new label
    cardCreateLabelColor                        // Selecting color for new label
    cardChecklistPane                           // Navigating checklist items
    cardAttachmentsPane                         // Navigating attachments
    cardActivityPane                            // Scrolling activity feed
    cardAddComment                              // Writing comment (textarea active)
    cardAddChecklist                            // Typing new checklist name
    cardAddCheckItem                            // Typing new checklist item name
    cardAddAttachment                           // Typing URL for new attachment
    cardConfirmDeleteChecklist                  // Awaiting y/n to delete checklist
    cardConfirmDeleteAttachment                 // Awaiting y/n to delete attachment
    cardMoveBoard                               // Selecting target board with filter
    cardMoveBoardList                           // Selecting target list on target board
)
```

**card.go — `checkRef`**:

```go
// checkRef identifies a specific checklist item by its checklist and item indices.
type checkRef struct {
    cl int // checklist index in m.checklists
    it int // item index within that checklist's CheckItems
}
```

---

## 3. Group CardModel fields by concern

**Severity**: MEDIUM
**File**: `internal/tui/card.go:42-85`

### Problem

30+ fields with no grouping or comments. An LLM cannot tell which fields are
meaningful in which mode.

### Changes

Add section comments:

```go
type CardModel struct {
    // Core data
    client *trello.Client
    card   trello.Card
    lists  []trello.List
    boardID string

    // Card data (fetched async on Init)
    boardMembers []trello.Member
    boardLabels  []trello.Label
    checklists   []trello.Checklist
    attachments  []trello.Attachment
    actions      []trello.Action

    // Navigation state
    listIndex int
    listName  string
    mode      cardMode

    // Edit inputs (active in corresponding edit modes)
    titleEdit      textinput.Model
    descEdit       textarea.Model
    dueInput       textinput.Model
    pickerFilter   textinput.Model
    labelNameInput textinput.Model
    commentInput   textarea.Model
    checklistInput textinput.Model

    // Move-to-board state (cardMoveBoard / cardMoveBoardList modes)
    allBoards       []trello.Board
    boardIndex      int
    targetBoard     trello.Board
    targetLists     []trello.List
    targetListIndex int

    // Cursor positions per mode/pane
    moveIndex     int
    memberIndex   int
    labelIndex    int
    labelColorIdx int
    checkItemIdx  int
    attachmentIdx int
    activityIdx   int

    // Scroll positions per pane
    infoScroll int
    clScroll   int
    attScroll  int
    actScroll  int

    // Display state
    width     int
    height    int
    statusMsg string
    showHelp  bool

    // Loading indicators
    loadingCL  bool
    loadingAtt bool
    loadingCom bool
}
```

---

## Checklist

- [x] Create `CLAUDE.md` with architecture guide
- [x] Document all `boardMode` constants with inline comments (17 modes)
- [x] Document all `cardMode` constants with inline comments (20 modes)
- [x] Document `boardListMode` constants (3 modes)
- [x] Add doc comment to `checkRef` struct with field descriptions
- [x] Group `CardModel` fields with section comments (8 groups)
- [x] Group `BoardModel` fields with section comments (8 groups)
- [x] Review for accuracy — `go vet` and `go test` pass
