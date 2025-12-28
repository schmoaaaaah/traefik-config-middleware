package aggregator_test

import (
	"os"
	"path/filepath"
	"testing"

	"traefik-config-middleware/pkg/aggregator"
)

func TestLoadConfig_ValidFile(t *testing.T) {
	cfg, err := aggregator.LoadConfig(filepath.Join("testdata", "valid_config.yml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify downstream count
	if len(cfg.Downstream) != 2 {
		t.Errorf("expected 2 downstream instances, got %d", len(cfg.Downstream))
	}

	// Verify first downstream
	ds1 := cfg.Downstream[0]
	if ds1.Name != "traefik-downstream-1" {
		t.Errorf("expected name 'traefik-downstream-1', got '%s'", ds1.Name)
	}
	if ds1.APIURL != "http://traefik-1:8081" {
		t.Errorf("expected api_url 'http://traefik-1:8081', got '%s'", ds1.APIURL)
	}
	if ds1.BackendOverride != "http://backend-1:8080" {
		t.Errorf("expected backend_override 'http://backend-1:8080', got '%s'", ds1.BackendOverride)
	}
	if ds1.TLS == nil || ds1.TLS.CertResolver != "myresolver" {
		t.Errorf("expected TLS cert_resolver 'myresolver'")
	}
	if len(ds1.EntryPoints) != 1 || ds1.EntryPoints[0] != "websecure" {
		t.Errorf("expected entrypoints ['websecure'], got %v", ds1.EntryPoints)
	}
	if len(ds1.Middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(ds1.Middlewares))
	}
	if len(ds1.IgnoreEntryPoints) != 1 || ds1.IgnoreEntryPoints[0] != "traefik" {
		t.Errorf("expected ignore_entrypoints ['traefik'], got %v", ds1.IgnoreEntryPoints)
	}
	if !ds1.WildcardFix {
		t.Error("expected wildcard_fix to be true")
	}

	// Verify second downstream
	ds2 := cfg.Downstream[1]
	if ds2.APIKey != "secret-api-key" {
		t.Errorf("expected api_key 'secret-api-key', got '%s'", ds2.APIKey)
	}

	// Verify poll interval
	if cfg.PollInterval != "15s" {
		t.Errorf("expected poll_interval '15s', got '%s'", cfg.PollInterval)
	}

	// Verify http timeout
	if cfg.HTTPTimeout != "5s" {
		t.Errorf("expected http_timeout '5s', got '%s'", cfg.HTTPTimeout)
	}

	// Verify log level
	if cfg.LogLevel != "debug" {
		t.Errorf("expected log_level 'debug', got '%s'", cfg.LogLevel)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := aggregator.LoadConfig("nonexistent_file.yml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected file not found error, got: %v", err)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	_, err := aggregator.LoadConfig(filepath.Join("testdata", "invalid_config.yml"))
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoadConfig_DefaultPollInterval(t *testing.T) {
	cfg, err := aggregator.LoadConfig(filepath.Join("testdata", "minimal_config.yml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Should default to 30s when not specified
	if cfg.PollInterval != "30s" {
		t.Errorf("expected default poll_interval '30s', got '%s'", cfg.PollInterval)
	}
}

func TestLoadConfig_MultipleDownstreams(t *testing.T) {
	cfg, err := aggregator.LoadConfig(filepath.Join("testdata", "valid_config.yml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(cfg.Downstream) != 2 {
		t.Errorf("expected 2 downstream instances, got %d", len(cfg.Downstream))
	}

	// Verify each downstream has required fields
	for i, ds := range cfg.Downstream {
		if ds.Name == "" {
			t.Errorf("downstream %d: expected non-empty name", i)
		}
		if ds.APIURL == "" {
			t.Errorf("downstream %d: expected non-empty api_url", i)
		}
	}
}
