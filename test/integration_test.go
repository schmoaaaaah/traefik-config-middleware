package aggregator_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"traefik-config-middleware/pkg/aggregator"
)

// Helper to create a mock Traefik API server
func createMockTraefikServer(t *testing.T, routers []aggregator.TraefikRouter) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/http/routers" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(routers)
	}))
}

func TestAggregateConfigs_SingleDownstream(t *testing.T) {
	// Create mock server
	routers := []aggregator.TraefikRouter{
		{
			Name:        "test-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "test-service",
			Rule:        "Host(`example.com`) && PathPrefix(`/`)",
		},
	}
	server := createMockTraefikServer(t, routers)
	defer server.Close()

	// Setup config
	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:   "test-downstream",
				APIURL: server.URL,
			},
		},
	}

	// Create aggregator and run aggregation
	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	// Verify results
	cachedConfig := agg.GetCachedConfig()

	if len(cachedConfig.HTTP.Routers) != 1 {
		t.Errorf("expected 1 router, got %d", len(cachedConfig.HTTP.Routers))
	}
	if len(cachedConfig.HTTP.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(cachedConfig.HTTP.Services))
	}

	// Check router name format
	expectedRouterName := "test-downstream-test-router"
	if _, exists := cachedConfig.HTTP.Routers[expectedRouterName]; !exists {
		t.Errorf("expected router '%s' to exist", expectedRouterName)
	}
}

func TestAggregateConfigs_MultipleDownstreams(t *testing.T) {
	// Create mock servers
	routers1 := []aggregator.TraefikRouter{
		{
			Name:        "router1@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "service1",
			Rule:        "Host(`app1.example.com`)",
		},
	}
	server1 := createMockTraefikServer(t, routers1)
	defer server1.Close()

	routers2 := []aggregator.TraefikRouter{
		{
			Name:        "router2@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "service2",
			Rule:        "Host(`app2.example.com`)",
		},
	}
	server2 := createMockTraefikServer(t, routers2)
	defer server2.Close()

	// Setup config
	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{Name: "downstream1", APIURL: server1.URL},
			{Name: "downstream2", APIURL: server2.URL},
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

func TestAggregateConfigs_DownstreamError(t *testing.T) {
	// Create one working server and one failing
	routers := []aggregator.TraefikRouter{
		{
			Name:        "working-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "working-service",
			Rule:        "Host(`working.example.com`)",
		},
	}
	workingServer := createMockTraefikServer(t, routers)
	defer workingServer.Close()

	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failingServer.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{Name: "working", APIURL: workingServer.URL},
			{Name: "failing", APIURL: failingServer.URL},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	// Should still have routers from the working downstream
	if len(cachedConfig.HTTP.Routers) != 1 {
		t.Errorf("expected 1 router (from working downstream), got %d", len(cachedConfig.HTTP.Routers))
	}
}

func TestAggregateConfigs_EntryPointOverride(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "test-router@kubernetes",
			EntryPoints: []string{"web"},
			Service:     "test-service",
			Rule:        "Host(`example.com`)",
		},
	}
	server := createMockTraefikServer(t, routers)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:        "test-downstream",
				APIURL:      server.URL,
				EntryPoints: []string{"websecure"}, // Override to websecure
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	router := cachedConfig.HTTP.Routers["test-downstream-test-router"]
	if len(router.EntryPoints) != 1 || router.EntryPoints[0] != "websecure" {
		t.Errorf("expected entrypoints ['websecure'], got %v", router.EntryPoints)
	}
}

func TestAggregateConfigs_MiddlewareInjection(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "test-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "test-service",
			Rule:        "Host(`example.com`)",
		},
	}
	server := createMockTraefikServer(t, routers)
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

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	router := cachedConfig.HTTP.Routers["test-downstream-test-router"]
	if len(router.Middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(router.Middlewares))
	}
	if router.Middlewares[0] != "auth@file" {
		t.Errorf("expected first middleware 'auth@file', got '%s'", router.Middlewares[0])
	}
}

func TestAggregateConfigs_TLSWithCertResolver(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "secure-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "secure-service",
			Rule:        "Host(`secure.example.com`)",
			TLS:         map[string]interface{}{"options": "default"},
		},
	}
	server := createMockTraefikServer(t, routers)
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

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	router := cachedConfig.HTTP.Routers["test-downstream-secure-router"]
	if router.TLS == nil {
		t.Fatal("expected TLS config to be present")
	}
	if router.TLS["certResolver"] != "letsencrypt" {
		t.Errorf("expected certResolver 'letsencrypt', got '%v'", router.TLS["certResolver"])
	}
}

func TestAggregateConfigs_IgnoreEntryPoints(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "dashboard-router@internal",
			EntryPoints: []string{"traefik"},
			Service:     "api@internal",
			Rule:        "PathPrefix(`/dashboard`)",
		},
		{
			Name:        "app-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "app-service",
			Rule:        "Host(`app.example.com`)",
		},
	}
	server := createMockTraefikServer(t, routers)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:              "test-downstream",
				APIURL:            server.URL,
				IgnoreEntryPoints: []string{"traefik"},
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	// Should only have the app router, not the dashboard router
	if len(cachedConfig.HTTP.Routers) != 1 {
		t.Errorf("expected 1 router (ignored dashboard), got %d", len(cachedConfig.HTTP.Routers))
	}
	if _, exists := cachedConfig.HTTP.Routers["test-downstream-app-router"]; !exists {
		t.Error("expected app-router to exist")
	}
	if _, exists := cachedConfig.HTTP.Routers["test-downstream-dashboard-router"]; exists {
		t.Error("expected dashboard-router to be ignored")
	}
}

