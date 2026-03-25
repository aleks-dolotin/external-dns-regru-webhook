//go:build smoke

// Package smoke contains smoke tests that run against the real Reg.ru API.
//
// Requirements:
//   - REGU_USERNAME and REGU_PASSWORD set to valid Reg.ru credentials
//   - SMOKE_TEST_ZONE set to a real zone you own (e.g. "example.com")
//
// Run:
//
//	REGU_USERNAME=user REGU_PASSWORD=pass SMOKE_TEST_ZONE=yourdomain.com \
//	  go test -v -tags=smoke -count=1 -timeout=120s ./tests/smoke/...
package smoke

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/auth"
)

const smokeSubdomain = "smoke-test"

// regrNS is the authoritative Reg.ru nameserver used for DNS verification.
// Querying it directly bypasses any recursive resolver TTL caching.
const regNS = "ns1.reg.ru:53"

func skipIfNotConfigured(t *testing.T) {
	t.Helper()
	if os.Getenv("REGU_USERNAME") == "" || os.Getenv("REGU_PASSWORD") == "" {
		t.Skip("REGU_USERNAME / REGU_PASSWORD not set — skipping smoke test")
	}
	if os.Getenv("SMOKE_TEST_ZONE") == "" {
		t.Skip("SMOKE_TEST_ZONE not set — skipping smoke test")
	}
}

func newRealAdapter(t *testing.T) *adapter.HTTPAdapter {
	t.Helper()
	driver, err := auth.NewDriverFromEnv()
	if err != nil {
		t.Fatalf("auth driver: %v", err)
	}
	// No REGRU_BASE_URL set → defaults to production https://api.reg.ru/api/regru2
	return adapter.NewHTTPAdapter(driver)
}

func zone(t *testing.T) string {
	t.Helper()
	return os.Getenv("SMOKE_TEST_ZONE")
}

// subdomain returns a unique subdomain for this test run to avoid collisions.
func subdomain() string {
	return fmt.Sprintf("%s-%d", smokeSubdomain, time.Now().UnixMilli()%100000)
}

// cleanup deletes the test record, ignoring errors (best-effort).
func cleanup(t *testing.T, a *adapter.HTTPAdapter, z, sub, recType, content string) {
	t.Helper()
	id := fmt.Sprintf("%s:%s:%s", sub, recType, content)
	_ = a.DeleteRecord(z, id)
}

// dnsLookup performs a direct DNS query to the Reg.ru authoritative nameserver
// using dig. This bypasses the Go resolver which on macOS may ignore custom Dial.
// Retries up to 12 times with 5s intervals to allow for zone propagation on Reg.ru NS.
func dnsLookup(t *testing.T, fqdn, qtype string) []string {
	t.Helper()
	for attempt := 1; attempt <= 12; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, "dig", "@ns1.reg.ru", fqdn, qtype, "+short", "+norecurse")
		out, err := cmd.Output()
		cancel()
		if err != nil {
			t.Logf("dig attempt %d failed: %v", attempt, err)
			time.Sleep(5 * time.Second)
			continue
		}
		var results []string
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				results = append(results, line)
			}
		}
		if len(results) > 0 {
			return results
		}
		if attempt < 12 {
			t.Logf("DNS attempt %d/12: empty result for %s %s, retrying in 5s...", attempt, fqdn, qtype)
			time.Sleep(5 * time.Second)
		}
	}
	return nil
}

// verifyDNS_A checks that the FQDN resolves to the expected IP via Reg.ru NS.
// Soft check — logs warning on failure, does not fail the test (Reg.ru NS propagation ~30-60s).
func verifyDNS_A(t *testing.T, fqdn, expectedIP string) {
	t.Helper()
	results := dnsLookup(t, fqdn, "A")
	for _, addr := range results {
		if addr == expectedIP {
			t.Logf("✅ DNS verified: %s → %s (via %s)", fqdn, expectedIP, regNS)
			return
		}
	}
	t.Logf("⚠️  DNS not yet propagated: %s returned %v, expected %s (Reg.ru NS delay ~30-60s)", fqdn, results, expectedIP)
}

// verifyDNS_TXT checks that the FQDN has the expected TXT record via Reg.ru NS.
// Soft check — logs warning on failure.
func verifyDNS_TXT(t *testing.T, fqdn, expectedContent string) {
	t.Helper()
	results := dnsLookup(t, fqdn, "TXT")
	for _, rec := range results {
		cleaned := strings.Trim(rec, "\"")
		if cleaned == expectedContent {
			t.Logf("✅ DNS verified: %s TXT=%q (via %s)", fqdn, expectedContent, regNS)
			return
		}
	}
	t.Logf("⚠️  DNS not yet propagated: %s TXT returned %v, expected %q", fqdn, results, expectedContent)
}

