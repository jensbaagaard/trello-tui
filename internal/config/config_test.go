package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_FromEnvVars(t *testing.T) {
	t.Setenv("TRELLO_API_KEY", "test-key")
	t.Setenv("TRELLO_TOKEN", "test-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key")
	}
	if cfg.Token != "test-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "test-token")
	}
}

func TestLoad_IgnoresPartialEnvVars(t *testing.T) {
	t.Setenv("TRELLO_API_KEY", "test-key")
	t.Setenv("TRELLO_TOKEN", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when only one env var is set and no config file exists")
	}
}

func TestLoad_FromConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "trello-tui")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := `{"api_key": "file-key", "token": "file-token"}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(data), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := loadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "file-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "file-key")
	}
	if cfg.Token != "file-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "file-token")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "trello-tui")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := loadFromDir(tmpDir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid config file") {
		t.Errorf("error = %q, want it to contain 'invalid config file'", err.Error())
	}
}

func TestLoad_MissingFields(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "trello-tui")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tests := []struct {
		name string
		json string
	}{
		{"empty api_key", `{"api_key": "", "token": "tok"}`},
		{"empty token", `{"api_key": "key", "token": ""}`},
		{"both empty", `{"api_key": "", "token": ""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(tt.json), 0o600); err != nil {
				t.Fatalf("write: %v", err)
			}

			_, err := loadFromDir(tmpDir)
			if err == nil {
				t.Fatal("expected error for missing fields")
			}
			if !strings.Contains(err.Error(), "must contain both") {
				t.Errorf("error = %q, want it to contain 'must contain both'", err.Error())
			}
		})
	}
}

func TestLoad_FixesInsecurePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "trello-tui")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := `{"api_key": "file-key", "token": "file-token"}`
	configFile := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configFile, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := loadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
}

func TestLoad_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := loadFromDir(tmpDir)
	if err == nil {
		t.Fatal("expected error when no credentials found")
	}
	if !strings.Contains(err.Error(), "no credentials found") {
		t.Errorf("error = %q, want it to contain 'no credentials found'", err.Error())
	}
	if !strings.Contains(err.Error(), "TRELLO_API_KEY") {
		t.Errorf("error should contain instructions with TRELLO_API_KEY")
	}
}
