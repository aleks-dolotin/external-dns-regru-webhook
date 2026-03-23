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
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------- helpers ----------

// generateTestRSAKey generates a 2048-bit RSA key pair for tests.
func generateTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return key
}

// marshalPKCS1PEM encodes an RSA private key to PEM (PKCS1).
func marshalPKCS1PEM(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// marshalPKCS8PEM encodes an RSA private key to PEM (PKCS8).
func marshalPKCS8PEM(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal PKCS8: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})
}

// writeTempKeyFile writes PEM data to a temporary file and returns its path.
func writeTempKeyFile(t *testing.T, pemData []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test_key.pem")
	if err := os.WriteFile(path, pemData, 0600); err != nil {
		t.Fatalf("write temp key: %v", err)
	}
	return path
}

// ---------- TokenDriver tests ----------

func TestTokenDriver_PrepareAuth_Success(t *testing.T) {
	d := &TokenDriver{Creds: Credentials{Username: "user", Password: "pass"}}
	params, err := d.PrepareAuth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["username"] != "user" {
		t.Errorf("expected username 'user', got %q", params["username"])
	}
	if params["password"] != "pass" {
		t.Errorf("expected password 'pass', got %q", params["password"])
	}
}

func TestTokenDriver_PrepareAuth_MissingUsername(t *testing.T) {
	d := &TokenDriver{Creds: Credentials{Username: "", Password: "pass"}}
	_, err := d.PrepareAuth()
	if !errors.Is(err, ErrMissingCredentials) {
		t.Errorf("expected ErrMissingCredentials, got %v", err)
	}
}

func TestTokenDriver_PrepareAuth_MissingPassword(t *testing.T) {
	d := &TokenDriver{Creds: Credentials{Username: "user", Password: ""}}
	_, err := d.PrepareAuth()
	if !errors.Is(err, ErrMissingCredentials) {
		t.Errorf("expected ErrMissingCredentials, got %v", err)
	}
}

func TestTokenDriver_PrepareAuth_BothMissing(t *testing.T) {
	d := &TokenDriver{Creds: Credentials{}}
	_, err := d.PrepareAuth()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------- RSASignatureDriver tests ----------

func TestRSADriver_PrepareAuth_Success(t *testing.T) {
	key := generateTestRSAKey(t)
	d := &RSASignatureDriver{Username: "testuser", PrivateKey: key}

	params, err := d.PrepareAuth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got %q", params["username"])
	}
	if params["sig"] == "" {
		t.Fatal("expected non-empty sig")
	}

	// Verify the signature is valid using public key.
	sigBytes, err := base64.StdEncoding.DecodeString(params["sig"])
	if err != nil {
		t.Fatalf("decode base64 sig: %v", err)
	}
	hashed := sha256.Sum256([]byte("testuser"))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, hashed[:], sigBytes); err != nil {
		t.Errorf("signature verification failed: %v", err)
	}
}

func TestRSADriver_PrepareAuth_MissingUsername(t *testing.T) {
	key := generateTestRSAKey(t)
	d := &RSASignatureDriver{Username: "", PrivateKey: key}
	_, err := d.PrepareAuth()
	if !errors.Is(err, ErrMissingCredentials) {
		t.Errorf("expected ErrMissingCredentials, got %v", err)
	}
}

func TestRSADriver_PrepareAuth_NilKey(t *testing.T) {
	d := &RSASignatureDriver{Username: "user", PrivateKey: nil}
	_, err := d.PrepareAuth()
	if !errors.Is(err, ErrMissingRSAKey) {
		t.Errorf("expected ErrMissingRSAKey, got %v", err)
	}
}

func TestRSADriver_SignaturesDiffer(t *testing.T) {
	key := generateTestRSAKey(t)
	d1 := &RSASignatureDriver{Username: "user1", PrivateKey: key}
	d2 := &RSASignatureDriver{Username: "user2", PrivateKey: key}

	p1, _ := d1.PrepareAuth()
	p2, _ := d2.PrepareAuth()

	if p1["sig"] == p2["sig"] {
		t.Error("different usernames should produce different signatures")
	}
}

// ---------- ParseRSAPrivateKey tests ----------

