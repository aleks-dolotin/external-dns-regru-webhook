package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadTimeouts(t *testing.T) {
	tests := []struct {
		name           string
		regruHTTP      string
		webhookRead    string
		webhookWrite   string
		want           Timeouts
		wantErrorParts []string
	}{
		{
			name: "defaults when variables are empty",
			want: Timeouts{
				RegruHTTP:    10 * time.Second,
				WebhookRead:  5 * time.Second,
				WebhookWrite: 60 * time.Second,
			},
		},
		{
			name:         "valid overrides",
			regruHTTP:    "7500ms",
			webhookRead:  "3s",
			webhookWrite: "1m30s",
			want: Timeouts{
				RegruHTTP:    7500 * time.Millisecond,
				WebhookRead:  3 * time.Second,
				WebhookWrite: 90 * time.Second,
			},
		},
		{
			name:           "malformed API timeout",
			regruHTTP:      "10",
			wantErrorParts: []string{EnvRegruHTTPTimeout, "10"},
		},
		{
			name:           "malformed read timeout",
			webhookRead:    "garbage",
			wantErrorParts: []string{EnvWebhookReadTimeout, "garbage"},
		},
		{
			name:           "malformed write timeout",
			webhookWrite:   "60",
			wantErrorParts: []string{EnvWebhookWriteTimeout, "60"},
		},
		{
			name:           "zero API timeout",
			regruHTTP:      "0s",
			wantErrorParts: []string{EnvRegruHTTPTimeout, "0s"},
		},
		{
			name:           "negative API timeout",
			regruHTTP:      "-1s",
			wantErrorParts: []string{EnvRegruHTTPTimeout, "-1s"},
		},
		{
			name:           "zero read timeout",
			webhookRead:    "0s",
			wantErrorParts: []string{EnvWebhookReadTimeout, "0s"},
		},
		{
			name:           "negative read timeout",
			webhookRead:    "-1s",
			wantErrorParts: []string{EnvWebhookReadTimeout, "-1s"},
		},
		{
			name:           "zero write timeout",
			webhookWrite:   "0s",
			wantErrorParts: []string{EnvWebhookWriteTimeout, "0s"},
		},
		{
			name:           "negative write timeout",
			webhookWrite:   "-1s",
			wantErrorParts: []string{EnvWebhookWriteTimeout, "-1s"},
		},
		{
			name:           "write timeout equals API timeout",
			regruHTTP:      "10s",
			webhookWrite:   "10s",
			wantErrorParts: []string{EnvWebhookWriteTimeout, EnvRegruHTTPTimeout},
		},
		{
			name:           "write timeout is less than API timeout",
			regruHTTP:      "20s",
			webhookWrite:   "10s",
			wantErrorParts: []string{EnvWebhookWriteTimeout, EnvRegruHTTPTimeout},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvRegruHTTPTimeout, tt.regruHTTP)
			t.Setenv(EnvWebhookReadTimeout, tt.webhookRead)
			t.Setenv(EnvWebhookWriteTimeout, tt.webhookWrite)

			got, err := LoadTimeouts()
			if len(tt.wantErrorParts) > 0 {
				if err == nil {
					t.Fatalf("LoadTimeouts() error = nil, want error containing %q", tt.wantErrorParts)
				}
				for _, part := range tt.wantErrorParts {
					if !strings.Contains(err.Error(), part) {
						t.Fatalf("LoadTimeouts() error = %q, want it to contain %q", err, part)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadTimeouts() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("LoadTimeouts() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoadTimeoutsDefaultsWhenVariablesAreUnset(t *testing.T) {
	for _, name := range []string{
		EnvRegruHTTPTimeout,
		EnvWebhookReadTimeout,
		EnvWebhookWriteTimeout,
	} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}

	got, err := LoadTimeouts()
	if err != nil {
		t.Fatalf("LoadTimeouts() unexpected error: %v", err)
	}
	want := Timeouts{
		RegruHTTP:    DefaultRegruHTTPTimeout,
		WebhookRead:  DefaultWebhookReadTimeout,
		WebhookWrite: DefaultWebhookWriteTimeout,
	}
	if got != want {
		t.Errorf("LoadTimeouts() = %+v, want %+v", got, want)
	}
}
