// Package retry provides a retry policy with exponential backoff and jitter
// for at-least-once processing of operations against external APIs.
package retry

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
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

// RetryAfterInfo holds parsed Retry-After header information (Story 5.2).
type RetryAfterInfo struct {
	Wait   time.Duration
	Reason string // "retry-after-header" or "backoff"
}

// ParseRetryAfter parses the Retry-After header from an HTTP response.
// Supports both seconds (integer) and HTTP-date formats per RFC 7231.
// Returns zero duration if header is absent or unparsable.
func ParseRetryAfter(resp *http.Response) RetryAfterInfo {
	if resp == nil {
		return RetryAfterInfo{}
	}
	val := resp.Header.Get("Retry-After")
	if val == "" {
		return RetryAfterInfo{}
	}
	val = strings.TrimSpace(val)

	// Try integer seconds first.
	if secs, err := strconv.Atoi(val); err == nil && secs > 0 {
		return RetryAfterInfo{
			Wait:   time.Duration(secs) * time.Second,
			Reason: "retry-after-header",
		}
	}

	// Try HTTP-date format (RFC 1123).
	if t, err := http.ParseTime(val); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return RetryAfterInfo{
			Wait:   d,
			Reason: "retry-after-header",
		}
	}

	return RetryAfterInfo{}
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

// DoWithRetryAfter executes fn with retries, honoring Retry-After from 429 responses (Story 5.2).
// getRetryAfter is called after each failure to get the Retry-After wait from the last response.
// onBackoff is called with the actual wait duration before each retry.
func (p Policy) DoWithRetryAfter(
	ctx context.Context,
	fn func() error,
	getRetryAfter func() time.Duration,
	onRetry func(attempt int, err error),
	onBackoff func(wait time.Duration),
) Result {
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

		// Story 5.2: honor Retry-After header if available, otherwise use exponential backoff.
		var wait time.Duration
		if getRetryAfter != nil {
			wait = getRetryAfter()
		}
		if wait <= 0 {
			wait = p.backoff(attempt)
		} else {
			// Cap Retry-After to MaxBackoff for safety.
			if wait > p.MaxBackoff {
				wait = p.MaxBackoff
			}
		}

		if onBackoff != nil {
			onBackoff(wait)
		}

		select {
		case <-ctx.Done():
			return Result{Attempts: attempt, Err: ctx.Err()}
		case <-time.After(wait):
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
		// ±25% jitter. Guard against overflow: 2*quarter must be strictly positive
		// for rand.Int63n (panics on n <= 0).
		quarter := int64(d) / 4
		if quarter > 0 && 2*quarter > 0 {
			d = time.Duration(int64(d) - quarter + rand.Int63n(2*quarter))
		}
	}
	return d
}
