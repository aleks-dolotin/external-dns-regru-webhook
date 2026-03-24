package adapter

import (
	"testing"
)

func TestExternalIDCache_InMemory(t *testing.T) {
	c := NewExternalIDCache()

	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}

	// Cache miss
	if got := c.Get(key); got != "" {
		t.Errorf("expected empty on miss, got %q", got)
	}

	// Set and get
	c.Set(key, "12345")
	if got := c.Get(key); got != "12345" {
		t.Errorf("expected 12345, got %q", got)
	}
	if c.Len() != 1 {
		t.Errorf("expected len=1, got %d", c.Len())
	}

	// Delete
	c.Delete(key)
	if got := c.Get(key); got != "" {
		t.Errorf("expected empty after delete, got %q", got)
	}
	if c.Len() != 0 {
		t.Errorf("expected len=0 after delete, got %d", c.Len())
	}
}

func TestExternalIDCache_FallbackOnMiss(t *testing.T) {
	// Demonstrates cache miss behavior — caller should fall back to API
	c := NewExternalIDCache()

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
