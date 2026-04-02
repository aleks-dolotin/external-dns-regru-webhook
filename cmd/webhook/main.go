// external-dns webhook provider for Reg.ru DNS API v2.
//
// Implements the standard external-dns webhook protocol:
//   GET  /           — negotiation, returns domain filter
//   GET  /records    — list current DNS records
//   POST /records    — apply changes (create/update/delete)
//   POST /adjustendpoints — adjust endpoints (no-op)
//
// Runs as a sidecar container alongside external-dns.
// Default port: 8888 (external-dns webhook default).
//
// Required environment variables:
//   REGU_USERNAME   — Reg.ru API username
//   REGU_PASSWORD   — Reg.ru API password
//   DOMAIN_FILTER   — comma-separated list of zones (e.g. "dolotin.ru")
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/auth"
	regprovider "github.com/aleks-dolotin/external-dns-regru-webhook/internal/provider"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider/webhook/api"
)

var Version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(Version)
		return
	}

	// Load credentials
	driver, err := auth.NewDriverFromEnv()
	if err != nil {
		log.Fatalf("auth: %v", err)
	}

	// Create Reg.ru adapter
	httpAdapter := adapter.NewHTTPAdapter(driver)

	// Parse domain filter
	domainFilterStr := os.Getenv("DOMAIN_FILTER")
	if domainFilterStr == "" {
		log.Fatal("DOMAIN_FILTER environment variable is required (e.g. \"dolotin.ru\")")
	}
	domains := strings.Split(domainFilterStr, ",")
	for i := range domains {
		domains[i] = strings.TrimSpace(domains[i])
	}
	domainFilter := endpoint.NewDomainFilter(domains)

	// Create provider
	p := regprovider.NewRegrProvider(httpAdapter, *domainFilter)

	// Start webhook HTTP server on port 8888 (external-dns default)
	port := os.Getenv("WEBHOOK_PORT")
	if port == "" {
		port = "8888"
	}

	log.Printf("Starting external-dns webhook provider for Reg.ru (version=%s, port=%s, domains=%v)", Version, port, domains)

	api.StartHTTPApi(
		p,
		nil, // no started signal needed
		5_000_000_000,  // readTimeout: 5s
		10_000_000_000, // writeTimeout: 10s
		":"+port,
	)
}
