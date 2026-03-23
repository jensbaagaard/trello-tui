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
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Desc   string  `json:"desc"`
	IDList string  `json:"idList"`
	Pos    float64 `json:"pos"`
	Closed bool    `json:"closed"`
	Labels []Label `json:"labels"`
}

type Label struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}
