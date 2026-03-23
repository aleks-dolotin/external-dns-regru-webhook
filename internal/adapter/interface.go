package adapter

// Record represents a DNS record in the adapter domain model
type Record struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
}

// BulkAction represents a single action in a bulk update_records request.
type BulkAction struct {
	Action    string // e.g. "add_alias", "add_cname", "remove_record"
	Subdomain string
	Content   string
	RecType   string // needed for remove_record: record_type field
}

// Adapter defines the contract for Reg.ru interactions
type Adapter interface {
	FindRecord(zone, name, typ string) (*Record, error)
	CreateRecord(zone string, r *Record) error
	UpdateRecord(zone string, r *Record) error
	DeleteRecord(zone string, id string) error
	BulkUpdate(zone string, actions []BulkAction) error
}
