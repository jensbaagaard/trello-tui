package trello

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (c *Client) ArchiveCard(cardID string) error {
	return c.request("PUT", fmt.Sprintf("/cards/%s", cardID), map[string]string{
		"closed": "true",
	}, nil)
}
