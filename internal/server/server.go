package server

import (
	"blueshorts/internal/cache"
	"blueshorts/internal/model"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type FetchFunc func(folder string) (*model.JSONFeed, error)

type Options struct {
	APIKey string
	Feeds  map[string]string
	Fetch  FetchFunc
	TTL    time.Duration
}

type Server struct {
	apiKey string
	feeds  map[string]string
	fetch  FetchFunc
	cache  *cache.TTL
}

func New(opts Options) http.Handler {
	if opts.Fetch == nil {
		panic("server: Fetch function is required")
	}
	if opts.TTL == 0 {
		opts.TTL = time.Minute * 5
	}

	s := &Server{
		apiKey: opts.APIKey,
		feeds:  opts.Feeds,
		fetch:  opts.Fetch,
		cache:  cache.New(opts.TTL),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/feeds/", s.handleFeed)
	return mux
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("key") != s.apiKey {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	name := strings.Split(strings.TrimPrefix(r.URL.Path, "/feeds/"), ".")[0]
	folder, ok := s.feeds[name]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if f, ok := s.cache.Get(name); ok {
		writeJSON(w, f)
		return
	}

	feed, err := s.fetch(folder)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	s.cache.Set(name, feed)
	writeJSON(w, feed)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
