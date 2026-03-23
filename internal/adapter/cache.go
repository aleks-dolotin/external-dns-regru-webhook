package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// ExternalIDCache provides a thread-safe, optionally persistent cache
// for mapping DNS record identifiers to Reg.ru external IDs (service_id).
type ExternalIDCache struct {
	mu       sync.RWMutex
	entries  map[string]string // CacheKey.String() → external_id
	filePath string            // empty = in-memory only
}

// NewExternalIDCache creates a new cache. If filePath is non-empty,
// the cache will be persisted to disk on writes and loaded on creation.
func NewExternalIDCache(filePath string) (*ExternalIDCache, error) {
	c := &ExternalIDCache{
		entries:  make(map[string]string),
		filePath: filePath,
	}
	if filePath != "" {
		if err := c.load(); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("cache load: %w", err)
		}
	}
	return c, nil
}

// Get returns the external_id for the given key, or "" if not cached.
func (c *ExternalIDCache) Get(key CacheKey) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.entries[key.String()]
}

// Set stores the external_id for the given key and persists if configured.
func (c *ExternalIDCache) Set(key CacheKey, externalID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key.String()] = externalID
	if c.filePath != "" {
		return c.saveLocked()
	}
	return nil
}

// Delete removes the entry for the given key.
func (c *ExternalIDCache) Delete(key CacheKey) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key.String())
	if c.filePath != "" {
		return c.saveLocked()
	}
	return nil
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

// load reads the cache from the JSON file. Caller must NOT hold the lock.
func (c *ExternalIDCache) load() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &c.entries)
}

// saveLocked writes the cache to the JSON file atomically using a temp file + rename
// pattern. This ensures the cache file is never partially written on crash.
// Caller MUST hold c.mu write lock.
func (c *ExternalIDCache) saveLocked() error {
	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}

	dir := filepath.Dir(c.filePath)
	tmp, err := os.CreateTemp(dir, ".cache-*.tmp")
	if err != nil {
		return fmt.Errorf("cache create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("cache write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("cache sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("cache close temp: %w", err)
	}

	if err := os.Rename(tmpName, c.filePath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("cache rename: %w", err)
	}
	return nil
}
