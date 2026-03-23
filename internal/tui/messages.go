package tui

import "github.com/jensbaagaard/trello-tui/internal/trello"

type BoardsFetchedMsg struct {
	Boards []trello.Board
	Err    error
}

type ListsFetchedMsg struct {
	Lists []trello.List
	Err   error
}

type CardsFetchedMsg struct {
	ListID string
	Cards  []trello.Card
	Err    error
}

type AllCardsFetchedMsg struct {
	CardsByList map[string][]trello.Card
	Err         error
}

type CardUpdatedMsg struct {
	Card trello.Card
	Err  error
}

type CardCreatedMsg struct {
	Card trello.Card
	Err  error
}

type CardArchivedMsg struct {
	CardID string
	Err    error
}

type CardMovedMsg struct {
	Card       trello.Card
	FromListID string
	ToListID   string
	Err        error
}

type ErrMsg struct {
	Err error
}
