package config

import (
	"os"
	"path/filepath"
	"testing"
)

const goodTOML = `
[server]
api_key = "your-secret-api-key"

[imap]
host     = "imap.example.com"
port     = 993
username = "you@example.com"
password = "hunter2"

[feeds]
news   = "Newsletters"
alerts = "Notifications/Alerts"
`

func tmpFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write tmp file: %v", err)
	}
	return path
}

func TestLoad_Valid(t *testing.T) {
	path := tmpFile(t, goodTOML)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.APIKey != "your-secret-api-key" {
		t.Fatalf("api_key mismatch: %q", cfg.Server.APIKey)
	}

	if cfg.IMAP.Host != "imap.example.com" ||
		cfg.IMAP.Port != 993 ||
		cfg.IMAP.Username != "you@example.com" ||
		cfg.IMAP.Password != "hunter2" {
		t.Fatalf("imap section mismatch: %+v", cfg.IMAP)
	}

	wantFeeds := 2
	if len(cfg.Feeds) != wantFeeds {
		t.Fatalf("expected %d feeds, got %d", wantFeeds, len(cfg.Feeds))
	}
}

func TestLoad_Errors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := Load("no-such-file.toml"); err == nil {
			t.Fatalf("expected error for missing file")
		}
	})

	t.Run("invalid toml", func(t *testing.T) {
		path := tmpFile(t, "::: not valid toml")
		if _, err := Load(path); err == nil {
			t.Fatalf("expected error for bad toml")
		}
	})
}
