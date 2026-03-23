package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testClient(handler http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	c := NewClient("test-key", "test-token")
	c.baseURL = srv.URL
	return c, srv
}

func TestGetBoards(t *testing.T) {
	boards := []Board{
		{ID: "b1", Name: "Board One"},
		{ID: "b2", Name: "Board Two"},
	}
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/members/me/boards") {
			t.Errorf("path = %s, want /members/me/boards", r.URL.Path)
		}
		json.NewEncoder(w).Encode(boards)
	}))
	defer srv.Close()

	got, err := c.GetBoards()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "Board One" {
		t.Errorf("Name = %q, want %q", got[0].Name, "Board One")
	}
}

func TestGetLists(t *testing.T) {
	lists := []List{
		{ID: "l1", Name: "To Do", IDBoard: "b1"},
		{ID: "l2", Name: "Done", IDBoard: "b1"},
	}
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/boards/b1/lists") {
			t.Errorf("path = %s, want /boards/b1/lists", r.URL.Path)
		}
		json.NewEncoder(w).Encode(lists)
	}))
	defer srv.Close()

	got, err := c.GetLists("b1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[1].Name != "Done" {
		t.Errorf("Name = %q, want %q", got[1].Name, "Done")
	}
}

func TestGetCards(t *testing.T) {
	cards := []Card{
		{ID: "c1", Name: "Card One", IDList: "l1", Members: []Member{{ID: "m1", FullName: "Alice"}}},
	}
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/lists/l1/cards") {
			t.Errorf("path = %s, want /lists/l1/cards", r.URL.Path)
		}
		if r.URL.Query().Get("members") != "true" {
			t.Error("expected members=true query param")
		}
		json.NewEncoder(w).Encode(cards)
	}))
	defer srv.Close()

	got, err := c.GetCards("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Name != "Card One" {
		t.Errorf("Name = %q, want %q", got[0].Name, "Card One")
	}
	if len(got[0].Members) != 1 || got[0].Members[0].FullName != "Alice" {
		t.Errorf("Members not deserialized correctly")
	}
}

func TestCreateCard(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/cards") {
			t.Errorf("path = %s, want /cards", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("idList") != "l1" {
			t.Errorf("idList = %q, want %q", q.Get("idList"), "l1")
		}
		if q.Get("name") != "New Card" {
			t.Errorf("name = %q, want %q", q.Get("name"), "New Card")
		}
		json.NewEncoder(w).Encode(Card{ID: "c-new", Name: "New Card", IDList: "l1"})
	}))
	defer srv.Close()

	got, err := c.CreateCard("l1", "New Card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "c-new" {
		t.Errorf("ID = %q, want %q", got.ID, "c-new")
	}
}

func TestUpdateCard(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/cards/c1") {
			t.Errorf("path = %s, want /cards/c1", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("name") != "Updated" {
			t.Errorf("name = %q, want %q", q.Get("name"), "Updated")
		}
		json.NewEncoder(w).Encode(Card{ID: "c1", Name: "Updated"})
	}))
	defer srv.Close()

	got, err := c.UpdateCard("c1", map[string]string{"name": "Updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Updated" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated")
	}
}

func TestMoveCard(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("idList") != "l2" {
			t.Errorf("idList = %q, want %q", q.Get("idList"), "l2")
		}
		json.NewEncoder(w).Encode(Card{ID: "c1", IDList: "l2"})
	}))
	defer srv.Close()

	got, err := c.MoveCard("c1", "l2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IDList != "l2" {
		t.Errorf("IDList = %q, want %q", got.IDList, "l2")
	}
}

func TestArchiveCard(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		q := r.URL.Query()
		if q.Get("closed") != "true" {
			t.Errorf("closed = %q, want %q", q.Get("closed"), "true")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := c.ArchiveCard("c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPIError(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid token"))
	}))
	defer srv.Close()

	_, err := c.GetBoards()
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %q, want it to contain '401'", err.Error())
	}
	if !strings.Contains(err.Error(), "invalid token") {
		t.Errorf("error = %q, want it to contain 'invalid token'", err.Error())
	}
}

func TestAuthParams(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("key") != "test-key" {
			t.Errorf("key = %q, want %q", q.Get("key"), "test-key")
		}
		if q.Get("token") != "test-token" {
			t.Errorf("token = %q, want %q", q.Get("token"), "test-token")
		}
		json.NewEncoder(w).Encode([]Board{})
	}))
	defer srv.Close()

	c.GetBoards()
}
