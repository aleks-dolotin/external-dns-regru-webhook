package config

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// TemplateData holds the variables available inside FQDN templates.
// Template variables: .Name, .Namespace, .Zone, .Labels (map).
type TemplateData struct {
	Name      string
	Namespace string
	Zone      string
	Labels    map[string]string
}

// RenderFQDN executes the given Go text/template string against data and
// returns the resulting FQDN. Returns an error if the template is invalid
// or execution fails (e.g. missing required variable).
func RenderFQDN(tmpl string, data TemplateData) (string, error) {
	t, err := template.New("fqdn").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("config: template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("config: template execution error: %w", err)
	}

	result := strings.TrimSpace(buf.String())
	if result == "" {
		return "", fmt.Errorf("config: template produced empty FQDN")
	}

	return result, nil
}

// RenderFQDNForZone is a convenience wrapper that uses the zone mapping's
// template and TTL/priority defaults to render an FQDN.
func RenderFQDNForZone(zm *ZoneMapping, name, namespace string, labels map[string]string) (string, error) {
	if zm == nil {
		return "", fmt.Errorf("config: zone mapping is nil")
	}

	data := TemplateData{
		Name:      name,
		Namespace: namespace,
		Zone:      zm.Zone,
		Labels:    labels,
	}

	return RenderFQDN(zm.Template, data)
}
