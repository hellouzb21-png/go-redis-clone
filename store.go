package main

import (
	"sync"
	"time"
)

type Item struct {
	Value  string
	Expiry int64 // 0 if no expiry, otherwise Unix timestamp in seconds
}

type Store struct {
	mu   sync.RWMutex
	data map[string]Item
}

func NewStore() *Store {
	s := &Store{
		data: make(map[string]Item),
	}
	// Start active expiry goroutine
	go s.activeExpiry()
	return s
}

func (s *Store) Set(key, value string, ttlSeconds int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var expiry int64 = 0
	if ttlSeconds > 0 {
		expiry = time.Now().Unix() + int64(ttlSeconds)
	}

	s.data[key] = Item{
		Value:  value,
		Expiry: expiry,
	}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	item, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return "", false
	}

	// Passive expiry check
	if item.Expiry > 0 && time.Now().Unix() > item.Expiry {
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return "", false
	}

	return item.Value, true
}

func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// activeExpiry periodically sweeps the store for expired keys
func (s *Store) activeExpiry() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		now := time.Now().Unix()
		
		s.mu.Lock()
		// Go's map iteration is random, which is good for a random sweep like real Redis
		// To be perfectly efficient on huge maps, we'd only sample a subset.
		// For simplicity, we sweep up to 20 keys per second.
		checked := 0
		for k, v := range s.data {
			if checked >= 20 {
				break
			}
			if v.Expiry > 0 && now > v.Expiry {
				delete(s.data, k)
			}
			checked++
		}
		s.mu.Unlock()
	}
}
