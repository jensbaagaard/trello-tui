package trello

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type Client struct {
	apiKey     string
	token      string
	httpClient *http.Client
	baseURL    string
}

func NewClient(apiKey, token string) *Client {
	return &Client{
		apiKey:     apiKey,
		token:      token,
		httpClient: &http.Client{},
		baseURL:    "https://api.trello.com/1",
	}
}

func (c *Client) authParams() url.Values {
	return url.Values{
		"key":   {c.apiKey},
		"token": {c.token},
	}
}

func (c *Client) get(endpoint string, params url.Values, result interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	for k, v := range c.authParams() {
		params[k] = v
	}

	u := fmt.Sprintf("%s%s?%s", c.baseURL, endpoint, params.Encode())
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) request(method, endpoint string, body map[string]string, result interface{}) error {
	params := c.authParams()
	for k, v := range body {
		params.Set(k, v)
	}

	var reqBody io.Reader
	u := fmt.Sprintf("%s%s?%s", c.baseURL, endpoint, params.Encode())

	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) GetBoards() ([]Board, error) {
	var boards []Board
	params := url.Values{"filter": {"open"}}
	err := c.get("/members/me/boards", params, &boards)
	return boards, err
}

func (c *Client) GetLists(boardID string) ([]List, error) {
	var lists []List
	params := url.Values{"filter": {"open"}}
	err := c.get(fmt.Sprintf("/boards/%s/lists", boardID), params, &lists)
	return lists, err
}

func (c *Client) GetCards(listID string) ([]Card, error) {
	var cards []Card
	params := url.Values{"members": {"true"}}
	err := c.get(fmt.Sprintf("/lists/%s/cards", listID), params, &cards)
	return cards, err
}

func (c *Client) CreateCard(listID, name string) (Card, error) {
	var card Card
	err := c.request("POST", "/cards", map[string]string{
		"idList": listID,
		"name":   name,
	}, &card)
	return card, err
}

func (c *Client) UpdateCard(cardID string, fields map[string]string) (Card, error) {
	var card Card
	err := c.request("PUT", fmt.Sprintf("/cards/%s", cardID), fields, &card)
	return card, err
}

func (c *Client) MoveCard(cardID, listID string) (Card, error) {
	return c.UpdateCard(cardID, map[string]string{"idList": listID})
}

func (c *Client) MoveCardToBoard(cardID, boardID, listID string) (Card, error) {
	return c.UpdateCard(cardID, map[string]string{"idBoard": boardID, "idList": listID})
}

func (c *Client) ArchiveCard(cardID string) error {
	return c.request("PUT", fmt.Sprintf("/cards/%s", cardID), map[string]string{
		"closed": "true",
	}, nil)
}

func (c *Client) GetArchivedCards(boardID string) ([]Card, error) {
	var cards []Card
	params := url.Values{"members": {"true"}}
	err := c.get(fmt.Sprintf("/boards/%s/cards/closed", boardID), params, &cards)
	return cards, err
}

func (c *Client) RestoreCard(cardID string) (Card, error) {
	return c.UpdateCard(cardID, map[string]string{"closed": "false"})
}

func (c *Client) GetBoardMembers(boardID string) ([]Member, error) {
	var members []Member
	err := c.get(fmt.Sprintf("/boards/%s/members", boardID), nil, &members)
	return members, err
}

func (c *Client) GetBoardLabels(boardID string) ([]Label, error) {
	var labels []Label
	err := c.get(fmt.Sprintf("/boards/%s/labels", boardID), nil, &labels)
	return labels, err
}

func (c *Client) AddMemberToCard(cardID, memberID string) error {
	return c.request("POST", fmt.Sprintf("/cards/%s/idMembers", cardID), map[string]string{"value": memberID}, nil)
}

func (c *Client) RemoveMemberFromCard(cardID, memberID string) error {
	return c.request("DELETE", fmt.Sprintf("/cards/%s/idMembers/%s", cardID, memberID), nil, nil)
}

func (c *Client) AddLabelToCard(cardID, labelID string) error {
	return c.request("POST", fmt.Sprintf("/cards/%s/idLabels", cardID), map[string]string{"value": labelID}, nil)
}

func (c *Client) RemoveLabelFromCard(cardID, labelID string) error {
	return c.request("DELETE", fmt.Sprintf("/cards/%s/idLabels/%s", cardID, labelID), nil, nil)
}

func (c *Client) GetChecklists(cardID string) ([]Checklist, error) {
	var checklists []Checklist
	err := c.get(fmt.Sprintf("/cards/%s/checklists", cardID), nil, &checklists)
	return checklists, err
}

func (c *Client) GetActions(cardID string) ([]Action, error) {
	var actions []Action
	params := url.Values{"filter": {"commentCard,updateCard,createCard,addMemberToCard,removeMemberFromCard,addAttachmentToCard"}}
	err := c.get(fmt.Sprintf("/cards/%s/actions", cardID), params, &actions)
	return actions, err
}

func (c *Client) ToggleCheckItem(cardID, checkItemID string, complete bool) error {
	state := "incomplete"
	if complete {
		state = "complete"
	}
	return c.request("PUT", fmt.Sprintf("/cards/%s/checkItem/%s", cardID, checkItemID),
		map[string]string{"state": state}, nil)
}

