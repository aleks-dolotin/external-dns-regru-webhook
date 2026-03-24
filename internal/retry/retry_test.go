package retry

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestDo_SuccessFirstAttempt(t *testing.T) {
	p := Policy{MaxAttempts: 3, InitialBackoff: time.Millisecond}
	calls := 0
	r := p.Do(context.Background(), func() error {
		calls++
		return nil
	}, nil)

	if !r.Success {
		t.Error("expected success")
	}
	if r.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", r.Attempts)
	}
	if r.Err != nil {
		t.Errorf("expected nil error, got %v", r.Err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	p := Policy{MaxAttempts: 5, InitialBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond}
	calls := 0
	r := p.Do(context.Background(), func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	}, nil)

	if !r.Success {
		t.Error("expected success")
	}
	if r.Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", r.Attempts)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDo_ExhaustedRetries(t *testing.T) {
	p := Policy{MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}
	errPerm := errors.New("permanent")
	r := p.Do(context.Background(), func() error {
		return errPerm
	}, nil)

	if r.Success {
		t.Error("expected failure after exhaustion")
	}
	if r.Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", r.Attempts)
	}
	if !errors.Is(r.Err, errPerm) {
		t.Errorf("expected errPerm, got %v", r.Err)
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := Policy{MaxAttempts: 10, InitialBackoff: time.Second}

	calls := 0
	r := p.Do(ctx, func() error {
		calls++
		if calls == 1 {
			cancel() // cancel during first retry wait
		}
		return errors.New("err")
	}, nil)

	if r.Success {
		t.Error("expected failure on context cancel")
	}
	if r.Err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", r.Err)
	}
}

func TestDo_OnRetryCallback(t *testing.T) {
	p := Policy{MaxAttempts: 4, InitialBackoff: time.Millisecond}
	var retryAttempts []int
	calls := 0

	p.Do(context.Background(), func() error {
		calls++
		if calls < 3 {
			return errors.New("retry me")
		}
		return nil
	}, func(attempt int, err error) {
		retryAttempts = append(retryAttempts, attempt)
	})

	if len(retryAttempts) != 2 {
		t.Fatalf("expected 2 onRetry calls, got %d", len(retryAttempts))
	}
	if retryAttempts[0] != 2 || retryAttempts[1] != 3 {
		t.Errorf("expected retry attempts [2,3], got %v", retryAttempts)
	}
}

func TestBackoff_Exponential(t *testing.T) {
	p := Policy{InitialBackoff: 100 * time.Millisecond, MaxBackoff: 10 * time.Second, Jitter: false}

	expected := []time.Duration{
		100 * time.Millisecond, // attempt 1
		200 * time.Millisecond, // attempt 2
		400 * time.Millisecond, // attempt 3
		800 * time.Millisecond, // attempt 4
	}
	for i, exp := range expected {
		got := p.backoff(i + 1)
		if got != exp {
			t.Errorf("attempt %d: expected %v, got %v", i+1, exp, got)
		}
	}
}

func TestBackoff_CappedAtMax(t *testing.T) {
	p := Policy{InitialBackoff: time.Second, MaxBackoff: 5 * time.Second, Jitter: false}

	got := p.backoff(10) // 1s * 2^9 = 512s, capped to 5s
	if got != 5*time.Second {
		t.Errorf("expected 5s cap, got %v", got)
	}
}

func TestBackoff_Jitter(t *testing.T) {
	p := Policy{InitialBackoff: time.Second, MaxBackoff: 60 * time.Second, Jitter: true}

	// Run multiple times to verify jitter varies
	values := make(map[time.Duration]struct{})
	for i := 0; i < 50; i++ {
		values[p.backoff(3)] = struct{}{} // base = 4s, jitter ±25% → 3s–5s
	}
	if len(values) < 2 {
		t.Error("expected jitter to produce varying backoff values")
	}
}

func TestDefaultPolicy(t *testing.T) {
	if DefaultPolicy.MaxAttempts != 5 {
		t.Errorf("expected MaxAttempts 5, got %d", DefaultPolicy.MaxAttempts)
	}
	if DefaultPolicy.InitialBackoff != time.Second {
		t.Errorf("expected InitialBackoff 1s, got %v", DefaultPolicy.InitialBackoff)
	}
	if DefaultPolicy.MaxBackoff != 60*time.Second {
		t.Errorf("expected MaxBackoff 60s, got %v", DefaultPolicy.MaxBackoff)
	}
	if !DefaultPolicy.Jitter {
		t.Error("expected Jitter enabled")
	}
}

