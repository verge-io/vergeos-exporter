# VergeOS Exporter Tests

This directory contains tests for the VergeOS exporter.

## Drive State Monitoring Test

The `drive_state_test.go` file contains a test for the drive state monitoring functionality. This test verifies that all drive states (online, offline, repairing, initializing, verifying, noredundant, outofspace) are properly tracked by the exporter.

### Running the Test

You can run the test using the provided shell script:

```bash
./run_drive_state_test.sh --verge-url=https://your-vergeos-instance --verge-user=admin --verge-pass=password
```

#### Options

- `--exporter-url=URL`: URL of the VergeOS exporter (default: http://localhost:9888)
- `--metrics-path=PATH`: Path to metrics endpoint (default: /metrics)
- `--verge-url=URL`: VergeOS API URL (required)
- `--verge-user=USER`: VergeOS API username (required)
- `--verge-pass=PASS`: VergeOS API password (required)
- `--start-exporter`: Start the exporter as part of the test
- `--help`: Show help message

### Test Procedure

The test performs the following steps:

1. Connects to the VergeOS exporter (or starts it if `--start-exporter` is specified)
2. Fetches metrics from the exporter
3. Parses the drive state metrics
4. Prints a table of drive states for each tier
5. Verifies that all expected drive states are present for each tier

### Expected Output

The test will output a table of drive states for each tier, showing the count of drives in each state:

```
Drive State Metrics:
------------------------------------------------------------
System          Tier       State           Count     
------------------------------------------------------------
test-system     0          online          3         
test-system     0          offline         1         
test-system     0          repairing       1         
test-system     0          initializing    1         
test-system     0          verifying       1         
test-system     0          noredundant     0         
test-system     0          outofspace      0         
test-system     1          online          4         
test-system     1          offline         0         
test-system     1          repairing       0         
test-system     1          initializing    0         
test-system     1          verifying       0         
test-system     1          noredundant     1         
test-system     1          outofspace      1         
------------------------------------------------------------

Verifying all states are present:
All expected states are present for all tiers.
```

If any states are missing, the test will output warnings:

```
Verifying all states are present:
WARNING: State 'repairing' is missing for tier 1
WARNING: State 'initializing' is missing for tier 1
```

## Unit Tests

The unit tests for the drive state monitoring functionality are in `collectors/storage_test.go`. These tests use mock API responses to verify that the exporter correctly counts drives in each state.

To run the unit tests:

```bash
go test -v ./collectors