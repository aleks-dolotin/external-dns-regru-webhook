package auth

import (
	"errors"
	"os"
)

// Environment variable names for Reg.ru credentials (populated from K8s Secret).
const (
	EnvUsername = "REGU_USERNAME"
	EnvPassword = "REGU_PASSWORD"
)

// ErrMissingCredentials is returned when required credential environment variables are not set.
var ErrMissingCredentials = errors.New("auth: missing required credentials")

// Validate checks that both Username and Password are non-empty.
func (c Credentials) Validate() error {
	if c.Username == "" || c.Password == "" {
		return ErrMissingCredentials
	}
	return nil
}

// EnvSecretProvider reads Reg.ru credentials from environment variables
// that Kubernetes populates from a Secret (via secretKeyRef in the deployment manifest).
type EnvSecretProvider struct{}

// LoadCredentials reads REGU_USERNAME and REGU_PASSWORD from environment variables.
// Returns ErrMissingCredentials if either variable is unset or empty.
func (p *EnvSecretProvider) LoadCredentials() (Credentials, error) {
	creds := Credentials{
		Username: os.Getenv(EnvUsername),
		Password: os.Getenv(EnvPassword),
	}
	if err := creds.Validate(); err != nil {
		return Credentials{}, err
	}
	return creds, nil
}