func (c *Client) GetAttachments(cardID string) ([]Attachment, error) {
	var attachments []Attachment
	err := c.get(fmt.Sprintf("/cards/%s/attachments", cardID), nil, &attachments)
	return attachments, err
}

func (c *Client) DownloadAttachment(cardID string, att Attachment) (string, error) {
	if !att.IsUpload {
		return att.URL, nil
	}

	u := fmt.Sprintf("%s/cards/%s/attachments/%s/download/%s",
		c.baseURL, cardID, att.ID, att.Name)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf(
		`OAuth oauth_consumer_key="%s", oauth_token="%s"`,
		c.apiKey, c.token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading attachment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download error %d: %s", resp.StatusCode, string(body))
	}

	ext := filepath.Ext(att.Name)
	tmp, err := os.CreateTemp("", "trello-att-*"+ext)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	return tmp.Name(), nil
}

func (c *Client) CreateLabel(boardID, name, color string) (Label, error) {
	var label Label
	err := c.request("POST", "/labels", map[string]string{
		"idBoard": boardID,
		"name":    name,
		"color":   color,
	}, &label)
	return label, err
}

func (c *Client) UpdateLabel(labelID, name, color string) (Label, error) {
	var label Label
	err := c.request("PUT", fmt.Sprintf("/labels/%s", labelID), map[string]string{
		"name":  name,
		"color": color,
	}, &label)
	return label, err
}

func (c *Client) DeleteLabel(labelID string) error {
	return c.request("DELETE", fmt.Sprintf("/labels/%s", labelID), nil, nil)
}

func (c *Client) AddComment(cardID, text string) (Action, error) {
	var action Action
	err := c.request("POST", fmt.Sprintf("/cards/%s/actions/comments", cardID),
		map[string]string{"text": text}, &action)
	return action, err
}

func (c *Client) CreateChecklist(cardID, name string) (Checklist, error) {
	var cl Checklist
	err := c.request("POST", fmt.Sprintf("/cards/%s/checklists", cardID),
		map[string]string{"name": name}, &cl)
	return cl, err
}

func (c *Client) CreateCheckItem(checklistID, name string) (CheckItem, error) {
	var item CheckItem
	err := c.request("POST", fmt.Sprintf("/checklists/%s/checkItems", checklistID),
		map[string]string{"name": name}, &item)
	return item, err
}

func (c *Client) AddAttachmentURL(cardID, url string) (Attachment, error) {
	var att Attachment
	err := c.request("POST", fmt.Sprintf("/cards/%s/attachments", cardID),
		map[string]string{"url": url}, &att)
	return att, err
}

func (c *Client) DeleteChecklist(checklistID string) error {
	return c.request("DELETE", fmt.Sprintf("/checklists/%s", checklistID), nil, nil)
}

func (c *Client) DeleteAttachment(cardID, attachmentID string) error {
	return c.request("DELETE", fmt.Sprintf("/cards/%s/attachments/%s", cardID, attachmentID), nil, nil)
}

func (c *Client) CreateList(boardID, name string) (List, error) {
	var list List
	err := c.request("POST", fmt.Sprintf("/boards/%s/lists", boardID), map[string]string{
		"name": name,
	}, &list)
	return list, err
}

func (c *Client) UpdateList(listID string, fields map[string]string) (List, error) {
	var list List
	err := c.request("PUT", fmt.Sprintf("/lists/%s", listID), fields, &list)
	return list, err
}

func (c *Client) ArchiveList(listID string) error {
	return c.request("PUT", fmt.Sprintf("/lists/%s/closed", listID), map[string]string{
		"value": "true",
	}, nil)
}

func (c *Client) GetOrganizations() ([]Organization, error) {
	var orgs []Organization
	err := c.get("/members/me/organizations", nil, &orgs)
	return orgs, err
}

func (c *Client) CreateBoard(name, idOrganization string) (Board, error) {
	var board Board
	err := c.request("POST", "/boards", map[string]string{
		"name":           name,
		"idOrganization": idOrganization,
	}, &board)
	return board, err
}

func (c *Client) AddMemberToBoard(boardID, email string) (Member, error) {
	var member Member
	err := c.request("PUT", fmt.Sprintf("/boards/%s/members", boardID), map[string]string{
		"email": email,
		"type":  "normal",
	}, &member)
	return member, err
}

func (c *Client) RemoveMemberFromBoard(boardID, memberID string) error {
	return c.request("DELETE", fmt.Sprintf("/boards/%s/members/%s", boardID, memberID), nil, nil)
}

func (c *Client) SearchCards(query string, page int) ([]SearchCard, error) {
	var result SearchResult
	params := url.Values{
		"query":        {query},
		"modelTypes":   {"cards"},
		"card_board":   {"true"},
		"card_list":    {"true"},
		"card_members": {"true"},
		"cards_limit":  {"20"},
		"cards_page":   {fmt.Sprintf("%d", page)},
		"partial":      {"true"},
	}
	err := c.get("/search", params, &result)
	return result.Cards, err
}