func TestParseRSAPrivateKey_PKCS1(t *testing.T) {
	key := generateTestRSAKey(t)
	pemData := marshalPKCS1PEM(key)

	parsed, err := ParseRSAPrivateKey(pemData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !key.Equal(parsed) {
		t.Error("parsed key does not match original")
	}
}

func TestParseRSAPrivateKey_PKCS8(t *testing.T) {
	key := generateTestRSAKey(t)
	pemData := marshalPKCS8PEM(t, key)

	parsed, err := ParseRSAPrivateKey(pemData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !key.Equal(parsed) {
		t.Error("parsed key does not match original")
	}
}

func TestParseRSAPrivateKey_InvalidPEM(t *testing.T) {
	_, err := ParseRSAPrivateKey([]byte("not a PEM"))
	if !errors.Is(err, ErrInvalidRSAKey) {
		t.Errorf("expected ErrInvalidRSAKey, got %v", err)
	}
}

func TestParseRSAPrivateKey_EmptyInput(t *testing.T) {
	_, err := ParseRSAPrivateKey([]byte{})
	if !errors.Is(err, ErrInvalidRSAKey) {
		t.Errorf("expected ErrInvalidRSAKey, got %v", err)
	}
}

func TestParseRSAPrivateKey_GarbagePEMBlock(t *testing.T) {
	garbage := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: []byte("garbage"),
	})
	_, err := ParseRSAPrivateKey(garbage)
	if !errors.Is(err, ErrInvalidRSAKey) {
		t.Errorf("expected ErrInvalidRSAKey, got %v", err)
	}
}

// ---------- NewDriverFromEnv tests ----------

func TestNewDriverFromEnv_DefaultToken(t *testing.T) {
	// No AUTH_DRIVER set → defaults to token.
	t.Setenv(EnvAuthDriver, "")
	t.Setenv(EnvUsername, "user")
	t.Setenv(EnvPassword, "pass")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := driver.(*TokenDriver); !ok {
		t.Errorf("expected *TokenDriver, got %T", driver)
	}
}

func TestNewDriverFromEnv_ExplicitToken(t *testing.T) {
	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvUsername, "user")
	t.Setenv(EnvPassword, "pass")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := driver.(*TokenDriver); !ok {
		t.Errorf("expected *TokenDriver, got %T", driver)
	}
}

func TestNewDriverFromEnv_TokenMissingCreds(t *testing.T) {
	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvUsername, "")
	t.Setenv(EnvPassword, "")

	_, err := NewDriverFromEnv()
	if !errors.Is(err, ErrMissingCredentials) {
		t.Errorf("expected ErrMissingCredentials, got %v", err)
	}
}

func TestNewDriverFromEnv_RSAInlineKey(t *testing.T) {
	key := generateTestRSAKey(t)
	pemData := marshalPKCS1PEM(key)

	t.Setenv(EnvAuthDriver, "rsa")
	t.Setenv(EnvUsername, "rsauser")
	t.Setenv(EnvPassword, "")
	t.Setenv(EnvRSAPrivateKey, string(pemData))
	t.Setenv(EnvRSAPrivateKeyPath, "")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rsaDriver, ok := driver.(*RSASignatureDriver)
	if !ok {
		t.Fatalf("expected *RSASignatureDriver, got %T", driver)
	}
	if rsaDriver.Username != "rsauser" {
		t.Errorf("expected username 'rsauser', got %q", rsaDriver.Username)
	}
}

func TestNewDriverFromEnv_RSAKeyFile(t *testing.T) {
	key := generateTestRSAKey(t)
	pemData := marshalPKCS1PEM(key)
	keyPath := writeTempKeyFile(t, pemData)

	t.Setenv(EnvAuthDriver, "rsa")
	t.Setenv(EnvUsername, "rsauser")
	t.Setenv(EnvPassword, "")
	t.Setenv(EnvRSAPrivateKey, "")
	t.Setenv(EnvRSAPrivateKeyPath, keyPath)

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := driver.(*RSASignatureDriver); !ok {
		t.Fatalf("expected *RSASignatureDriver, got %T", driver)
	}
}

func TestNewDriverFromEnv_RSAMissingKey(t *testing.T) {
	t.Setenv(EnvAuthDriver, "rsa")
	t.Setenv(EnvUsername, "user")
	t.Setenv(EnvPassword, "")
	t.Setenv(EnvRSAPrivateKey, "")
	t.Setenv(EnvRSAPrivateKeyPath, "")

	_, err := NewDriverFromEnv()
	if !errors.Is(err, ErrMissingRSAKey) {
		t.Errorf("expected ErrMissingRSAKey, got %v", err)
	}
}

