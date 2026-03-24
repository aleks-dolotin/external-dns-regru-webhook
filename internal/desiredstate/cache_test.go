package desiredstate

import (
	"testing"
)

func TestCache_PutAndForZone(t *testing.T) {
	c := New()
	c.Put("example.com", "app.example.com", "A", "1.2.3.4", 300)
	c.Put("example.com", "api.example.com", "A", "5.6.7.8", 300)
	c.Put("other.com", "web.other.com", "CNAME", "cdn.other.com", 600)

	records := c.ForZone("example.com")
	if len(records) != 2 {
		t.Fatalf("expected 2 records for example.com, got %d", len(records))
	}

	records = c.ForZone("other.com")
	if len(records) != 1 {
		t.Fatalf("expected 1 record for other.com, got %d", len(records))
	}

	records = c.ForZone("nonexistent.com")
	if len(records) != 0 {
		t.Fatalf("expected 0 records for nonexistent.com, got %d", len(records))
	}
}

func TestCache_PutOverwrites(t *testing.T) {
	c := New()
	c.Put("example.com", "app.example.com", "A", "1.2.3.4", 300)
	c.Put("example.com", "app.example.com", "A", "9.9.9.9", 600)

	records := c.ForZone("example.com")
	if len(records) != 1 {
		t.Fatalf("expected 1 record (overwritten), got %d", len(records))
	}
	if records[0].Content != "9.9.9.9" {
		t.Errorf("expected content '9.9.9.9', got %q", records[0].Content)
	}
	if records[0].TTL != 600 {
		t.Errorf("expected TTL 600, got %d", records[0].TTL)
	}
}

func TestCache_Remove(t *testing.T) {
	c := New()
	c.Put("example.com", "app.example.com", "A", "1.2.3.4", 300)
	c.Put("example.com", "api.example.com", "A", "5.6.7.8", 300)

	c.Remove("example.com", "app.example.com", "A")

	records := c.ForZone("example.com")
	if len(records) != 1 {
		t.Fatalf("expected 1 record after remove, got %d", len(records))
	}
	if records[0].FQDN != "api.example.com" {
		t.Errorf("expected api.example.com, got %q", records[0].FQDN)
	}
}

func TestCache_RemoveNonexistent(t *testing.T) {
	c := New()
	// Should not panic.
	c.Remove("example.com", "nope.example.com", "A")
	if c.Len() != 0 {
		t.Errorf("expected 0 records, got %d", c.Len())
	}
}

func TestCache_All(t *testing.T) {
	c := New()
	c.Put("a.com", "x.a.com", "A", "1.1.1.1", 300)
	c.Put("b.com", "y.b.com", "CNAME", "z.b.com", 600)

	all := c.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 records, got %d", len(all))
	}
}

func TestCache_Len(t *testing.T) {
	c := New()
	if c.Len() != 0 {
		t.Errorf("expected 0, got %d", c.Len())
	}
	c.Put("a.com", "x.a.com", "A", "1.1.1.1", 300)
	if c.Len() != 1 {
		t.Errorf("expected 1, got %d", c.Len())
	}
	c.Put("a.com", "x.a.com", "A", "2.2.2.2", 300) // overwrite, same key
	if c.Len() != 1 {
		t.Errorf("expected 1 after overwrite, got %d", c.Len())
	}
}

func TestCache_DifferentTypeSameNameAreDistinct(t *testing.T) {
	c := New()
	c.Put("example.com", "app.example.com", "A", "1.2.3.4", 300)
	c.Put("example.com", "app.example.com", "AAAA", "::1", 300)

	records := c.ForZone("example.com")
	if len(records) != 2 {
		t.Fatalf("expected 2 records (A and AAAA), got %d", len(records))
	}
}
