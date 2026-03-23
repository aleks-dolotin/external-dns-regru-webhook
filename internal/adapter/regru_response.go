package adapter

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ReguResponse is the top-level response envelope from the Reg.ru API v2.
type ReguResponse struct {
	Result string      `json:"result"`
	Answer *ReguAnswer `json:"answer,omitempty"`
	// Top-level error fields (present when result == "error").
	ErrorCode string `json:"error_code,omitempty"`
	ErrorText string `json:"error_text,omitempty"`
}

// ReguAnswer contains the answer section of the API response.
type ReguAnswer struct {
	Domains []DomainResult `json:"domains"`
}

// DomainResult represents a per-domain result inside the answer.domains array.
// NOTE: ServiceID is returned at the domain level by Reg.ru API, not per-record.
// All records within the same domain share the same ServiceID value. This is an
// API limitation — do not rely on ServiceID for unique per-record identification.
type DomainResult struct {
	Dname     string           `json:"dname"`
	Result    string           `json:"result"`
	ServiceID json.Number      `json:"service_id,omitempty"`
	Rrs       []ResourceRecord `json:"rrs,omitempty"`
	// Fields for update_records responses.
	ActionList []ActionResult `json:"action_list,omitempty"`
	// Error fields for per-domain errors.
	ErrorCode string `json:"error_code,omitempty"`
	ErrorText string `json:"error_text,omitempty"`
}

// ResourceRecord represents a single DNS resource record returned by
// zone/get_resource_records.
type ResourceRecord struct {
	Subname  string `json:"subname"`
	Rectype  string `json:"rectype"`
	Content  string `json:"content"`
	Priority string `json:"prio"`
	State    string `json:"state,omitempty"`
}

// ActionResult represents the result of a single action in zone/update_records.
type ActionResult struct {
	Action string `json:"action"`
	Result string `json:"result"`
}

// buildFormBody constructs the application/x-www-form-urlencoded request body
// required by the Reg.ru API v2. Auth parameters (username/password) are placed
// inside the input_data JSON object, not as top-level form fields.
// The function does NOT mutate the caller's inputData map.
func buildFormBody(inputData map[string]interface{}, authParams map[string]string) (string, error) {
	// Shallow-copy inputData so we don't mutate the caller's map.
	merged := make(map[string]interface{}, len(inputData)+len(authParams))
	for k, v := range inputData {
		merged[k] = v
	}
	for k, v := range authParams {
		merged[k] = v
	}

	jsonBytes, err := json.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("marshal input_data: %w", err)
	}

	form := url.Values{}
	form.Set("input_format", "json")
	form.Set("input_data", string(jsonBytes))
	return form.Encode(), nil
}
