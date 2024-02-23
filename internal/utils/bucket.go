package utils

import (
	"errors"
	"sync"
	"time"
)

type TokenBucket struct {
	tokens        int
	maxTokens     int
	fillRate      time.Duration
	tokenAdded    chan struct{}
	tokenConsumed chan struct{}
	mu            sync.Mutex
}

func NewTokenBucket(maxTokens int, fillRate time.Duration) *TokenBucket {
	tb := &TokenBucket{
		tokens:        maxTokens,
		maxTokens:     maxTokens,
		fillRate:      fillRate,
		tokenAdded:    make(chan struct{}, 1),
		tokenConsumed: make(chan struct{}, 1),
	}

	go tb.Run()
	return tb
}

func (tb *TokenBucket) Run() {
	ticker := time.NewTicker(tb.fillRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tb.mu.Lock()
			if tb.tokens < tb.maxTokens {
				tb.tokens++
				tb.tokenAdded <- struct{}{}
			}
			tb.mu.Unlock()
		case <-tb.tokenConsumed:
			tb.mu.Lock()
			tb.tokens--
			tb.mu.Unlock()
		}
	}
}

func (tb *TokenBucket) GetToken() bool {
	select {
	case <-tb.tokenAdded:
		return true
	default:
		return false
	}
}

func (tb *TokenBucket) ConsumeToken() {
	tb.tokenConsumed <- struct{}{}
}

func TransmitData(tb *TokenBucket, data int) error {
	if tb.GetToken() {
		tb.ConsumeToken()
		return nil
	}

	return errors.New("data transmission denied. No tokens available")
}
