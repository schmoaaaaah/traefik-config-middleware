package aggregator_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"traefik-config-middleware/pkg/aggregator"
)

// Helper to create a mock passthrough server that returns HTTPProxyConfig
func createMockPassthroughServer(t *testing.T, config aggregator.HTTPProxyConfig) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	}))
}

func TestFetchPassthroughConfig_Success(t *testing.T) {
	mockConfig := aggregator.HTTPProxyConfig{
		HTTP: aggregator.HTTPBlock{
			Routers: map[string]aggregator.HTTPRouter{
				"test-router": {
					Rule:        "Host(`example.com`)",
					Service:     "test-service",
					EntryPoints: []string{"websecure"},
				},
			},
			Services: map[string]aggregator.HTTPService{
				"test-service": {
					LoadBalancer: aggregator.LoadBalancer{
						Servers: []aggregator.Server{{URL: "http://backend:80"}},
					},
				},
			},
		},
	}

	server := createMockPassthroughServer(t, mockConfig)
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:        "passthrough-test",
		APIURL:      server.URL,
		Passthrough: true,
	}

	client := &http.Client{}
	config, err := aggregator.FetchPassthroughConfig(ds, client)
	if err != nil {
		t.Fatalf("FetchPassthroughConfig failed: %v", err)
	}

	if len(config.HTTP.Routers) != 1 {
		t.Errorf("expected 1 router, got %d", len(config.HTTP.Routers))
	}
	if len(config.HTTP.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(config.HTTP.Services))
	}
}

