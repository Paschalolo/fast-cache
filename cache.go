package main

import (
	"sync"
	"time"
)

type CacheItem struct {
	Value      string
	ExpiryTime time.Time
}

type Cache struct {
	sync.RWMutex
	items map[string]CacheItem
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]CacheItem),
	}
}

func (c *Cache) Set(Key, value string, ttl time.Duration) {
	c.Lock()
	defer c.Unlock()
	c.items[Key] = CacheItem{
		Value:      value,
		ExpiryTime: time.Now().Add(ttl),
	}

}

func (c *Cache) Get(key string) (string, bool) {
	c.RLock()
	defer c.RUnlock()
	item, found := c.items[key]
	if !found || time.Now().After(item.ExpiryTime) {
		return "", false
	}
	return item.Value, true
}

func (c *Cache) startEvictionTicker(d time.Duration) {
	ticker := time.NewTicker(d)
	go func() {
		for range ticker.C {
			go c.evictExpiredItems()
		}
	}()
}

func (c *Cache) evictExpiredItems() {
	c.Lock()
	defer c.Unlock()
	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiryTime) {
			delete(c.items, key)
		}
	}
}
