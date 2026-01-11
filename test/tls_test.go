package aggregator_test

import (
	"reflect"
	"testing"

	"traefik-config-middleware/pkg/aggregator"
)

func TestBuildTLSConfig_WithCertResolver(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		TLS: &aggregator.TLSConfig{
			CertResolver: "letsencrypt",
		},
	}
	rule := "Host(`example.com`)"

	result := aggregator.BuildTLSConfig(ds, rule, nil)

	if result["certResolver"] != "letsencrypt" {
		t.Errorf("expected certResolver 'letsencrypt', got '%v'", result["certResolver"])
	}
}

func TestBuildTLSConfig_PreserveExistingOptions(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		TLS: &aggregator.TLSConfig{
			CertResolver: "myresolver",
		},
	}
	rule := "Host(`example.com`)"
	existingTLS := map[string]interface{}{
		"options": "custom-tls-options",
	}

	result := aggregator.BuildTLSConfig(ds, rule, existingTLS)

	if result["options"] != "custom-tls-options" {
		t.Errorf("expected options 'custom-tls-options', got '%v'", result["options"])
	}
	if result["certResolver"] != "myresolver" {
		t.Errorf("expected certResolver 'myresolver', got '%v'", result["certResolver"])
	}
}

func TestBuildTLSConfig_ExtractDomains(t *testing.T) {
	ds := aggregator.DownstreamConfig{}
	rule := "Host(`example.com`)"

	result := aggregator.BuildTLSConfig(ds, rule, nil)

	domains, ok := result["domains"].([]aggregator.TLSDomain)
	if !ok {
		t.Fatal("expected domains to be []TLSDomain")
	}
	if len(domains) != 1 {
		t.Fatalf("expected 1 domain entry, got %d", len(domains))
	}
	if domains[0].Main != "example.com" {
		t.Errorf("expected main domain 'example.com', got '%s'", domains[0].Main)
	}
}

func TestBuildTLSConfig_MultipleDomains(t *testing.T) {
	ds := aggregator.DownstreamConfig{}
	rule := "Host(`main.example.com`) || Host(`alt1.example.com`) || Host(`alt2.example.com`)"

	result := aggregator.BuildTLSConfig(ds, rule, nil)

	domains, ok := result["domains"].([]aggregator.TLSDomain)
	if !ok {
		t.Fatal("expected domains to be []TLSDomain")
	}
	if len(domains) != 1 {
		t.Fatalf("expected 1 domain entry with SANs, got %d", len(domains))
	}
	if domains[0].Main != "main.example.com" {
		t.Errorf("expected main domain 'main.example.com', got '%s'", domains[0].Main)
	}
	if len(domains[0].Sans) != 2 {
		t.Fatalf("expected 2 SANs, got %d", len(domains[0].Sans))
	}
	expectedSans := []string{"alt1.example.com", "alt2.example.com"}
	if !reflect.DeepEqual(domains[0].Sans, expectedSans) {
		t.Errorf("expected SANs %v, got %v", expectedSans, domains[0].Sans)
	}
}

func TestBuildTLSConfig_NoTLS(t *testing.T) {
	ds := aggregator.DownstreamConfig{}
	rule := "PathPrefix(`/api`)" // No Host() pattern

	result := aggregator.BuildTLSConfig(ds, rule, nil)

	// Should be empty when no TLS config and no domains
	if len(result) != 0 {
		t.Errorf("expected empty TLS config, got %v", result)
	}
}

func TestBuildTLSConfig_WildcardDomain(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		WildcardFix: true,
	}
	rule := "HostRegexp(`^[a-zA-Z0-9-]+\\.pages\\.example\\.com$`)"

	result := aggregator.BuildTLSConfig(ds, rule, nil)

	domains, ok := result["domains"].([]aggregator.TLSDomain)
	if !ok {
		t.Fatal("expected domains to be []TLSDomain")
	}
	if len(domains) != 1 {
		t.Fatalf("expected 1 domain entry, got %d", len(domains))
	}
	if domains[0].Main != "*.pages.example.com" {
		t.Errorf("expected main domain '*.pages.example.com', got '%s'", domains[0].Main)
	}
}

func TestBuildTLSConfig_OverwriteExistingDomains(t *testing.T) {
	ds := aggregator.DownstreamConfig{}
	rule := "Host(`new.example.com`)"
	existingTLS := map[string]interface{}{
		"domains": []aggregator.TLSDomain{{Main: "old.example.com"}},
		"options": "default",
	}

	result := aggregator.BuildTLSConfig(ds, rule, existingTLS)

	// Domains should be rebuilt, not preserved
	domains, ok := result["domains"].([]aggregator.TLSDomain)
	if !ok {
		t.Fatal("expected domains to be []TLSDomain")
	}
	if domains[0].Main != "new.example.com" {
		t.Errorf("expected domains to be rebuilt with 'new.example.com', got '%s'", domains[0].Main)
	}
	// Options should still be preserved
	if result["options"] != "default" {
		t.Errorf("expected options to be preserved, got '%v'", result["options"])
	}
}

func TestBuildTLSConfig_CertResolverOverridesExisting(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		TLS: &aggregator.TLSConfig{
			CertResolver: "new-resolver",
		},
	}
	rule := "Host(`example.com`)"
	existingTLS := map[string]interface{}{
		"certResolver": "old-resolver",
	}

	result := aggregator.BuildTLSConfig(ds, rule, existingTLS)

	// certResolver from ds.TLS should override existing
	if result["certResolver"] != "new-resolver" {
		t.Errorf("expected certResolver 'new-resolver', got '%v'", result["certResolver"])
	}
}

