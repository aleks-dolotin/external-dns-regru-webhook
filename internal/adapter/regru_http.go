package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/auth"
)

// ErrPermissionDenied is returned when the API responds with a permission/authorization error.
// This typically means the credentials lack required zone-management permissions.
var ErrPermissionDenied = errors.New("adapter: permission denied — verify credentials have zone-management permissions (see docs/runbooks/reg-adapter-runbook.md)")

// ErrAuthenticationFailed is returned when the API rejects the provided credentials.
var ErrAuthenticationFailed = errors.New("adapter: authentication failed — check credentials")

// HTTPAdapter is a simple implementation that posts to Reg.ru-like endpoints
type HTTPAdapter struct {
	client     *http.Client
	baseURL    string
	authDriver auth.AuthDriver
	cache      *ExternalIDCache // optional; nil = no caching
}

// NewHTTPAdapter creates an HTTPAdapter with the given AuthDriver for request authentication.
// If driver is nil, requests are sent without authentication credentials.
func NewHTTPAdapter(driver auth.AuthDriver) *HTTPAdapter {
	base := os.Getenv("REGRU_BASE_URL")
	if base == "" {
		base = "http://localhost:8081/api/regru2"
	}
	return &HTTPAdapter{
		client:     &http.Client{Timeout: 10 * time.Second},
		baseURL:    base,
		authDriver: driver,
	}
}

// SetCache attaches an ExternalIDCache to the adapter. When set, the adapter
// populates the cache on successful FindRecord/CreateRecord/UpdateRecord and
// evicts entries on DeleteRecord.
func (h *HTTPAdapter) SetCache(c *ExternalIDCache) {
	h.cache = c
}

// cacheSet stores an external_id in the cache (no-op if cache is nil or id is empty).
func (h *HTTPAdapter) cacheSet(zone, name, recType, id string) {
	if h.cache != nil && id != "" {
		_ = h.cache.Set(CacheKey{Zone: zone, Name: name, RecType: recType}, id)
	}
}

// cacheEvict removes an entry from the cache (no-op if cache is nil).
func (h *HTTPAdapter) cacheEvict(zone, name, recType string) {
	if h.cache != nil {
		_ = h.cache.Delete(CacheKey{Zone: zone, Name: name, RecType: recType})
	}
}

// LookupExternalID returns the external_id for the given record.
// It checks the cache first (AC-1); on cache miss it falls back to a
// Reg.ru API call via FindRecord and caches the result (AC-2).
func (h *HTTPAdapter) LookupExternalID(zone, name, recType string) (string, error) {
	if h.cache != nil {
		if id := h.cache.Get(CacheKey{Zone: zone, Name: name, RecType: recType}); id != "" {
			return id, nil
		}
	}
	// Cache miss — fall back to API.
	rec, err := h.FindRecord(zone, name, recType)
	if err != nil {
		return "", err
	}
	if rec == nil {
		return "", nil
	}
	return rec.ID, nil // FindRecord already populates cache
}

// ReconcileCache refreshes all cache entries by querying the Reg.ru API (AC-3).
// Stale entries (record no longer exists) are evicted.
//
// Instead of calling FindRecord per cache entry (N+1), entries are grouped by
// zone and each zone is fetched once via zone/get_resource_records. All cached
// keys for that zone are then reconciled against the returned record set in memory.
func (h *HTTPAdapter) ReconcileCache(ctx context.Context) {
	if h.cache == nil {
		return
	}

	byZone := h.cache.KeysByZone()
	for zone, keys := range byZone {
		if ctx.Err() != nil {
			return
		}

		domain, err := h.fetchZoneRecords(zone)
		if err != nil {
			log.Printf("cache reconcile zone %s: %v", zone, err)
			continue
		}

		// Build lookup set from zone records: "subname\trectype" → ResourceRecord
		type rrKey struct{ name, rectype string }
		liveRecords := make(map[rrKey]bool)
		if domain != nil {
			for _, rr := range domain.Rrs {
				liveRecords[rrKey{
					name:    strings.ToLower(rr.Subname),
					rectype: strings.ToLower(rr.Rectype),
				}] = true
			}
		}

		// Reconcile each cached key against the live zone data.
		for _, key := range keys {
			rk := rrKey{
				name:    strings.ToLower(key.Name),
				rectype: strings.ToLower(key.RecType),
			}
			if !liveRecords[rk] {
				// Record no longer exists in zone — evict from cache.
				h.cacheEvict(key.Zone, key.Name, key.RecType)
			}
			// If record exists, the cache entry stays; service_id refresh
			// happens naturally on the next FindRecord hit.
		}
	}
}

