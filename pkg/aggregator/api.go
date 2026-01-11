package aggregator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const maxErrorBodyLen = 256

// FetchDownstreamRouters fetches router configurations from a downstream Traefik API.
func FetchDownstreamRouters(ds DownstreamConfig, client *http.Client) ([]TraefikRouter, error) {
	apiEndpoint, err := url.JoinPath(ds.APIURL, "/api/http/routers")
	if err != nil {
		return nil, fmt.Errorf("invalid API URL: %w", err)
	}

	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, err
	}

	if ds.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+ds.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if len(bodyStr) > maxErrorBodyLen {
			bodyStr = bodyStr[:maxErrorBodyLen] + "...(truncated)"
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, bodyStr)
	}

	// Traefik API returns an array, not a map
	var routersArray []TraefikRouter
	if err := json.NewDecoder(resp.Body).Decode(&routersArray); err != nil {
		return nil, err
	}

	return routersArray, nil
}

// FetchPassthroughConfig fetches a full HTTPProxyConfig from a passthrough downstream.
// Unlike FetchDownstreamRouters, this fetches directly from the api_url without appending a path.
func FetchPassthroughConfig(ds DownstreamConfig, client *http.Client) (*HTTPProxyConfig, error) {
	req, err := http.NewRequest("GET", ds.APIURL, nil)
	if err != nil {
		return nil, err
	}

	if ds.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+ds.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if len(bodyStr) > maxErrorBodyLen {
			bodyStr = bodyStr[:maxErrorBodyLen] + "...(truncated)"
		}
		return nil, fmt.Errorf("passthrough API returned status %d: %s", resp.StatusCode, bodyStr)
	}

	var config HTTPProxyConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode passthrough config: %w", err)
	}

	return &config, nil
}
