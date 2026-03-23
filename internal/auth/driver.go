package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Environment variable for selecting the authentication driver.
const EnvAuthDriver = "AUTH_DRIVER"

// Supported driver names.
const (
	DriverToken = "token"
	DriverRSA   = "rsa"
)

// Environment variable names for credential configuration.
const (
	EnvUsername          = "REGU_USERNAME"
	EnvPassword          = "REGU_PASSWORD"
	EnvRSAPrivateKey     = "REGU_RSA_PRIVATE_KEY"      // PEM-encoded private key (inline)
	EnvRSAPrivateKeyPath = "REGU_RSA_PRIVATE_KEY_PATH" // Path to PEM-encoded private key file
	EnvCredentialsPath   = "REGU_CREDENTIALS_PATH"     // Directory with secret files (K8s volume mount)
)

// Sentinel errors for driver operations.
var (
	ErrMissingCredentials = errors.New("auth: missing required credentials")
	ErrUnsupportedDriver  = errors.New("auth: unsupported driver type")
	ErrMissingRSAKey      = errors.New("auth: RSA private key not provided")
	ErrInvalidRSAKey      = errors.New("auth: invalid RSA private key")
)

// Credentials holds basic credential representation.
type Credentials struct {
	Username string
	Password string
}

// Validate checks that both Username and Password are non-empty.
func (c Credentials) Validate() error {
	if c.Username == "" || c.Password == "" {
		return ErrMissingCredentials
	}
	return nil
}

// AuthDriver interface abstracts authentication mechanism for Reg.ru.
type AuthDriver interface {
	// PrepareAuth returns key-value pairs to include in API request payloads.
	PrepareAuth() (map[string]string, error)
}

// TokenDriver is a simple username/password driver.
type TokenDriver struct {
	Creds Credentials
}

// PrepareAuth returns username and password as request parameters.
func (t *TokenDriver) PrepareAuth() (map[string]string, error) {
	if t.Creds.Username == "" || t.Creds.Password == "" {
		return nil, ErrMissingCredentials
	}
	return map[string]string{
		"username": t.Creds.Username,
		"password": t.Creds.Password,
	}, nil
}

// RSASignatureDriver implements authentication using RSA signature.
// Per Reg.ru API v2 spec, the signature is computed over the username string
// using SHA-256 with the private key, then base64-encoded.
type RSASignatureDriver struct {
	Username   string
	PrivateKey *rsa.PrivateKey
}

// PrepareAuth returns username and RSA signature as request parameters.
func (r *RSASignatureDriver) PrepareAuth() (map[string]string, error) {
	if r.Username == "" {
		return nil, ErrMissingCredentials
	}
	if r.PrivateKey == nil {
		return nil, ErrMissingRSAKey
	}

	hashed := sha256.Sum256([]byte(r.Username))
	sig, err := rsa.SignPKCS1v15(rand.Reader, r.PrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, fmt.Errorf("auth: RSA signing failed: %w", err)
	}

	return map[string]string{
		"username": r.Username,
		"sig":      base64.StdEncoding.EncodeToString(sig),
	}, nil
}

// ParseRSAPrivateKey parses a PEM-encoded RSA private key (PKCS1 or PKCS8).
func ParseRSAPrivateKey(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, ErrInvalidRSAKey
	}

	// Try PKCS8 first, then PKCS1.
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, ErrInvalidRSAKey
		}
		return rsaKey, nil
	}

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRSAKey, err)
	}
	return rsaKey, nil
}

// readCredentialFile reads a single credential value from a file inside basePath.
// Returns empty string if file does not exist (caller falls back to env var).
func readCredentialFile(basePath, filename string) string {
	data, err := os.ReadFile(filepath.Join(basePath, filename))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// resolveCredential returns a credential value by first trying the file-based
// secret mount (REGU_CREDENTIALS_PATH/<filename>), then falling back to the
// environment variable. File-based reading is required for K8s secret rotation
// without pod restart — env vars set via secretKeyRef are immutable at runtime.
func resolveCredential(filename, envVar string) string {
	if basePath := os.Getenv(EnvCredentialsPath); basePath != "" {
		if v := readCredentialFile(basePath, filename); v != "" {
			return v
		}
	}
	return os.Getenv(envVar)
}

// NewDriverFromEnv creates the appropriate AuthDriver based on the AUTH_DRIVER
// environment variable. Defaults to "token" when AUTH_DRIVER is not set.
//
// Credentials are resolved in order:
//  1. File from REGU_CREDENTIALS_PATH/<key> (K8s secret volume mount — supports live rotation)
//  2. Environment variable fallback (REGU_USERNAME, REGU_PASSWORD, etc.)
func NewDriverFromEnv() (AuthDriver, error) {
	driverType := strings.ToLower(os.Getenv(EnvAuthDriver))
	if driverType == "" {
		driverType = DriverToken
	}

	switch driverType {
	case DriverToken:
		return newTokenDriverFromEnv()
	case DriverRSA:
		return newRSADriverFromEnv()
	default:
		return nil, fmt.Errorf("%w: %q (supported: token, rsa)", ErrUnsupportedDriver, driverType)
	}
}

// newTokenDriverFromEnv constructs a TokenDriver. Reads from secret files first,
// then falls back to environment variables.
func newTokenDriverFromEnv() (*TokenDriver, error) {
	creds := Credentials{
		Username: resolveCredential("username", EnvUsername),
		Password: resolveCredential("password", EnvPassword),
	}
	if err := creds.Validate(); err != nil {
		return nil, err
	}
	return &TokenDriver{Creds: creds}, nil
}

// newRSADriverFromEnv constructs an RSASignatureDriver. Reads username from
// secret files first, then falls back to environment variables. RSA key is
// resolved from inline env, file env path, or secret mount directory.
func newRSADriverFromEnv() (*RSASignatureDriver, error) {
	username := resolveCredential("username", EnvUsername)
	if username == "" {
		return nil, ErrMissingCredentials
	}

	// Try inline PEM env var first.
	pemData := []byte(os.Getenv(EnvRSAPrivateKey))

	// Then try explicit file path env var.
	if len(pemData) == 0 {
		keyPath := os.Getenv(EnvRSAPrivateKeyPath)
		if keyPath != "" {
			var err error
			pemData, err = os.ReadFile(keyPath)
			if err != nil {
				return nil, fmt.Errorf("auth: reading RSA key file: %w", err)
			}
		}
	}

	// Finally try secret mount directory.
	if len(pemData) == 0 {
		if basePath := os.Getenv(EnvCredentialsPath); basePath != "" {
			candidate := filepath.Join(basePath, "rsa-key")
			if data, err := os.ReadFile(candidate); err == nil {
				pemData = data
			}
		}
	}

	if len(pemData) == 0 {
		return nil, ErrMissingRSAKey
	}

	key, err := ParseRSAPrivateKey(pemData)
	if err != nil {
		return nil, err
	}

	return &RSASignatureDriver{
		Username:   username,
		PrivateKey: key,
	}, nil
}
