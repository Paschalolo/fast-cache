package main

import (
	"container/list"
	"log"
	"sync"
	"time"
)

type CacheItem struct {
	Value      string
	ExpiryTime time.Time
}

type Cache struct {
	sync.RWMutex
	items    map[string]*list.Element
	eviction *list.List
	capacity int
}

type entry struct {
	key   string
	value CacheItem
}

func NewCache(capacity int) *Cache {
	return &Cache{
		items:    make(map[string]*list.Element),
		eviction: list.New(),
		capacity: capacity,
	}
}

func (c *Cache) Set(Key, value string, ttl time.Duration) {
	duration := ttl * time.Minute
	c.Lock()
	defer c.Unlock()
	if elem, found := c.items[Key]; found {
		c.eviction.Remove(elem)
		delete(c.items, Key)
	}

	// Evict the least recently used item if the cache is at capacity
	if c.eviction.Len() >= c.capacity {
		c.evictLRU()
	}
	item := CacheItem{
		Value:      value,
		ExpiryTime: time.Now().Add(duration),
	}
	elem := c.eviction.PushFront(&entry{Key, item})
	c.items[Key] = elem
}

func (c *Cache) Get(key string) (string, bool) {
	c.RLock()
	defer c.RUnlock()
	log.Println("looking")
	elem, found := c.items[key]
	log.Println(found)
	if !found || time.Now().After(elem.Value.(*entry).value.ExpiryTime) {
		return "", false
	}
	return elem.Value.(*entry).value.Value, true
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
	for key, elem := range c.items {
		if now.After(elem.Value.(*entry).value.ExpiryTime) {
			c.eviction.Remove(elem)
			delete(c.items, key)
		}
	}
}

func (c *Cache) evictLRU() {
	elem := c.eviction.Back()
	if elem != nil {
		c.eviction.Remove(elem)
		kv := elem.Value.(*entry)
		delete(c.items, kv.key)
	}
}
