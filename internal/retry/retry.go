// Package retry provides a retry policy with exponential backoff and jitter
// for at-least-once processing of operations against external APIs.
package retry

import (
	"context"
	"math/rand"
	"time"
)

// Policy defines the retry behaviour for transient failures.
type Policy struct {
	MaxAttempts    int           // total attempts (1 = no retry)
	InitialBackoff time.Duration // backoff for first retry
	MaxBackoff     time.Duration // upper cap on backoff
	Jitter         bool          // add randomised jitter to backoff
}

// DefaultPolicy is the standard retry policy per Story 2.2 tech notes.
var DefaultPolicy = Policy{
	MaxAttempts:    5,
	InitialBackoff: 1 * time.Second,
	MaxBackoff:     60 * time.Second,
	Jitter:         true,
}

// Result captures the outcome of a retried operation.
type Result struct {
	Attempts int   // how many attempts were made
	Err      error // nil on success; last error on exhaustion or context cancel
	Success  bool
}

// Do executes fn with retries according to the policy.
// onRetry is called before each retry sleep (attempt starts at 2).
// Returns as soon as fn succeeds, context is cancelled, or max attempts exhausted.
func (p Policy) Do(ctx context.Context, fn func() error, onRetry func(attempt int, err error)) Result {
	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return Result{Attempts: attempt, Success: true}
		}

		if attempt == p.MaxAttempts {
			return Result{Attempts: attempt, Err: err}
		}

		if onRetry != nil {
			onRetry(attempt+1, err)
		}

		backoff := p.backoff(attempt)
		select {
		case <-ctx.Done():
			return Result{Attempts: attempt, Err: ctx.Err()}
		case <-time.After(backoff):
		}
	}

	// Should not reach here, but safety net.
	return Result{Attempts: p.MaxAttempts, Err: nil}
}

// backoff calculates the sleep duration for the given attempt (1-based).
// Uses exponential backoff: InitialBackoff * 2^(attempt-1), capped at MaxBackoff.
func (p Policy) backoff(attempt int) time.Duration {
	d := p.InitialBackoff
	for i := 1; i < attempt; i++ {
		d *= 2
		if d > p.MaxBackoff {
			d = p.MaxBackoff
			break
		}
	}
	if p.Jitter && d > 0 {
		// ±25% jitter
		quarter := int64(d) / 4
		if quarter > 0 {
			d = time.Duration(int64(d) - quarter + rand.Int63n(2*quarter))
		}
	}
	return d
}
