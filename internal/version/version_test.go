package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"v0.4.0", "v0.3.0", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.3.1", "v0.3.0", true},
		{"v0.3.0", "v0.3.0", false},
		{"v0.2.0", "v0.3.0", false},
		{"v0.3.0", "dev", false},
		{"dev", "v0.3.0", false},
		{"", "v0.3.0", false},
		{"v0.3.0", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got := IsNewer(tt.latest, tt.current)
		if got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"v1.2.3", true},
		{"1.2.3", true},
		{"dev", false},
		{"", false},
		{"v1.2", false},
		{"v1.2.3.4", false},
		{"vabc", false},
	}
	for _, tt := range tests {
		got := parseSemver(tt.input)
		if (got != nil) != tt.valid {
			t.Errorf("parseSemver(%q) valid=%v, want %v", tt.input, got != nil, tt.valid)
		}
	}
}

func TestCheckLatest(t *testing.T) {
	tags := []ghTag{
		{Name: "v0.1.0"},
		{Name: "v0.3.0"},
		{Name: "v0.2.0"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tags)
	}))
	defer srv.Close()

	old := repoURL
	repoURL = srv.URL
	defer func() { repoURL = old }()

	got := CheckLatest()
	if got != "v0.3.0" {
		t.Errorf("CheckLatest() = %q, want %q", got, "v0.3.0")
	}
}

func TestCheckLatestServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := repoURL
	repoURL = srv.URL
	defer func() { repoURL = old }()

	got := CheckLatest()
	if got != "" {
		t.Errorf("CheckLatest() on error = %q, want empty", got)
	}
}
