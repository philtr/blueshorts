package cache

import (
	"blueshorts/internal/model"
	"sync"
	"testing"
	"time"
)

func TestTTLCache_Basic(t *testing.T) {
	ttl := 50 * time.Millisecond
	c := New(ttl)

	key := "foo"
	feed := &model.JSONFeed{Title: "bar"}

	if _, ok := c.Get(key); ok {
		t.Fatalf("expected miss on empty cache")
	}

	c.Set(key, feed)

	if got, ok := c.Get(key); !ok || got != feed {
		t.Fatalf("expected hit with same pointer; ok=%v got=%v", ok, got)
	}

	time.Sleep(ttl + 10*time.Millisecond)

	if _, ok := c.Get(key); ok {
		t.Fatalf("expected miss after ttl expiry")
	}
}

func TestTTLCache_Concurrent(t *testing.T) {
	ttl := time.Second
	c := New(ttl)

	key := "foo"
	feed := &model.JSONFeed{}

	var wg sync.WaitGroup
	workers := 8
	iterations := 500

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				c.Set(key, feed)
				c.Get(key)
			}
		}()
	}
	wg.Wait()
}
