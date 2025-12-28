package aggregator

import (
	"regexp"
	"strings"
)

// ConvertRegexpToWildcard converts a HostRegexp pattern to a wildcard domain
// if it matches common wildcard prefix patterns like ^[a-zA-Z0-9-]+\.
func ConvertRegexpToWildcard(pattern string) string {
	wildcardPrefixes := []string{
		`^[a-zA-Z0-9-]+\.`,
		`^[a-zA-Z0-9_-]+\.`,
		`^[^.]+\.`,
		`^.+\.`,
		`^.*\.`,
	}

	for _, prefix := range wildcardPrefixes {
		if strings.HasPrefix(pattern, prefix) {
			remainder := strings.TrimPrefix(pattern, prefix)
			remainder = strings.TrimSuffix(remainder, "$")
			domain := strings.ReplaceAll(remainder, `\.`, ".")
			return "*." + domain
		}
	}

	return ""
}

// ExtractDomainsFromRule parses Host() and HostRegexp() patterns from a Traefik rule
// and returns a list of domains. HostRegexp patterns are only processed if wildcardFix is true.
func ExtractDomainsFromRule(rule string, wildcardFix bool) []string {
	var domains []string

	// Extract Host(`domain`) patterns
	hostRegex := regexp.MustCompile("Host\\(`([^`]+)`\\)")
	for _, match := range hostRegex.FindAllStringSubmatch(rule, -1) {
		if len(match) > 1 {
			domains = append(domains, match[1])
		}
	}

	// Extract HostRegexp() patterns (only if wildcardFix enabled)
	if wildcardFix {
		hostRegexpRegex := regexp.MustCompile("HostRegexp\\(`([^`]+)`\\)")
		for _, match := range hostRegexpRegex.FindAllStringSubmatch(rule, -1) {
			if len(match) > 1 {
				domain := ConvertRegexpToWildcard(match[1])
				if domain != "" {
					domains = append(domains, domain)
				}
			}
		}
	}

	return domains
}
