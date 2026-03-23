package adapter

// Record represents a DNS record in the adapter domain model
type Record struct {
    ID      string `json:"id,omitempty"`
    Name    string `json:"name"`
    Type    string `json:"type"`
    Content string `json:"content"`
    TTL     int    `json:"ttl"`
    Priority int   `json:"priority,omitempty"`
}

// Adapter defines the contract for Reg.ru interactions
type Adapter interface {
    FindRecord(zone, name, typ string) (*Record, error)
    CreateRecord(zone string, r *Record) error
    UpdateRecord(zone string, r *Record) error
    DeleteRecord(zone string, id string) error
    BulkUpdate(zone string, actions []interface{}) error
}

