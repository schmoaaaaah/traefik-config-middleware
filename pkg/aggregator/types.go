package aggregator

// Config represents the application configuration
type Config struct {
	Downstream   []DownstreamConfig `yaml:"downstream"`
	PollInterval string             `yaml:"poll_interval"`
	HTTPTimeout  string             `yaml:"http_timeout"`
	LogLevel     string             `yaml:"log_level"`
}

// TLSConfig holds TLS-specific configuration for a downstream
type TLSConfig struct {
	CertResolver  string `yaml:"cert_resolver"`
	StripResolver bool   `yaml:"strip_resolver"`
}

// TLSDomain represents a single domain entry for TLS certificates
type TLSDomain struct {
	Main string   `json:"main"`
	Sans []string `json:"sans,omitempty"`
}

// DownstreamConfig represents configuration for a single downstream Traefik instance
type DownstreamConfig struct {
	Name              string     `yaml:"name"`
	APIURL            string     `yaml:"api_url"`
	BackendOverride   string     `yaml:"backend_override"`
	APIKey            string     `yaml:"api_key"`
	TLS               *TLSConfig `yaml:"tls"`
	EntryPoints       []string   `yaml:"entrypoints"`
	Middlewares       []string   `yaml:"middlewares"`
	IgnoreEntryPoints []string   `yaml:"ignore_entrypoints"`
	WildcardFix       bool       `yaml:"wildcard_fix"`
	Passthrough       bool       `yaml:"passthrough"`
	ServerTransport   string     `yaml:"server_transport"`
}

// TraefikRouter represents a router from the Traefik API
type TraefikRouter struct {
	Name        string                 `json:"name"`
	EntryPoints []string               `json:"entryPoints"`
	Service     string                 `json:"service"`
	Rule        string                 `json:"rule"`
	TLS         map[string]interface{} `json:"tls,omitempty"`
}

// HTTPRouter represents an HTTP router in the output configuration
type HTTPRouter struct {
	Rule        string                 `json:"rule"`
	Service     string                 `json:"service"`
	EntryPoints []string               `json:"entryPoints"`
	Middlewares []string               `json:"middlewares,omitempty"`
	TLS         map[string]interface{} `json:"tls,omitempty"`
}

// Server represents a backend server
type Server struct {
	URL string `json:"url"`
}

// LoadBalancer represents load balancer configuration
type LoadBalancer struct {
	ServersTransport string   `json:"serversTransport,omitempty"`
	Servers          []Server `json:"servers"`
}

// HTTPService represents an HTTP service in the output configuration
type HTTPService struct {
	LoadBalancer LoadBalancer `json:"loadBalancer"`
}

// HTTPBlock contains routers, services, and middlewares
type HTTPBlock struct {
	Routers     map[string]HTTPRouter  `json:"routers"`
	Services    map[string]HTTPService `json:"services"`
	Middlewares map[string]interface{} `json:"middlewares,omitempty"`
}

// HTTPProxyConfig is the complete output configuration
type HTTPProxyConfig struct {
	HTTP HTTPBlock `json:"http"`
}
