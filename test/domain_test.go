package aggregator_test

import (
	"reflect"
	"testing"

	"traefik-config-middleware/pkg/aggregator"
)

func TestExtractDomainsFromRule_SingleHost(t *testing.T) {
	rule := "Host(`example.com`) && PathPrefix(`/`)"
	domains := aggregator.ExtractDomainsFromRule(rule, false)

	expected := []string{"example.com"}
	if !reflect.DeepEqual(domains, expected) {
		t.Errorf("expected %v, got %v", expected, domains)
	}
}

func TestExtractDomainsFromRule_MultipleHosts(t *testing.T) {
	rule := "Host(`host1.example.com`) || Host(`host2.example.com`)"
	domains := aggregator.ExtractDomainsFromRule(rule, false)

	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0] != "host1.example.com" {
		t.Errorf("expected first domain 'host1.example.com', got '%s'", domains[0])
	}
	if domains[1] != "host2.example.com" {
		t.Errorf("expected second domain 'host2.example.com', got '%s'", domains[1])
	}
}

func TestExtractDomainsFromRule_HostRegexp_WildcardFixEnabled(t *testing.T) {
	rule := "HostRegexp(`^[a-zA-Z0-9-]+\\.pages\\.example\\.com$`) && PathPrefix(`/`)"
	domains := aggregator.ExtractDomainsFromRule(rule, true)

	expected := []string{"*.pages.example.com"}
	if !reflect.DeepEqual(domains, expected) {
		t.Errorf("expected %v, got %v", expected, domains)
	}
}

func TestExtractDomainsFromRule_HostRegexp_WildcardFixDisabled(t *testing.T) {
	rule := "HostRegexp(`^[a-zA-Z0-9-]+\\.pages\\.example\\.com$`) && PathPrefix(`/`)"
	domains := aggregator.ExtractDomainsFromRule(rule, false)

	// Should return empty slice when wildcardFix is false
	if len(domains) != 0 {
		t.Errorf("expected empty domains when wildcardFix is false, got %v", domains)
	}
}

func TestExtractDomainsFromRule_ComplexRule(t *testing.T) {
	rule := "Host(`api.example.com`) && PathPrefix(`/v1`) || Host(`web.example.com`) && PathPrefix(`/`)"
	domains := aggregator.ExtractDomainsFromRule(rule, false)

	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0] != "api.example.com" {
		t.Errorf("expected 'api.example.com', got '%s'", domains[0])
	}
	if domains[1] != "web.example.com" {
		t.Errorf("expected 'web.example.com', got '%s'", domains[1])
	}
}

func TestExtractDomainsFromRule_MixedHostAndHostRegexp(t *testing.T) {
	rule := "Host(`static.example.com`) || HostRegexp(`^[a-zA-Z0-9-]+\\.cdn\\.example\\.com$`)"
	domains := aggregator.ExtractDomainsFromRule(rule, true)

	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0] != "static.example.com" {
		t.Errorf("expected 'static.example.com', got '%s'", domains[0])
	}
	if domains[1] != "*.cdn.example.com" {
		t.Errorf("expected '*.cdn.example.com', got '%s'", domains[1])
	}
}

func TestExtractDomainsFromRule_NoDomains(t *testing.T) {
	rule := "PathPrefix(`/api`)"
	domains := aggregator.ExtractDomainsFromRule(rule, false)

	if len(domains) != 0 {
		t.Errorf("expected no domains, got %v", domains)
	}
}

