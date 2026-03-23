package retry

import (
	"context"
	"errors"
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
