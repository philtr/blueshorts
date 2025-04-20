package imap

import (
	"blueshorts/internal/model"
	"fmt"
	"io"
	"mime"
	"strings"

	"github.com/emersion/go-imap"
	imapClient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

// dialTLS is a seam for testing; overwrite it in tests to return a stub client.
var dialTLS = func(addr string) (client, error) {
	return imapClient.DialTLS(addr, nil)
}

// client defines just the methods fetch uses, making it easy to stub.
type client interface {
	Login(user, pass string) error
	Logout() error
	Select(name string, readOnly bool) (*imap.MailboxStatus, error)
	Fetch(seq *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
}

// NewFetcher returns a closure that knows how to grab a folder using the
// provided IMAP credentials.
func NewFetcher(cfg Config) func(folder string) (*model.JSONFeed, error) {
	return func(folder string) (*model.JSONFeed, error) {
		return fetch(cfg, folder)
	}
}

func fetch(cfg Config, folder string) (*model.JSONFeed, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	c, err := dialTLS(addr)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	if err := c.Login(cfg.Username, cfg.Password); err != nil {
		return nil, err
	}

	mbox, err := c.Select(folder, true)
	if err != nil {
		return nil, err
	}

	seqset := new(imap.SeqSet)
	total := mbox.Messages
	var from uint32 = 1
	if total > 25 {
		from = total - 25 + 1
	}
	seqset.AddRange(from, total)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, section.FetchItem()}

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() { done <- c.Fetch(seqset, items, messages) }()

	feed := &model.JSONFeed{
		Version: "https://jsonfeed.org/version/1.1",
		Title:   strings.TrimPrefix(folder, "/"),
		Items:   []model.Message{},
	}

	for msg := range messages {
		item := model.Message{
			ID:    msg.Envelope.MessageId,
			Title: msg.Envelope.Subject,
			Date:  msg.Envelope.Date,
		}
		if r := msg.GetBody(section); r != nil {
			mr, err := mail.CreateReader(r)
			if err == nil {
				var htmlBody, textBody string
				for {
					p, err := mr.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						continue
					}
					ct := p.Header.Get("Content-Type")
					media, _, perr := mime.ParseMediaType(ct)
					if perr != nil {
						media = strings.ToLower(strings.Split(ct, ";")[0])
					}
					b, _ := io.ReadAll(p.Body)
					switch media {
					case "text/html":
						if htmlBody == "" {
							htmlBody = string(b)
						}
					case "text/plain":
						if textBody == "" {
							textBody = string(b)
						}
					}
				}
				item.ContentHTML = htmlBody
				item.ContentText = textBody
			}
		}
		feed.Items = append(feed.Items, item)
	}

	if err := <-done; err != nil {
		return nil, err
	}

	return feed, nil
}
