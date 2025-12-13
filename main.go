package main

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/url"
    "os"
    "strings"
    "sync"
    "time"

    "gopkg.in/yaml.v3"
)

const (
    defaultPollInterval = 30 * time.Second
    defaultHTTPTimeout  = 10 * time.Second
    defaultConfigFile   = "config.yml"
    defaultListenAddr   = ":8080"
    maxErrorBodyLen     = 256
)

type Config struct {
    Downstream   []DownstreamConfig `yaml:"downstream"`
    PollInterval string             `yaml:"poll_interval"`
    HTTPTimeout  string             `yaml:"http_timeout"`
    LogLevel     string             `yaml:"log_level"`
}

type TLSConfig struct {
    CertResolver string `yaml:"cert_resolver"`
}

type DownstreamConfig struct {
    Name               string     `yaml:"name"`
    APIURL             string     `yaml:"api_url"`
    BackendOverride    string     `yaml:"backend_override"`
    APIKey             string     `yaml:"api_key"`
    TLS                *TLSConfig `yaml:"tls"`
    Middlewares        []string   `yaml:"middlewares"`
    IgnoreEntryPoints  []string   `yaml:"ignore_entrypoints"`
}

type TraefikRouter struct {
    Name        string                 `json:"name"`
    EntryPoints []string               `json:"entryPoints"`
    Service     string                 `json:"service"`
    Rule        string                 `json:"rule"`
    TLS         map[string]interface{} `json:"tls,omitempty"`
}

type HTTPRouter struct {
    Rule        string                 `json:"rule"`
    Service     string                 `json:"service"`
    EntryPoints []string               `json:"entryPoints"`
    Middlewares []string               `json:"middlewares,omitempty"`
    TLS         map[string]interface{} `json:"tls,omitempty"`
}

type Server struct {
    URL string `json:"url"`
}

type LoadBalancer struct {
    Servers []Server `json:"servers"`
}

type HTTPService struct {
    LoadBalancer LoadBalancer `json:"loadBalancer"`
}

type HTTPBlock struct {
    Routers  map[string]HTTPRouter  `json:"routers"`
    Services map[string]HTTPService `json:"services"`
}

type HTTPProxyConfig struct {
    HTTP HTTPBlock `json:"http"`
}

var (
    config       Config
    cachedConfig HTTPProxyConfig
    configMutex  sync.RWMutex
    httpClient   = &http.Client{
        Timeout: 10 * time.Second,
    }
)

// loadConfig loads the application configuration from the specified YAML file.
// Expected YAML structure:
//   - downstream: list of downstream Traefik instances to aggregate
//   - poll_interval: how often to refresh config (default: 30s)
//   - http_timeout: timeout for API calls (default: 10s)
//   - log_level: logging verbosity
//
// If poll_interval is not specified, defaults to 30s.
func loadConfig(filename string) error {
    data, err := os.ReadFile(filename)
    if err != nil {
        return err
    }

    err = yaml.Unmarshal(data, &config)
    if err != nil {
        return err
    }

    if config.PollInterval == "" {
        config.PollInterval = "30s"
    }

    log.Printf("Loaded config with %d downstream instances", len(config.Downstream))
    return nil
}


func fetchDownstreamRouters(ds DownstreamConfig) ([]TraefikRouter, error) {
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

    resp, err := httpClient.Do(req)
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

func getBackendURL(ds DownstreamConfig, useTLS bool) string {
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

func shouldIgnoreRouter(router TraefikRouter, ignoreEntryPoints []string) bool {
    if len(ignoreEntryPoints) == 0 {
        return false
    }

    // Check if any of the router's entrypoints are in the ignore list
    for _, routerEP := range router.EntryPoints {
        for _, ignoreEP := range ignoreEntryPoints {
            if routerEP == ignoreEP {
                return true
            }
        }
    }

    return false
}

// aggregateConfigs fetches router configurations from all downstream Traefik instances
// and builds a unified HTTPProxyConfig. Errors from individual downstreams are logged
// but don't stop processing of other downstreams. Updates cachedConfig atomically.
func aggregateConfigs() {
    newConfig := HTTPProxyConfig{}
    newConfig.HTTP.Routers = make(map[string]HTTPRouter)
    newConfig.HTTP.Services = make(map[string]HTTPService)

    for _, ds := range config.Downstream {
        routers, err := fetchDownstreamRouters(ds)
        if err != nil {
            log.Printf("Error fetching from %s: %v", ds.Name, err)
            continue
        }

        log.Printf("Processing %s with %d routers", ds.Name, len(routers))

        for _, router := range routers {
            // Skip routers with ignored entrypoints
            if shouldIgnoreRouter(router, ds.IgnoreEntryPoints) {
                log.Printf("  Skipping router %s (ignored entrypoint)", router.Name)
                continue
            }

            // Determine if this router uses TLS
            useTLS := len(router.TLS) > 0

            // Get backend URL with protocol matching
            backendURL := getBackendURL(ds, useTLS)

            // Generate unique names for router and service
            // Use router name without provider suffix if available
            routerBaseName := router.Name
            if idx := strings.Index(routerBaseName, "@"); idx != -1 {
                routerBaseName = routerBaseName[:idx]
            }

            httpRouterName := fmt.Sprintf("%s-%s", ds.Name, routerBaseName)
            httpServiceName := fmt.Sprintf("service-%s-%s", ds.Name, routerBaseName)

            // Create HTTP router preserving original rule
            httpRouter := HTTPRouter{
                Rule:        router.Rule,
                Service:     httpServiceName,
                EntryPoints: router.EntryPoints,
                Middlewares: ds.Middlewares, // User-defined middlewares from config
            }

            // Apply user-configured TLS settings
            if ds.TLS != nil && ds.TLS.CertResolver != "" {
                httpRouter.TLS = map[string]interface{}{
                    "certResolver": ds.TLS.CertResolver,
                }
            }

            newConfig.HTTP.Routers[httpRouterName] = httpRouter

            // Create HTTP service pointing to downstream Traefik
            httpService := HTTPService{}
            httpService.LoadBalancer.Servers = []Server{
                {URL: backendURL},
            }
            newConfig.HTTP.Services[httpServiceName] = httpService

            log.Printf("  Added HTTP route: %s -> %s (TLS: %v)", router.Rule, backendURL, useTLS)
        }
    }

    configMutex.Lock()
    cachedConfig = newConfig
    configMutex.Unlock()

    log.Printf("Config aggregation complete: %d routers, %d services",
        len(newConfig.HTTP.Routers), len(newConfig.HTTP.Services))
}

func getTraefikConfig(w http.ResponseWriter, r *http.Request) {
    configMutex.RLock()
    defer configMutex.RUnlock()

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
    aggregateConfigs()

    for range ticker.C {
        aggregateConfigs()
    }
}

func main() {
    configPath := os.Getenv("CONFIG_PATH")
    if configPath == "" {
        configPath = defaultConfigFile
    }

    if err := loadConfig(configPath); err != nil {
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

    http.HandleFunc("/traefik-config", getTraefikConfig)
    http.HandleFunc("/health", healthCheck)

    go pollLoop()

    log.Printf("SNI Config Aggregator starting on %s", defaultListenAddr)
    log.Fatal(http.ListenAndServe(defaultListenAddr, nil))
}