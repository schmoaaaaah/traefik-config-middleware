package aggregator

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Aggregator manages the configuration aggregation from downstream Traefik instances
type Aggregator struct {
	config       *Config
	cachedConfig HTTPProxyConfig
	configMutex  sync.RWMutex
	httpClient   *http.Client
}

// NewAggregator creates a new Aggregator with the given configuration and HTTP client
func NewAggregator(config *Config, client *http.Client) *Aggregator {
	return &Aggregator{
		config:     config,
		httpClient: client,
	}
}

// GetCachedConfig returns the current cached configuration (thread-safe)
func (a *Aggregator) GetCachedConfig() HTTPProxyConfig {
	a.configMutex.RLock()
	defer a.configMutex.RUnlock()
	return a.cachedConfig
}

// AggregateConfigs fetches router configurations from all downstream Traefik instances
// and builds a unified HTTPProxyConfig. Errors from individual downstreams are logged
// but don't stop processing of other downstreams.
func (a *Aggregator) AggregateConfigs() {
	newConfig := HTTPProxyConfig{}
	newConfig.HTTP.Routers = make(map[string]HTTPRouter)
	newConfig.HTTP.Services = make(map[string]HTTPService)

	for _, ds := range a.config.Downstream {
		routers, err := FetchDownstreamRouters(ds, a.httpClient)
		if err != nil {
			log.Printf("Error fetching from %s: %v", ds.Name, err)
			continue
		}

		log.Printf("Processing %s with %d routers", ds.Name, len(routers))

		for _, router := range routers {
			// Skip routers with ignored entrypoints
			if ShouldIgnoreRouter(router, ds.IgnoreEntryPoints) {
				log.Printf("  Skipping router %s (ignored entrypoint)", router.Name)
				continue
			}

			// Determine if this router uses TLS
			useTLS := len(router.TLS) > 0

			// Get backend URL with protocol matching
			backendURL := GetBackendURL(ds, useTLS)

			// Generate unique names for router and service
			// Use router name without provider suffix if available
			routerBaseName := router.Name
			if idx := strings.Index(routerBaseName, "@"); idx != -1 {
				routerBaseName = routerBaseName[:idx]
			}

			httpRouterName := fmt.Sprintf("%s-%s", ds.Name, routerBaseName)
			httpServiceName := fmt.Sprintf("service-%s-%s", ds.Name, routerBaseName)

			// Determine entrypoints - use override if specified
			entryPoints := router.EntryPoints
			if len(ds.EntryPoints) > 0 {
				entryPoints = ds.EntryPoints
			}

			// Create HTTP router preserving original rule
			httpRouter := HTTPRouter{
				Rule:        router.Rule,
				Service:     httpServiceName,
				EntryPoints: entryPoints,
				Middlewares: ds.Middlewares, // User-defined middlewares from config
			}

			// Build TLS config with domain extraction
			if ds.TLS != nil || len(router.TLS) > 0 {
				tlsConfig := BuildTLSConfig(ds, router.Rule, router.TLS)
				if len(tlsConfig) > 0 {
					httpRouter.TLS = tlsConfig
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

	a.configMutex.Lock()
	a.cachedConfig = newConfig
	a.configMutex.Unlock()

	log.Printf("Config aggregation complete: %d routers, %d services",
		len(newConfig.HTTP.Routers), len(newConfig.HTTP.Services))
}
