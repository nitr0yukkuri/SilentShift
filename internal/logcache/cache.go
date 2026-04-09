package logcache

import (
	"sync"
	"time"
)

type Entry struct {
	Author    string    `json:"author"`
	Message   string    `json:"message"`
	ChannelID string    `json:"channelId"`
	At        time.Time `json:"at"`
}

type Cache struct {
	mu       sync.RWMutex
	entries  []Entry
	capacity int
	idx      int
	filled   bool
}

func NewCache(capacity int) *Cache {
	if capacity < 8 {
		capacity = 8
	}

	return &Cache{
		entries:  make([]Entry, capacity),
		capacity: capacity,
	}
}

func (c *Cache) Add(e Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[c.idx] = e
	c.idx = (c.idx + 1) % c.capacity
	if c.idx == 0 {
		c.filled = true
	}
}

func (c *Cache) LastN(n int) []Entry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if n <= 0 {
		return nil
	}

	total := c.idx
	if c.filled {
		total = c.capacity
	}
	if total == 0 {
		return nil
	}
	if n > total {
		n = total
	}

	out := make([]Entry, 0, n)
	start := (c.idx - n + c.capacity) % c.capacity
	for i := 0; i < n; i++ {
		at := (start + i) % c.capacity
		out = append(out, c.entries[at])
	}

	return out
}
