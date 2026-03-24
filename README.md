# trello-tui

A terminal-based Trello client built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea). Browse boards, manage cards, and move work across lists — all from your terminal.

## Features

- **Board browser** with fuzzy filtering
- **Kanban board view** with responsive columns and horizontal scrolling
- **Card management** — create, edit, archive, and move cards between lists
- **Card detail view** with multiple panes:
  - Info (title, description, labels, members, due date, URL)
  - Checklists with interactive toggle
  - Attachments with system viewer integration
  - Activity feed with interlaced comments
- **Label management** — create, edit, and delete board labels; create labels inline from the card picker
- **Inline editing** — title, description, due date, members, and labels
- **Comments** — view and post comments from the activity pane
- **Card filtering** — search across titles, descriptions, members, labels, and due dates
- **Visual indicators** — color-coded labels, member initials, due date warnings, checklist progress
- **Version checking** — notifies you when a newer release is available
- **Keyboard-driven** — no mouse needed

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
trello-tui
```

### Keybindings

#### Board List

| Key       | Action          |
| --------- | --------------- |
| `j` / `k` | Navigate boards |
| `/`       | Filter boards   |
| `enter`   | Open board      |
| `q`       | Quit            |

#### Board View

| Key              | Action                               |
| ---------------- | ------------------------------------ |
| `left` / `right` | Switch lists                         |
| `j` / `k`        | Navigate cards                       |
| `n`              | New card                             |
| `c`              | Archive card (confirms with `y`/`n`) |
| `,` / `.`        | Move card left / right               |
| `<` / `>`        | Move card to first / last list       |
| `L`              | Manage board labels                  |
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

#### Card Detail

| Key       | Action                    |
| --------- | ------------------------- |
| `t`       | Edit title                |
| `e`       | Edit description          |
| `m`       | Move to list (picker)     |
| `a`       | Add / remove members      |
| `l`       | Add / remove labels       |
| `ctrl+n`  | New label (from picker)   |
| `d`       | Set / clear due date      |
| `,` / `.` | Move card left / right    |
| `<` / `>` | Move to first / last list |
| `tab`     | Cycle panes               |
| `esc`     | Back to board             |

#### Checklist Pane

| Key             | Action       |
| --------------- | ------------ |
| `j` / `k`       | Navigate items |
| `enter` / `space` | Toggle item  |
| `tab`           | Next pane    |

#### Attachments Pane

| Key             | Action                   |
| --------------- | ------------------------ |
| `j` / `k`       | Navigate attachments     |
| `o` / `enter`   | Open with system viewer  |
| `tab`           | Next pane                |

#### Activity Pane

| Key       | Action        |
| --------- | ------------- |
| `j` / `k` | Scroll        |
| `n`       | New comment   |
| `tab`     | Next pane     |

## License

MIT
