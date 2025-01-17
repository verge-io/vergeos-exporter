# VergeOS Exporter

A Prometheus exporter for VergeOS that collects metrics about VSAN tiers, clusters, and nodes.

## Features

- VSAN Tier Metrics:
  - Capacity, usage, and allocation statistics
  - Transaction and repair counts
  - Drive status and temperature monitoring
  - Node and drive availability tracking
  - Performance metrics (read/write operations)

- Cluster Metrics:
  - Total and online nodes
  - RAM and CPU utilization
  - Running machines statistics

## Installation

### Prebuilt Binaries

Prebuilt binaries for Linux, Windows, and macOS (both amd64 and arm64) are available on the [Releases](https://github.com/verge-io/vergeos-exporter/releases) page.

1. Download the appropriate binary for your system
2. Extract the archive:
   ```bash
   # For Linux/macOS:
   tar xzf vergeos-exporter_Linux_x86_64.tar.gz
   # For Windows:
   # Extract the .zip file using Windows Explorer
   ```
3. Move the binary to your preferred location

### Building from Source

If you prefer to build from source:

1. Clone the repository
2. Build the exporter:
```bash
go build -o vergeos-exporter
```

## Usage

```bash
./vergeos-exporter [flags]
```

### Flags

- `-web.listen-address`: Address to listen on for web interface and telemetry (default: ":9100")
- `-vergeos.url`: VergeOS API URL (required)
- `-vergeos.username`: VergeOS API username (required)
- `-vergeos.password`: VergeOS API password (required)

### Example

```bash
./vergeos-exporter -vergeos.url="http://vergeos-server" -vergeos.username="admin" -vergeos.password="password"
```

## Metrics

See [metrics.md](metrics.md) for a complete list of exported metrics.

## Development

### Prerequisites

- Go 1.21 or higher
- Access to a VergeOS instance

### Building

```bash
go build
```

### Testing

```bash
go test ./...
```

### Creating a Release

1. Tag the release:
```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

2. GitHub Actions will automatically:
   - Build binaries for all supported platforms
   - Create a new GitHub release
   - Upload the binaries and checksums

## License

This project is licensed under the MIT License - see the LICENSE file for details.
