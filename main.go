package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/emersion/go-imap"
	imapClient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

// Config holds server, IMAP and feed settings
// config is expected at /data/config.toml
type Config struct {
	Server struct {
		APIKey string `toml:"api_key"`
	} `toml:"server"`

	IMAP struct {
		Host     string `toml:"host"`
		Port     int    `toml:"port"`
		Username string `toml:"username"`
		Password string `toml:"password"`
	} `toml:"imap"`

	Feeds map[string]string `toml:"feeds"`
}

// Message represents a single email in the JSON Feed
type Message struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url,omitempty"`
	Date        time.Time `json:"date_published"`
	ContentHTML string    `json:"content_html,omitempty"`
	ContentText string    `json:"content_text,omitempty"`
}

// JSONFeed conforms to JSON Feed spec v1.1
type JSONFeed struct {
	Version     string    `json:"version"`
	Title       string    `json:"title"`
	HomePageURL string    `json:"home_page_url,omitempty"`
	FeedURL     string    `json:"feed_url,omitempty"`
	Items       []Message `json:"items"`
}

var (
	config   Config
	cache    = make(map[string]JSONFeed)
	cacheExp = make(map[string]time.Time)
	mu       sync.Mutex
	cacheTTL = 5 * time.Minute
)

func main() {
	if _, err := toml.DecodeFile("/data/config.toml", &config); err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	http.HandleFunc("/feeds/", feedHandler)
	addr := ":8080"
	log.Printf("Listening on %s...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func feedHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key != config.Server.APIKey {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/feeds/"), ".")
	feedName := parts[0]
	imapFolder, ok := config.Feeds[feedName]
	if !ok {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	mu.Lock()
	exp, found := cacheExp[feedName]
	if found && time.Now().Before(exp) {
		jsonResponse(w, cache[feedName])
		mu.Unlock()
		return
	}
	mu.Unlock()

	feed, err := fetchFeed(imapFolder)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching feed: %v", err), http.StatusInternalServerError)
		return
	}

	mu.Lock()
	cache[feedName] = *feed
	cacheExp[feedName] = time.Now().Add(cacheTTL)
	mu.Unlock()

	jsonResponse(w, *feed)
}

func jsonResponse(w http.ResponseWriter, feed JSONFeed) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

func fetchFeed(folder string) (*JSONFeed, error) {
	addr := fmt.Sprintf("%s:%d", config.IMAP.Host, config.IMAP.Port)
	c, err := imapClient.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	if err := c.Login(config.IMAP.Username, config.IMAP.Password); err != nil {
		return nil, err
	}

	mbox, err := c.Select(folder, true)
	if err != nil {
		return nil, err
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(1, mbox.Messages)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, section.FetchItem()}

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() { done <- c.Fetch(seqset, items, messages) }()

	feed := &JSONFeed{
		Version: "https://jsonfeed.org/version/1.1",
		Title:   strings.TrimPrefix(folder, "/"),
		Items:   []Message{},
	}

	for msg := range messages {
		item := Message{
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
					if err == io.EOF { break }
					if err != nil { continue }
					ctHeader := p.Header.Get("Content-Type")
					mediaType, _, parseErr := mime.ParseMediaType(ctHeader)
					if parseErr != nil {
						mediaType = strings.ToLower(strings.Split(ctHeader, ";")[0])
					}
					bodyBytes, _ := io.ReadAll(p.Body)
					body := string(bodyBytes)
					switch mediaType {
					case "text/html":
						if htmlBody == "" { htmlBody = body }
					case "text/plain":
						if textBody == "" { textBody = body }
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
