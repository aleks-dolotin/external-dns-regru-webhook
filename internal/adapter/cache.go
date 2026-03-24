package adapter

import (
	"fmt"
	"strings"
	"sync"
)

// CacheKey uniquely identifies a DNS record in the cache.
type CacheKey struct {
	Zone    string `json:"zone"`
	Name    string `json:"name"`
	RecType string `json:"rec_type"`
}

func (k CacheKey) String() string {
	return fmt.Sprintf("%s/%s/%s", k.Zone, k.Name, k.RecType)
}

// ExternalIDCache provides a thread-safe in-memory cache for mapping
// DNS record identifiers to Reg.ru external IDs (service_id).
// Cache is ephemeral — it does not survive pod restarts. Consistency
// is maintained via reconciliation against the Reg.ru API.
type ExternalIDCache struct {
	mu      sync.RWMutex
	entries map[string]string // CacheKey.String() → external_id
}

// NewExternalIDCache creates a new in-memory cache.
func NewExternalIDCache() *ExternalIDCache {
	return &ExternalIDCache{
		entries: make(map[string]string),
	}
}

// Get returns the external_id for the given key, or "" if not cached.
func (c *ExternalIDCache) Get(key CacheKey) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.entries[key.String()]
}

// Set stores the external_id for the given key.
func (c *ExternalIDCache) Set(key CacheKey, externalID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key.String()] = externalID
}

// Delete removes the entry for the given key.
func (c *ExternalIDCache) Delete(key CacheKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key.String())
}

// Len returns the number of entries in the cache.
func (c *ExternalIDCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Keys returns all cache keys. Used for background reconciliation.
func (c *ExternalIDCache) Keys() []CacheKey {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]CacheKey, 0, len(c.entries))
	for k := range c.entries {
		parts := strings.SplitN(k, "/", 3)
		if len(parts) == 3 {
			keys = append(keys, CacheKey{Zone: parts[0], Name: parts[1], RecType: parts[2]})
		}
	}
	return keys
}

// KeysByZone returns cache keys grouped by zone. This allows callers
// to batch API calls per zone instead of per record, avoiding N+1 queries.
func (c *ExternalIDCache) KeysByZone() map[string][]CacheKey {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string][]CacheKey)
	for k := range c.entries {
		parts := strings.SplitN(k, "/", 3)
		if len(parts) == 3 {
			key := CacheKey{Zone: parts[0], Name: parts[1], RecType: parts[2]}
			result[key.Zone] = append(result[key.Zone], key)
		}
	}
	return result
}
