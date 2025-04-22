package server

import (
	"blueshorts/internal/cache"
	"blueshorts/internal/model"
	"encoding/json"
	"log"
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
	log.Printf("initializing server: feeds=%v ttl=%s apiKey=%s", opts.Feeds, opts.TTL, opts.APIKey)

	s := &Server{
		apiKey: opts.APIKey,
		feeds:  opts.Feeds,
		fetch:  opts.Fetch,
		cache:  cache.New(opts.TTL),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/feeds/", s.handleFeed)

	log.Println("handler registered: GET /feeds/{name}.json")
	return mux
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	log.Printf("incoming request: %s %s from %s", r.Method, r.URL.RequestURI(), r.RemoteAddr)

	if r.URL.Query().Get("key") != s.apiKey {
		log.Printf("forbidden: invalid api key from %s", r.RemoteAddr)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	name := strings.Split(strings.TrimPrefix(r.URL.Path, "/feeds/"), ".")[0]
	folder, ok := s.feeds[name]
	if !ok {
		log.Printf("not found: feed %q", name)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if f, ok := s.cache.Get(name); ok {
		log.Printf("cache hit: %s", name)
		writeJSON(w, f)
		return
	}

	log.Printf("cache miss: fetching feed %s from folder %s", name, folder)
	feed, err := s.fetch(folder)
	if err != nil {
		log.Printf("upstream error fetching %s: %v", name, err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	s.cache.Set(name, feed)
	log.Printf("fetched & cached feed %s", name)
	writeJSON(w, feed)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error writing JSON response: %v", err)
	} else {
		log.Printf("response written: %T", v)
	}
}
