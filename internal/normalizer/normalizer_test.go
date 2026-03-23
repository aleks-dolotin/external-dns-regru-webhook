package normalizer

import (
	"strings"
	"testing"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
)

func TestNormalize_ValidEvent(t *testing.T) {
	event := DNSEndpointEvent{
		SourceEventID: "evt-123",
		ResourceRef: adapter.ResourceRef{
			Kind:      "Ingress",
			Namespace: "default",
			Name:      "my-app",
		},
		Zone:       "example.com",
		FQDN:       "app.example.com",
		RecordType: "A",
		Content:    "1.2.3.4",
		TTL:        300,
		Action:     "create",
	}

	op, err := Normalize(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// correlating_id (OpID) must be a non-empty UUID
	if op.OpID == "" {
		t.Error("expected non-empty OpID (correlating_id)")
	}
	if len(strings.Split(op.OpID, "-")) != 5 {
		t.Errorf("expected UUID format (5 groups), got %q", op.OpID)
	}

	if op.Zone != "example.com" {
		t.Errorf("expected zone 'example.com', got %q", op.Zone)
	}
	if op.Name != "app.example.com" {
		t.Errorf("expected name 'app.example.com', got %q", op.Name)
	}
	if op.Type != "A" {
		t.Errorf("expected type 'A', got %q", op.Type)
	}
	if op.Content != "1.2.3.4" {
		t.Errorf("expected content '1.2.3.4', got %q", op.Content)
	}
	if op.TTL != 300 {
		t.Errorf("expected TTL 300, got %d", op.TTL)
	}
	if op.Action != "create" {
		t.Errorf("expected action 'create', got %q", op.Action)
	}
	if op.SourceEvent != "evt-123" {
		t.Errorf("expected source_event 'evt-123', got %q", op.SourceEvent)
	}
	if op.ResourceRef.Kind != "Ingress" {
		t.Errorf("expected resource kind 'Ingress', got %q", op.ResourceRef.Kind)
	}
	if op.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestNormalize_UniqueCorrelatingID(t *testing.T) {
	event := DNSEndpointEvent{
		Zone:       "example.com",
		FQDN:       "a.example.com",
		RecordType: "A",
		Action:     "create",
	}

	ids := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		op, err := Normalize(event)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if _, dup := ids[op.OpID]; dup {
			t.Fatalf("duplicate correlating_id at iteration %d: %s", i, op.OpID)
		}
		ids[op.OpID] = struct{}{}
	}
}

func TestNormalize_MissingZone(t *testing.T) {
	event := DNSEndpointEvent{
		FQDN:       "app.example.com",
		RecordType: "A",
		Action:     "create",
	}

	_, err := Normalize(event)
	if err == nil {
		t.Fatal("expected error for missing zone")
	}
	if !strings.Contains(err.Error(), "zone") {
		t.Errorf("expected error about zone, got %q", err.Error())
	}
}

func TestNormalize_MissingFQDN(t *testing.T) {
	event := DNSEndpointEvent{
		Zone:       "example.com",
		RecordType: "A",
		Action:     "create",
	}

	_, err := Normalize(event)
	if err == nil {
		t.Fatal("expected error for missing fqdn")
	}
	if !strings.Contains(err.Error(), "fqdn") {
		t.Errorf("expected error about fqdn, got %q", err.Error())
	}
}

func TestNormalize_MissingRecordType(t *testing.T) {
	event := DNSEndpointEvent{
		Zone:   "example.com",
		FQDN:   "app.example.com",
		Action: "create",
	}

	_, err := Normalize(event)
	if err == nil {
		t.Fatal("expected error for missing record_type")
	}
	if !strings.Contains(err.Error(), "record_type") {
		t.Errorf("expected error about record_type, got %q", err.Error())
	}
}

func TestNormalize_MissingAction(t *testing.T) {
	event := DNSEndpointEvent{
		Zone:       "example.com",
		FQDN:       "app.example.com",
		RecordType: "CNAME",
	}

	_, err := Normalize(event)
	if err == nil {
		t.Fatal("expected error for missing action")
	}
	if !strings.Contains(err.Error(), "action") {
		t.Errorf("expected error about action, got %q", err.Error())
	}
}

func TestNormalize_InvalidAction(t *testing.T) {
	event := DNSEndpointEvent{
		Zone:       "example.com",
		FQDN:       "app.example.com",
		RecordType: "A",
		Action:     "invalid",
	}

	_, err := Normalize(event)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
	if !strings.Contains(err.Error(), "action") {
		t.Errorf("expected error about action, got %q", err.Error())
	}
}

func TestNormalize_AllValidActions(t *testing.T) {
	for _, action := range []string{"create", "update", "delete"} {
		event := DNSEndpointEvent{
			Zone:       "example.com",
			FQDN:       "app.example.com",
			RecordType: "A",
			Action:     action,
		}
		op, err := Normalize(event)
		if err != nil {
			t.Errorf("action %q: unexpected error: %v", action, err)
		}
		if op.Action != action {
			t.Errorf("expected action %q, got %q", action, op.Action)
		}
	}
}

func TestNormalize_K8sMetadataPreserved(t *testing.T) {
	meta := map[string]interface{}{
		"labels": map[string]interface{}{
			"app": "nginx",
		},
	}
	event := DNSEndpointEvent{
		Zone:       "example.com",
		FQDN:       "app.example.com",
		RecordType: "A",
		Action:     "create",
		K8sMeta:    meta,
	}

	op, err := Normalize(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.K8sMeta == nil {
		t.Fatal("expected K8sMeta to be preserved")
	}
	labels, ok := op.K8sMeta["labels"].(map[string]interface{})
	if !ok {
		t.Fatal("expected labels in K8sMeta")
	}
	if labels["app"] != "nginx" {
		t.Errorf("expected label app=nginx, got %v", labels["app"])
	}
}

func TestNormalize_DeleteWithoutContent(t *testing.T) {
	event := DNSEndpointEvent{
		Zone:       "example.com",
		FQDN:       "old.example.com",
		RecordType: "CNAME",
		Action:     "delete",
	}

	op, err := Normalize(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.Content != "" {
		t.Errorf("expected empty content for delete, got %q", op.Content)
	}
}

func TestNormalizeBatch_Mixed(t *testing.T) {
	events := []DNSEndpointEvent{
		{Zone: "a.com", FQDN: "x.a.com", RecordType: "A", Action: "create", Content: "1.1.1.1"},
		{Zone: "b.com", FQDN: "y.b.com", RecordType: "CNAME", Action: "delete"},
	}

	ops, errs := NormalizeBatch(events)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(ops) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(ops))
	}
	if ops[0].Zone != "a.com" {
		t.Errorf("op[0] zone: expected 'a.com', got %q", ops[0].Zone)
	}
	if ops[1].Zone != "b.com" {
		t.Errorf("op[1] zone: expected 'b.com', got %q", ops[1].Zone)
	}
	// Each should have unique OpID
	if ops[0].OpID == ops[1].OpID {
		t.Error("expected unique OpIDs for different operations")
	}
}

func TestNormalizeBatch_PartialErrors(t *testing.T) {
	events := []DNSEndpointEvent{
		{Zone: "a.com", FQDN: "x.a.com", RecordType: "A", Action: "create"},
		{Zone: "", FQDN: "y.b.com", RecordType: "A", Action: "create"}, // invalid: missing zone
		{Zone: "c.com", FQDN: "z.c.com", RecordType: "TXT", Action: "update"},
	}

	ops, errs := NormalizeBatch(events)
	if len(ops) != 2 {
		t.Errorf("expected 2 valid operations, got %d", len(ops))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestNormalizeBatch_Empty(t *testing.T) {
	ops, errs := NormalizeBatch(nil)
	if len(ops) != 0 {
		t.Errorf("expected 0 ops for nil input, got %d", len(ops))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors for nil input, got %d", len(errs))
	}
}

func TestGenerateUUID_Format(t *testing.T) {
	id, err := generateUUID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("expected 5 UUID groups, got %d in %q", len(parts), id)
	}
	// 8-4-4-4-12 hex chars
	expectedLens := []int{8, 4, 4, 4, 12}
	for i, p := range parts {
		if len(p) != expectedLens[i] {
			t.Errorf("group %d: expected %d chars, got %d in %q", i, expectedLens[i], len(p), id)
		}
	}
}
