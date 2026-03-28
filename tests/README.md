# VergeOS Exporter Tests

This directory contains unit tests for the VergeOS exporter collectors.

## Test Structure

```
tests/
├── testhelpers.go     # Shared test utilities and mock types
├── storage_test.go    # VSAN tier metrics tests
├── node_test.go       # Node metrics tests
├── cluster_test.go    # Cluster metrics tests
├── network_test.go    # Network collector tests (info metric only due to SDK gaps)
└── system_test.go     # System version metrics tests
```

## Running Tests

Run all tests:

```bash
go test ./tests/...
```

Run with verbose output:

```bash
go test -v ./tests/...
```

Run a specific test:

```bash
go test -v ./tests/... -run TestStorageTierMetrics
```

## Test Coverage

The tests cover:

### Storage Collector (`storage_test.go`)
- VSAN tier capacity, usage, and allocation metrics
- Tier encryption and redundancy status
- Bad drives and fullwalk progress
- **Bug #27**: Phantom tier filtering (non-contiguous tiers)
- **Bug #28**: Stale metrics prevention (status transitions)
- Edge case: No tiers configured

### Node Collector (`node_test.go`)
- Physical node enumeration by cluster
- IPMI status reporting
- RAM total and allocation metrics
- Stale metrics prevention when nodes are removed
- Multiple cluster support

### Cluster Collector (`cluster_test.go`)
- Cluster count and status metrics
- Health status (online state detection)
- Enabled/disabled status
- RAM, cores, and running machines metrics
- Stale metrics prevention
- Offline cluster detection

### Network Collector (`network_test.go`)
- Info metric emission (SDK gaps prevent NIC metrics)
- Descriptor count verification
- Verification that NIC metrics are not emitted

### System Collector (`system_test.go`)
- Version and hash metrics
- Stale metrics prevention on version changes
- Different version format handling

## Test Helpers

The `testhelpers.go` file provides shared utilities:

- `CreateTestSDKClient()` - Creates an SDK client for testing
- `NewBaseMockServer()` - Creates a mock HTTP server with version/settings support
- Mock types for API responses (StorageTierMock, ClusterTierMock, etc.)
- Helper functions for JSON response writing and auth checking

## SDK Gaps

Some metrics are not tested because they cannot be implemented due to SDK gaps:

- **Drive metrics**: SDK lacks `node_display` and `statuslist` fields
- **NIC metrics**: SDK doesn't capture dashboard NIC data
- **Node dashboard metrics**: SDK doesn't expose `machine.stats` fields

See `.claude/GAPS.md` for comprehensive documentation of SDK limitations.