func TestNewDriverFromEnv_RSAMissingUsername(t *testing.T) {
	key := generateTestRSAKey(t)
	pemData := marshalPKCS1PEM(key)

	t.Setenv(EnvAuthDriver, "rsa")
	t.Setenv(EnvUsername, "")
	t.Setenv(EnvPassword, "")
	t.Setenv(EnvRSAPrivateKey, string(pemData))

	_, err := NewDriverFromEnv()
	if !errors.Is(err, ErrMissingCredentials) {
		t.Errorf("expected ErrMissingCredentials, got %v", err)
	}
}

func TestNewDriverFromEnv_RSABadKeyFile(t *testing.T) {
	t.Setenv(EnvAuthDriver, "rsa")
	t.Setenv(EnvUsername, "user")
	t.Setenv(EnvPassword, "")
	t.Setenv(EnvRSAPrivateKey, "")
	t.Setenv(EnvRSAPrivateKeyPath, "/nonexistent/path/key.pem")

	_, err := NewDriverFromEnv()
	if err == nil {
		t.Fatal("expected error for nonexistent key file")
	}
}

func TestNewDriverFromEnv_RSAInvalidPEM(t *testing.T) {
	t.Setenv(EnvAuthDriver, "rsa")
	t.Setenv(EnvUsername, "user")
	t.Setenv(EnvPassword, "")
	t.Setenv(EnvRSAPrivateKey, "not-valid-pem-data")
	t.Setenv(EnvRSAPrivateKeyPath, "")

	_, err := NewDriverFromEnv()
	if !errors.Is(err, ErrInvalidRSAKey) {
		t.Errorf("expected ErrInvalidRSAKey, got %v", err)
	}
}

func TestNewDriverFromEnv_UnsupportedDriver(t *testing.T) {
	t.Setenv(EnvAuthDriver, "oauth2")
	t.Setenv(EnvUsername, "user")
	t.Setenv(EnvPassword, "pass")

	_, err := NewDriverFromEnv()
	if !errors.Is(err, ErrUnsupportedDriver) {
		t.Errorf("expected ErrUnsupportedDriver, got %v", err)
	}
}

func TestNewDriverFromEnv_CaseInsensitive(t *testing.T) {
	t.Setenv(EnvAuthDriver, "TOKEN")
	t.Setenv(EnvUsername, "user")
	t.Setenv(EnvPassword, "pass")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := driver.(*TokenDriver); !ok {
		t.Errorf("expected *TokenDriver for AUTH_DRIVER=TOKEN, got %T", driver)
	}
}

func TestNewDriverFromEnv_RSACaseInsensitive(t *testing.T) {
	key := generateTestRSAKey(t)
	pemData := marshalPKCS1PEM(key)

	t.Setenv(EnvAuthDriver, "RSA")
	t.Setenv(EnvUsername, "user")
	t.Setenv(EnvRSAPrivateKey, string(pemData))
	t.Setenv(EnvRSAPrivateKeyPath, "")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := driver.(*RSASignatureDriver); !ok {
		t.Errorf("expected *RSASignatureDriver for AUTH_DRIVER=RSA, got %T", driver)
	}
}

// ---------- Integration: RSA driver round-trip ----------

func TestRSADriver_RoundTrip_VerifySignature(t *testing.T) {
	key := generateTestRSAKey(t)
	pemData := marshalPKCS1PEM(key)
	keyPath := writeTempKeyFile(t, pemData)

	t.Setenv(EnvAuthDriver, "rsa")
	t.Setenv(EnvUsername, "roundtrip-user")
	t.Setenv(EnvRSAPrivateKey, "")
	t.Setenv(EnvRSAPrivateKeyPath, keyPath)
	t.Setenv(EnvCredentialsPath, "")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("NewDriverFromEnv: %v", err)
	}

	params, err := driver.PrepareAuth()
	if err != nil {
		t.Fatalf("PrepareAuth: %v", err)
	}

	// Verify signature with public key.
	sigBytes, err := base64.StdEncoding.DecodeString(params["sig"])
	if err != nil {
		t.Fatalf("decode sig: %v", err)
	}
	hashed := sha256.Sum256([]byte("roundtrip-user"))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, hashed[:], sigBytes); err != nil {
		t.Errorf("round-trip signature verification failed: %v", err)
	}
}

// ---------- File-based credential reading (K8s secret volume mount) ----------

// writeSecretDir creates a temp directory simulating a K8s secret volume mount.
func writeSecretDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatalf("write secret file %s: %v", name, err)
		}
	}
	return dir
}

