package cache

import (
	"blueshorts/internal/model"
	"sync"
	"time"
)

type TTL struct {
	data map[string]*model.JSONFeed
	exp  map[string]time.Time
	mu   sync.Mutex
	ttl  time.Duration
}

func New(ttl time.Duration) *TTL {
	return &TTL{
		data: make(map[string]*model.JSONFeed),
		exp:  make(map[string]time.Time),
		ttl:  ttl,
	}
}

func (c *TTL) Get(key string) (*model.JSONFeed, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.exp[key]; ok && time.Now().Before(t) {
		return c.data[key], true
	}
	return nil, false
}

func (c *TTL) Set(key string, val *model.JSONFeed) {
	c.mu.Lock()
	c.data[key] = val
	c.exp[key] = time.Now().Add(c.ttl)
	c.mu.Unlock()
}
