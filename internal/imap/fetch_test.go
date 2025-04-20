package imap

import (
	"testing"
	"time"

	"github.com/emersion/go-imap"
)

type stub struct {
	msgs []*imap.Message
}

func (s *stub) Login(_, _ string) error { return nil }
func (s *stub) Logout() error           { return nil }
func (s *stub) Select(_ string, _ bool) (*imap.MailboxStatus, error) {
	return &imap.MailboxStatus{Messages: uint32(len(s.msgs))}, nil
}

func (s *stub) Fetch(_ *imap.SeqSet, _ []imap.FetchItem, ch chan *imap.Message) error {
	for _, m := range s.msgs {
		ch <- m
	}
	close(ch)
	return nil
}

func TestFetch(t *testing.T) {
	// build two dummy messages
	now := time.Now()
	st := &stub{
		msgs: []*imap.Message{
			{Envelope: &imap.Envelope{MessageId: "a", Subject: "one", Date: now}},
			{Envelope: &imap.Envelope{MessageId: "b", Subject: "two", Date: now.Add(time.Minute)}},
		},
	}

	// swap in our stub and restore afterwards
	old := dialTLS
	dialTLS = func(_ string) (client, error) { return st, nil }
	defer func() { dialTLS = old }()

	cfg := Config{Host: "x", Port: 993, Username: "u", Password: "p"}
	feed, err := fetch(cfg, "/Inbox")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if feed.Title != "Inbox" {
		t.Fatalf("want title Inbox, got %q", feed.Title)
	}
	if len(feed.Items) != 2 || feed.Items[0].ID != "a" || feed.Items[1].ID != "b" {
		t.Fatalf("items not copied correctly: %+v", feed.Items)
	}
}
