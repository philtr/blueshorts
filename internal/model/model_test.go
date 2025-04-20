package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMessage_OmitEmpty(t *testing.T) {
	now := time.Date(2025, 4, 19, 21, 0, 0, 0, time.UTC)

	msg := Message{
		ID:    "1",
		Title: "hello",
		Date:  now,
		// URL, ContentHTML, ContentText intentionally left empty
	}

	j, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(j)

	// mandatory fields present
	if !strings.Contains(s, `"id":"1"`) ||
		!strings.Contains(s, `"title":"hello"`) ||
		!strings.Contains(s, `"date_published":"2025-04-19T21:00:00Z"`) {
		t.Fatalf("missing expected fields: %s", s)
	}

	// optional fields omitted
	if strings.Contains(s, `"url":`) ||
		strings.Contains(s, `"content_html":`) ||
		strings.Contains(s, `"content_text":`) {
		t.Fatalf("unexpected optional field in %s", s)
	}
}

func TestJSONFeed_RoundTrip(t *testing.T) {
	now := time.Now().UTC()

	orig := JSONFeed{
		Version: "https://jsonfeed.org/version/1.1",
		Title:   "inbox",
		Items: []Message{
			{
				ID:          "a",
				Title:       "first",
				URL:         "https://example.com/a",
				Date:        now,
				ContentHTML: "<p>hi</p>",
				ContentText: "hi",
			},
		},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got JSONFeed
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Version != orig.Version || got.Title != orig.Title {
		t.Fatalf("header mismatch: %+v vs %+v", got, orig)
	}

	if len(got.Items) != 1 ||
		got.Items[0].ID != orig.Items[0].ID ||
		got.Items[0].ContentHTML != orig.Items[0].ContentHTML {
		t.Fatalf("item mismatch: %+v vs %+v", got.Items[0], orig.Items[0])
	}
}