func TestAggregateConfigs_ConcurrentAccess(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "test-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "test-service",
			Rule:        "Host(`example.com`)",
		},
	}
	server := createMockTraefikServer(t, routers)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{Name: "test-downstream", APIURL: server.URL},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})

	// Run aggregation in background while reading config
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// Writer goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			agg.AggregateConfigs()
		}()
	}

	// Reader goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = agg.GetCachedConfig()
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		if err != nil {
			t.Errorf("concurrent access error: %v", err)
		}
	}
}

func TestAggregateConfigs_BackendURLConstruction(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "test-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "test-service",
			Rule:        "Host(`example.com`)",
			TLS:         map[string]interface{}{"options": "default"},
		},
	}
	server := createMockTraefikServer(t, routers)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:            "test-downstream",
				APIURL:          server.URL,
				BackendOverride: "https://backend.internal:8443",
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	service := cachedConfig.HTTP.Services["service-test-downstream-test-router"]
	if len(service.LoadBalancer.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(service.LoadBalancer.Servers))
	}
	if service.LoadBalancer.Servers[0].URL != "https://backend.internal:8443" {
		t.Errorf("expected backend URL 'https://backend.internal:8443', got '%s'", service.LoadBalancer.Servers[0].URL)
	}
}

func TestAggregateConfigs_ServerTransport(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "test-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "test-service",
			Rule:        "Host(`example.com`)",
			TLS:         map[string]interface{}{"options": "default"},
		},
	}
	server := createMockTraefikServer(t, routers)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:            "test-downstream",
				APIURL:          server.URL,
				ServerTransport: "insecure-transport",
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	service := cachedConfig.HTTP.Services["service-test-downstream-test-router"]
	if service.LoadBalancer.ServersTransport != "insecure-transport" {
		t.Errorf("expected serversTransport 'insecure-transport', got '%s'", service.LoadBalancer.ServersTransport)
	}
}

func TestAggregateConfigs_ServerTransportNotSet(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "test-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "test-service",
			Rule:        "Host(`example.com`)",
		},
	}
	server := createMockTraefikServer(t, routers)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:   "test-downstream",
				APIURL: server.URL,
				// ServerTransport not set
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	service := cachedConfig.HTTP.Services["service-test-downstream-test-router"]
	if service.LoadBalancer.ServersTransport != "" {
		t.Errorf("expected empty serversTransport, got '%s'", service.LoadBalancer.ServersTransport)
	}
}

func TestAggregateConfigs_WildcardFix(t *testing.T) {
	routers := []aggregator.TraefikRouter{
		{
			Name:        "wildcard-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "wildcard-service",
			Rule:        "HostRegexp(`^[a-zA-Z0-9-]+\\.pages\\.example\\.com$`)",
			TLS:         map[string]interface{}{"options": "default"},
		},
	}
	server := createMockTraefikServer(t, routers)
	defer server.Close()

	cfg := &aggregator.Config{
		Downstream: []aggregator.DownstreamConfig{
			{
				Name:        "test-downstream",
				APIURL:      server.URL,
				WildcardFix: true,
				TLS: &aggregator.TLSConfig{
					CertResolver: "letsencrypt",
				},
			},
		},
	}

	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	cachedConfig := agg.GetCachedConfig()

	router := cachedConfig.HTTP.Routers["test-downstream-wildcard-router"]
	if router.TLS == nil {
		t.Fatal("expected TLS config to be present")
	}

	domains, ok := router.TLS["domains"].([]aggregator.TLSDomain)
	if !ok {
		t.Fatal("expected domains to be []TLSDomain")
	}
	if len(domains) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(domains))
	}
	if domains[0].Main != "*.pages.example.com" {
		t.Errorf("expected wildcard domain '*.pages.example.com', got '%s'", domains[0].Main)
	}
}

func TestIntegration_FullFlow(t *testing.T) {
	// Test the full flow: load config, fetch routers, aggregate, get config

	routers := []aggregator.TraefikRouter{
		{
			Name:        "app-router@kubernetes",
			EntryPoints: []string{"websecure"},
			Service:     "app-service",
			Rule:        "Host(`app.example.com`) && PathPrefix(`/`)",
			TLS:         map[string]interface{}{"options": "default"},
		},
	}
	server := createMockTraefikServer(t, routers)
	defer server.Close()

	// Create test config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `downstream:
  - name: test-app
    api_url: ` + server.URL + `
    tls:
      cert_resolver: letsencrypt
    middlewares:
      - auth@file
poll_interval: 10s
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load config
	cfg, err := aggregator.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Create aggregator and aggregate configs
	agg := aggregator.NewAggregator(cfg, &http.Client{})
	agg.AggregateConfigs()

	// Verify results
	result := agg.GetCachedConfig()

	// Verify router exists with expected configuration
	router, exists := result.HTTP.Routers["test-app-app-router"]
	if !exists {
		t.Fatal("expected router 'test-app-app-router' to exist")
	}

	if router.Rule != "Host(`app.example.com`) && PathPrefix(`/`)" {
		t.Errorf("unexpected rule: %s", router.Rule)
	}

	if len(router.Middlewares) != 1 || router.Middlewares[0] != "auth@file" {
		t.Errorf("expected middlewares ['auth@file'], got %v", router.Middlewares)
	}

	if router.TLS == nil {
		t.Error("expected TLS config to be present")
	}
}
