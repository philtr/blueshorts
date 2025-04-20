package server

import (
	"blueshorts/internal/imap"
	"blueshorts/internal/model"
	"blueshorts/internal/server"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emersion/go-imap"
)

// ---------- IMAP stub ----------

type stubClient struct {
	msgs []*imap.Message
}

func (s *stubClient) Login(_, _ string) error { return nil }
func (s *stubClient) Logout() error           { return nil }
func (s *stubClient) Select(_ string, _ bool) (*imap.MailboxStatus, error) {
	return &imap.MailboxStatus{Messages: uint32(len(s.msgs))}, nil
}

func (s *stubClient) Fetch(_ *imap.SeqSet, _ []imap.FetchItem, ch chan *imap.Message) error {
	for _, m := range s.msgs {
		ch <- m
	}
	close(ch)
	return nil
}

// ---------- test ----------

func TestEndToEnd_FeedEndpoint(t *testing.T) {
	// build two dummy IMAP envelopes
	now := time.Now().UTC()
	stub := &stubClient{
		msgs: []*imap.Message{
			{Envelope: &imap.Envelope{MessageId: "a", Subject: "one", Date: now}},
			{Envelope: &imap.Envelope{MessageId: "b", Subject: "two", Date: now.Add(time.Minute)}},
		},
	}

	// swap in stub dialer + restore afterwards
	oldDial := imap.DialTLS
	imap.DialTLS = func(string) (imap.Client, error) { return stub, nil }
	defer func() { imap.DialTLS = oldDial }()

	// server wiring
	fetch := imap.NewFetcher(imap.Config{
		Host: "irrelevant", Port: 993, Username: "x", Password: "y",
	})

	srv := server.New(server.Options{
		APIKey: "test‑key",
		Feeds:  map[string]string{"Inbox": "/inbox"}, // route → folder
		Fetch:  fetch,
		TTL:    time.Minute,
	})

	ts := httptest.NewServer(srv)
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/inbox", nil)
	req.Header.Set("X-API-Key", "test‑key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}

	var feed model.JSONFeed
	if err := json.NewDecoder(resp.Body).Decode(&feed); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if feed.Title != "Inbox" || len(feed.Items) != 2 {
		t.Fatalf("bad feed: %+v", feed)
	}
	if feed.Items[0].ID != "a" || feed.Items[1].ID != "b" {
		t.Fatalf("bad item ids: %+v", feed.Items)
	}
}
