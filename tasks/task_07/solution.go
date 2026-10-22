package main

import (
	"container/list"
	"sync"
)

type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity    int
	ll          list.List
	items       map[K]*list.Element
	mutex_cache sync.Mutex
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V] { capacity: capacity, ll: list.List{}, items: make(map[K]*list.Element) }
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	c.mutex_cache.Lock()
	defer c.mutex_cache.Unlock()

	if c.capacity <= 0 {
		var zero V
		return zero, false
	}

	key_in_ll, value_in_ll := c.items[key]

	if !value_in_ll {
		var zero V
		return zero, false
	}

	c.ll.MoveToFront(key_in_ll)

	return key_in_ll.Value.(*entry[K, V]).value, true
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mutex_cache.Lock()
	defer c.mutex_cache.Unlock()

	if c.capacity <= 0 {
		return
	}

	key_in_ll, value_in_ll := c.items[key]
	
	if value_in_ll {
		key_in_ll.Value.(*entry[K, V]).value = value
		return
	}

	if c.ll.Len() >= c.capacity {
		most_oldest := c.ll.Back()
		if most_oldest != nil {
			c.ll.Remove(most_oldest)
			delete(c.items, most_oldest.Value.(*entry[K, V]).key)
		}
	}	
	
	new_element := c.ll.PushFront(&entry[K, V]{key: key, value: value})
	c.items[key] = new_element
	
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
