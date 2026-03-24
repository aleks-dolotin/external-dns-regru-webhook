// Package config provides ConfigMap-based zone-namespace mapping configuration
// with hot-reload support and runtime validation.
//
// Story 3.1: Express zone-namespace mappings via ConfigMap
// Story 3.2: Validate config at runtime with clear errors
// Story 3.3: Template-driven FQDN generation
package config

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"go.yaml.in/yaml/v2"
)

// DefaultReloadInterval is the default hot-reload polling interval.
const DefaultReloadInterval = 30 * time.Second

// ZoneMapping represents a single zone configuration entry.
type ZoneMapping struct {
	Zone         string   `yaml:"zone"`
	Namespaces   []string `yaml:"namespaces"`
	Template     string   `yaml:"template"`
	TTL          int      `yaml:"ttl,omitempty"`
	Priority     int      `yaml:"priority,omitempty"`
	QuotaPerHour int      `yaml:"quota_per_hour,omitempty"` // Story 9.2: per-namespace quota (0 = no quota)
}

// MappingsConfig is the top-level structure for mappings.yaml.
type MappingsConfig struct {
	Zones []ZoneMapping `yaml:"zones"`
}

// Store holds the current validated configuration and supports
// thread-safe reads with hot-reload.
type Store struct {
	mu       sync.RWMutex
	config   *MappingsConfig
	filePath string
	modTime  time.Time

	// OnReload is called after a successful config reload.
	OnReload func(*MappingsConfig)

	// OnReloadError is called when a reload fails validation.
	OnReloadError func(error)
}

// NewStore creates a new configuration store and loads the initial
// configuration from the given file path. Returns an error if the
// initial load fails.
func NewStore(filePath string) (*Store, error) {
	s := &Store{
		filePath: filePath,
	}

	cfg, modTime, err := s.loadFromFile()
	if err != nil {
		return nil, fmt.Errorf("config: initial load failed: %w", err)
	}

	s.config = cfg
	s.modTime = modTime
	return s, nil
}

// Get returns the current validated configuration. Thread-safe.
func (s *Store) Get() *MappingsConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// FindZone looks up a zone mapping by zone name. Returns nil if not found.
func (s *Store) FindZone(zone string) *ZoneMapping {
	cfg := s.Get()
	if cfg == nil {
		return nil
	}
	for i := range cfg.Zones {
		if cfg.Zones[i].Zone == zone {
			return &cfg.Zones[i]
		}
	}
	return nil
}

// IsNamespaceAllowed checks whether a namespace is allowed for a given zone.
// Returns true if the zone has no namespace restrictions (empty list) or if
// the namespace is in the allowed list.
// Story 9.1: checks across all entries for the same zone (multi-entry support).
func (s *Store) IsNamespaceAllowed(zone, namespace string) bool {
	cfg := s.Get()
	if cfg == nil {
		return false
	}
	for _, zm := range cfg.Zones {
		if zm.Zone != zone {
			continue
		}
		// Empty namespaces list means all namespaces are allowed.
		if len(zm.Namespaces) == 0 {
			return true
		}
		for _, ns := range zm.Namespaces {
			if ns == namespace {
				return true
			}
		}
	}
	return false
}

// FindZoneForNamespace returns the specific ZoneMapping for a (zone, namespace) pair.
// If a zone entry has an empty namespaces list (wildcard), it matches any namespace.
// Returns nil if no matching entry is found.
// Story 9.1: enables per-namespace template differentiation for the same zone.
func (s *Store) FindZoneForNamespace(zone, namespace string) *ZoneMapping {
	cfg := s.Get()
	if cfg == nil {
		return nil
	}
	for i := range cfg.Zones {
		if cfg.Zones[i].Zone != zone {
			continue
		}
		// Empty namespaces list = wildcard: matches any namespace.
		if len(cfg.Zones[i].Namespaces) == 0 {
			return &cfg.Zones[i]
		}
		for _, ns := range cfg.Zones[i].Namespaces {
			if ns == namespace {
				return &cfg.Zones[i]
			}
		}
	}
	return nil
}

// ZonesForNamespace returns all zone names where the given namespace is allowed.
// A zone with an empty Namespaces list (wildcard) matches every namespace.
// Story 8.1: used by force-resync to resolve namespace → zones.
// Story 9.1: deduplicates results when same zone has multiple entries.
func (s *Store) ZonesForNamespace(namespace string) []string {
	cfg := s.Get()
	if cfg == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var zones []string
	for _, zm := range cfg.Zones {
		if _, dup := seen[zm.Zone]; dup {
			continue
		}
		if len(zm.Namespaces) == 0 {
			// Wildcard — all namespaces allowed.
			seen[zm.Zone] = struct{}{}
			zones = append(zones, zm.Zone)
			continue
		}
		for _, ns := range zm.Namespaces {
			if ns == namespace {
				seen[zm.Zone] = struct{}{}
				zones = append(zones, zm.Zone)
				break
			}
		}
	}
	return zones
}

// Reload attempts to reload the configuration from the file.
// If the file hasn't changed (same mod time), this is a no-op.
// If validation fails, the previous valid config is retained.
func (s *Store) Reload() error {
	cfg, modTime, err := s.loadFromFile()
	if err != nil {
		if s.OnReloadError != nil {
			s.OnReloadError(err)
		}
		return err
	}

	s.mu.Lock()
	// Check if file actually changed.
	if modTime.Equal(s.modTime) {
		s.mu.Unlock()
		return nil
	}

	s.config = cfg
	s.modTime = modTime
	s.mu.Unlock()

	log.Printf("config: reloaded mappings from %s (mod_time=%s, zones=%d)",
		s.filePath, modTime.Format(time.RFC3339), len(cfg.Zones))

	if s.OnReload != nil {
		s.OnReload(cfg)
	}
	return nil
}

// RunReloader starts a periodic reload loop that checks for config changes
// at the given interval. It blocks until the done channel is closed.
func (s *Store) RunReloader(done <-chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := s.Reload(); err != nil {
				log.Printf("config: reload error: %v", err)
			}
		}
	}
}

// loadFromFile reads, parses, and validates the config file.
func (s *Store) loadFromFile() (*MappingsConfig, time.Time, error) {
	info, err := os.Stat(s.filePath)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("config: cannot stat %s: %w", s.filePath, err)
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("config: cannot read %s: %w", s.filePath, err)
	}

	var cfg MappingsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, time.Time{}, fmt.Errorf("config: YAML parse error in %s: %w", s.filePath, err)
	}

	if err := Validate(&cfg); err != nil {
		return nil, time.Time{}, fmt.Errorf("config: validation error: %w", err)
	}

	return &cfg, info.ModTime(), nil
}