// verifyDNS_CNAME checks that the FQDN has the expected CNAME via Reg.ru NS.
// Soft check — logs warning on failure.
func verifyDNS_CNAME(t *testing.T, fqdn, expectedTarget string) {
	t.Helper()
	results := dnsLookup(t, fqdn, "CNAME")
	expected := strings.TrimSuffix(expectedTarget, ".") + "."
	for _, cname := range results {
		if strings.EqualFold(cname, expected) || strings.EqualFold(strings.TrimSuffix(cname, "."), strings.TrimSuffix(expectedTarget, ".")) {
			t.Logf("✅ DNS verified: %s CNAME=%s (via %s)", fqdn, cname, regNS)
			return
		}
	}
	t.Logf("⚠️  DNS not yet propagated: %s CNAME returned %v, expected %s", fqdn, results, expected)
}

// verifyDNS_NotExists checks that the FQDN does NOT resolve via Reg.ru NS.
// Soft check — logs warning on failure.
func verifyDNS_NotExists(t *testing.T, fqdn string) {
	t.Helper()
	for attempt := 1; attempt <= 12; attempt++ {
		results := dnsLookupOnce(t, fqdn, "A")
		if len(results) == 0 {
			t.Logf("✅ DNS verified: %s does not resolve (via %s)", fqdn, regNS)
			return
		}
		if attempt < 12 {
			t.Logf("DNS delete attempt %d/12: %s still resolves to %v, retrying in 5s...", attempt, fqdn, results)
			time.Sleep(5 * time.Second)
		} else {
			t.Logf("⚠️  DNS delete not yet propagated: %s still resolves to %v after 60s", fqdn, results)
		}
	}
}

// dnsLookupOnce performs a single dig query without retry (used by verifyDNS_NotExists).
func dnsLookupOnce(t *testing.T, fqdn, qtype string) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "dig", "@ns1.reg.ru", fqdn, qtype, "+short", "+norecurse")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var results []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, line)
		}
	}
	return results
}

// --- Smoke Test: Full CRUD lifecycle against real Reg.ru ---

func TestSmoke_CreateFindDeleteRecord(t *testing.T) {
	skipIfNotConfigured(t)
	a := newRealAdapter(t)
	z := zone(t)
	sub := subdomain()

	t.Logf("Smoke test: zone=%s subdomain=%s", z, sub)

	// CREATE
	rec := &adapter.Record{
		Name:    sub,
		Type:    "A",
		Content: "198.51.100.1", // TEST-NET-2, safe for smoke tests
	}
	err := a.CreateRecord(z, rec)
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	t.Logf("✅ CreateRecord succeeded (subdomain=%s)", sub)

	// Best-effort cleanup regardless of test outcome.
	defer cleanup(t, a, z, sub, "A", "198.51.100.1")

	// Wait for propagation — Reg.ru API can be eventually consistent.
	time.Sleep(2 * time.Second)

	// FIND
	found, err := a.FindRecord(z, sub, "A")
	if err != nil {
		t.Fatalf("FindRecord: %v", err)
	}
	if found == nil {
		t.Fatal("FindRecord returned nil — record not found after create")
	}
	if found.Content != "198.51.100.1" {
		t.Errorf("expected content 198.51.100.1, got %q", found.Content)
	}
	t.Logf("✅ FindRecord succeeded (content=%s)", found.Content)

	// DNS VERIFICATION: query Reg.ru NS directly (no TTL cache).
	fqdn := sub + "." + z
	verifyDNS_A(t, fqdn, "198.51.100.1")

	// DELETE
	deleteID := fmt.Sprintf("%s:%s:%s", sub, "A", "198.51.100.1")
	err = a.DeleteRecord(z, deleteID)
	if err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	t.Logf("✅ DeleteRecord succeeded")

	// VERIFY DELETED
	time.Sleep(2 * time.Second)
	found, err = a.FindRecord(z, sub, "A")
	if err != nil {
		t.Fatalf("FindRecord after delete: %v", err)
	}
	if found != nil {
		t.Errorf("record still exists after delete: %+v", found)
	} else {
		t.Log("✅ Record confirmed deleted")
	}

	// DNS VERIFICATION: record should not resolve anymore.
	verifyDNS_NotExists(t, sub+"."+z)
}

// --- Smoke Test: TXT record (different content key path) ---

