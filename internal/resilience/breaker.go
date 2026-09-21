package resilience

import (
	"errors"
	"sync"
	"time"
)

// ErrOpen is returned when the circuit breaker is open.
var ErrOpen = errors.New("circuit breaker open")

type breakerState int

const (
	breakerClosed   breakerState = 0
	breakerOpen     breakerState = 1
	breakerHalfOpen breakerState = 2
)

// Breaker trips after consecutive failures and blocks calls for cooldown.
type Breaker struct {
	mu               sync.Mutex
	maxFailures      int
	cooldown         time.Duration
	halfOpenMaxCalls int
	failures         int
	openUntil        time.Time
	state            breakerState
	halfOpenInFlight int
}

func NewBreaker(maxFailures int, cooldown time.Duration) *Breaker {
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &Breaker{
		maxFailures:      maxFailures,
		cooldown:         cooldown,
		halfOpenMaxCalls: 1,
	}
}

func (b *Breaker) Call(fn func() error) error {
	b.mu.Lock()
	now := time.Now()
	switch b.state {
	case breakerOpen:
		if now.Before(b.openUntil) {
			b.mu.Unlock()
			return ErrOpen
		}
		b.state = breakerHalfOpen
		b.halfOpenInFlight = 1
	case breakerHalfOpen:
		if b.halfOpenInFlight >= b.halfOpenMaxCalls {
			b.mu.Unlock()
			return ErrOpen
		}
		b.halfOpenInFlight++
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		if b.state == breakerHalfOpen {
			b.state = breakerOpen
			b.openUntil = time.Now().Add(b.cooldown)
			b.failures = 0
			b.halfOpenInFlight = 0
			return err
		}
		b.failures++
		if b.failures >= b.maxFailures {
			b.state = breakerOpen
			b.openUntil = time.Now().Add(b.cooldown)
			b.failures = 0
		}
		return err
	}

	b.state = breakerClosed
	b.failures = 0
	b.halfOpenInFlight = 0
	return nil
}
