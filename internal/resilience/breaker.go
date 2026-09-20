package resilience

import (
	"errors"
	"sync"
	"time"
)

// ErrOpen is returned when the circuit breaker is open.
var ErrOpen = errors.New("circuit breaker open")

// Breaker trips after consecutive failures and blocks calls for cooldown.
type Breaker struct {
	mu          sync.Mutex
	maxFailures int
	cooldown    time.Duration
	failures    int
	openUntil   time.Time
}

func NewBreaker(maxFailures int, cooldown time.Duration) *Breaker {
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &Breaker{maxFailures: maxFailures, cooldown: cooldown}
}

func (b *Breaker) Call(fn func() error) error {
	b.mu.Lock()
	if time.Now().Before(b.openUntil) {
		b.mu.Unlock()
		return ErrOpen
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		b.failures++
		if b.failures >= b.maxFailures {
			b.openUntil = time.Now().Add(b.cooldown)
			b.failures = 0
		}
		return err
	}
	b.failures = 0
	return nil
}
