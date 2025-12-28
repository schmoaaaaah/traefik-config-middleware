package aggregator_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"traefik-config-middleware/pkg/aggregator"
)

// createTestAggregator creates an aggregator with the given config for testing
func createTestAggregator(cfg *aggregator.Config) *aggregator.Aggregator {
	client := &http.Client{}
	return aggregator.NewAggregator(cfg, client)
}

func TestGetCachedConfig_ReturnsJSON(t *testing.T) {
	// Create a mock downstream server
	routers := []aggregator.TraefikRouter{
		{
			Name:        "test-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "test-service",
			Rule:        "Host(`example.com`)",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/http/routers" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(routers)
	}))
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:   "test-downstream",
				APIURL: server.URL,
			},
		},
	}

	agg := createTestAggregator(cfg)
	agg.AggregateConfigs()

	result := agg.GetCachedConfig()

	if len(result.HTTP.Routers) != 1 {
		t.Errorf("expected 1 router, got %d", len(result.HTTP.Routers))
	}
	if len(result.HTTP.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(result.HTTP.Services))
	}
}

func TestGetCachedConfig_EmptyConfig(t *testing.T) {
	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{},
	}

	agg := createTestAggregator(cfg)
	agg.AggregateConfigs()

	result := agg.GetCachedConfig()

	if len(result.HTTP.Routers) != 0 {
		t.Errorf("expected 0 routers, got %d", len(result.HTTP.Routers))
	}
	if len(result.HTTP.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(result.HTTP.Services))
	}
}

func TestGetCachedConfig_WithTLS(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "secure-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "secure-service",
			Rule:        "Host(`secure.example.com`)",
			TLS:         map[string]interface{}{"options": "default"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/http/routers" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(routers)
	}))
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:   "test-downstream",
				APIURL: server.URL,
				TLS: &aggregator.TLSConfig{
					CertResolver: "letsencrypt",
				},
			},
		},
	}

	agg := createTestAggregator(cfg)
	agg.AggregateConfigs()

	result := agg.GetCachedConfig()

	router := result.HTTP.Routers["test-downstream-secure-router"]
	if router.TLS == nil {
		t.Error("expected TLS config to be present")
	}
}

func TestGetCachedConfig_WithMiddlewares(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "app-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "app-service",
			Rule:        "Host(`app.example.com`)",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/http/routers" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(routers)
	}))
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:        "test-downstream",
				APIURL:      server.URL,
				Middlewares: []string{"auth@file", "compress@file"},
			},
		},
	}

	agg := createTestAggregator(cfg)
	agg.AggregateConfigs()

	result := agg.GetCachedConfig()

	router := result.HTTP.Routers["test-downstream-app-router"]
	if len(router.Middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(router.Middlewares))
	}
}
