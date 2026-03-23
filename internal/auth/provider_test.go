package auth

import (
	"testing"
)

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
