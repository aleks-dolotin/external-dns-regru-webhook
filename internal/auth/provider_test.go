package auth

import (
	"testing"
)

func TestLoadCredentials_Success(t *testing.T) {
	t.Setenv(EnvUsername, "testuser")
	t.Setenv(EnvPassword, "testpass")

	provider := &EnvSecretProvider{}
	creds, err := provider.LoadCredentials()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if creds.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", creds.Username)
	}
	if creds.Password != "testpass" {
		t.Errorf("expected password 'testpass', got %q", creds.Password)
	}
}

func TestLoadCredentials_MissingUsername(t *testing.T) {
	t.Setenv(EnvUsername, "")
	t.Setenv(EnvPassword, "testpass")

	provider := &EnvSecretProvider{}
	_, err := provider.LoadCredentials()
	if err == nil {
		t.Fatal("expected error when username missing, got nil")
	}
	if err != ErrMissingCredentials {
		t.Errorf("expected ErrMissingCredentials, got %v", err)
	}
}

func TestLoadCredentials_MissingPassword(t *testing.T) {
	t.Setenv(EnvUsername, "testuser")
	t.Setenv(EnvPassword, "")

	provider := &EnvSecretProvider{}
	_, err := provider.LoadCredentials()
	if err == nil {
		t.Fatal("expected error when password missing, got nil")
	}
	if err != ErrMissingCredentials {
		t.Errorf("expected ErrMissingCredentials, got %v", err)
	}
}

func TestLoadCredentials_BothMissing(t *testing.T) {
	t.Setenv(EnvUsername, "")
	t.Setenv(EnvPassword, "")

	provider := &EnvSecretProvider{}
	_, err := provider.LoadCredentials()
	if err == nil {
		t.Fatal("expected error when both missing, got nil")
	}
}

func TestLoadCredentials_EmptyValues(t *testing.T) {
	t.Setenv(EnvUsername, "")
	t.Setenv(EnvPassword, "")

	provider := &EnvSecretProvider{}
	_, err := provider.LoadCredentials()
	if err == nil {
		t.Fatal("expected error when values empty, got nil")
	}
}

func TestCredentials_Validate_Valid(t *testing.T) {
	creds := Credentials{Username: "user", Password: "pass"}
	if err := creds.Validate(); err != nil {
		t.Fatalf("expected no error for valid creds, got %v", err)
	}
}

func TestCredentials_Validate_EmptyUsername(t *testing.T) {
	creds := Credentials{Username: "", Password: "pass"}
	if err := creds.Validate(); err == nil {
		t.Fatal("expected error for empty username")
	}
}

func TestCredentials_Validate_EmptyPassword(t *testing.T) {
	creds := Credentials{Username: "user", Password: ""}
	if err := creds.Validate(); err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestCredentials_Validate_BothEmpty(t *testing.T) {
	creds := Credentials{}
	if err := creds.Validate(); err == nil {
		t.Fatal("expected error for empty creds")
	}
}
