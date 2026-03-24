package trello

type Board struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Closed bool   `json:"closed"`
}

type List struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IDBoard string `json:"idBoard"`
	Pos     float64 `json:"pos"`
	Closed  bool   `json:"closed"`
}

type Card struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Desc        string   `json:"desc"`
	IDList      string   `json:"idList"`
	Pos         float64  `json:"pos"`
	Closed      bool     `json:"closed"`
	Due         string   `json:"due"`
	DueComplete bool     `json:"dueComplete"`
	Labels      []Label  `json:"labels"`
	ShortURL    string   `json:"shortUrl"`
	Members     []Member `json:"members"`
}

type Label struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Member struct {
	ID       string `json:"id"`
	FullName string `json:"fullName"`
	Username string `json:"username"`
}

type CheckItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"` // "complete" or "incomplete"
}

type Checklist struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	CheckItems []CheckItem `json:"checkItems"`
}

type CommentData struct {
	Text string `json:"text"`
}

type Comment struct {
	ID            string      `json:"id"`
	Date          string      `json:"date"`
	Data          CommentData `json:"data"`
	MemberCreator Member      `json:"memberCreator"`
}
