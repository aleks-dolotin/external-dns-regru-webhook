package adapter

import "time"

// ResourceRef identifies the Kubernetes resource that triggered the operation.
type ResourceRef struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// Operation is the normalized representation of a DNS change request.
// It is the canonical DTO flowing through the system: event intake → queue → worker → adapter.
type Operation struct {
	OpID        string                 `json:"op_id"` // correlating_id — unique UUID v4 per operation
	ResourceRef ResourceRef            `json:"resource_ref"`
	Action      string                 `json:"action"` // create, update, delete
	Zone        string                 `json:"zone"`   // DNS zone (e.g. "example.com")
	Name        string                 `json:"name"`   // FQDN of the record (e.g. "app.example.com")
	Type        string                 `json:"type"`   // record type: A, AAAA, CNAME, TXT
	Content     string                 `json:"content,omitempty"`
	TTL         int                    `json:"ttl,omitempty"`
	Priority    int                    `json:"priority,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	SourceEvent string                 `json:"source_event_id,omitempty"`
	K8sMeta     map[string]interface{} `json:"k8s_metadata,omitempty"`
}
