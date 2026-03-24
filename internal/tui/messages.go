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

type BoardMembersFetchedMsg struct {
	Members []trello.Member
	Err     error
}

type BoardLabelsFetchedMsg struct {
	Labels []trello.Label
	Err    error
}

type MemberToggledMsg struct {
	Err error
}

type LabelToggledMsg struct {
	Err error
}

type LabelCreatedMsg struct {
	Label trello.Label
	Err   error
}

type LabelUpdatedMsg struct {
	Label trello.Label
	Err   error
}

type LabelDeletedMsg struct {
	LabelID string
	Err     error
}

type ChecklistsFetchedMsg struct {
	Checklists []trello.Checklist
	Err        error
}

type ActionsFetchedMsg struct {
	Actions []trello.Action
	Err     error
}

type AttachmentsFetchedMsg struct {
	Attachments []trello.Attachment
	Err         error
}

type AttachmentOpenedMsg struct {
	Err error
}

type CheckItemToggledMsg struct {
	Err error
}

type CommentAddedMsg struct {
	Action trello.Action
	Err    error
}

type VersionCheckMsg struct {
	UpdateNotice string
}
