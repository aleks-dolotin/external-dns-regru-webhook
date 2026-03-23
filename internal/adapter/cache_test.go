package adapter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExternalIDCache_InMemory(t *testing.T) {
	c, err := NewExternalIDCache("")
	if err != nil {
		t.Fatalf("NewExternalIDCache: %v", err)
	}

	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}

	// Cache miss
	if got := c.Get(key); got != "" {
		t.Errorf("expected empty on miss, got %q", got)
	}

	// Set and get
	if err := c.Set(key, "12345"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := c.Get(key); got != "12345" {
		t.Errorf("expected 12345, got %q", got)
	}
	if c.Len() != 1 {
		t.Errorf("expected len=1, got %d", c.Len())
	}

	// Delete
	if err := c.Delete(key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := c.Get(key); got != "" {
		t.Errorf("expected empty after delete, got %q", got)
	}
	if c.Len() != 0 {
		t.Errorf("expected len=0 after delete, got %d", c.Len())
	}
}

func TestExternalIDCache_Persisted(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "cache.json")

	// Create and populate cache
	c1, err := NewExternalIDCache(fp)
	if err != nil {
		t.Fatalf("NewExternalIDCache: %v", err)
	}
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	if err := c1.Set(key, "99999"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(fp); err != nil {
		t.Fatalf("cache file should exist: %v", err)
	}

	// Create second cache from same file — should load persisted data
	c2, err := NewExternalIDCache(fp)
	if err != nil {
		t.Fatalf("NewExternalIDCache (reload): %v", err)
	}
	if got := c2.Get(key); got != "99999" {
		t.Errorf("expected 99999 from persisted cache, got %q", got)
	}
}

func TestExternalIDCache_SurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "cache.json")

	// Populate with multiple entries
	c, err := NewExternalIDCache(fp)
	if err != nil {
		t.Fatalf("NewExternalIDCache: %v", err)
	}
	keys := []CacheKey{
		{Zone: "a.com", Name: "www", RecType: "A"},
		{Zone: "b.com", Name: "mail", RecType: "CNAME"},
		{Zone: "c.com", Name: "@", RecType: "TXT"},
	}
	ids := []string{"111", "222", "333"}
	for i, k := range keys {
		if err := c.Set(k, ids[i]); err != nil {
			t.Fatalf("Set %v: %v", k, err)
		}
	}

	// Simulate restart
	c2, err := NewExternalIDCache(fp)
	if err != nil {
		t.Fatalf("restart NewExternalIDCache: %v", err)
	}
	for i, k := range keys {
		if got := c2.Get(k); got != ids[i] {
			t.Errorf("key %v: expected %s, got %q", k, ids[i], got)
		}
	}
	if c2.Len() != 3 {
		t.Errorf("expected 3 entries after restart, got %d", c2.Len())
	}
}

func TestExternalIDCache_FallbackOnMiss(t *testing.T) {
	// Demonstrates cache miss behavior — caller should fall back to API
	c, err := NewExternalIDCache("")
	if err != nil {
		t.Fatalf("NewExternalIDCache: %v", err)
	}

	key := CacheKey{Zone: "example.com", Name: "missing", RecType: "A"}
	if got := c.Get(key); got != "" {
		t.Errorf("expected empty for cache miss, got %q", got)
	}
}

func TestCacheKey_String(t *testing.T) {
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	expected := "example.com/www/A"
	if got := key.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