func TestFetchPassthroughConfig_WithAPIKey(t *testing.T) {
	var capturedAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"http":{"routers":{},"services":{}}}`))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:        "passthrough-test",
		APIURL:      server.URL,
		Passthrough: true,
		APIKey:      "my-secret-key",
	}

	client := &http.Client{}
	_, err := aggregator.FetchPassthroughConfig(ds, client)
	if err != nil {
		t.Fatalf("FetchPassthroughConfig failed: %v", err)
	}

	expectedAuth := "Bearer my-secret-key"
	if capturedAuthHeader != expectedAuth {
		t.Errorf("expected Authorization header '%s', got '%s'", expectedAuth, capturedAuthHeader)
	}
}

func TestFetchPassthroughConfig_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:        "passthrough-test",
		APIURL:      server.URL,
		Passthrough: true,
	}

	client := &http.Client{}
	_, err := aggregator.FetchPassthroughConfig(ds, client)
	if err == nil {
		t.Error("expected error for non-200 status, got nil")
	}
}

func TestFetchPassthroughConfig_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:        "passthrough-test",
		APIURL:      server.URL,
		Passthrough: true,
	}

	client := &http.Client{}
	_, err := aggregator.FetchPassthroughConfig(ds, client)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestAggregateConfigs_Passthrough(t *testing.T) {
	// Create mock passthrough config
	mockConfig := aggregator.HTTPProxyConfig{
		HTTP: aggregator.HTTPBlock{
			Routers: map[string]aggregator.HTTPRouter{
				"upstream-router": {
					Rule:        "Host(`upstream.example.com`)",
					Service:     "upstream-service",
					EntryPoints: []string{"websecure"},
				},
			},
			Services: map[string]aggregator.HTTPService{
				"upstream-service": {
					LoadBalancer: aggregator.LoadBalancer{
						Servers: []aggregator.Server{{URL: "http://upstream-backend:80"}},
					},
				},
			},
		},
	}

	server := createMockPassthroughServer(t, mockConfig)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:        "passthrough-downstream",
				APIURL:      server.URL,
				Passthrough: true,
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	// Check that router exists with prefixed name
	expectedRouterName := "passthrough-downstream-upstream-router"
	if _, exists := cachedConfig.HTTP.Routers[expectedRouterName]; !exists {
		t.Errorf("expected router '%s' to exist, got routers: %v", expectedRouterName, getKeys(cachedConfig.HTTP.Routers))
	}

	// Check that service exists with prefixed name
	expectedServiceName := "passthrough-downstream-upstream-service"
	if _, exists := cachedConfig.HTTP.Services[expectedServiceName]; !exists {
		t.Errorf("expected service '%s' to exist, got services: %v", expectedServiceName, getServiceKeys(cachedConfig.HTTP.Services))
	}

	// Check that router's service reference is updated to prefixed name
	router := cachedConfig.HTTP.Routers[expectedRouterName]
	if router.Service != expectedServiceName {
		t.Errorf("expected router service '%s', got '%s'", expectedServiceName, router.Service)
	}
}

func TestAggregateConfigs_PassthroughWithMultipleRouters(t *testing.T) {
	mockConfig := aggregator.HTTPProxyConfig{
		HTTP: aggregator.HTTPBlock{
			Routers: map[string]aggregator.HTTPRouter{
				"router1": {
					Rule:        "Host(`app1.example.com`)",
					Service:     "service1",
					EntryPoints: []string{"websecure"},
				},
				"router2": {
					Rule:        "Host(`app2.example.com`)",
					Service:     "service2",
					EntryPoints: []string{"websecure"},
				},
			},
			Services: map[string]aggregator.HTTPService{
				"service1": {
					LoadBalancer: aggregator.LoadBalancer{
						Servers: []aggregator.Server{{URL: "http://backend1:80"}},
					},
				},
				"service2": {
					LoadBalancer: aggregator.LoadBalancer{
						Servers: []aggregator.Server{{URL: "http://backend2:80"}},
					},
				},
			},
		},
	}

	server := createMockPassthroughServer(t, mockConfig)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:        "multi-passthrough",
				APIURL:      server.URL,
				Passthrough: true,
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	if len(cachedConfig.HTTP.Routers) != 2 {
		t.Errorf("expected 2 routers, got %d", len(cachedConfig.HTTP.Routers))
	}
	if len(cachedConfig.HTTP.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(cachedConfig.HTTP.Services))
	}
}

func TestAggregateConfigs_MixedPassthroughAndRegular(t *testing.T) {
	// Create passthrough server
	passthroughConfig := aggregator.HTTPProxyConfig{
		HTTP: aggregator.HTTPBlock{
			Routers: map[string]aggregator.HTTPRouter{
				"passthrough-router": {
					Rule:        "Host(`passthrough.example.com`)",
					Service:     "passthrough-service",
					EntryPoints: []string{"websecure"},
				},
			},
			Services: map[string]aggregator.HTTPService{
				"passthrough-service": {
					LoadBalancer: aggregator.LoadBalancer{
						Servers: []aggregator.Server{{URL: "http://passthrough-backend:80"}},
					},
				},
			},
		},
	}
	passthroughServer := createMockPassthroughServer(t, passthroughConfig)
	defer passthroughServer.Close()

	// Create regular Traefik API server
	regularRouters := []aggregator.TraefikRouter{
		{
			Name:        "regular-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "regular-service",
			Rule:        "Host(`regular.example.com`)",
		},
	}
	regularServer := createMockTraefikServer(t, regularRouters)
	defer regularServer.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:        "passthrough-ds",
				APIURL:      passthroughServer.URL,
				Passthrough: true,
			},
			{
				Name:   "regular-ds",
				APIURL: regularServer.URL,
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	// Should have routers from both
	if len(cachedConfig.HTTP.Routers) != 2 {
		t.Errorf("expected 2 routers (1 passthrough + 1 regular), got %d", len(cachedConfig.HTTP.Routers))
	}

	// Verify passthrough router
	if _, exists := cachedConfig.HTTP.Routers["passthrough-ds-passthrough-router"]; !exists {
		t.Error("expected passthrough router to exist")
	}

	// Verify regular router
	if _, exists := cachedConfig.HTTP.Routers["regular-ds-regular-router"]; !exists {
		t.Error("expected regular router to exist")
	}
}

func TestAggregateConfigs_PassthroughError(t *testing.T) {
	// Create failing passthrough server
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failingServer.Close()

	// Create working regular server
	regularRouters := []aggregator.TraefikRouter{
		{
			Name:        "working-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "working-service",
			Rule:        "Host(`working.example.com`)",
		},
	}
	workingServer := createMockTraefikServer(t, regularRouters)
	defer workingServer.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:        "failing-passthrough",
				APIURL:      failingServer.URL,
				Passthrough: true,
			},
			{
				Name:   "working-regular",
				APIURL: workingServer.URL,
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	// Should still have the working downstream's routers
	if len(cachedConfig.HTTP.Routers) != 1 {
		t.Errorf("expected 1 router (from working downstream), got %d", len(cachedConfig.HTTP.Routers))
	}

	if _, exists := cachedConfig.HTTP.Routers["working-regular-working-router"]; !exists {
		t.Error("expected working router to exist")
	}
}

func TestAggregateConfigs_PassthroughPreservesConfig(t *testing.T) {
	// Test that passthrough preserves all router fields
	mockConfig := aggregator.HTTPProxyConfig{
		HTTP: aggregator.HTTPBlock{
			Routers: map[string]aggregator.HTTPRouter{
				"full-router": {
					Rule:        "Host(`full.example.com`) && PathPrefix(`/api`)",
					Service:     "full-service",
					EntryPoints: []string{"websecure", "web"},
					Middlewares: []string{"auth", "compress"},
					TLS:         map[string]interface{}{"certResolver": "letsencrypt"},
				},
			},
			Services: map[string]aggregator.HTTPService{
				"full-service": {
					LoadBalancer: aggregator.LoadBalancer{
						Servers: []aggregator.Server{{URL: "http://full-backend:80"}},
					},
				},
			},
		},
	}

	server := createMockPassthroughServer(t, mockConfig)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:        "preserve-test",
				APIURL:      server.URL,
				Passthrough: true,
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	router := cachedConfig.HTTP.Routers["preserve-test-full-router"]

	// Verify all fields are preserved
	if router.Rule != "Host(`full.example.com`) && PathPrefix(`/api`)" {
		t.Errorf("rule not preserved: %s", router.Rule)
	}
	if len(router.EntryPoints) != 2 {
		t.Errorf("expected 2 entrypoints, got %d", len(router.EntryPoints))
	}
	if len(router.Middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(router.Middlewares))
	}
	// Middleware references should be prefixed
	if router.Middlewares[0] != "preserve-test-auth" {
		t.Errorf("expected middleware 'preserve-test-auth', got '%s'", router.Middlewares[0])
	}
	if router.TLS == nil {
		t.Error("expected TLS config to be preserved")
	}
	if router.TLS["certResolver"] != "letsencrypt" {
		t.Errorf("expected certResolver 'letsencrypt', got '%v'", router.TLS["certResolver"])
	}
}

func TestAggregateConfigs_PassthroughWithMiddlewares(t *testing.T) {
	// Test that middlewares are passed through with prefixed names
	mockConfig := aggregator.HTTPProxyConfig{
		HTTP: aggregator.HTTPBlock{
			Routers: map[string]aggregator.HTTPRouter{
				"my-router": {
					Rule:        "Host(`example.com`)",
					Service:     "my-service",
					EntryPoints: []string{"websecure"},
					Middlewares: []string{"auth", "ratelimit"},
				},
			},
			Services: map[string]aggregator.HTTPService{
				"my-service": {
					LoadBalancer: aggregator.LoadBalancer{
						Servers: []aggregator.Server{{URL: "http://backend:80"}},
					},
				},
			},
			Middlewares: map[string]interface{}{
				"auth": map[string]interface{}{
					"basicAuth": map[string]interface{}{
						"users": []string{"admin:$apr1$..."},
					},
				},
				"ratelimit": map[string]interface{}{
					"rateLimit": map[string]interface{}{
						"average": 100,
						"burst":   50,
					},
				},
			},
		},
	}

	server := createMockPassthroughServer(t, mockConfig)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:        "mw-test",
				APIURL:      server.URL,
				Passthrough: true,
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	// Check middlewares are passed through with prefixed names
	if len(cachedConfig.HTTP.Middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(cachedConfig.HTTP.Middlewares))
	}

	if _, exists := cachedConfig.HTTP.Middlewares["mw-test-auth"]; !exists {
		t.Error("expected middleware 'mw-test-auth' to exist")
	}
	if _, exists := cachedConfig.HTTP.Middlewares["mw-test-ratelimit"]; !exists {
		t.Error("expected middleware 'mw-test-ratelimit' to exist")
	}

	// Check router middleware references are prefixed
	router := cachedConfig.HTTP.Routers["mw-test-my-router"]
	if len(router.Middlewares) != 2 {
		t.Errorf("expected 2 middleware references, got %d", len(router.Middlewares))
	}
	if router.Middlewares[0] != "mw-test-auth" {
		t.Errorf("expected middleware ref 'mw-test-auth', got '%s'", router.Middlewares[0])
	}
	if router.Middlewares[1] != "mw-test-ratelimit" {
		t.Errorf("expected middleware ref 'mw-test-ratelimit', got '%s'", router.Middlewares[1])
	}
}

// Helper functions
func getKeys(m map[string]aggregator.HTTPRouter) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getServiceKeys(m map[string]aggregator.HTTPService) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
