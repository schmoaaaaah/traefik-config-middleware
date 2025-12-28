package aggregator_test

import (
	"testing"

	"traefik-config-middleware/pkg/aggregator"
)

func TestGetBackendURL_WithOverride_FullURL_HTTP(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL:          "http://traefik:8081",
		BackendOverride: "http://custom-backend:9000",
	}

	result := aggregator.GetBackendURL(ds, false)
	expected := "http://custom-backend:9000"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetBackendURL_WithOverride_FullURL_HTTPS(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL:          "http://traefik:8081",
		BackendOverride: "https://secure-backend:443",
	}

	result := aggregator.GetBackendURL(ds, true)
	expected := "https://secure-backend:443"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetBackendURL_WithOverride_HostOnly_HTTP(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL:          "http://traefik:8081",
		BackendOverride: "custom-backend:9000",
	}

	result := aggregator.GetBackendURL(ds, false)
	expected := "http://custom-backend:9000"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetBackendURL_WithOverride_HostOnly_HTTPS(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL:          "http://traefik:8081",
		BackendOverride: "secure-backend:443",
	}

	result := aggregator.GetBackendURL(ds, true)
	expected := "https://secure-backend:443"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetBackendURL_FromAPIURL_HTTP(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL: "http://traefik-host:8081",
	}

	result := aggregator.GetBackendURL(ds, false)
	expected := "http://traefik-host:8081"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetBackendURL_FromAPIURL_HTTPS(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL: "https://traefik-host:8082",
	}

	result := aggregator.GetBackendURL(ds, true)
	expected := "https://traefik-host:8082"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetBackendURL_DefaultPort_HTTP(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL: "http://traefik-host",
	}

	result := aggregator.GetBackendURL(ds, false)
	expected := "http://traefik-host:80"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetBackendURL_DefaultPort_HTTPS(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL: "https://traefik-host",
	}

	result := aggregator.GetBackendURL(ds, true)
	expected := "https://traefik-host:443"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetBackendURL_PreserveExistingPort(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL: "http://traefik-host:8080",
	}

	result := aggregator.GetBackendURL(ds, false)
	expected := "http://traefik-host:8080"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetBackendURL_WithPath(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL: "http://traefik-host:8081/api/endpoint",
	}

	result := aggregator.GetBackendURL(ds, false)
	expected := "http://traefik-host:8081"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetBackendURL_TLSChangesProtocol(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL: "http://traefik-host:8081",
	}

	// Without TLS
	resultHTTP := aggregator.GetBackendURL(ds, false)
	if resultHTTP != "http://traefik-host:8081" {
		t.Errorf("expected http protocol, got '%s'", resultHTTP)
	}

	// With TLS (should change to https)
	resultHTTPS := aggregator.GetBackendURL(ds, true)
	if resultHTTPS != "https://traefik-host:8081" {
		t.Errorf("expected https protocol, got '%s'", resultHTTPS)
	}
}

func TestGetBackendURL_OverrideWithHTTPSProtocol_IgnoresTLSFlag(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		APIURL:          "http://traefik:8081",
		BackendOverride: "https://secure-backend:443",
	}

	// Even with useTLS=false, the override with https:// should be used as-is
	result := aggregator.GetBackendURL(ds, false)
	expected := "https://secure-backend:443"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}
