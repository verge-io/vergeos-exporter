# VergeOS Exporter

A Prometheus exporter for VergeOS that collects metrics about VSAN tiers, clusters, nodes, storage, and network.

## Tech Stack

- **Language**: Go 1.23.4
- **Framework**: Prometheus client library (`github.com/prometheus/client_golang`)
- **Build Tool**: GoReleaser (cross-platform builds)
- **CI/CD**: GitHub Actions (release automation on tags)

## Project Structure

```
vergeos-exporter/
├── main.go              # Entry point, CLI flags, HTTP server setup
├── collectors/          # Prometheus collectors for different VergeOS resources
│   ├── base.go          # BaseCollector with shared HTTP/auth logic
│   ├── collector.go     # Collector interface definition
│   ├── types.go         # API response types and JSON unmarshaling
│   ├── storage.go       # VSAN tier and drive metrics
│   ├── node.go          # Physical node metrics (CPU, RAM, IPMI)
│   ├── cluster.go       # Cluster-level metrics
│   ├── network.go       # NIC metrics
│   └── system.go        # System version info
├── tests/               # Additional test files
├── examples/            # Docker Compose example, Grafana dashboard
├── metrics.md           # Complete metrics reference
└── .goreleaser.yml      # Cross-platform release configuration
```

## Commands

```bash
# Build
go build

# Run tests
go test ./...

# Run locally
./vergeos-exporter -verge.url="https://VERGEURL" -verge.username="admin" -verge.password="password"

# Create release (triggers CI)
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## Architecture

### Collector Pattern

Each collector implements `prometheus.Collector` interface:

```go
type Collector interface {
    Describe(ch chan<- *prometheus.Desc)
    Collect(ch chan<- prometheus.Metric)
}
```

Collectors embed `BaseCollector` for shared functionality:
- HTTP client with configurable timeout
- Basic authentication to VergeOS API
- TLS configuration (defaults to insecure for self-signed certs)
- System name retrieval from settings API

### API Integration

- All collectors fetch `cloud_name` from `/api/v4/settings` for metric labeling
- Uses VergeOS API v4 endpoints with JSON responses
- Types in `collectors/types.go` map directly to API response structures

### Adding a New Collector

1. Create `collectors/<resource>.go`
2. Embed `BaseCollector` in your struct
3. Define prometheus metrics in constructor with appropriate labels
4. Implement `Describe()` and `Collect()` methods
5. Register in `main.go` with `prometheus.MustRegister()`

## Conventions

### Metric Naming

- Prefix: `vergeos_`
- Format: `vergeos_<resource>_<measurement>`
- Examples: `vergeos_drive_read_ops`, `vergeos_vsan_tier_capacity`

### Label Naming

- `system_name`: VergeOS cloud name (from settings)
- `cluster`: Cluster display name
- `node_name`: Physical node name
- `tier`: VSAN tier number (0, 1, etc.)

### Counter vs Gauge

- **Counter**: cumulative values that only increase (ops, bytes, errors)
- **Gauge**: values that can go up or down (utilization, temperature, counts)

### Test Pattern

Tests use `httptest.Server` to mock VergeOS API responses:

```go
mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Route handling based on r.URL.Path
}))
defer mockServer.Close()
collector := NewCollector(mockServer.URL, mockServer.Client(), "user", "pass")
```

## VergeOS API Reference

Key endpoints used:
- `/api/v4/settings` - System settings including `cloud_name`
- `/api/v4/nodes?filter=physical eq true` - Physical node list
- `/api/v4/nodes/{id}?fields=dashboard` - Node details with stats
- `/api/v4/storage_tiers?fields=most` - VSAN tier overview
- `/api/v4/cluster_tiers?fields=all` - Detailed tier status
- `/api/v4/machine_drives` - Drive state information
- `/api/v4/clusters` - Cluster list and details

## Default Port

The exporter listens on `:9888` by default. Verify with:
```bash
curl -s http://localhost:9888/metrics
```
