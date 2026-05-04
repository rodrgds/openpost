package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	windowStart time.Time
	count       int
}

func New() *Limiter {
	return &Limiter{buckets: make(map[string]*bucket)}
}

func (l *Limiter) Allow(key string, limit int, window time.Duration) bool {
	now := time.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()

	current, ok := l.buckets[key]
	if !ok || now.Sub(current.windowStart) >= window {
		l.buckets[key] = &bucket{
			windowStart: now,
			count:       1,
		}
		return true
	}

	if current.count >= limit {
		return false
	}

	current.count++
	return true
}
