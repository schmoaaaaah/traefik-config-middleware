package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"traefik-config-middleware/pkg/aggregator"
)

const (
	defaultPollInterval = 30 * time.Second
	defaultHTTPTimeout  = 10 * time.Second
	defaultConfigFile   = "config.yml"
	defaultListenAddr   = ":8080"
)

var (
	config     *aggregator.Config
	agg        *aggregator.Aggregator
	httpClient = &http.Client{
		Timeout: defaultHTTPTimeout,
	}
)

func getTraefikConfig(w http.ResponseWriter, r *http.Request) {
	cachedConfig := agg.GetCachedConfig()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cachedConfig); err != nil {
		log.Printf("Error encoding config response: %v", err)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func pollLoop() {
	duration, err := time.ParseDuration(config.PollInterval)
	if err != nil {
		duration = defaultPollInterval
	}

	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	// Initial fetch
	agg.AggregateConfigs()

	for range ticker.C {
		agg.AggregateConfigs()
	}
}

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigFile
	}

	var err error
	config, err = aggregator.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Configure HTTP client timeout from config
	timeout := defaultHTTPTimeout
	if config.HTTPTimeout != "" {
		if parsed, err := time.ParseDuration(config.HTTPTimeout); err == nil {
			timeout = parsed
		}
	}
	httpClient.Timeout = timeout

	// Create aggregator with config and HTTP client
	agg = aggregator.NewAggregator(config, httpClient)

	http.HandleFunc("/traefik-config", getTraefikConfig)
	http.HandleFunc("/health", healthCheck)

	go pollLoop()

	log.Printf("SNI Config Aggregator starting on %s", defaultListenAddr)
	log.Fatal(http.ListenAndServe(defaultListenAddr, nil))
}
