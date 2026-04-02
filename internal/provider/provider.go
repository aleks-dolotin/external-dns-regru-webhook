// Package provider implements the external-dns webhook Provider interface
// for Reg.ru DNS API v2.
package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

// RegrProvider implements provider.Provider for Reg.ru DNS.
type RegrProvider struct {
	provider.BaseProvider
	adapter      adapter.Adapter
	domainFilter *endpoint.DomainFilter
	zones        []string
}

// NewRegrProvider creates a new Reg.ru provider.
func NewRegrProvider(a adapter.Adapter, domainFilter endpoint.DomainFilter) *RegrProvider {
	return &RegrProvider{
		adapter:      a,
		domainFilter: &domainFilter,
		zones:        domainFilter.Filters,
	}
}

// Records returns the current DNS records from Reg.ru for all configured zones.
func (p *RegrProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	var endpoints []*endpoint.Endpoint

	for _, zone := range p.zones {
		records, err := p.listRecords(zone)
		if err != nil {
			return nil, fmt.Errorf("listing records for zone %s: %w", zone, err)
		}
		endpoints = append(endpoints, records...)
	}

	return endpoints, nil
}

// ApplyChanges applies the given set of changes to the Reg.ru DNS.
func (p *RegrProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	for _, ep := range changes.Create {
		zone := p.findZone(ep.DNSName)
		if zone == "" {
			continue
		}
		subdomain := extractSubdomain(ep.DNSName, zone)
		for _, target := range ep.Targets {
			rec := &adapter.Record{
				Name:    subdomain,
				Type:    ep.RecordType,
				Content: target,
				TTL:     int(ep.RecordTTL),
			}
			if err := p.adapter.CreateRecord(zone, rec); err != nil {
				return fmt.Errorf("create %s.%s (%s): %w", subdomain, zone, ep.RecordType, err)
			}
		}
	}

	for _, ep := range changes.UpdateNew {
		zone := p.findZone(ep.DNSName)
		if zone == "" {
			continue
		}
		subdomain := extractSubdomain(ep.DNSName, zone)
		for _, target := range ep.Targets {
			rec := &adapter.Record{
				Name:    subdomain,
				Type:    ep.RecordType,
				Content: target,
				TTL:     int(ep.RecordTTL),
			}
			if err := p.adapter.UpdateRecord(zone, rec); err != nil {
				return fmt.Errorf("update %s.%s (%s): %w", subdomain, zone, ep.RecordType, err)
			}
		}
	}

	for _, ep := range changes.Delete {
		zone := p.findZone(ep.DNSName)
		if zone == "" {
			continue
		}
		subdomain := extractSubdomain(ep.DNSName, zone)
		for _, target := range ep.Targets {
			deleteID := fmt.Sprintf("%s:%s:%s", subdomain, ep.RecordType, target)
			if err := p.adapter.DeleteRecord(zone, deleteID); err != nil {
				return fmt.Errorf("delete %s.%s (%s): %w", subdomain, zone, ep.RecordType, err)
			}
		}
	}

	return nil
}

// GetDomainFilter returns the domain filter for this provider.
func (p *RegrProvider) GetDomainFilter() endpoint.DomainFilterInterface {
	return p.domainFilter
}

// listRecords fetches all records for a zone and converts them to endpoints.
func (p *RegrProvider) listRecords(zone string) ([]*endpoint.Endpoint, error) {
	// FindRecord with empty name and type returns all records for the zone.
	// We need to use the adapter's underlying method.
	// For now, use a well-known approach: list via get_resource_records.
	httpAdapter, ok := p.adapter.(*adapter.HTTPAdapter)
	if !ok {
		return nil, fmt.Errorf("adapter does not support ListRecords")
	}

	records, err := httpAdapter.ListRecords(zone)
	if err != nil {
		return nil, err
	}

	endpointMap := make(map[string]*endpoint.Endpoint)
	for _, r := range records {
		fqdn := r.Name + "." + zone
		if r.Name == "" || r.Name == "@" {
			fqdn = zone
		}
		key := fqdn + "/" + r.Type
		if ep, exists := endpointMap[key]; exists {
			ep.Targets = append(ep.Targets, r.Content)
		} else {
			endpointMap[key] = &endpoint.Endpoint{
				DNSName:    fqdn,
				RecordType: r.Type,
				Targets:    endpoint.Targets{r.Content},
				RecordTTL:  endpoint.TTL(r.TTL),
			}
		}
	}

	var endpoints []*endpoint.Endpoint
	for _, ep := range endpointMap {
		endpoints = append(endpoints, ep)
	}
	return endpoints, nil
}

// findZone returns the zone that matches the given FQDN.
func (p *RegrProvider) findZone(fqdn string) string {
	for _, zone := range p.zones {
		if strings.HasSuffix(fqdn, "."+zone) || fqdn == zone {
			return zone
		}
	}
	return ""
}

// extractSubdomain extracts the subdomain from an FQDN relative to the zone.
// "app.dolotin.ru" with zone "dolotin.ru" → "app"
// "dolotin.ru" with zone "dolotin.ru" → "@"
func extractSubdomain(fqdn, zone string) string {
	if fqdn == zone {
		return "@"
	}
	return strings.TrimSuffix(fqdn, "."+zone)
}
