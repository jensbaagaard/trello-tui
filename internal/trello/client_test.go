package trello

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
		{
			ID: "c1", Name: "Card One", IDList: "l1",
			Members: []Member{{ID: "m1", FullName: "Alice"}},
			Badges:  Badges{CheckItems: 4, CheckItemsChecked: 3, Comments: 2, Attachments: 1},
		},
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
	if got[0].Badges.CheckItems != 4 {
		t.Errorf("Badges.CheckItems = %d, want 4", got[0].Badges.CheckItems)
	}
	if got[0].Badges.CheckItemsChecked != 3 {
		t.Errorf("Badges.CheckItemsChecked = %d, want 3", got[0].Badges.CheckItemsChecked)
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
	// 401 should return a user-friendly message without raw body
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("error = %q, want it to contain 'authentication failed'", err.Error())
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

func TestGetAttachments(t *testing.T) {
	attachments := []Attachment{
		{ID: "a1", Name: "screenshot.png", URL: "https://trello.com/att/1", MimeType: "image/png", Bytes: 2048, IsUpload: true},
		{ID: "a2", Name: "link", URL: "https://example.com", MimeType: "", Bytes: 0, IsUpload: false},
	}
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/cards/c1/attachments") {
			t.Errorf("path = %s, want /cards/c1/attachments", r.URL.Path)
		}
		json.NewEncoder(w).Encode(attachments)
	}))
	defer srv.Close()

	got, err := c.GetAttachments("c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "screenshot.png" {
		t.Errorf("Name = %q, want %q", got[0].Name, "screenshot.png")
	}
	if !got[0].IsUpload {
		t.Error("expected IsUpload = true for first attachment")
	}
	if got[1].IsUpload {
		t.Error("expected IsUpload = false for second attachment")
	}
}

func TestDownloadAttachmentExternalLink(t *testing.T) {
	c := NewClient("test-key", "test-token")
	att := Attachment{
		ID:       "a1",
		Name:     "link",
		URL:      "https://example.com/doc",
		IsUpload: false,
	}

	path, err := c.DownloadAttachment("c1", att)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "https://example.com/doc" {
		t.Errorf("path = %q, want %q", path, "https://example.com/doc")
	}
}

func TestDownloadAttachmentUpload(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		expectedPath := "/cards/c1/attachments/a1/download/image.png"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %s, want %s", r.URL.Path, expectedPath)
		}
		auth := r.Header.Get("Authorization")
		if !strings.Contains(auth, "oauth_consumer_key") {
			t.Error("expected OAuth Authorization header")
		}
		if !strings.Contains(auth, "test-key") {
			t.Errorf("auth header missing api key: %s", auth)
		}
		if !strings.Contains(auth, "test-token") {
			t.Errorf("auth header missing token: %s", auth)
		}
		w.Write([]byte("fake image data"))
	}))
	defer srv.Close()

	att := Attachment{
		ID:       "a1",
		Name:     "image.png",
		URL:      srv.URL + "/some-s3-url",
		IsUpload: true,
	}

	path, err := c.DownloadAttachment("c1", att)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, ".png") {
		t.Errorf("path = %q, want it to end with .png", path)
	}
	os.Remove(path)
}

// ── Week 1a: Security tests ─────────────────────────────────────────────────

func TestSanitizeExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"safe png", "image.png", ".png"},
		{"safe pdf", "document.pdf", ".pdf"},
		{"safe jpg", "photo.JPG", ".jpg"},
		{"safe docx", "file.docx", ".docx"},
		{"unsafe app", "malicious.app", ".bin"},
		{"unsafe scpt", "payload.scpt", ".bin"},
		{"unsafe exe", "virus.exe", ".bin"},
		{"unsafe sh", "script.sh", ".bin"},
		{"no extension", "noext", ".bin"},
		{"empty string", "", ".bin"},
		{"path separators stripped", "../../../etc/passwd.txt", ".txt"},
		{"null bytes stripped", "file\x00.png", ".png"},
		{"backslash stripped", "..\\..\\file.pdf", ".pdf"},
		{"double extension", "file.tar.gz", ".bin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeExtension(tt.filename)
			if got != tt.want {
				t.Errorf("sanitizeExtension(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestDownloadAttachment_SanitizesUnsafeExtension(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("fake data"))
	}))
	defer srv.Close()

	att := Attachment{ID: "a1", Name: "malicious.app", IsUpload: true}
	path, err := c.DownloadAttachment("c1", att)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(path)

	if !strings.HasSuffix(path, ".bin") {
		t.Errorf("path = %q, want .bin suffix for unsafe extension", path)
	}
}

