package handler

import (
	"sync"
	"time"
)

type failEntry struct {
	count        int
	windowStart  time.Time
	blockedUntil time.Time
}

// failLimiter counts failed logins per client IP. After maxFails failures
// inside window, the IP is blocked for blockFor.
type failLimiter struct {
	mu       sync.Mutex
	entries  map[string]*failEntry
	maxFails int
	window   time.Duration
	blockFor time.Duration
}

func newFailLimiter(maxFails int, window, blockFor time.Duration) *failLimiter {
	return &failLimiter{
		entries:  make(map[string]*failEntry),
		maxFails: maxFails,
		window:   window,
		blockFor: blockFor,
	}
}

func (l *failLimiter) blocked(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[ip]
	return ok && time.Now().Before(e.blockedUntil)
}

func (l *failLimiter) fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.prune(now)
	e, ok := l.entries[ip]
	if !ok || now.Sub(e.windowStart) > l.window {
		e = &failEntry{windowStart: now}
		l.entries[ip] = e
	}
	e.count++
	if e.count >= l.maxFails {
		e.blockedUntil = now.Add(l.blockFor)
		e.count = 0
		e.windowStart = now
	}
}

func (l *failLimiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, ip)
}

// prune drops entries that are neither blocked nor inside their window.
// Caller must hold l.mu.
func (l *failLimiter) prune(now time.Time) {
	for ip, e := range l.entries {
		if now.After(e.blockedUntil) && now.Sub(e.windowStart) > l.window {
			delete(l.entries, ip)
		}
	}
}
