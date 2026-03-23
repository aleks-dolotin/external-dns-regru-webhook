package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/yourorg/externaldns-regru-sidecar/internal/auth"
)

// HTTPAdapter is a simple implementation that posts to Reg.ru-like endpoints
type HTTPAdapter struct {
	client     *http.Client
	baseURL    string
	authDriver auth.AuthDriver
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

// authParams returns authentication key-value pairs from the AuthDriver.
// Returns an empty map if no driver is configured.
func (h *HTTPAdapter) authParams() map[string]string {
	if h.authDriver == nil {
		return map[string]string{}
	}
	params, err := h.authDriver.PrepareAuth()
	if err != nil {
		return map[string]string{}
	}
	return params
}

func (h *HTTPAdapter) FindRecord(zone, _, _ string) (*Record, error) {
	// For simplicity, call get_resource_records
	endpoint := fmt.Sprintf("%s/zone/get_resource_records", h.baseURL)
	payload := map[string]interface{}{
		"input_format": "json",
		"input_data":   map[string]interface{}{"domains": []map[string]interface{}{{"dname": zone}}},
	}
	// Inject auth params into payload
	for k, v := range h.authParams() {
		payload[k] = v
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	resp, err := h.client.Post(endpoint, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// naive: return nil (not found)
	return nil, nil
}

func (h *HTTPAdapter) CreateRecord(zone string, r *Record) error {
	endpoint := fmt.Sprintf("%s/zone/update_records", h.baseURL)
	// Create a basic action list payload
	domains := []map[string]interface{}{{
		"dname":       zone,
		"action_list": []map[string]interface{}{{"action": fmt.Sprintf("add_%s", r.Type), "subdomain": r.Name, "ipaddr": r.Content}},
	}}
	payload := map[string]interface{}{"input_format": "json", "input_data": map[string]interface{}{"domains": domains}}
	// Inject auth params into payload
	for k, v := range h.authParams() {
		payload[k] = v
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	resp, err := h.client.Post(endpoint, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("reg.ru returned status %d", resp.StatusCode)
	}
	return nil
}

func (h *HTTPAdapter) UpdateRecord(zone string, r *Record) error {
	// placeholder: reuse CreateRecord for now
	return h.CreateRecord(zone, r)
}

func (h *HTTPAdapter) DeleteRecord(_ string, _ string) error {
	// not implemented
	return nil
}

func (h *HTTPAdapter) BulkUpdate(_ string, _ []interface{}) error {
	// not implemented
	return nil
}