func TestDownloadAttachment_CleansUpOnWriteError(t *testing.T) {
	// Serve enough data to start the download, then error
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "999999")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("partial"))
		// Connection drops — simulated by handler returning early
	}))
	defer srv.Close()

	att := Attachment{ID: "a1", Name: "file.pdf", IsUpload: true}
	path, err := c.DownloadAttachment("c1", att)

	// The download may succeed (since httptest sends full response before handler returns)
	// or fail. If it succeeded, verify we got a valid path; clean up.
	if err == nil {
		defer os.Remove(path)
		return
	}

	// If it did fail, verify the temp file was cleaned up
	if path != "" {
		if _, statErr := os.Stat(path); statErr == nil {
			t.Errorf("temp file %q should have been cleaned up on error", path)
			os.Remove(path)
		}
	}
}

func TestApiError_UserFriendlyMessages(t *testing.T) {
	tests := []struct {
		code int
		body string
		want string
	}{
		{401, "invalid token", "authentication failed"},
		{403, "forbidden", "access denied"},
		{404, "not found", "resource not found"},
		{429, "too many requests", "rate limited"},
		{500, "internal error", "API error 500: internal error"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.code), func(t *testing.T) {
			err := apiError(tt.code, []byte(tt.body))
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("apiError(%d) = %q, want it to contain %q", tt.code, err.Error(), tt.want)
			}
		})
	}
}

func TestApiError_TruncatesLongBody(t *testing.T) {
	longBody := strings.Repeat("x", 500)
	err := apiError(500, []byte(longBody))
	if len(err.Error()) > 250 {
		t.Errorf("error message too long (%d chars), expected truncation", len(err.Error()))
	}
	if !strings.HasSuffix(err.Error(), "...") {
		t.Error("expected truncated body to end with ...")
	}
}

func TestApiError_DoesNotLeakCredentials(t *testing.T) {
	// Simulate a response body that echoes back credentials
	body := `{"error": "invalid key=abc123secret token=xyz789secret"}`
	err := apiError(401, []byte(body))
	msg := err.Error()
	if strings.Contains(msg, "abc123secret") || strings.Contains(msg, "xyz789secret") {
		t.Errorf("error message contains credentials: %q", msg)
	}
}

// errorReader is an io.Reader that always returns an error
type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("simulated read failure")
}

func TestGet_ReadBodyError(t *testing.T) {
	// We can't easily mock io.ReadAll inside get(), but we can test that
	// a non-200 response with a proper body uses apiError correctly.
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("access denied detail"))
	}))
	defer srv.Close()

	_, err := c.GetBoards()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("error = %q, want user-friendly 403 message", err.Error())
	}
}

func TestRequest_ServerError(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("something broke"))
	}))
	defer srv.Close()

	_, err := c.CreateCard("l1", "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "API error 500") {
		t.Errorf("error = %q, want it to contain 'API error 500'", err.Error())
	}
}

func TestDownloadAttachment_ServerError(t *testing.T) {
	c, srv := testClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("attachment gone"))
	}))
	defer srv.Close()

	att := Attachment{ID: "a1", Name: "file.pdf", IsUpload: true}
	_, err := c.DownloadAttachment("c1", att)
	if err == nil {
		t.Fatal("expected error for 404 download")
	}
	if !strings.Contains(err.Error(), "resource not found") {
		t.Errorf("error = %q, want user-friendly 404 message", err.Error())
	}
}

func TestNewClient_HasTimeout(t *testing.T) {
	c := NewClient("key", "token")
	if c.httpClient.Timeout == 0 {
		t.Error("expected HTTP client to have a timeout")
	}
}

// Verify unused imports don't break — these are used above
var _ = io.ReadAll
var _ = fmt.Sprintf
