package config

import (
	"fmt"
	"strings"
	"text/template"
)

// ValidationError collects field-level validation problems.
type ValidationError struct {
	Errors []FieldError
}

// FieldError describes a single validation issue.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (ve *ValidationError) Error() string {
	msgs := make([]string, 0, len(ve.Errors))
	for _, fe := range ve.Errors {
		msgs = append(msgs, fmt.Sprintf("%s: %s", fe.Field, fe.Message))
	}
	return fmt.Sprintf("config validation failed (%d errors): %s",
		len(ve.Errors), strings.Join(msgs, "; "))
}

// Validate checks a MappingsConfig for correctness. Returns a *ValidationError
// with field-level detail if any issues are found. Returns nil if valid.
func Validate(cfg *MappingsConfig) error {
	if cfg == nil {
		return &ValidationError{Errors: []FieldError{
			{Field: "config", Message: "config is nil"},
		}}
	}

	if len(cfg.Zones) == 0 {
		return &ValidationError{Errors: []FieldError{
			{Field: "zones", Message: "at least one zone mapping is required"},
		}}
	}

	var errs []FieldError
	seen := make(map[string]struct{})

	for i, zm := range cfg.Zones {
		prefix := fmt.Sprintf("zones[%d]", i)

		// Zone name must be non-empty and look like a domain.
		if zm.Zone == "" {
			errs = append(errs, FieldError{
				Field:   prefix + ".zone",
				Message: "zone name is required",
			})
		} else if !isValidZone(zm.Zone) {
			errs = append(errs, FieldError{
				Field:   prefix + ".zone",
				Message: fmt.Sprintf("invalid zone format %q: must be a valid domain name", zm.Zone),
			})
		}

		// Duplicate zone detection.
		if zm.Zone != "" {
			if _, dup := seen[zm.Zone]; dup {
				errs = append(errs, FieldError{
					Field:   prefix + ".zone",
					Message: fmt.Sprintf("duplicate zone %q", zm.Zone),
				})
			}
			seen[zm.Zone] = struct{}{}
		}

		// Template must be non-empty and valid Go template syntax.
		if zm.Template == "" {
			errs = append(errs, FieldError{
				Field:   prefix + ".template",
				Message: "FQDN template is required",
			})
		} else if err := validateTemplate(zm.Template); err != nil {
			errs = append(errs, FieldError{
				Field:   prefix + ".template",
				Message: fmt.Sprintf("invalid template syntax: %v", err),
			})
		}

		// TTL must be non-negative.
		if zm.TTL < 0 {
			errs = append(errs, FieldError{
				Field:   prefix + ".ttl",
				Message: fmt.Sprintf("TTL must be >= 0, got %d", zm.TTL),
			})
		}

		// Priority must be non-negative.
		if zm.Priority < 0 {
			errs = append(errs, FieldError{
				Field:   prefix + ".priority",
				Message: fmt.Sprintf("priority must be >= 0, got %d", zm.Priority),
			})
		}

		// Namespace entries must be non-empty strings.
		for j, ns := range zm.Namespaces {
			if strings.TrimSpace(ns) == "" {
				errs = append(errs, FieldError{
					Field:   fmt.Sprintf("%s.namespaces[%d]", prefix, j),
					Message: "namespace name must not be empty",
				})
			}
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

// isValidZone checks that a zone looks like a valid domain name.
// Allows alphanumeric, hyphens, underscores (for _acme-challenge etc.), and dots.
func isValidZone(zone string) bool {
	if len(zone) == 0 || len(zone) > 253 {
		return false
	}
	if !strings.Contains(zone, ".") {
		return false
	}
	parts := strings.Split(zone, ".")
	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
		for _, c := range part {
			if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' && c != '_' {
				return false
			}
		}
		// Labels must not start or end with a hyphen.
		if part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
	}
	return true
}

// validateTemplate checks that a template string is parseable by text/template.
func validateTemplate(tmpl string) error {
	_, err := template.New("check").Parse(tmpl)
	return err
}