func TestDo_SingleAttempt(t *testing.T) {
	p := Policy{MaxAttempts: 1, InitialBackoff: time.Millisecond}
	r := p.Do(context.Background(), func() error {
		return errors.New("fail")
	}, nil)

	if r.Success {
		t.Error("expected failure with max 1 attempt")
	}
	if r.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", r.Attempts)
	}
}

func TestDo_ConcurrentSafe(t *testing.T) {
	p := Policy{MaxAttempts: 3, InitialBackoff: time.Millisecond}
	var count int64

	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			p.Do(context.Background(), func() error {
				atomic.AddInt64(&count, 1)
				return nil
			}, nil)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	if atomic.LoadInt64(&count) != 10 {
		t.Errorf("expected 10 calls, got %d", count)
	}
}

// --- Story 5.2: ParseRetryAfter and DoWithRetryAfter tests ---

func TestParseRetryAfter_Seconds(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "30")
	info := ParseRetryAfter(resp)
	if info.Wait != 30*time.Second {
		t.Errorf("expected 30s, got %v", info.Wait)
	}
	if info.Reason != "retry-after-header" {
		t.Errorf("expected reason retry-after-header, got %s", info.Reason)
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	future := time.Now().Add(60 * time.Second)
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", future.UTC().Format(http.TimeFormat))
	info := ParseRetryAfter(resp)
	// Should be roughly 60 seconds (within tolerance).
	if info.Wait < 50*time.Second || info.Wait > 70*time.Second {
		t.Errorf("expected ~60s, got %v", info.Wait)
	}
}

func TestParseRetryAfter_Empty(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	info := ParseRetryAfter(resp)
	if info.Wait != 0 {
		t.Errorf("expected 0, got %v", info.Wait)
	}
}

func TestParseRetryAfter_NilResponse(t *testing.T) {
	info := ParseRetryAfter(nil)
	if info.Wait != 0 {
		t.Errorf("expected 0, got %v", info.Wait)
	}
}

func TestDoWithRetryAfter_HonorsRetryAfterHeader(t *testing.T) {
	p := Policy{MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: 5 * time.Second, Jitter: false}

	calls := 0
	retryAfterWait := 10 * time.Millisecond
	var observedBackoff time.Duration

	r := p.DoWithRetryAfter(context.Background(),
		func() error {
			calls++
			if calls < 2 {
				return errors.New("429")
			}
			return nil
		},
		func() time.Duration { return retryAfterWait },
		nil,
		func(wait time.Duration) { observedBackoff = wait },
	)

	if !r.Success {
		t.Error("expected success")
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
	if observedBackoff != retryAfterWait {
		t.Errorf("expected backoff %v from Retry-After, got %v", retryAfterWait, observedBackoff)
	}
}

func TestDoWithRetryAfter_FallsBackToExponential(t *testing.T) {
	p := Policy{MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: time.Second, Jitter: false}

	calls := 0
	var backoffs []time.Duration

	p.DoWithRetryAfter(context.Background(),
		func() error {
			calls++
			if calls < 3 {
				return errors.New("500")
			}
			return nil
		},
		func() time.Duration { return 0 }, // no Retry-After
		nil,
		func(wait time.Duration) { backoffs = append(backoffs, wait) },
	)

	if len(backoffs) != 2 {
		t.Fatalf("expected 2 backoffs, got %d", len(backoffs))
	}
	// First backoff = InitialBackoff = 1ms, second = 2ms.
	if backoffs[0] != time.Millisecond {
		t.Errorf("first backoff: expected 1ms, got %v", backoffs[0])
	}
	if backoffs[1] != 2*time.Millisecond {
		t.Errorf("second backoff: expected 2ms, got %v", backoffs[1])
	}
}

func TestDoWithRetryAfter_CapsRetryAfterToMaxBackoff(t *testing.T) {
	p := Policy{MaxAttempts: 2, InitialBackoff: time.Millisecond, MaxBackoff: 50 * time.Millisecond, Jitter: false}

	var observedBackoff time.Duration
	p.DoWithRetryAfter(context.Background(),
		func() error { return errors.New("429") },
		func() time.Duration { return 10 * time.Second }, // way over MaxBackoff
		nil,
		func(wait time.Duration) { observedBackoff = wait },
	)

	if observedBackoff != 50*time.Millisecond {
		t.Errorf("expected capped to 50ms, got %v", observedBackoff)
	}
}
