package aggregator_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"traefik-config-middleware/pkg/aggregator"
)

func TestFetchDownstreamRouters_Success(t *testing.T) {
	// Create mock server
	mockResponse := `[
		{
			"name": "test-router@kubernetes",
			"entryPoints": ["websecure"],
			"service": "test-service",
			"rule": "Host('example.com')"
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify endpoint
		if r.URL.Path != "/api/http/routers" {
			t.Errorf("expected path '/api/http/routers', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: server.URL,
	}

	client := &http.Client{}
	routers, err := aggregator.FetchDownstreamRouters(ds, client)
	if err != nil {
		t.Fatalf("FetchDownstreamRouters failed: %v", err)
	}

	if len(routers) != 1 {
		t.Fatalf("expected 1 router, got %d", len(routers))
	}
	if routers[0].Name != "test-router@kubernetes" {
		t.Errorf("expected router name 'test-router@kubernetes', got '%s'", routers[0].Name)
	}
}

func TestFetchDownstreamRouters_WithAPIKey(t *testing.T) {
	var capturedAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: server.URL,
		APIKey: "my-secret-key",
	}

	client := &http.Client{}
	_, err := aggregator.FetchDownstreamRouters(ds, client)
	if err != nil {
		t.Fatalf("FetchDownstreamRouters failed: %v", err)
	}

	expectedAuth := "Bearer my-secret-key"
	if capturedAuthHeader != expectedAuth {
		t.Errorf("expected Authorization header '%s', got '%s'", expectedAuth, capturedAuthHeader)
	}
}

func TestFetchDownstreamRouters_NoAPIKey(t *testing.T) {
	var capturedAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: server.URL,
		APIKey: "",
	}

	client := &http.Client{}
	_, err := aggregator.FetchDownstreamRouters(ds, client)
	if err != nil {
		t.Fatalf("FetchDownstreamRouters failed: %v", err)
	}

	if capturedAuthHeader != "" {
		t.Errorf("expected no Authorization header, got '%s'", capturedAuthHeader)
	}
}

func TestFetchDownstreamRouters_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: server.URL,
	}

	client := &http.Client{}
	_, err := aggregator.FetchDownstreamRouters(ds, client)
	if err == nil {
		t.Error("expected error for non-200 status, got nil")
	}
}

func TestFetchDownstreamRouters_404Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: server.URL,
	}

	client := &http.Client{}
	_, err := aggregator.FetchDownstreamRouters(ds, client)
	if err == nil {
		t.Error("expected error for 404 status, got nil")
	}
}

func TestFetchDownstreamRouters_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: server.URL,
	}

	client := &http.Client{}
	_, err := aggregator.FetchDownstreamRouters(ds, client)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestFetchDownstreamRouters_EmptyArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: server.URL,
	}

	client := &http.Client{}
	routers, err := aggregator.FetchDownstreamRouters(ds, client)
	if err != nil {
		t.Fatalf("FetchDownstreamRouters failed: %v", err)
	}

	if len(routers) != 0 {
		t.Errorf("expected 0 routers, got %d", len(routers))
	}
}

func TestFetchDownstreamRouters_InvalidAPIURL(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: "://invalid-url",
	}

	client := &http.Client{}
	_, err := aggregator.FetchDownstreamRouters(ds, client)
	if err == nil {
		t.Error("expected error for invalid API URL, got nil")
	}
}

func TestFetchDownstreamRouters_NetworkError(t *testing.T) {
	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: "http://localhost:99999", // Port that won't be listening
	}

	client := &http.Client{}
	_, err := aggregator.FetchDownstreamRouters(ds, client)
	if err == nil {
		t.Error("expected error for network failure, got nil")
	}
}

func TestFetchDownstreamRouters_MultipleRouters(t *testing.T) {
	mockResponse := `[
		{
			"name": "router1@kubernetes",
			"entryPoints": ["websecure"],
			"service": "service1",
			"rule": "Host('example1.com')"
		},
		{
			"name": "router2@kubernetes",
			"entryPoints": ["web"],
			"service": "service2",
			"rule": "Host('example2.com')"
		},
		{
			"name": "router3@kubernetes",
			"entryPoints": ["websecure"],
			"service": "service3",
			"rule": "Host('example3.com')",
			"tls": {"options": "default"}
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: server.URL,
	}

	client := &http.Client{}
	routers, err := aggregator.FetchDownstreamRouters(ds, client)
	if err != nil {
		t.Fatalf("FetchDownstreamRouters failed: %v", err)
	}

	if len(routers) != 3 {
		t.Errorf("expected 3 routers, got %d", len(routers))
	}
}

func TestFetchDownstreamRouters_LongErrorBody(t *testing.T) {
	// Create a response body longer than maxErrorBodyLen (256)
	longBody := make([]byte, 500)
	for i := range longBody {
		longBody[i] = 'x'
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(longBody)
	}))
	defer server.Close()

	ds := aggregator.DownstreamConfig{
		Name:   "test-downstream",
		APIURL: server.URL,
	}

	client := &http.Client{}
	_, err := aggregator.FetchDownstreamRouters(ds, client)
	if err == nil {
		t.Error("expected error for non-200 status, got nil")
	}
	// Error message should be truncated
	errStr := err.Error()
	if len(errStr) > 500 {
		t.Error("error message should be truncated for long responses")
	}
}
