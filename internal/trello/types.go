package trello

type Board struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Closed bool   `json:"closed"`
}

type Organization struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Name        string `json:"name"` // short name / slug
}

type List struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IDBoard string `json:"idBoard"`
	Pos     float64 `json:"pos"`
	Closed  bool   `json:"closed"`
}

type Badges struct {
	CheckItems        int `json:"checkItems"`
	CheckItemsChecked int `json:"checkItemsChecked"`
	Comments          int `json:"comments"`
	Attachments       int `json:"attachments"`
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
	Badges      Badges   `json:"badges"`
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

type Attachment struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	MimeType string `json:"mimeType"`
	Bytes    int    `json:"bytes"`
	Date     string `json:"date"`
	IsUpload bool   `json:"isUpload"`
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

type ActionList struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ActionRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ActionData struct {
	Text       string      `json:"text"`
	List       *ActionList `json:"list"`
	ListBefore *ActionList `json:"listBefore"`
	ListAfter  *ActionList `json:"listAfter"`
	Member     *Member     `json:"member"`
	Attachment *ActionRef  `json:"attachment"`
}

type Action struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	Date          string     `json:"date"`
	Data          ActionData `json:"data"`
	MemberCreator Member     `json:"memberCreator"`
}

type SearchResult struct {
	Cards []SearchCard `json:"cards"`
}

type SearchCard struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Desc        string   `json:"desc"`
	IDBoard     string   `json:"idBoard"`
	IDList      string   `json:"idList"`
	Closed      bool     `json:"closed"`
	Due         string   `json:"due"`
	DueComplete bool     `json:"dueComplete"`
	Labels      []Label  `json:"labels"`
	ShortURL    string   `json:"shortUrl"`
	Members     []Member `json:"members"`
	Badges      Badges   `json:"badges"`
	Board       Board    `json:"board"`
	List        List     `json:"list"`
}

func (sc SearchCard) ToCard() Card {
	return Card{
		ID:          sc.ID,
		Name:        sc.Name,
		Desc:        sc.Desc,
		IDList:      sc.IDList,
		Closed:      sc.Closed,
		Due:         sc.Due,
		DueComplete: sc.DueComplete,
		Labels:      sc.Labels,
		ShortURL:    sc.ShortURL,
		Members:     sc.Members,
		Badges:      sc.Badges,
	}
}
