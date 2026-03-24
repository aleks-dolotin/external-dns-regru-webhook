package quota

import (
	"testing"
	"time"
)

func TestAllowRequest_WithinQuota(t *testing.T) {
	m := New([]NamespaceQuota{
		{Namespace: "team-a", LimitPerHour: 3},
	})

	for i := 0; i < 3; i++ {
		if !m.AllowRequest("team-a") {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}
}

func TestAllowRequest_ExceedsQuota(t *testing.T) {
	m := New([]NamespaceQuota{
		{Namespace: "team-a", LimitPerHour: 2},
	})

	if !m.AllowRequest("team-a") {
		t.Fatal("first request should be allowed")
	}
	if !m.AllowRequest("team-a") {
		t.Fatal("second request should be allowed")
	}
	if m.AllowRequest("team-a") {
		t.Fatal("third request should be blocked (quota=2)")
	}
}

func TestAllowRequest_NoQuotaNamespace(t *testing.T) {
	m := New([]NamespaceQuota{
		{Namespace: "team-a", LimitPerHour: 1},
	})

	// team-b has no quota — should always be allowed.
	for i := 0; i < 100; i++ {
		if !m.AllowRequest("team-b") {
			t.Fatalf("no-quota namespace should always be allowed, blocked at %d", i)
		}
	}
}

func TestAllowRequest_WindowReset(t *testing.T) {
	now := time.Date(2026, 3, 24, 10, 30, 0, 0, time.UTC)
	m := New([]NamespaceQuota{
		{Namespace: "team-a", LimitPerHour: 1},
	})
	m.now = func() time.Time { return now }
	// Re-initialize counter with test clock.
	m.counters["team-a"].windowEnd = nextWindowEnd(now)

	if !m.AllowRequest("team-a") {
		t.Fatal("first request should be allowed")
	}
	if m.AllowRequest("team-a") {
		t.Fatal("second request should be blocked")
	}

	// Advance past window end (11:00).
	now = time.Date(2026, 3, 24, 11, 0, 1, 0, time.UTC)
	m.now = func() time.Time { return now }

	if !m.AllowRequest("team-a") {
		t.Fatal("request after window reset should be allowed")
	}
}

func TestCurrentUsage(t *testing.T) {
	m := New([]NamespaceQuota{
		{Namespace: "team-a", LimitPerHour: 10},
	})

	used, limit := m.CurrentUsage("team-a")
	if used != 0 || limit != 10 {
		t.Fatalf("expected (0, 10), got (%d, %d)", used, limit)
	}

	m.AllowRequest("team-a")
	m.AllowRequest("team-a")

	used, limit = m.CurrentUsage("team-a")
	if used != 2 || limit != 10 {
		t.Fatalf("expected (2, 10), got (%d, %d)", used, limit)
	}

	// No quota namespace.
	used, limit = m.CurrentUsage("team-b")
	if used != 0 || limit != 0 {
		t.Fatalf("expected (0, 0) for no-quota, got (%d, %d)", used, limit)
	}
}

func TestUpdateQuotas_NewLimits(t *testing.T) {
	m := New([]NamespaceQuota{
		{Namespace: "team-a", LimitPerHour: 5},
	})

	m.AllowRequest("team-a")
	m.AllowRequest("team-a")

	// Update: increase team-a quota, add team-b.
	m.UpdateQuotas([]NamespaceQuota{
		{Namespace: "team-a", LimitPerHour: 100},
		{Namespace: "team-b", LimitPerHour: 50},
	})

	// team-a retains usage but new limit.
	used, limit := m.CurrentUsage("team-a")
	if used != 2 || limit != 100 {
		t.Fatalf("expected (2, 100) after update, got (%d, %d)", used, limit)
	}

	// team-b is new.
	used, limit = m.CurrentUsage("team-b")
	if used != 0 || limit != 50 {
		t.Fatalf("expected (0, 50) for new team-b, got (%d, %d)", used, limit)
	}
}

func TestUpdateQuotas_RemoveQuota(t *testing.T) {
	m := New([]NamespaceQuota{
		{Namespace: "team-a", LimitPerHour: 5},
		{Namespace: "team-b", LimitPerHour: 10},
	})

	// Remove team-b quota.
	m.UpdateQuotas([]NamespaceQuota{
		{Namespace: "team-a", LimitPerHour: 5},
	})

	// team-b should now be unrestricted.
	if !m.AllowRequest("team-b") {
		t.Fatal("team-b should be unrestricted after quota removal")
	}
}

func TestRejectionMessage(t *testing.T) {
	msg := RejectionMessage("team-a", 100)
	if msg != "namespace team-a: quota exceeded (limit: 100/hr)" {
		t.Errorf("unexpected message: %s", msg)
	}
}

func TestNew_ZeroQuota_Ignored(t *testing.T) {
	m := New([]NamespaceQuota{
		{Namespace: "team-a", LimitPerHour: 0},
	})

	// LimitPerHour=0 means no quota — should always be allowed.
	for i := 0; i < 100; i++ {
		if !m.AllowRequest("team-a") {
			t.Fatalf("zero-quota namespace should be unrestricted, blocked at %d", i)
		}
	}
}
