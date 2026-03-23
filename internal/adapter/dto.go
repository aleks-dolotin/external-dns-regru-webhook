package adapter

import "time"

type ResourceRef struct {
    Kind      string `json:"kind"`
    Namespace string `json:"namespace"`
    Name      string `json:"name"`
}

type Operation struct {
    OpID        string                 `json:"op_id"`
    ResourceRef ResourceRef            `json:"resource_ref"`
    Action      string                 `json:"action"`
    Zone        string                 `json:"zone"`
    Name        string                 `json:"name"`
    Type        string                 `json:"type"`
    Content     string                 `json:"content,omitempty"`
    TTL         int                    `json:"ttl,omitempty"`
    Priority    int                    `json:"priority,omitempty"`
    Timestamp   time.Time              `json:"timestamp"`
    SourceEvent string                 `json:"source_event_id,omitempty"`
    K8sMeta     map[string]interface{} `json:"k8s_metadata,omitempty"`
}

