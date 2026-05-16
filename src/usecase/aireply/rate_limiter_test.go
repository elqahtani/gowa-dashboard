package aireply

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_FirstCallAllowed(t *testing.T) {
	rl := NewRateLimiter(50 * time.Millisecond)
	assert.True(t, rl.Allow("chat-a"))
}

func TestRateLimiter_BlocksWithinInterval(t *testing.T) {
	rl := NewRateLimiter(50 * time.Millisecond)
	assert.True(t, rl.Allow("chat-a"))
	assert.False(t, rl.Allow("chat-a"))
}

func TestRateLimiter_RecoversAfterInterval(t *testing.T) {
	rl := NewRateLimiter(20 * time.Millisecond)
	assert.True(t, rl.Allow("chat-a"))
	time.Sleep(30 * time.Millisecond)
	assert.True(t, rl.Allow("chat-a"))
}

func TestRateLimiter_PerKeyIsolated(t *testing.T) {
	rl := NewRateLimiter(50 * time.Millisecond)
	assert.True(t, rl.Allow("chat-a"))
	assert.True(t, rl.Allow("chat-b"))
	assert.False(t, rl.Allow("chat-a"))
	assert.False(t, rl.Allow("chat-b"))
}
