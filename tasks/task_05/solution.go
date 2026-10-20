package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{ capacity: capacity, items: make(map[K]V), }
}

func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	if c.capacity <= 0 {
		var zero V
		return zero, false
	}

	value, answer := c.items[k]
	if !answer {
		var zero V
		return zero, false
	}
	return value, true
}

func (c *Cache[K, V]) Set(k K, v V) bool {
	if c.capacity <= 0 {
		return false
	}

	_, answer := c.items[k]
	if !answer {
		if c.capacity <= len(c.items) {
			return false
		}	
	}

	c.items[k] = v
	return true
}
