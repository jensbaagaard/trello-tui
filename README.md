# trello-tui

A terminal-based Trello client built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea). Browse boards, manage cards, and move work across lists — all from your terminal.

## Features

- Browse and filter your Trello boards
- Kanban-style board view with scrollable columns
- Create, edit, archive, and move cards between lists
- Color-coded labels and member badges on cards
- Card detail view with description editing
- Keyboard-driven — no mouse needed

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
| `enter`          | Open card detail                     |
| `r`              | Refresh                              |
| `esc`            | Back to board list                   |

#### Card Detail

| Key       | Action                    |
| --------- | ------------------------- |
| `t` / `e` | Edit title                |
| `E`       | Edit description          |
| `m`       | Move to list (picker)     |
| `,` / `.` | Move card left / right    |
| `<` / `>` | Move to first / last list |
| `esc`     | Back to board             |

## License

MIT
