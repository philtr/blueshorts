package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// reset clears globals but does NOT overwrite the fetch func.
func reset() {
	cache = make(map[string]JSONFeed)
	cacheExp = make(map[string]time.Time)
	// leave `fetch` alone; each test decides what it should be
}

func stubServer(t *testing.T, hits *int) *httptest.Server {
	t.Helper()

	reset() // first, clear caches + keep fetch=whatever

	// now stub fetch
	fetch = func(string) (*JSONFeed, error) {
		*hits++
		return &JSONFeed{
			Version: "https://jsonfeed.org/version/1.1",
			Title:   "INBOX",
			Items:   []Message{{ID: "1", Title: "hi", Date: time.Now()}},
		}, nil
	}

	cfg := Config{
		Server: struct {
			APIKey string `toml:"api_key"`
		}{APIKey: "secret"},
		Feeds: map[string]string{"inbox": "INBOX"},
	}

	ts := httptest.NewServer(newServer(cfg))
	t.Cleanup(func() { ts.Close(); reset() })
	return ts
}

func TestOK(t *testing.T) {
	var hits int
	srv := stubServer(t, &hits)

	res, err := http.Get(srv.URL + "/feeds/inbox.json?key=secret")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("want 200 got %d", res.StatusCode)
	}
	var feed JSONFeed
	if err := json.NewDecoder(res.Body).Decode(&feed); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if feed.Title != "INBOX" {
		t.Errorf("bad title %q", feed.Title)
	}
	if hits != 1 {
		t.Errorf("fetch called %d times, want 1", hits)
	}
}

func TestForbidden(t *testing.T) {
	var hits int
	srv := stubServer(t, &hits)

	res, _ := http.Get(srv.URL + "/feeds/inbox.json?key=wrong")
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 got %d", res.StatusCode)
	}
	if hits != 0 {
		t.Errorf("fetch should not run, ran %d", hits)
	}
}

func TestNotFound(t *testing.T) {
	var hits int
	srv := stubServer(t, &hits)

	res, _ := http.Get(srv.URL + "/feeds/missing.json?key=secret")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 got %d", res.StatusCode)
	}
	if hits != 0 {
		t.Errorf("fetch should not run, ran %d", hits)
	}
}

func TestCache(t *testing.T) {
	var hits int
	srv := stubServer(t, &hits)

	url := srv.URL + "/feeds/inbox.json?key=secret"
	for i := 0; i < 2; i++ {
		if res, _ := http.Get(url); res.StatusCode != http.StatusOK {
			t.Fatalf("round %d: not 200", i)
		}
	}
	if hits != 1 {
		t.Errorf("cache miss: fetch ran %d times, want 1", hits)
	}
}