func TestNewDriverFromEnv_FileBasedToken(t *testing.T) {
	secretDir := writeSecretDir(t, map[string]string{
		"username": "fileuser\n",
		"password": "filepass\n",
	})

	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvCredentialsPath, secretDir)
	t.Setenv(EnvUsername, "")
	t.Setenv(EnvPassword, "")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	td, ok := driver.(*TokenDriver)
	if !ok {
		t.Fatalf("expected *TokenDriver, got %T", driver)
	}
	if td.Creds.Username != "fileuser" {
		t.Errorf("expected 'fileuser', got %q", td.Creds.Username)
	}
	if td.Creds.Password != "filepass" {
		t.Errorf("expected 'filepass', got %q", td.Creds.Password)
	}
}

func TestNewDriverFromEnv_FileOverridesEnv(t *testing.T) {
	secretDir := writeSecretDir(t, map[string]string{
		"username": "file-wins",
		"password": "file-wins-pass",
	})

	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvCredentialsPath, secretDir)
	t.Setenv(EnvUsername, "env-loses")
	t.Setenv(EnvPassword, "env-loses-pass")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params, _ := driver.PrepareAuth()
	if params["username"] != "file-wins" {
		t.Errorf("file should override env, got %q", params["username"])
	}
}

func TestNewDriverFromEnv_FileFallbackToEnv(t *testing.T) {
	// Empty secret dir — should fall back to env vars.
	emptyDir := t.TempDir()

	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvCredentialsPath, emptyDir)
	t.Setenv(EnvUsername, "envuser")
	t.Setenv(EnvPassword, "envpass")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params, _ := driver.PrepareAuth()
	if params["username"] != "envuser" {
		t.Errorf("should fall back to env, got %q", params["username"])
	}
}

func TestNewDriverFromEnv_RSAFromSecretMount(t *testing.T) {
	key := generateTestRSAKey(t)
	pemData := marshalPKCS1PEM(key)

	secretDir := writeSecretDir(t, map[string]string{
		"username": "rsa-file-user",
		"rsa-key":  string(pemData),
	})

	t.Setenv(EnvAuthDriver, "rsa")
	t.Setenv(EnvCredentialsPath, secretDir)
	t.Setenv(EnvUsername, "")
	t.Setenv(EnvRSAPrivateKey, "")
	t.Setenv(EnvRSAPrivateKeyPath, "")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rd, ok := driver.(*RSASignatureDriver)
	if !ok {
		t.Fatalf("expected *RSASignatureDriver, got %T", driver)
	}
	if rd.Username != "rsa-file-user" {
		t.Errorf("expected 'rsa-file-user', got %q", rd.Username)
	}
	// Verify signature works.
	params, err := driver.PrepareAuth()
	if err != nil {
		t.Fatalf("PrepareAuth: %v", err)
	}
	if params["sig"] == "" {
		t.Error("expected non-empty sig")
	}
}

func TestReloadableDriver_Reload_PicksUpFileChanges(t *testing.T) {
	// Simulates K8s secret rotation: write initial creds, reload, then write new creds, reload again.
	secretDir := writeSecretDir(t, map[string]string{
		"username": "initial-user",
		"password": "initial-pass",
	})

	t.Setenv(EnvAuthDriver, "token")
	t.Setenv(EnvCredentialsPath, secretDir)
	t.Setenv(EnvUsername, "")
	t.Setenv(EnvPassword, "")

	driver, err := NewDriverFromEnv()
	if err != nil {
		t.Fatalf("initial load: %v", err)
	}

	rd := NewReloadableDriver(driver, time.Hour)

	// Verify initial creds.
	params, _ := rd.PrepareAuth()
	if params["username"] != "initial-user" {
		t.Fatalf("expected 'initial-user', got %q", params["username"])
	}

	// Simulate K8s secret update — overwrite files.
	if err := os.WriteFile(filepath.Join(secretDir, "username"), []byte("rotated-user"), 0600); err != nil {
		t.Fatalf("write rotated username: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, "password"), []byte("rotated-pass"), 0600); err != nil {
		t.Fatalf("write rotated password: %v", err)
	}

	// Reload should pick up new files.
	if err := rd.Reload(); err != nil {
		t.Fatalf("reload after file change: %v", err)
	}

	params, _ = rd.PrepareAuth()
	if params["username"] != "rotated-user" {
		t.Errorf("expected 'rotated-user' after rotation, got %q", params["username"])
	}
	if params["password"] != "rotated-pass" {
		t.Errorf("expected 'rotated-pass' after rotation, got %q", params["password"])
	}
}
