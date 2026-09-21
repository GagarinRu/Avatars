package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestBreakerOpensAfterFailures(t *testing.T) {
	b := NewBreaker(2, time.Minute)
	errTest := errors.New("fail")

	if err := b.Call(func() error { return errTest }); !errors.Is(err, errTest) {
		t.Fatalf("expected test error")
	}
	if err := b.Call(func() error { return errTest }); !errors.Is(err, errTest) {
		t.Fatalf("expected test error on second failure")
	}
	if err := b.Call(func() error { return nil }); !errors.Is(err, ErrOpen) {
		t.Fatalf("expected circuit open, got %v", err)
	}
}

func TestBreakerResetsOnSuccess(t *testing.T) {
	b := NewBreaker(2, time.Minute)
	errTest := errors.New("fail")

	_ = b.Call(func() error { return errTest })
	if err := b.Call(func() error { return nil }); err != nil {
		t.Fatalf("expected success to reset breaker")
	}
	if err := b.Call(func() error { return errTest }); !errors.Is(err, errTest) {
		t.Fatalf("expected single failure after reset")
	}
}