func TestConvertRegexpToWildcard_Pattern1(t *testing.T) {
	// Pattern: ^[a-zA-Z0-9-]+\.
	pattern := `^[a-zA-Z0-9-]+\.example\.com$`
	result := aggregator.ConvertRegexpToWildcard(pattern)

	expected := "*.example.com"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestConvertRegexpToWildcard_Pattern2(t *testing.T) {
	// Pattern: ^[a-zA-Z0-9_-]+\.
	pattern := `^[a-zA-Z0-9_-]+\.subdomain\.example\.com$`
	result := aggregator.ConvertRegexpToWildcard(pattern)

	expected := "*.subdomain.example.com"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestConvertRegexpToWildcard_Pattern3(t *testing.T) {
	// Pattern: ^[^.]+\.
	pattern := `^[^.]+\.wildcard\.example\.com$`
	result := aggregator.ConvertRegexpToWildcard(pattern)

	expected := "*.wildcard.example.com"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestConvertRegexpToWildcard_Pattern4(t *testing.T) {
	// Pattern: ^.+\.
	pattern := `^.+\.any\.example\.com$`
	result := aggregator.ConvertRegexpToWildcard(pattern)

	expected := "*.any.example.com"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestConvertRegexpToWildcard_Pattern5(t *testing.T) {
	// Pattern: ^.*\.
	pattern := `^.*\.star\.example\.com$`
	result := aggregator.ConvertRegexpToWildcard(pattern)

	expected := "*.star.example.com"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestConvertRegexpToWildcard_NoMatch(t *testing.T) {
	// Pattern that doesn't match any wildcard prefix
	pattern := `example\.com$`
	result := aggregator.ConvertRegexpToWildcard(pattern)

	if result != "" {
		t.Errorf("expected empty string for non-matching pattern, got '%s'", result)
	}
}

func TestConvertRegexpToWildcard_ComplexDomain(t *testing.T) {
	pattern := `^[a-zA-Z0-9-]+\.pages\.gitlab\.example\.com$`
	result := aggregator.ConvertRegexpToWildcard(pattern)

	expected := "*.pages.gitlab.example.com"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestConvertRegexpToWildcard_WithoutDollar(t *testing.T) {
	// Pattern without trailing $
	pattern := `^[a-zA-Z0-9-]+\.example\.com`
	result := aggregator.ConvertRegexpToWildcard(pattern)

	expected := "*.example.com"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestExtractDomainsFromRule_HostAndMultipleHostRegexp(t *testing.T) {
	// Test case: static Host + multiple HostRegexp patterns
	rule := "Host(`static.example.com`) || HostRegexp(`^[a-zA-Z0-9-]+\\.api\\.example\\.com$`) || HostRegexp(`^[a-zA-Z0-9-]+\\.cdn\\.example\\.com$`)"
	domains := aggregator.ExtractDomainsFromRule(rule, true)

	if len(domains) != 3 {
		t.Fatalf("expected 3 domains, got %d: %v", len(domains), domains)
	}
	if domains[0] != "static.example.com" {
		t.Errorf("expected first domain 'static.example.com', got '%s'", domains[0])
	}
	if domains[1] != "*.api.example.com" {
		t.Errorf("expected second domain '*.api.example.com', got '%s'", domains[1])
	}
	if domains[2] != "*.cdn.example.com" {
		t.Errorf("expected third domain '*.cdn.example.com', got '%s'", domains[2])
	}
}

func TestExtractDomainsFromRule_MultipleHostRegexpOnly(t *testing.T) {
	// Test case: multiple HostRegexp patterns without static Host
	rule := "HostRegexp(`^[a-zA-Z0-9-]+\\.api\\.example\\.com$`) || HostRegexp(`^[a-zA-Z0-9-]+\\.cdn\\.example\\.com$`) || HostRegexp(`^[a-zA-Z0-9-]+\\.pages\\.example\\.com$`)"
	domains := aggregator.ExtractDomainsFromRule(rule, true)

	if len(domains) != 3 {
		t.Fatalf("expected 3 domains, got %d: %v", len(domains), domains)
	}
	if domains[0] != "*.api.example.com" {
		t.Errorf("expected first domain '*.api.example.com', got '%s'", domains[0])
	}
	if domains[1] != "*.cdn.example.com" {
		t.Errorf("expected second domain '*.cdn.example.com', got '%s'", domains[1])
	}
	if domains[2] != "*.pages.example.com" {
		t.Errorf("expected third domain '*.pages.example.com', got '%s'", domains[2])
	}
}
