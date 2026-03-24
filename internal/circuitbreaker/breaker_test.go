package circuitbreaker

import (
	"sync"
	"testing"
	"time"
)

func newTestManager(cfg Config) (*Manager, *fakeClock) {
	clk := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	mgr := NewManager(cfg)
	mgr.now = clk.Now
	return mgr, clk
}

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestCircuitStartsClosed(t *testing.T) {
	mgr, _ := newTestManager(DefaultConfig())
	if s := mgr.GetState("zone.com"); s != StateClosed {
		t.Errorf("expected closed, got %v", s)
	}
}

func TestAllowRequest_ClosedState(t *testing.T) {
	mgr, _ := newTestManager(DefaultConfig())
	if err := mgr.AllowRequest("zone.com"); err != nil {
		t.Errorf("closed circuit should allow: %v", err)
	}
}

func TestCircuitOpens_OnHighErrorRate(t *testing.T) {
	cfg := Config{
		ErrorThreshold: 0.5,
		Window:         60 * time.Second,
		CooldownPeriod: 5 * time.Minute,
		HalfOpenProbes: 3,
	}
	mgr, _ := newTestManager(cfg)

	// 2 successes + 5 failures = 71% error rate → should open.
	mgr.RecordSuccess("zone.com")
	mgr.RecordSuccess("zone.com")
	for i := 0; i < 5; i++ {
		mgr.RecordFailure("zone.com")
	}

	if s := mgr.GetState("zone.com"); s != StateOpen {
		t.Errorf("expected open, got %v", s)
	}

	// Verify requests are rejected.
	if err := mgr.AllowRequest("zone.com"); err == nil {
		t.Error("open circuit should reject requests")
	}
}

func TestCircuitOpen_OtherZonesUnaffected(t *testing.T) {
	mgr, _ := newTestManager(DefaultConfig())

	// Open circuit for zone-a.
	mgr.RecordSuccess("zone-a.com")
	for i := 0; i < 10; i++ {
		mgr.RecordFailure("zone-a.com")
	}
	if s := mgr.GetState("zone-a.com"); s != StateOpen {
		t.Fatalf("zone-a should be open, got %v", s)
	}

	// zone-b should be unaffected.
	if err := mgr.AllowRequest("zone-b.com"); err != nil {
		t.Errorf("zone-b should not be affected: %v", err)
	}
}

func TestCircuitTransitionsToHalfOpen_AfterCooldown(t *testing.T) {
	cfg := Config{
		ErrorThreshold: 0.5,
		Window:         60 * time.Second,
		CooldownPeriod: 5 * time.Minute,
		HalfOpenProbes: 3,
	}
	mgr, clk := newTestManager(cfg)

	// Open the circuit.
	mgr.RecordSuccess("zone.com")
	for i := 0; i < 10; i++ {
		mgr.RecordFailure("zone.com")
	}
	if s := mgr.GetState("zone.com"); s != StateOpen {
		t.Fatalf("expected open, got %v", s)
	}

	// Advance past cooldown.
	clk.Advance(6 * time.Minute)

	// Request should be allowed (transition to half-open).
	if err := mgr.AllowRequest("zone.com"); err != nil {
		t.Errorf("should allow after cooldown: %v", err)
	}
	if s := mgr.GetState("zone.com"); s != StateHalfOpen {
		t.Errorf("expected half-open, got %v", s)
	}
}

func TestCircuitCloses_AfterSuccessfulProbes(t *testing.T) {
	cfg := Config{
		ErrorThreshold: 0.5,
		Window:         60 * time.Second,
		CooldownPeriod: 5 * time.Minute,
		HalfOpenProbes: 3,
	}
	mgr, clk := newTestManager(cfg)

	// Open → half-open.
	mgr.RecordSuccess("zone.com")
	for i := 0; i < 10; i++ {
		mgr.RecordFailure("zone.com")
	}
	clk.Advance(6 * time.Minute)
	_ = mgr.AllowRequest("zone.com") // triggers half-open

	// Probe successes.
	mgr.RecordSuccess("zone.com")
	mgr.RecordSuccess("zone.com")
	mgr.RecordSuccess("zone.com")

	if s := mgr.GetState("zone.com"); s != StateClosed {
		t.Errorf("expected closed after probes, got %v", s)
	}
}

