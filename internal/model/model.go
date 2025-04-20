package model

import "time"

type Message struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url,omitempty"`
	Date        time.Time `json:"date_published"`
	ContentHTML string    `json:"content_html,omitempty"`
	ContentText string    `json:"content_text,omitempty"`
}

type JSONFeed struct {
	Version     string    `json:"version"`
	Title       string    `json:"title"`
	HomePageURL string    `json:"home_page_url,omitempty"`
	FeedURL     string    `json:"feed_url,omitempty"`
	Items       []Message `json:"items"`
}
