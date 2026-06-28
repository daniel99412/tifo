package espn

import "sync"

type IDCache struct {
	mu   sync.RWMutex
	m    map[int]string
}

func NewIDCache() *IDCache {
	return &IDCache{m: make(map[int]string)}
}

func (c *IDCache) Get(fotmobID int) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.m[fotmobID]
	return id, ok
}

func (c *IDCache) Set(fotmobID int, espnID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[fotmobID] = espnID
}