func TestCircuitReopens_OnHalfOpenFailure(t *testing.T) {
	cfg := Config{
		ErrorThreshold: 0.5,
		Window:         60 * time.Second,
		CooldownPeriod: 5 * time.Minute,
		HalfOpenProbes: 3,
	}
	mgr, clk := newTestManager(cfg)

	// Open → half-open.
	mgr.RecordSuccess("zone.com")
	for i := 0; i < 10; i++ {
		mgr.RecordFailure("zone.com")
	}
	clk.Advance(6 * time.Minute)
	_ = mgr.AllowRequest("zone.com")

	// Failure in half-open → re-open.
	mgr.RecordFailure("zone.com")

	if s := mgr.GetState("zone.com"); s != StateOpen {
		t.Errorf("expected re-open, got %v", s)
	}
}

func TestOnStateChange_Callback(t *testing.T) {
	mgr, _ := newTestManager(DefaultConfig())
	var transitions []struct{ zone, from, to string }
	mgr.OnStateChange = func(zone string, from, to State) {
		transitions = append(transitions, struct{ zone, from, to string }{zone, from.String(), to.String()})
	}

	mgr.RecordSuccess("z.com")
	for i := 0; i < 10; i++ {
		mgr.RecordFailure("z.com")
	}

	if len(transitions) == 0 {
		t.Fatal("expected at least one state transition callback")
	}
	last := transitions[len(transitions)-1]
	if last.to != "open" {
		t.Errorf("expected transition to open, got %s", last.to)
	}
}

func TestForceClose(t *testing.T) {
	mgr, _ := newTestManager(DefaultConfig())

	// Open circuit.
	mgr.RecordSuccess("z.com")
	for i := 0; i < 10; i++ {
		mgr.RecordFailure("z.com")
	}
	if mgr.GetState("z.com") != StateOpen {
		t.Fatal("expected open")
	}

	mgr.ForceClose("z.com")
	if mgr.GetState("z.com") != StateClosed {
		t.Error("expected closed after ForceClose")
	}
}

func TestZoneStates(t *testing.T) {
	mgr, _ := newTestManager(DefaultConfig())
	mgr.RecordSuccess("a.com")
	mgr.RecordSuccess("b.com")

	states := mgr.ZoneStates()
	if len(states) != 2 {
		t.Errorf("expected 2 zones, got %d", len(states))
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr, _ := newTestManager(DefaultConfig())
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			zone := "zone.com"
			_ = mgr.AllowRequest(zone)
			if id%3 == 0 {
				mgr.RecordFailure(zone)
			} else {
				mgr.RecordSuccess(zone)
			}
		}(i)
	}
	wg.Wait()
}

func TestWindowReset(t *testing.T) {
	cfg := Config{
		ErrorThreshold: 0.5,
		Window:         10 * time.Second,
		CooldownPeriod: 5 * time.Minute,
		HalfOpenProbes: 3,
	}
	mgr, clk := newTestManager(cfg)

	// Record failures but not enough to trip.
	mgr.RecordSuccess("z.com")
	mgr.RecordSuccess("z.com")
	mgr.RecordSuccess("z.com")
	mgr.RecordFailure("z.com")

	// Advance past window — counters should reset.
	clk.Advance(15 * time.Second)

	// Now record successes — should stay closed.
	for i := 0; i < 5; i++ {
		mgr.RecordSuccess("z.com")
	}
	mgr.RecordFailure("z.com")

	if s := mgr.GetState("z.com"); s != StateClosed {
		t.Errorf("expected closed after window reset, got %v", s)
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		s    State
		want string
	}{
		{StateClosed, "closed"},
		{StateHalfOpen, "half-open"},
		{StateOpen, "open"},
		{State(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}
