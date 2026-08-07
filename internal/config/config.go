// Package config loads and validates webhook runtime configuration.
package config

import (
	"fmt"
	"os"
	"time"
)

const (
	EnvRegruHTTPTimeout    = "REGRU_HTTP_TIMEOUT"
	EnvWebhookReadTimeout  = "WEBHOOK_READ_TIMEOUT"
	EnvWebhookWriteTimeout = "WEBHOOK_WRITE_TIMEOUT"

	DefaultRegruHTTPTimeout    = 10 * time.Second
	DefaultWebhookReadTimeout  = 5 * time.Second
	DefaultWebhookWriteTimeout = 60 * time.Second
)

// Timeouts contains the deadlines used by the Reg.ru client and webhook server.
type Timeouts struct {
	RegruHTTP    time.Duration
	WebhookRead  time.Duration
	WebhookWrite time.Duration
}

// LoadTimeouts reads timeout settings from the environment and validates them.
// Missing and empty variables use their defaults.
func LoadTimeouts() (Timeouts, error) {
	regruHTTP, err := parsePositiveDuration(EnvRegruHTTPTimeout, DefaultRegruHTTPTimeout)
	if err != nil {
		return Timeouts{}, err
	}

	webhookRead, err := parsePositiveDuration(EnvWebhookReadTimeout, DefaultWebhookReadTimeout)
	if err != nil {
		return Timeouts{}, err
	}

	webhookWrite, err := parsePositiveDuration(EnvWebhookWriteTimeout, DefaultWebhookWriteTimeout)
	if err != nil {
		return Timeouts{}, err
	}

	if webhookWrite <= regruHTTP {
		return Timeouts{}, fmt.Errorf(
			"%s (%s) must be greater than %s (%s)",
			EnvWebhookWriteTimeout,
			webhookWrite,
			EnvRegruHTTPTimeout,
			regruHTTP,
		)
	}

	return Timeouts{
		RegruHTTP:    regruHTTP,
		WebhookRead:  webhookRead,
		WebhookWrite: webhookWrite,
	}, nil
}

func parsePositiveDuration(name string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s has invalid duration %q: %w", name, value, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration, got %q", name, value)
	}

	return duration, nil
}
