package aggregator

// BuildTLSConfig constructs a TLS configuration map with domain extraction.
// It merges existing TLS options with certResolver from config and extracted domains.
func BuildTLSConfig(ds DownstreamConfig, rule string, existingTLS map[string]interface{}) map[string]interface{} {
	tlsConfig := make(map[string]interface{})

	// Preserve existing TLS options (e.g., "options": "default")
	if existingTLS != nil {
		for k, v := range existingTLS {
			if k != "domains" { // We'll rebuild domains
				tlsConfig[k] = v
			}
		}
	}

	// Add/override certResolver from downstream config
	if ds.TLS != nil && ds.TLS.CertResolver != "" {
		tlsConfig["certResolver"] = ds.TLS.CertResolver
	}

	// Extract and add domains from rule
	domains := ExtractDomainsFromRule(rule, ds.WildcardFix)
	if len(domains) > 0 {
		tlsDomain := TLSDomain{Main: domains[0]}
		if len(domains) > 1 {
			tlsDomain.Sans = domains[1:]
		}
		tlsConfig["domains"] = []TLSDomain{tlsDomain}
	}

	return tlsConfig
}
