package adapter

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"
)

// HTTPAdapter is a simple implementation that posts to Reg.ru-like endpoints
type HTTPAdapter struct {
    client  *http.Client
    baseURL string
}

func NewHTTPAdapter() *HTTPAdapter {
    base := os.Getenv("REGRU_BASE_URL")
    if base == "" {
        base = "http://localhost:8081/api/regru2"
    }
    return &HTTPAdapter{client: &http.Client{Timeout: 10 * time.Second}, baseURL: base}
}

func (h *HTTPAdapter) FindRecord(zone, name, typ string) (*Record, error) {
    // For simplicity, call get_resource_records
    endpoint := fmt.Sprintf("%s/zone/get_resource_records", h.baseURL)
    payload := map[string]interface{}{
        "input_format": "json",
        "input_data": map[string]interface{}{"domains": []map[string]interface{}{{"dname": zone}}},
    }
    b, _ := json.Marshal(payload)
    resp, err := h.client.Post(endpoint, "application/x-www-form-urlencoded", bytes.NewBuffer(b))
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
    domains := []map[string]interface{}{ {
        "dname": zone,
        "action_list": []map[string]interface{}{ {"action": "add_%s", "subdomain": r.Name, "ipaddr": r.Content} },
    } }
    payload := map[string]interface{}{"input_format": "json", "input_data": map[string]interface{}{"domains": domains}}
    b, _ := json.Marshal(payload)
    resp, err := h.client.Post(endpoint, "application/x-www-form-urlencoded", bytes.NewBuffer(b))
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

func (h *HTTPAdapter) DeleteRecord(zone string, id string) error {
    // not implemented
    return nil
}

func (h *HTTPAdapter) BulkUpdate(zone string, actions []interface{}) error {
    // not implemented
    return nil
}

