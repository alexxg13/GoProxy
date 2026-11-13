package lrucache

import (
	"container/list"
	"sync"
	"time"
)

type entry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
}

type Cache[K comparable, V any] struct {
	mu       sync.RWMutex
	capacity int
	ttl      time.Duration
	items    map[K]*list.Element
	order    *list.List
}

func New[K comparable, V any](capacity int, ttl time.Duration) *Cache[K, V] {
	if capacity <= 0 {
		capacity = 1024
	}
	return &Cache[K, V]{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[K]*list.Element),
		order:    list.New(),
	}
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	item := elem.Value.(*entry[K, V])
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		c.removeElement(elem)
		var zero V
		return zero, false
	}

	c.order.MoveToFront(elem)
	return item.value, true
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		item := elem.Value.(*entry[K, V])
		item.value = value
		if c.ttl > 0 {
			item.expiresAt = time.Now().Add(c.ttl)
		}
		return
	}

	expiresAt := time.Time{}
	if c.ttl > 0 {
		expiresAt = time.Now().Add(c.ttl)
	}

	elem := c.order.PushFront(&entry[K, V]{key: key, value: value, expiresAt: expiresAt})
	c.items[key] = elem

	if c.order.Len() > c.capacity {
		back := c.order.Back()
		if back != nil {
			c.removeElement(back)
		}
	}
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}

func (c *Cache[K, V]) removeElement(elem *list.Element) {
	item := elem.Value.(*entry[K, V])
	delete(c.items, item.key)
	c.order.Remove(elem)
}
