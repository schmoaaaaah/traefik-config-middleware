# Traefik Config Middleware

Automatically aggregate and promote configurations from downstream Traefik instances to an upstream Traefik proxy. This middleware acts as a dynamic configuration provider, polling multiple Traefik instances and exposing their combined routing configuration via HTTP API.

## Features

- **Multi-instance aggregation**: Poll and combine configurations from multiple downstream Traefik instances
- **Automatic route discovery**: Dynamically discovers HTTP routers and creates corresponding upstream routes
- **TLS preservation**: Maintains TLS settings from downstream routers
- **Flexible backend routing**: Override backend URLs or auto-detect from API endpoints
- **Middleware injection**: Attach custom middlewares to all routes from specific downstream instances
- **Entry point filtering**: Ignore internal/admin routes using entry point filters
- **Configurable polling**: Adjustable poll intervals for configuration updates
- **Health checks**: Built-in health endpoint for monitoring

## Use Cases

- **Multi-cluster ingress**: Aggregate routes from multiple Kubernetes clusters into a single entry point
- **Hybrid cloud routing**: Combine routes from on-premise and cloud-hosted Traefik instances
- **Development environments**: Expose multiple developer Traefik instances through a single proxy
- **Service mesh integration**: Create a centralized routing layer for distributed Traefik deployments

## Configuration

Create a `config.yml` file in the same directory as the binary:

```yaml
downstream:
  - name: production-cluster
    api_url: http://traefik-prod.example.com:8080
    # Optional: Override backend URL (useful for private IPs/hostnames)
    backend_override: https://prod-internal.example.com
    # Optional: Add middlewares to all routes from this downstream
    middlewares:
      - auth@file
      - ratelimit@file
    # Optional: Ignore routes on specific entrypoints
    ignore_entrypoints:
      - traefik  # Ignore Traefik dashboard routes

  - name: staging-cluster
    api_url: http://traefik-staging.example.com:8080
    # Optional: API key for authenticated Traefik API
    api_key: your-api-key-here

  - name: dev-cluster
    api_url: http://traefik-dev.example.com:8080

# Optional: Poll interval (default: 30s)
poll_interval: 30s

# Optional: Log level (default: warn)
log_level: info
```

### Configuration Options

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `downstream` | array | Yes | - | List of downstream Traefik instances to poll |
| `downstream[].name` | string | Yes | - | Unique identifier for this downstream instance |
| `downstream[].api_url` | string | Yes | - | Traefik API URL (usually port 8080) |
| `downstream[].backend_override` | string | No | Auto-detected | Override the backend URL for proxying requests |
| `downstream[].api_key` | string | No | - | Bearer token for authenticated Traefik APIs |
| `downstream[].middlewares` | array | No | [] | Middlewares to attach to all routes from this instance |
| `downstream[].ignore_entrypoints` | array | No | [] | Skip routes using these entrypoints |
| `poll_interval` | string | No | 30s | How often to poll downstream instances |
| `log_level` | string | No | warn | Logging verbosity (debug, info, warn, error) |

## Usage

### 1. Start the Middleware

```bash
# Using Docker Compose (see compose.yaml)
docker-compose up -d

# Or run directly
./traefik-config-middleware
```

The service exposes two endpoints:
- `http://localhost:8080/traefik-config` - Dynamic configuration endpoint
- `http://localhost:8080/health` - Health check endpoint

### 2. Configure Upstream Traefik

Configure your upstream Traefik instance to use this middleware as a dynamic configuration provider:

```yaml
# traefik.yml or traefik.toml
providers:
  http:
    endpoint: "http://traefik-config-middleware:8080/traefik-config"
    pollInterval: "30s"
```

Or using Docker labels:

```yaml
# docker-compose.yml
services:
  traefik:
    image: traefik:v3.6.2
    command:
      - "--providers.http.endpoint=http://traefik-config-middleware:8080/traefik-config"
      - "--providers.http.pollInterval=30s"
```

### 3. Verify Configuration

Check the aggregated configuration:

```bash
curl http://localhost:8080/traefik-config | jq
```

Check service health:

```bash
curl http://localhost:8080/health
```

## How It Works

1. **Polling**: The middleware polls each downstream Traefik instance at the configured interval
2. **Aggregation**: HTTP routers from all downstream instances are collected and processed
3. **Route Generation**: For each downstream router:
   - A new HTTP router is created with the original rule
   - A service is created pointing to the downstream Traefik instance
   - TLS settings are preserved
   - Custom middlewares are attached if configured
   - Routes on ignored entrypoints are skipped
4. **Exposure**: The aggregated configuration is served via HTTP API
5. **Upstream Sync**: The upstream Traefik instance polls this API and applies the routes

## Architecture

```
┌─────────────────┐
│  Downstream     │
│  Traefik 1      │──┐
└─────────────────┘  │
                     │  Poll & Aggregate
┌─────────────────┐  │  ┌──────────────────────┐
│  Downstream     │──┼─→│   Config Middleware  │
│  Traefik 2      │  │  │  :8080/traefik-config│
└─────────────────┘  │  └──────────────────────┘
                     │             │
┌─────────────────┐  │             │ HTTP Provider
│  Downstream     │──┘             ↓
│  Traefik 3      │         ┌─────────────┐
└─────────────────┘         │  Upstream   │
                            │  Traefik    │
                            └─────────────┘
```

## Development

### Prerequisites

- Go 1.25 or later
- Docker (for containerized builds)

### Building

```bash
# Build binary
go build -o traefik-config-middleware

# Build Docker image
docker build -t traefik-config-middleware .
```

## Support

- **Issues**: [GitHub Issues](https://github.com/schmoaaaaah/traefik-config-middleware/issues)
- **Discussions**: [GitHub Discussions](https://github.com/schmoaaaaah/traefik-config-middleware/discussions)

## Related Projects

- [Traefik](https://github.com/traefik/traefik) - The Cloud Native Application Proxy
