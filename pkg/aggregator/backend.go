package aggregator

import (
	"strings"
)

// GetBackendURL determines the backend URL for a downstream configuration.
// If BackendOverride is set, it's used (with protocol added if missing).
// Otherwise, the URL is derived from APIURL with appropriate protocol and port.
func GetBackendURL(ds DownstreamConfig, useTLS bool) string {
	var protocol string
	var defaultPort string

	if useTLS {
		protocol = "https://"
		defaultPort = ":443"
	} else {
		protocol = "http://"
		defaultPort = ":80"
	}

	if ds.BackendOverride != "" {
		// If override contains protocol, use it as-is
		if strings.HasPrefix(ds.BackendOverride, "http://") || strings.HasPrefix(ds.BackendOverride, "https://") {
			return ds.BackendOverride
		}
		// Otherwise, add the protocol
		return protocol + ds.BackendOverride
	}

	// Extract host:port from api_url
	apiURL := ds.APIURL
	apiURL = strings.TrimPrefix(apiURL, "http://")
	apiURL = strings.TrimPrefix(apiURL, "https://")

	// Remove path if present
	if idx := strings.Index(apiURL, "/"); idx != -1 {
		apiURL = apiURL[:idx]
	}

	// Add default port if not specified, otherwise preserve existing port
	if !strings.Contains(apiURL, ":") {
		apiURL = apiURL + defaultPort
	}

	return protocol + apiURL
}