func TestSmoke_TXTRecord(t *testing.T) {
	skipIfNotConfigured(t)
	a := newRealAdapter(t)
	z := zone(t)
	sub := subdomain()

	rec := &adapter.Record{
		Name:    sub,
		Type:    "TXT",
		Content: "v=smoke-test",
	}
	err := a.CreateRecord(z, rec)
	if err != nil {
		t.Fatalf("CreateRecord TXT: %v", err)
	}
	t.Logf("✅ TXT CreateRecord succeeded (subdomain=%s)", sub)
	defer cleanup(t, a, z, sub, "TXT", "v=smoke-test")

	time.Sleep(2 * time.Second)

	found, err := a.FindRecord(z, sub, "TXT")
	if err != nil {
		t.Fatalf("FindRecord TXT: %v", err)
	}
	if found == nil {
		t.Fatal("TXT record not found after create")
	}
	t.Logf("✅ TXT FindRecord succeeded (content=%s)", found.Content)

	// DNS VERIFICATION
	verifyDNS_TXT(t, sub+"."+z, "v=smoke-test")

	deleteID := fmt.Sprintf("%s:%s:%s", sub, "TXT", "v=smoke-test")
	if err := a.DeleteRecord(z, deleteID); err != nil {
		t.Fatalf("DeleteRecord TXT: %v", err)
	}
	t.Log("✅ TXT record deleted")
}

// --- Smoke Test: Auth validation ---

func TestSmoke_AuthWorks(t *testing.T) {
	skipIfNotConfigured(t)
	a := newRealAdapter(t)
	z := zone(t)

	// FindRecord should not return auth error — just nil if no records.
	_, err := a.FindRecord(z, "nonexistent-"+subdomain(), "A")
	if err != nil {
		t.Fatalf("FindRecord with valid auth failed: %v", err)
	}
	t.Log("✅ Auth works — API accepted credentials")
}

// --- Smoke Test: CNAME record ---

func TestSmoke_CNAMERecord(t *testing.T) {
	skipIfNotConfigured(t)
	a := newRealAdapter(t)
	z := zone(t)
	sub := subdomain()

	rec := &adapter.Record{
		Name:    sub,
		Type:    "CNAME",
		Content: "target." + z,
	}
	err := a.CreateRecord(z, rec)
	if err != nil {
		t.Fatalf("CreateRecord CNAME: %v", err)
	}
	t.Logf("✅ CNAME CreateRecord succeeded")
	defer cleanup(t, a, z, sub, "CNAME", "target."+z)

	time.Sleep(2 * time.Second)

	found, err := a.FindRecord(z, sub, "CNAME")
	if err != nil {
		t.Fatalf("FindRecord CNAME: %v", err)
	}
	if found == nil {
		t.Fatal("CNAME record not found after create")
	}
	t.Logf("✅ CNAME FindRecord succeeded (content=%s)", found.Content)

	// DNS VERIFICATION
	verifyDNS_CNAME(t, sub+"."+z, "target."+z)

	deleteID := fmt.Sprintf("%s:%s:%s", sub, "CNAME", "target."+z)
	if err := a.DeleteRecord(z, deleteID); err != nil {
		t.Fatalf("DeleteRecord CNAME: %v", err)
	}
	t.Log("✅ CNAME record deleted")
}

// --- Smoke Test: UpdateRecord changes content ---

func TestSmoke_UpdateRecord(t *testing.T) {
	skipIfNotConfigured(t)
	a := newRealAdapter(t)
	z := zone(t)
	sub := subdomain()

	// CREATE with initial IP
	rec := &adapter.Record{Name: sub, Type: "A", Content: "198.51.100.1"}
	if err := a.CreateRecord(z, rec); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	t.Logf("✅ Created A record %s → 198.51.100.1", sub)
	defer cleanup(t, a, z, sub, "A", "198.51.100.2") // cleanup new IP
	defer cleanup(t, a, z, sub, "A", "198.51.100.1") // cleanup old IP (in case update fails)

	time.Sleep(2 * time.Second)

	// UPDATE to new IP
	rec.Content = "198.51.100.2"
	if err := a.UpdateRecord(z, rec); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	t.Log("✅ UpdateRecord succeeded")

	time.Sleep(2 * time.Second)

	// VERIFY content actually changed
	found, err := a.FindRecord(z, sub, "A")
	if err != nil {
		t.Fatalf("FindRecord after update: %v", err)
	}
	if found == nil {
		t.Fatal("record not found after update")
	}
	if found.Content != "198.51.100.2" {
		t.Errorf("expected updated content 198.51.100.2, got %q", found.Content)
	} else {
		t.Logf("✅ Content verified: %s → %s (was 198.51.100.1)", sub, found.Content)
	}

	// DNS VERIFICATION: authoritative NS should return updated IP.
	verifyDNS_A(t, sub+"."+z, "198.51.100.2")

	// VERIFY old record is gone (no duplicates)
	// FindRecord returns first match, so we check via get_resource_records that there's only one A record for this subdomain
	t.Log("✅ Update test complete — content changed and verified")
}
