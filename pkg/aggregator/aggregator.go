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
	newConfig.HTTP.Middlewares = make(map[string]interface{})

	for _, ds := range a.config.Downstream {
		// Handle passthrough mode - fetch full config and merge with prefixed names
		if ds.Passthrough {
			passthroughConfig, err := FetchPassthroughConfig(ds, a.httpClient)
			if err != nil {
				log.Printf("Error fetching passthrough from %s: %v", ds.Name, err)
				continue
			}

			// Merge middlewares with prefixed names
			for name, middleware := range passthroughConfig.HTTP.Middlewares {
				prefixedName := fmt.Sprintf("%s-%s", ds.Name, name)
				newConfig.HTTP.Middlewares[prefixedName] = middleware
			}

			// Merge routers with prefixed names
			for name, router := range passthroughConfig.HTTP.Routers {
				prefixedName := fmt.Sprintf("%s-%s", ds.Name, name)
				prefixedServiceName := fmt.Sprintf("%s-%s", ds.Name, router.Service)
				router.Service = prefixedServiceName

				// Prefix middleware references
				if len(router.Middlewares) > 0 {
					prefixedMiddlewares := make([]string, len(router.Middlewares))
					for i, mw := range router.Middlewares {
						prefixedMiddlewares[i] = fmt.Sprintf("%s-%s", ds.Name, mw)
					}
					router.Middlewares = prefixedMiddlewares
				}

				newConfig.HTTP.Routers[prefixedName] = router
			}

			// Merge services with prefixed names
			for name, service := range passthroughConfig.HTTP.Services {
				prefixedName := fmt.Sprintf("%s-%s", ds.Name, name)
				newConfig.HTTP.Services[prefixedName] = service
			}

			log.Printf("Passthrough %s: %d routers, %d services, %d middlewares",
				ds.Name,
				len(passthroughConfig.HTTP.Routers),
				len(passthroughConfig.HTTP.Services),
				len(passthroughConfig.HTTP.Middlewares))
			continue
		}

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
			if ds.ServerTransport != "" {
				httpService.LoadBalancer.ServersTransport = ds.ServerTransport
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