// fetchZoneRecords retrieves all resource records for a zone in a single
// API call to zone/get_resource_records. Returns the DomainResult containing
// all rrs, or nil if the zone is empty / not found.
func (h *HTTPAdapter) fetchZoneRecords(zone string) (*DomainResult, error) {
	endpoint := fmt.Sprintf("%s/zone/get_resource_records", h.baseURL)
	inputData := map[string]interface{}{
		"domains": []map[string]interface{}{{"dname": zone}},
	}
	reguResp, err := h.doRequest(endpoint, inputData)
	if err != nil {
		return nil, err
	}
	return checkDomainResult(reguResp)
}

// StartReconciliation launches a background goroutine that periodically
// refreshes all cache entries. It stops when ctx is cancelled.
func (h *HTTPAdapter) StartReconciliation(ctx context.Context, interval time.Duration) {
	if h.cache == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("cache reconciliation stopped")
				return
			case <-ticker.C:
				h.ReconcileCache(ctx)
			}
		}
	}()
}

// authParams returns authentication key-value pairs from the AuthDriver.
// Returns an empty map if no driver is configured.
func (h *HTTPAdapter) authParams() map[string]string {
	if h.authDriver == nil {
		return map[string]string{}
	}
	params, err := h.authDriver.PrepareAuth()
	if err != nil {
		log.Printf("WARNING: auth driver PrepareAuth failed: %v (request will be sent without credentials)", err)
		return map[string]string{}
	}
	return params
}

