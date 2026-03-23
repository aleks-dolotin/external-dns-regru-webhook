package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ===== Story 3.1: Store, load, hot-reload tests =====

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	return path
}

const validMappingsYAML = `zones:
  - zone: example.com
    namespaces: ["prod","staging"]
    template: "{{.Name}}.{{.Zone}}"
    ttl: 300
    priority: 10
  - zone: test.org
    namespaces: []
    template: "{{.Name}}-{{.Namespace}}.test.org"
`

func TestNewStore_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "mappings.yaml", validMappingsYAML)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := store.Get()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if len(cfg.Zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(cfg.Zones))
	}
	if cfg.Zones[0].Zone != "example.com" {
		t.Errorf("expected zone 'example.com', got %q", cfg.Zones[0].Zone)
	}
	if cfg.Zones[0].TTL != 300 {
		t.Errorf("expected TTL 300, got %d", cfg.Zones[0].TTL)
	}
	if len(cfg.Zones[0].Namespaces) != 2 {
		t.Errorf("expected 2 namespaces, got %d", len(cfg.Zones[0].Namespaces))
	}
}

func TestNewStore_FileNotFound(t *testing.T) {
	_, err := NewStore("/nonexistent/path/mappings.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestNewStore_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "bad.yaml", "not: [valid: yaml: {{")

	_, err := NewStore(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestNewStore_ValidationFails(t *testing.T) {
	dir := t.TempDir()
	// Empty zones list
	path := writeTestFile(t, dir, "empty.yaml", "zones: []\n")

	_, err := NewStore(path)
	if err == nil {
		t.Fatal("expected validation error for empty zones")
	}
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("expected validation error, got %q", err.Error())
	}
}

func TestStore_FindZone(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "mappings.yaml", validMappingsYAML)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	zm := store.FindZone("example.com")
	if zm == nil {
		t.Fatal("expected to find zone 'example.com'")
	}
	if zm.Zone != "example.com" {
		t.Errorf("expected zone name 'example.com', got %q", zm.Zone)
	}

	if store.FindZone("nonexistent.com") != nil {
		t.Error("expected nil for nonexistent zone")
	}
}

func TestStore_IsNamespaceAllowed(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "mappings.yaml", validMappingsYAML)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		zone      string
		namespace string
		want      bool
	}{
		{"example.com", "prod", true},
		{"example.com", "staging", true},
		{"example.com", "dev", false},
		{"test.org", "anything", true}, // empty namespaces => all allowed
		{"unknown.com", "prod", false}, // zone not found
	}

	for _, tt := range tests {
		got := store.IsNamespaceAllowed(tt.zone, tt.namespace)
		if got != tt.want {
			t.Errorf("IsNamespaceAllowed(%q, %q) = %v, want %v",
				tt.zone, tt.namespace, got, tt.want)
		}
	}
}

func TestStore_Reload_FileChanged(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "mappings.yaml", validMappingsYAML)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.Get().Zones) != 2 {
		t.Fatalf("initial: expected 2 zones")
	}

	// Wait a bit to ensure different mod time on some filesystems.
	time.Sleep(50 * time.Millisecond)

	updatedYAML := `zones:
  - zone: new.io
    template: "{{.Name}}.new.io"
`
	if err := os.WriteFile(path, []byte(updatedYAML), 0644); err != nil {
		t.Fatalf("failed to update file: %v", err)
	}

	// Track reload callback.
	reloaded := false
	store.OnReload = func(_ *MappingsConfig) { reloaded = true }

	if err := store.Reload(); err != nil {
		t.Fatalf("reload error: %v", err)
	}

	cfg := store.Get()
	if len(cfg.Zones) != 1 {
		t.Fatalf("after reload: expected 1 zone, got %d", len(cfg.Zones))
	}
	if cfg.Zones[0].Zone != "new.io" {
		t.Errorf("expected zone 'new.io', got %q", cfg.Zones[0].Zone)
	}
	if !reloaded {
		t.Error("OnReload callback was not called")
	}
}

func TestStore_Reload_InvalidFileRetainsPrevious(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "mappings.yaml", validMappingsYAML)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	originalZoneCount := len(store.Get().Zones)

	time.Sleep(50 * time.Millisecond)

	// Write invalid config.
	if err := os.WriteFile(path, []byte("zones: []\n"), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	var reloadErr error
	store.OnReloadError = func(e error) { reloadErr = e }

	err = store.Reload()
	if err == nil {
		t.Fatal("expected reload error for invalid config")
	}
	if reloadErr == nil {
		t.Error("OnReloadError callback was not called")
	}

	// Previous valid config is retained.
	if len(store.Get().Zones) != originalZoneCount {
		t.Errorf("expected previous config retained (%d zones), got %d",
			originalZoneCount, len(store.Get().Zones))
	}
}

func TestStore_RunReloader_StopsOnDone(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "mappings.yaml", validMappingsYAML)

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		store.RunReloader(done, 10*time.Millisecond)
		close(finished)
	}()

	// Let it tick a few times.
	time.Sleep(50 * time.Millisecond)
	close(done)

	select {
	case <-finished:
		// OK
	case <-time.After(time.Second):
		t.Fatal("RunReloader did not stop within timeout")
	}
}

