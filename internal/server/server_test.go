package server_test

import (
	"blueshorts/internal/model"
	"blueshorts/internal/server"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// returns an httptest.Server with a stubbed fetcher.
func stubServer(t *testing.T, hits *int) *httptest.Server {
	t.Helper()

	stubFetch := func(string) (*model.JSONFeed, error) {
		*hits++
		return &model.JSONFeed{
			Version: "https://jsonfeed.org/version/1.1",
			Title:   "INBOX",
			Items: []model.Message{
				{ID: "1", Title: "hi", Date: time.Now()},
			},
		}, nil
	}

	h := server.New(server.Options{
		APIKey: "secret",
		Feeds:  map[string]string{"inbox": "INBOX"},
		Fetch:  stubFetch,
		TTL:    time.Hour,
	})

	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
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
	var feed model.JSONFeed
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
