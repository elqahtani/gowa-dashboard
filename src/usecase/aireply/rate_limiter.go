package aireply

import (
	"sync"
	"time"
)

// RateLimiter is a tiny per-key minimum-interval limiter. Concurrency-safe.
type RateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     map[string]time.Time
}

func NewRateLimiter(interval time.Duration) *RateLimiter {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	return &RateLimiter{interval: interval, last: make(map[string]time.Time)}
}

// Allow returns true if the key has not been allowed within the configured
// interval. On a true result, the key's last-allowed timestamp is updated.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if prev, ok := r.last[key]; ok && now.Sub(prev) < r.interval {
		return false
	}
	r.last[key] = now
	return true
}

// SetInterval lets tests / config changes adjust the interval at runtime.
func (r *RateLimiter) SetInterval(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if d > 0 {
		r.interval = d
	}
}
