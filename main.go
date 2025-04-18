package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/emersion/go-imap"
	imapClient "github.com/emersion/go-imap/client"
)

// Config holds server, IMAP and feed settings
// config is expected at data/config.toml

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
	ContentText string    `json:"content_text"`
}

// JSONFeed conforms to JSON Feed spec v1
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
	// simple TTL cache
	cacheTTL = 5 * time.Minute
)

func main() {
	// Load configuration from data/config.toml
	if _, err := toml.DecodeFile("/data/config.toml", &config); err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	http.HandleFunc("/feeds/", feedHandler)
	addr := ":8080"
	log.Printf("Listening on %s...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func feedHandler(w http.ResponseWriter, r *http.Request) {
	// API key check
	key := r.URL.Query().Get("key")
	if key != config.Server.APIKey {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// extract feed name from URL
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/feeds/"), ".")
	feedName := parts[0]
	imapFolder, ok := config.Feeds[feedName]
	if !ok {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	// serve from cache if fresh
	mu.Lock()
	exp, found := cacheExp[feedName]
	if found && time.Now().Before(exp) {
		feed := cache[feedName]
		mu.Unlock()
		jsonResponse(w, feed)
		return
	}
	mu.Unlock()

	// fetch new feed
	feed, err := fetchFeed(imapFolder)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching feed: %v", err), http.StatusInternalServerError)
		return
	}

	// cache it
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
	// connect to IMAP
	addr := fmt.Sprintf("%s:%d", config.IMAP.Host, config.IMAP.Port)
	c, err := imapClient.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	// login
	if err := c.Login(config.IMAP.Username, config.IMAP.Password); err != nil {
		return nil, err
	}

	// select folder
	mbox, err := c.Select(folder, true)
	if err != nil {
		return nil, err
	}

	// fetch all messages
	seqset := new(imap.SeqSet)
	seqset.AddRange(1, mbox.Messages)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	feed := &JSONFeed{
		Version: "https://jsonfeed.org/version/1",
		Title:   fmt.Sprintf("IMAP: %s", folder),
	}

	for msg := range messages {
		item := Message{
			ID:          msg.Envelope.MessageId,
			Title:       msg.Envelope.Subject,
			Date:        msg.Envelope.Date,
			ContentText: "", // body fetching can be added here
		}
		feed.Items = append(feed.Items, item)
	}

	if err := <-done; err != nil {
		return nil, err
	}

	return feed, nil
}