// classifyHTTPError maps HTTP status codes to domain-specific errors.
func classifyHTTPError(resp *http.Response) error {
	switch {
	case resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w (HTTP %d)", ErrPermissionDenied, resp.StatusCode)
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("%w (HTTP %d)", ErrAuthenticationFailed, resp.StatusCode)
	case resp.StatusCode >= 400:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("reg.ru returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// classifyAPIError inspects the Reg.ru JSON response body for permission errors.
// Reg.ru API returns 200 with error details in the response body for some failures.
func classifyAPIError(body []byte) error {
	var result struct {
		Result    string `json:"result"`
		ErrorCode string `json:"error_code"`
		ErrorText string `json:"error_text"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil // cannot parse — not an API error
	}
	if result.Result == "error" {
		switch result.ErrorCode {
		case "ACCESS_DENIED_TO_OBJECT", "INVALID_CREDENTIALS", "NO_SUCH_USER":
			return fmt.Errorf("%w: %s — %s", ErrPermissionDenied, result.ErrorCode, result.ErrorText)
		case "INVALID_AUTH":
			return fmt.Errorf("%w: %s — %s", ErrAuthenticationFailed, result.ErrorCode, result.ErrorText)
		default:
			return fmt.Errorf("reg.ru API error: %s — %s", result.ErrorCode, result.ErrorText)
		}
	}
	return nil
}

// doRequest sends a form-encoded POST to the given endpoint and returns the
// parsed ReguResponse. It handles HTTP status classification, API-level error
// detection, response unmarshalling, and top-level result validation.
func (h *HTTPAdapter) doRequest(endpoint string, inputData map[string]interface{}) (*ReguResponse, error) {
	body, err := buildFormBody(inputData, h.authParams())
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := h.client.Post(endpoint, "application/x-www-form-urlencoded", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if httpErr := classifyHTTPError(resp); httpErr != nil {
		return nil, httpErr
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if apiErr := classifyAPIError(respBody); apiErr != nil {
		return nil, apiErr
	}

	var reguResp ReguResponse
	if err := json.Unmarshal(respBody, &reguResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if reguResp.Result != "success" {
		return nil, fmt.Errorf("reg.ru API error: result=%s", reguResp.Result)
	}
	return &reguResp, nil
}

// checkDomainResult validates the first domain result in the response and
// returns the DomainResult. Returns an error if the domain-level result is not "success".
func checkDomainResult(reguResp *ReguResponse) (*DomainResult, error) {
	if reguResp.Answer == nil || len(reguResp.Answer.Domains) == 0 {
		return nil, nil
	}
	domain := &reguResp.Answer.Domains[0]
	if domain.Result != "success" {
		if domain.ErrorCode != "" {
			return nil, fmt.Errorf("reg.ru domain error: %s — %s", domain.ErrorCode, domain.ErrorText)
		}
		return nil, fmt.Errorf("reg.ru domain result: %s", domain.Result)
	}
	return domain, nil
}

func (h *HTTPAdapter) FindRecord(zone, name, typ string) (*Record, error) {
	endpoint := fmt.Sprintf("%s/zone/get_resource_records", h.baseURL)

	inputData := map[string]interface{}{
		"domains": []map[string]interface{}{{"dname": zone}},
	}

	reguResp, err := h.doRequest(endpoint, inputData)
	if err != nil {
		return nil, err
	}

	domain, err := checkDomainResult(reguResp)
	if err != nil {
		return nil, err
	}
	if domain == nil {
		return nil, nil
	}

	for _, rr := range domain.Rrs {
		if strings.EqualFold(rr.Subname, name) && strings.EqualFold(rr.Rectype, typ) {
			rec := &Record{
				ID:      domain.ServiceID.String(),
				Name:    rr.Subname,
				Type:    rr.Rectype,
				Content: rr.Content,
			}
			h.cacheSet(zone, name, typ, rec.ID)
			return rec, nil
		}
	}

	return nil, nil
}

// ErrUnsupportedRecordType is returned when the DNS record type is not supported.
var ErrUnsupportedRecordType = errors.New("adapter: unsupported record type")

// recordTypeAction maps DNS record types to Reg.ru API action names.
func recordTypeAction(recType string) (string, error) {
	switch strings.ToUpper(recType) {
	case "A":
		return "add_alias", nil
	case "AAAA":
		return "add_aaaa", nil
	case "CNAME":
		return "add_cname", nil
	case "TXT":
		return "add_txt", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedRecordType, recType)
	}
}

// buildActionPayload constructs the action_list entry for a given record type.
// Different record types require different field names per Reg.ru API.
func buildActionPayload(action, subdomain, content string, recType string) map[string]interface{} {
	entry := map[string]interface{}{
		"action":    action,
		"subdomain": subdomain,
	}
	switch strings.ToUpper(recType) {
	case "A", "AAAA":
		entry["ipaddr"] = content
	case "CNAME":
		entry["canonical_name"] = content
	case "TXT":
		entry["text"] = content
	}
	return entry
}

func (h *HTTPAdapter) CreateRecord(zone string, r *Record) error {
	// FR14 idempotency: check if record already exists.
	existing, err := h.FindRecord(zone, r.Name, r.Type)
	if err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if existing != nil {
		// Record already exists — skip create, persist external_id.
		r.ID = existing.ID
		return nil
	}

	action, err := recordTypeAction(r.Type)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/zone/update_records", h.baseURL)
	actionEntry := buildActionPayload(action, r.Name, r.Content, r.Type)

	inputData := map[string]interface{}{
		"domains": []map[string]interface{}{{
			"dname":       zone,
			"action_list": []map[string]interface{}{actionEntry},
		}},
	}

	reguResp, err := h.doRequest(endpoint, inputData)
	if err != nil {
		return err
	}

	domain, err := checkDomainResult(reguResp)
	if err != nil {
		return err
	}
	// FR15: persist service_id as external_id.
	if domain != nil && domain.ServiceID.String() != "" {
		r.ID = domain.ServiceID.String()
	}

	// Populate cache after successful create.
	h.cacheSet(zone, r.Name, r.Type, r.ID)

	return nil
}

func (h *HTTPAdapter) UpdateRecord(zone string, r *Record) error {
	// Find the existing record first to get current content for removal.
	existing, err := h.FindRecord(zone, r.Name, r.Type)
	if err != nil {
		return fmt.Errorf("update lookup: %w", err)
	}

	if existing == nil {
		// No existing record — fall back to create (idempotent).
		return h.CreateRecord(zone, r)
	}

	// If content is the same, nothing to update.
	if existing.Content == r.Content {
		r.ID = existing.ID
		return nil
	}

	// Atomic update: remove old + add new in single update_records call.
	action, err := recordTypeAction(r.Type)
	if err != nil {
		return err
	}

	removeEntry := map[string]interface{}{
		"action":      "remove_record",
		"subdomain":   r.Name,
		"record_type": r.Type,
		"content":     existing.Content,
	}
	addEntry := buildActionPayload(action, r.Name, r.Content, r.Type)

	endpoint := fmt.Sprintf("%s/zone/update_records", h.baseURL)
	inputData := map[string]interface{}{
		"domains": []map[string]interface{}{{
			"dname":       zone,
			"action_list": []map[string]interface{}{removeEntry, addEntry},
		}},
	}

	reguResp, err := h.doRequest(endpoint, inputData)
	if err != nil {
		return err
	}

	domain, err := checkDomainResult(reguResp)
	if err != nil {
		return err
	}
	if domain != nil && domain.ServiceID.String() != "" {
		r.ID = domain.ServiceID.String()
	}

	// Update cache after successful update.
	h.cacheSet(zone, r.Name, r.Type, r.ID)

	return nil
}

func (h *HTTPAdapter) DeleteRecord(zone string, id string) error {
	// id is expected to be in format "subdomain:rectype:content" for precise deletion.
	// Parse the compound id.
	parts := strings.SplitN(id, ":", 3)
	if len(parts) < 2 {
		return fmt.Errorf("adapter: invalid delete id %q — expected subdomain:rectype[:content]", id)
	}
	subdomain := parts[0]
	recType := parts[1]
	content := ""
	if len(parts) == 3 {
		content = parts[2]
	}

	endpoint := fmt.Sprintf("%s/zone/remove_record", h.baseURL)
	inputData := map[string]interface{}{
		"domains":     []map[string]interface{}{{"dname": zone}},
		"subdomain":   subdomain,
		"record_type": recType,
	}
	if content != "" {
		inputData["content"] = content
	}

	reguResp, err := h.doRequest(endpoint, inputData)
	if err != nil {
		return err
	}

	if _, err := checkDomainResult(reguResp); err != nil {
		return err
	}

	// Evict from cache after successful delete.
	h.cacheEvict(zone, subdomain, recType)

	return nil
}

func (h *HTTPAdapter) BulkUpdate(zone string, actions []BulkAction) error {
	if len(actions) == 0 {
		return nil
	}

	actionList := make([]map[string]interface{}, 0, len(actions))
	for _, ba := range actions {
		entry := map[string]interface{}{
			"action":    ba.Action,
			"subdomain": ba.Subdomain,
		}
		if ba.Action == "remove_record" {
			entry["record_type"] = ba.RecType
			if ba.Content != "" {
				entry["content"] = ba.Content
			}
		} else {
			// Determine content field by record type.
			switch strings.ToUpper(ba.RecType) {
			case "A", "AAAA":
				entry["ipaddr"] = ba.Content
			case "CNAME":
				entry["canonical_name"] = ba.Content
			case "TXT":
				entry["text"] = ba.Content
			default:
				entry["content"] = ba.Content
			}
		}
		actionList = append(actionList, entry)
	}

	endpoint := fmt.Sprintf("%s/zone/update_records", h.baseURL)
	inputData := map[string]interface{}{
		"domains": []map[string]interface{}{{
			"dname":       zone,
			"action_list": actionList,
		}},
	}

	reguResp, err := h.doRequest(endpoint, inputData)
	if err != nil {
		return err
	}

	if _, err := checkDomainResult(reguResp); err != nil {
		return err
	}

	return nil
}