// ===== Story 3.2: Validation tests =====

func TestValidate_NilConfig(t *testing.T) {
	err := Validate(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(ve.Errors))
	}
}

func TestValidate_EmptyZones(t *testing.T) {
	err := Validate(&MappingsConfig{Zones: []ZoneMapping{}})
	if err == nil {
		t.Fatal("expected error for empty zones")
	}
	ve := err.(*ValidationError)
	if ve.Errors[0].Field != "zones" {
		t.Errorf("expected field 'zones', got %q", ve.Errors[0].Field)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &MappingsConfig{
		Zones: []ZoneMapping{
			{Zone: "example.com", Template: "{{.Name}}.example.com", TTL: 300},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestValidate_MissingZoneName(t *testing.T) {
	cfg := &MappingsConfig{
		Zones: []ZoneMapping{
			{Zone: "", Template: "{{.Name}}.example.com"},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing zone name")
	}
	ve := err.(*ValidationError)
	found := false
	for _, fe := range ve.Errors {
		if strings.Contains(fe.Field, ".zone") && strings.Contains(fe.Message, "required") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected zone required error, got %v", ve.Errors)
	}
}

func TestValidate_InvalidZoneFormat(t *testing.T) {
	tests := []struct {
		name string
		zone string
	}{
		{"no dot", "localhost"},
		{"starts with hyphen", "-bad.com"},
		{"ends with hyphen", "bad-.com"},
		{"empty label", "bad..com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &MappingsConfig{
				Zones: []ZoneMapping{
					{Zone: tt.zone, Template: "{{.Name}}.example.com"},
				},
			}
			err := Validate(cfg)
			if err == nil {
				t.Fatalf("expected error for zone %q", tt.zone)
			}
		})
	}
}

func TestValidate_ValidZoneFormats(t *testing.T) {
	tests := []string{
		"example.com",
		"sub.example.com",
		"a.b.c.d.example.com",
		"test-zone.org",
		"123.com",
		"_acme-challenge.example.com",
		"my_zone.example.com",
	}

	for _, zone := range tests {
		t.Run(zone, func(t *testing.T) {
			cfg := &MappingsConfig{
				Zones: []ZoneMapping{
					{Zone: zone, Template: "{{.Name}}." + zone},
				},
			}
			if err := Validate(cfg); err != nil {
				t.Errorf("expected zone %q to be valid, got: %v", zone, err)
			}
		})
	}
}

func TestValidate_DuplicateZone(t *testing.T) {
	cfg := &MappingsConfig{
		Zones: []ZoneMapping{
			{Zone: "example.com", Template: "{{.Name}}.example.com"},
			{Zone: "example.com", Template: "{{.Name}}.example.com"},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for duplicate zones")
	}
	ve := err.(*ValidationError)
	found := false
	for _, fe := range ve.Errors {
		if strings.Contains(fe.Message, "duplicate") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected duplicate zone error, got %v", ve.Errors)
	}
}

func TestValidate_MissingTemplate(t *testing.T) {
	cfg := &MappingsConfig{
		Zones: []ZoneMapping{
			{Zone: "example.com", Template: ""},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
	ve := err.(*ValidationError)
	found := false
	for _, fe := range ve.Errors {
		if strings.Contains(fe.Field, ".template") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected template error, got %v", ve.Errors)
	}
}

func TestValidate_InvalidTemplateSyntax(t *testing.T) {
	cfg := &MappingsConfig{
		Zones: []ZoneMapping{
			{Zone: "example.com", Template: "{{.Name"},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid template syntax")
	}
	ve := err.(*ValidationError)
	found := false
	for _, fe := range ve.Errors {
		if strings.Contains(fe.Message, "template syntax") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected template syntax error, got %v", ve.Errors)
	}
}

func TestValidate_NegativeTTL(t *testing.T) {
	cfg := &MappingsConfig{
		Zones: []ZoneMapping{
			{Zone: "example.com", Template: "{{.Name}}.example.com", TTL: -1},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for negative TTL")
	}
}

func TestValidate_NegativePriority(t *testing.T) {
	cfg := &MappingsConfig{
		Zones: []ZoneMapping{
			{Zone: "example.com", Template: "{{.Name}}.example.com", Priority: -5},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for negative priority")
	}
}

func TestValidate_EmptyNamespace(t *testing.T) {
	cfg := &MappingsConfig{
		Zones: []ZoneMapping{
			{Zone: "example.com", Template: "{{.Name}}.example.com", Namespaces: []string{"prod", ""}},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for empty namespace")
	}
	ve := err.(*ValidationError)
	found := false
	for _, fe := range ve.Errors {
		if strings.Contains(fe.Field, "namespaces") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected namespace error, got %v", ve.Errors)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &MappingsConfig{
		Zones: []ZoneMapping{
			{Zone: "", Template: ""},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	if len(ve.Errors) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(ve.Errors), ve.Errors)
	}
}

func TestValidationError_ErrorString(t *testing.T) {
	ve := &ValidationError{
		Errors: []FieldError{
			{Field: "zones[0].zone", Message: "zone name is required"},
			{Field: "zones[0].template", Message: "FQDN template is required"},
		},
	}
	s := ve.Error()
	if !strings.Contains(s, "2 errors") {
		t.Errorf("expected '2 errors' in message, got %q", s)
	}
	if !strings.Contains(s, "zone name is required") {
		t.Errorf("expected field error detail in message, got %q", s)
	}
}

// ===== Story 3.3: Template-driven FQDN generation tests =====

func TestRenderFQDN_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    string
		data    TemplateData
		want    string
		wantErr bool
	}{
		{
			name: "simple name.zone",
			tmpl: "{{.Name}}.{{.Zone}}",
			data: TemplateData{Name: "my-app", Zone: "example.com"},
			want: "my-app.example.com",
		},
		{
			name: "name-namespace.zone",
			tmpl: "{{.Name}}-{{.Namespace}}.{{.Zone}}",
			data: TemplateData{Name: "api", Namespace: "prod", Zone: "example.com"},
			want: "api-prod.example.com",
		},
		{
			name: "with labels",
			tmpl: `{{.Name}}.{{index .Labels "env"}}.{{.Zone}}`,
			data: TemplateData{
				Name:   "web",
				Zone:   "example.com",
				Labels: map[string]string{"env": "staging"},
			},
			want: "web.staging.example.com",
		},
		{
			name: "namespace only",
			tmpl: "{{.Namespace}}.{{.Zone}}",
			data: TemplateData{Namespace: "kube-system", Zone: "internal.io"},
			want: "kube-system.internal.io",
		},
		{
			name: "static prefix",
			tmpl: "app-{{.Name}}.{{.Zone}}",
			data: TemplateData{Name: "frontend", Zone: "example.com"},
			want: "app-frontend.example.com",
		},
		{
			name: "complex label template",
			tmpl: `{{.Name}}.{{index .Labels "tier"}}-{{.Namespace}}.{{.Zone}}`,
			data: TemplateData{
				Name:      "cache",
				Namespace: "prod",
				Zone:      "myzone.com",
				Labels:    map[string]string{"tier": "backend"},
			},
			want: "cache.backend-prod.myzone.com",
		},
		{
			name:    "invalid template syntax",
			tmpl:    "{{.Name",
			data:    TemplateData{Name: "test"},
			wantErr: true,
		},
		{
			name: "empty result",
			tmpl: "{{.Name}}",
			data: TemplateData{Name: ""},
			// empty Name renders empty string → should error
			wantErr: true,
		},
		{
			name:    "missing label key with missingkey=error",
			tmpl:    `{{index .Labels "missing"}}`,
			data:    TemplateData{Labels: map[string]string{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderFQDN(tt.tmpl, tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got result %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("RenderFQDN() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderFQDNForZone(t *testing.T) {
	zm := &ZoneMapping{
		Zone:     "example.com",
		Template: "{{.Name}}-{{.Namespace}}.{{.Zone}}",
		TTL:      300,
	}

	fqdn, err := RenderFQDNForZone(zm, "web", "prod", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fqdn != "web-prod.example.com" {
		t.Errorf("expected 'web-prod.example.com', got %q", fqdn)
	}
}

func TestRenderFQDNForZone_WithLabels(t *testing.T) {
	zm := &ZoneMapping{
		Zone:     "test.org",
		Template: `{{.Name}}.{{index .Labels "env"}}.{{.Zone}}`,
	}

	fqdn, err := RenderFQDNForZone(zm, "api", "default",
		map[string]string{"env": "staging"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fqdn != "api.staging.test.org" {
		t.Errorf("expected 'api.staging.test.org', got %q", fqdn)
	}
}

func TestRenderFQDNForZone_NilMapping(t *testing.T) {
	_, err := RenderFQDNForZone(nil, "web", "prod", nil)
	if err == nil {
		t.Fatal("expected error for nil zone mapping")
	}
}

// ===== Zone validation helper tests =====

func TestIsValidZone(t *testing.T) {
	tests := []struct {
		zone string
		want bool
	}{
		{"example.com", true},
		{"sub.example.com", true},
		{"a.b.c.d.e.f.com", true},
		{"test-zone.org", true},
		{"123.456.com", true},
		{"a.co", true},
		{"_acme-challenge.example.com", true},
		{"my_zone.example.com", true},
		{"", false},
		{"localhost", false},
		{"-bad.com", false},
		{"bad-.com", false},
		{"bad..com", false},
		{".com", false},
		{"com.", false},
	}

	for _, tt := range tests {
		t.Run(tt.zone, func(t *testing.T) {
			got := isValidZone(tt.zone)
			if got != tt.want {
				t.Errorf("isValidZone(%q) = %v, want %v", tt.zone, got, tt.want)
			}
		})
	}
}
