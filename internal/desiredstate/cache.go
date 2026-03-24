// Package desiredstate maintains an in-memory cache of desired DNS records
// derived from incoming events on /adapter/v1/events. This cache is the
// source of truth for force-resync reconciliation.
//
// Story 8.1: desired state provider for force-resync.
package desiredstate

import (
	"sync"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/reconciler"
)

// recordKey uniquely identifies a DNS record by zone + FQDN + type.
type recordKey struct {
	Zone       string
	FQDN       string
	RecordType string
}

// Cache stores the latest desired DNS records observed from incoming events.
// Thread-safe for concurrent reads and writes.
type Cache struct {
	mu      sync.RWMutex
	records map[recordKey]reconciler.DesiredRecord
}

// New creates an empty desired state cache.
func New() *Cache {
	return &Cache{
		records: make(map[recordKey]reconciler.DesiredRecord),
	}
}

// Put adds or updates a desired record. Called on create/update events.
func (c *Cache) Put(zone, fqdn, recordType, content string, ttl int) {
	key := recordKey{Zone: zone, FQDN: fqdn, RecordType: recordType}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records[key] = reconciler.DesiredRecord{
		Zone:       zone,
		FQDN:       fqdn,
		RecordType: recordType,
		Content:    content,
		TTL:        ttl,
	}
}

// Remove deletes a desired record. Called on delete events.
func (c *Cache) Remove(zone, fqdn, recordType string) {
	key := recordKey{Zone: zone, FQDN: fqdn, RecordType: recordType}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.records, key)
}

// ForZone returns all desired records for a specific zone.
func (c *Cache) ForZone(zone string) []reconciler.DesiredRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var result []reconciler.DesiredRecord
	for _, r := range c.records {
		if r.Zone == zone {
			result = append(result, r)
		}
	}
	return result
}

// All returns all desired records in the cache.
func (c *Cache) All() []reconciler.DesiredRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]reconciler.DesiredRecord, 0, len(c.records))
	for _, r := range c.records {
		result = append(result, r)
	}
	return result
}

// Len returns the number of records in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.records)
}
