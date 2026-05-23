package cache

import (
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/sync/singleflight"
)

const (
	minTTL         = 60 * time.Second
	maxTTL         = 24 * time.Hour
	defaultMaxSize = 10000
)

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*entry
	order   []string
	maxSize int
	group   singleflight.Group
	Stats   Stats
}

type entry struct {
	msg        *dns.Msg
	expiresAt  time.Time
	insertedAt time.Time
}

type Stats struct {
	Hits       uint64
	Misses     uint64
	Evictions  uint64
	TotalItems int
}

func New(maxSize int) *Cache {
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	return &Cache{
		entries: make(map[string]*entry),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

func makeKey(name string, qtype, qclass uint16) string {
	return strings.ToLower(name) + "|" + dns.TypeToString[qtype] + "|" + dns.ClassToString[qclass]
}

func (c *Cache) Get(name string, qtype, qclass uint16) (*dns.Msg, bool) {
	key := makeKey(name, qtype, qclass)

	c.mu.RLock()
	e, found := c.entries[key]
	c.mu.RUnlock()

	if !found {
		c.mu.Lock()
		c.Stats.Misses++
		c.mu.Unlock()
		return nil, false
	}

	now := time.Now()
	if now.After(e.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.Stats.Misses++
		c.Stats.TotalItems = len(c.entries)
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	c.Stats.Hits++
	c.mu.Unlock()

	return adjustTTLs(e.msg.Copy(), now.Sub(e.insertedAt)), true
}

func (c *Cache) Set(name string, qtype, qclass uint16, msg *dns.Msg) {
	if msg == nil || len(msg.Answer) == 0 {
		return
	}

	ttl := extractMinTTL(msg)
	if ttl < minTTL {
		ttl = minTTL
	}
	if ttl > maxTTL {
		ttl = maxTTL
	}

	key := makeKey(name, qtype, qclass)
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = &entry{
		msg:        msg.Copy(),
		expiresAt:  now.Add(ttl),
		insertedAt: now,
	}
	c.order = append(c.order, key)
	c.Stats.TotalItems = len(c.entries)
}

// singleflight prevents multiple goroutines from querying the same domain simultaneously
func (c *Cache) GetOrFetch(name string, qtype, qclass uint16, fetch func() (*dns.Msg, error)) (*dns.Msg, error) {
	if msg, ok := c.Get(name, qtype, qclass); ok {
		return msg, nil
	}

	key := makeKey(name, qtype, qclass)
	result, err, _ := c.group.Do(key, func() (interface{}, error) {
		msg, err := fetch()
		if err != nil {
			return nil, err
		}
		c.Set(name, qtype, qclass, msg)
		return msg, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*dns.Msg), nil
}

func (c *Cache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*entry)
	c.order = c.order[:0]
	c.Stats = Stats{}
}

func (c *Cache) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Stats
}

func (c *Cache) StartJanitor(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-stop:
				return
			}
		}
	}()
}

func (c *Cache) cleanup() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, key)
		}
	}
	c.Stats.TotalItems = len(c.entries)
}

func (c *Cache) evictOldest() {
	for len(c.order) > 0 {
		key := c.order[0]
		c.order = c.order[1:]
		if _, exists := c.entries[key]; exists {
			delete(c.entries, key)
			c.Stats.Evictions++
			return
		}
	}
}

func extractMinTTL(msg *dns.Msg) time.Duration {
	var min uint32 = 86400
	for _, rr := range msg.Answer {
		if ttl := rr.Header().Ttl; ttl < min {
			min = ttl
		}
	}
	for _, rr := range msg.Ns {
		if ttl := rr.Header().Ttl; ttl < min {
			min = ttl
		}
	}
	return time.Duration(min) * time.Second
}

// adjustTTLs subtracts time-in-cache from TTLs so clients see accurate remaining time
func adjustTTLs(msg *dns.Msg, age time.Duration) *dns.Msg {
	ageSecs := uint32(age.Seconds())
	for _, rr := range msg.Answer {
		h := rr.Header()
		if h.Ttl > ageSecs {
			h.Ttl -= ageSecs
		} else {
			h.Ttl = 1
		}
	}
	for _, rr := range msg.Ns {
		h := rr.Header()
		if h.Ttl > ageSecs {
			h.Ttl -= ageSecs
		} else {
			h.Ttl = 1
		}
	}
	for _, rr := range msg.Extra {
		h := rr.Header()
		if h.Ttl > ageSecs {
			h.Ttl -= ageSecs
		} else {
			h.Ttl = 1
		}
	}
	return msg
}
