# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Prometheus exporter for VergeOS that collects metrics about VSAN tiers, drives, clusters, nodes, network, and system info. Written in Go, uses the `goVergeOS` SDK for all API calls.

## Commands

```bash
# Build
go build

# Run all tests
go test ./...

# Run a single test
go test ./tests -run TestStorageTierMetrics -v

# Run locally (requires a VergeOS instance)
./vergeos-exporter -verge.url="https://VERGEURL" -verge.username="admin" -verge.password="password"

# Verify metrics endpoint
curl -s http://localhost:9888/metrics
```

## Architecture

### SDK-Based Collectors

All collectors use the `goVergeOS` SDK client (`vergeos.Client`) — no direct HTTP calls. The SDK is a local dependency via `replace` directive in `go.mod` (points to `../goVergeOS`).

**Data flow:** `main.go` creates one `vergeos.Client` → passes it to each collector constructor → collectors call SDK service methods (e.g., `client.StorageTiers.List(ctx)`) in their `Collect()` method.

### Collector Structure

Every collector embeds `BaseCollector` (which holds the SDK client and caches `systemName`). The pattern:

1. Constructor (`NewXxxCollector`) defines `prometheus.Desc` descriptors with label sets
2. `Describe()` sends all descriptors to the channel
3. `Collect()` fetches data via SDK, emits metrics with `prometheus.MustNewConstMetric`

The `MustNewConstMetric` pattern (vs persistent metric objects) is intentional — it prevents stale label values from persisting across scrapes (Bug #28).

### Collectors

| Collector | File | SDK Services Used |
|-----------|------|-------------------|
| Storage | `collectors/storage.go` | `StorageTiers`, `ClusterTiers`, `MachineDrivePhys`, `MachineDriveStats` |
| Node | `collectors/node.go` | `Nodes`, `MachineStats` |
| Cluster | `collectors/cluster.go` | `Clusters`, `ClusterStatus` |
| Network | `collectors/network.go` | `MachineNICs` |
| System | `collectors/system.go` | `Settings`, `UpdateSettings`, `UpdateSourcePackages` |

### Test Pattern

Tests live in `tests/` (separate package). They use `httptest.Server` to mock the VergeOS API, with shared helpers in `tests/testhelpers.go`:

- `NewBaseMockServer()` — creates a mock that handles version check + settings + auth, with a callback for resource-specific routes
- `CreateTestSDKClient()` — creates an SDK client pointed at the mock server
- Tests verify metrics using `prometheus/testutil` (count metrics, check values)

Mock types in `testhelpers.go` mirror the API JSON shapes — update them when the API changes.

## Key Conventions

### Metric Naming

- Prefix: `vergeos_`
- Format: `vergeos_<resource>_<measurement>`
- All metrics include `system_name` label (the VergeOS cloud name)
- Booleans become gauges: `1.0` = true, `0.0` = false (use `boolToFloat64()`)

### Known Bug Patterns

- **Bug #27**: Phantom tiers — `cluster_tiers` can return tiers not in `storage_tiers`. Filter using a `validTiers` set.
- **Bug #28**: Stale labels — always use `MustNewConstMetric`, never persistent metric objects.
- **Bug #34**: Fail-fast auth — credentials are validated at startup before registering collectors.

### Release

Uses GoReleaser (`.goreleaser.yml`) with GitHub Actions for cross-platform builds. Tag with `v*` to trigger.

## Port

Default listen address: `:9888`