func TestBuildTLSConfig_NilTLSConfig(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		TLS: nil,
	}
	rule := "Host(`example.com`)"

	result := aggregator.BuildTLSConfig(ds, rule, nil)

	// Should still extract domains even without TLS config
	domains, ok := result["domains"].([]aggregator.TLSDomain)
	if !ok {
		t.Fatal("expected domains to be []TLSDomain")
	}
	if domains[0].Main != "example.com" {
		t.Errorf("expected main domain 'example.com', got '%s'", domains[0].Main)
	}
	// Should not have certResolver
	if _, exists := result["certResolver"]; exists {
		t.Error("expected no certResolver when TLS config is nil")
	}
}

func TestBuildTLSConfig_HostWithMultipleWildcardSANs(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		WildcardFix: true,
		TLS: &aggregator.TLSConfig{
			CertResolver: "letsencrypt",
		},
	}
	// Static Host + multiple HostRegexp wildcards
	rule := "Host(`static.example.com`) || HostRegexp(`^[a-zA-Z0-9-]+\\.api\\.example\\.com$`) || HostRegexp(`^[a-zA-Z0-9-]+\\.cdn\\.example\\.com$`)"

	result := aggregator.BuildTLSConfig(ds, rule, nil)

	domains, ok := result["domains"].([]aggregator.TLSDomain)
	if !ok {
		t.Fatal("expected domains to be []TLSDomain")
	}
	if len(domains) != 1 {
		t.Fatalf("expected 1 domain entry with SANs, got %d", len(domains))
	}
	// Static host should be Main domain
	if domains[0].Main != "static.example.com" {
		t.Errorf("expected main domain 'static.example.com', got '%s'", domains[0].Main)
	}
	// Wildcard domains should be SANs
	if len(domains[0].Sans) != 2 {
		t.Fatalf("expected 2 SANs, got %d", len(domains[0].Sans))
	}
	expectedSans := []string{"*.api.example.com", "*.cdn.example.com"}
	if !reflect.DeepEqual(domains[0].Sans, expectedSans) {
		t.Errorf("expected SANs %v, got %v", expectedSans, domains[0].Sans)
	}
	// Verify certResolver is set
	if result["certResolver"] != "letsencrypt" {
		t.Errorf("expected certResolver 'letsencrypt', got '%v'", result["certResolver"])
	}
}

func TestBuildTLSConfig_MultipleWildcardsOnly(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		WildcardFix: true,
		TLS: &aggregator.TLSConfig{
			CertResolver: "letsencrypt",
		},
	}
	// Multiple HostRegexp wildcards without static Host
	rule := "HostRegexp(`^[a-zA-Z0-9-]+\\.api\\.example\\.com$`) || HostRegexp(`^[a-zA-Z0-9-]+\\.cdn\\.example\\.com$`) || HostRegexp(`^[a-zA-Z0-9-]+\\.pages\\.example\\.com$`)"

	result := aggregator.BuildTLSConfig(ds, rule, nil)

	domains, ok := result["domains"].([]aggregator.TLSDomain)
	if !ok {
		t.Fatal("expected domains to be []TLSDomain")
	}
	if len(domains) != 1 {
		t.Fatalf("expected 1 domain entry with SANs, got %d", len(domains))
	}
	// First wildcard becomes Main
	if domains[0].Main != "*.api.example.com" {
		t.Errorf("expected main domain '*.api.example.com', got '%s'", domains[0].Main)
	}
	// Remaining wildcards become SANs
	if len(domains[0].Sans) != 2 {
		t.Fatalf("expected 2 SANs, got %d", len(domains[0].Sans))
	}
	expectedSans := []string{"*.cdn.example.com", "*.pages.example.com"}
	if !reflect.DeepEqual(domains[0].Sans, expectedSans) {
		t.Errorf("expected SANs %v, got %v", expectedSans, domains[0].Sans)
	}
}

func TestBuildTLSConfig_StripResolver(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		TLS: &aggregator.TLSConfig{
			CertResolver:  "letsencrypt",
			StripResolver: true,
		},
	}
	rule := "Host(`example.com`)"

	result := aggregator.BuildTLSConfig(ds, rule, nil)

	// certResolver should be stripped
	if _, exists := result["certResolver"]; exists {
		t.Error("expected certResolver to be stripped")
	}
	// domains should still be present
	domains, ok := result["domains"].([]aggregator.TLSDomain)
	if !ok {
		t.Fatal("expected domains to be []TLSDomain")
	}
	if domains[0].Main != "example.com" {
		t.Errorf("expected main domain 'example.com', got '%s'", domains[0].Main)
	}
}

func TestBuildTLSConfig_StripResolverFromExisting(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		TLS: &aggregator.TLSConfig{
			StripResolver: true,
		},
	}
	rule := "Host(`example.com`)"
	existingTLS := map[string]interface{}{
		"certResolver": "existing-resolver",
		"options":      "default",
	}

	result := aggregator.BuildTLSConfig(ds, rule, existingTLS)

	// certResolver from existing TLS should be stripped
	if _, exists := result["certResolver"]; exists {
		t.Error("expected certResolver to be stripped from existing TLS")
	}
	// options should still be preserved
	if result["options"] != "default" {
		t.Errorf("expected options to be preserved, got '%v'", result["options"])
	}
}

func TestBuildTLSConfig_StripResolverFalse(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		TLS: &aggregator.TLSConfig{
			CertResolver:  "letsencrypt",
			StripResolver: false,
		},
	}
	rule := "Host(`example.com`)"

	result := aggregator.BuildTLSConfig(ds, rule, nil)

	// certResolver should NOT be stripped
	if result["certResolver"] != "letsencrypt" {
		t.Errorf("expected certResolver 'letsencrypt', got '%v'", result["certResolver"])
	}
}
