package adapter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/yourorg/externaldns-regru-sidecar/internal/auth"
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

func (h *HTTPAdapter) FindRecord(zone, _, _ string) (*Record, error) {
	endpoint := fmt.Sprintf("%s/zone/get_resource_records", h.baseURL)
	payload := map[string]interface{}{
		"input_format": "json",
		"input_data":   map[string]interface{}{"domains": []map[string]interface{}{{"dname": zone}}},
	}
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
	defer func() { _ = resp.Body.Close() }()

	if httpErr := classifyHTTPError(resp); httpErr != nil {
		return nil, httpErr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if apiErr := classifyAPIError(body); apiErr != nil {
		return nil, apiErr
	}
	return nil, nil
}

func (h *HTTPAdapter) CreateRecord(zone string, r *Record) error {
	endpoint := fmt.Sprintf("%s/zone/update_records", h.baseURL)
	domains := []map[string]interface{}{{
		"dname":       zone,
		"action_list": []map[string]interface{}{{"action": fmt.Sprintf("add_%s", r.Type), "subdomain": r.Name, "ipaddr": r.Content}},
	}}
	payload := map[string]interface{}{"input_format": "json", "input_data": map[string]interface{}{"domains": domains}}
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
	defer func() { _ = resp.Body.Close() }()

	if httpErr := classifyHTTPError(resp); httpErr != nil {
		return httpErr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	return classifyAPIError(body)
}

func (h *HTTPAdapter) UpdateRecord(zone string, r *Record) error {
	return h.CreateRecord(zone, r)
}

func (h *HTTPAdapter) DeleteRecord(_ string, _ string) error {
	return nil
}

func (h *HTTPAdapter) BulkUpdate(_ string, _ []interface{}) error {
	return nil
}
