# trello-tui

A terminal-based Trello client built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea). Browse boards, manage cards, and move work across lists — all from your terminal.

## Features

- **Board browser** with fuzzy filtering
- **Kanban board view** with responsive columns and horizontal scrolling
- **Card management** — create, edit, archive, and move cards between lists or across boards
- **Card detail view** with multiple panes:
  - Info (title, description, labels, members, due date, URL)
  - Checklists with interactive toggle, create/delete checklists and items
  - Attachments with system viewer integration, URL attachment support, and deletion
  - Activity feed with interlaced comments
- **Label management** — create, edit, and delete board labels; create labels inline from the card picker
- **Inline editing** — title, description, due date, members, and labels
- **Comments** — view and post comments from the activity pane
- **Archived cards** — browse and restore archived cards with filtering
- **Global card search** — search cards across all boards from the board list screen
- **Card filtering** — search across titles, descriptions, members, labels, and due dates
- **Visual indicators** — color-coded labels, member initials, due date warnings, checklist progress
- **Auto-refresh** — optionally re-fetch board data on a timer
- **CLI flags** — `--version`, `--board=<name>`, `--auto-refresh=<seconds>`
- **Version checking** — notifies you when a newer release is available
- **Keyboard-driven** — Trello-compatible hotkeys, no mouse needed

## Prerequisites

- [Go](https://go.dev/dl/) 1.25 or later
- Ensure `$HOME/go/bin` is in your `PATH`:
  ```bash
  export PATH="$PATH:$(go env GOPATH)/bin"
  ```

## Installation

```bash
go install github.com/jensbaagaard/trello-tui@latest
```

Or build from source:

```bash
git clone https://github.com/jensbaagaard/trello-tui.git
cd trello-tui
go build -o trello-tui .
```

## Configuration

You need a Trello API key and token. Get them here: https://trello.com/power-ups/admin

Then either set environment variables:

```bash
export TRELLO_API_KEY=your_key
export TRELLO_TOKEN=your_token
```

Or create a config file at `~/.config/trello-tui/config.json` (Linux) / `~/Library/Application Support/trello-tui/config.json` (macOS):

```json
{
  "api_key": "your_key",
  "token": "your_token"
}
```

## Usage

```bash
trello-tui                          # launch normally
trello-tui --board="Sprint"         # jump directly into a board (substring match)
trello-tui --auto-refresh=30        # refresh board data every 30 seconds
trello-tui --version                # print version and exit
```

### Keybindings

#### Board List

| Key       | Action          |
| --------- | --------------- |
| `j` / `k` | Navigate boards |
| `/`       | Filter boards   |
| `s`       | Search cards    |
| `enter`   | Open board      |
| `q`       | Quit            |

#### Card Search

| Key       | Action            |
| --------- | ----------------- |
| `enter`   | Search / open card |
| `j` / `k` | Navigate results  |
| `/`       | New search        |
| `esc`     | Back to boards    |

#### Board View

| Key              | Action                               |
| ---------------- | ------------------------------------ |
| `left` / `right` | Switch lists                         |
| `j` / `k`        | Navigate cards                       |
| `n`              | New card                             |
| `c`              | Archive card (confirms with `y`/`n`) |
| `,` / `.`        | Move card left / right               |
| `<` / `>`        | Move card to first / last list       |
| `a`              | View archived cards                  |
| `L`              | Manage board labels                  |
| `N`              | New list                             |
| `R`              | Rename current list                  |
| `C`              | Archive current list                 |
| `{` / `}`        | Move list left / right               |
| `/`              | Filter cards                         |
| `enter`          | Open card detail                     |
| `r`              | Refresh                              |
| `esc`            | Clear filter / back to board list    |

#### Label Manager (from board view)

| Key       | Action       |
| --------- | ------------ |
| `j` / `k` | Navigate     |
| `n`       | New label    |
| `e`       | Edit label   |
| `d`       | Delete label |
| `esc`     | Back         |

#### Archived Cards (from board view)

| Key            | Action        |
| -------------- | ------------- |
| `j` / `k`      | Navigate      |
| `enter` / `u`  | Restore card  |
| `/`            | Filter        |
| `r`            | Refresh       |
| `esc`          | Clear filter / back |

#### Card Detail

Keys marked with `*` match Trello's native shortcuts.

| Key       | Action                              |
| --------- | ----------------------------------- |
| `t`       | Edit title `*`                      |
| `e`       | Edit description `*`                |
| `m`       | Move to list (`B` for other board)  |
| `a`       | Add / remove members                |
| `l`       | Add / remove labels `*`             |
| `ctrl+n`  | New label (from picker)             |
| `d`       | Set / clear due date `*`            |
| `-`       | Add checklist `*`                   |
| `A`       | Attach URL                          |
| `c`       | Copy card URL                       |
| `,` / `.` | Move card left / right              |
| `<` / `>` | Move to first / last list           |
| `tab`     | Cycle panes                         |
| `esc`     | Back to board                       |

#### Checklist Pane

| Key               | Action         |
| ----------------- | -------------- |
| `j` / `k`         | Navigate items   |
| `enter` / `space`  | Toggle item      |
| `n`               | Add item         |
| `-`               | New checklist    |
| `d`               | Delete checklist |
| `tab`             | Next pane        |
| `esc`             | Back             |

#### Attachments Pane

| Key             | Action                   |
| --------------- | ------------------------ |
| `j` / `k`       | Navigate attachments     |
| `o` / `enter`   | Open with system viewer  |
| `a`             | Add URL attachment       |
| `d`             | Delete attachment        |
| `tab`           | Next pane                |
| `esc`           | Back                     |

#### Activity Pane

| Key       | Action        |
| --------- | ------------- |
| `j` / `k` | Scroll        |
| `n`       | New comment   |
| `tab`     | Next pane     |
| `esc`     | Back          |

## Roadmap

- [x] Move cards across boards (not just lists)
- [ ] Board search / filtering in the board list
- [x] CLI flags: `--version`, `--board=<name>`, `--auto-refresh=<seconds>`
- [x] Global card search across boards
- [x] View and restore archived cards
- [x] List management in board view (add, rename, archive, reorder)

## License

MIT
