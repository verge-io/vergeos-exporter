# VergeOS Exporter (ioMetrics)

A Prometheus exporter for VergeOS that collects metrics about VSAN tiers, clusters, nodes, and storage.

## Features

- VSAN Tier Metrics:
  - Capacity, usage, and allocation statistics
  - Transaction and repair counts
  - Drive status, temperature, and health monitoring
  - Comprehensive drive state monitoring (online, offline, repairing, initializing, verifying, noredundant, outofspace)
  - Node and drive availability tracking
  - Performance metrics (read/write operations, IOPS)
  - Storage pool utilization and redundancy status

- Cluster Metrics:
  - Total and online nodes
  - RAM, CPU, and disk utilization
  - Cluster health status and node synchronization

- Node Metrics:
  - CPU and memory usage per node
  - Network throughput and latency
  - Process and service status

- Virtual Network (VNet) Metrics:
  - Enabled/monitoring state per network
  - Gateway monitoring quality, latency, and packet-loss stats

## Metrics Format

The exporter supports both standard Prometheus text format and [OpenMetrics](https://openmetrics.io/) format via content negotiation. Prometheus 2.5.0+ will automatically request OpenMetrics format. Older scrapers continue to receive standard Prometheus text format — no configuration required.

## Installation

### Prebuilt Binaries

Prebuilt binaries for Linux, Windows, and macOS (both amd64 and arm64) are available on the [Releases](https://github.com/verge-io/vergeos-exporter/releases) page.

Note that the version number is included in the filename (e.g., vergeos-exporter_1.1.6_Darwin_x86_64.tar.gz), so ensure you download the correct version for your system.

1. Download the appropriate binary for your system
2. Extract the archive:
   ```bash
   # For Linux/macOS:
   tar xzf vergeos-exporter_Linux_x86_64.tar.gz
   # For Windows:
   # Extract the .zip file using Windows Explorer
   ```
3. Move the binary to your preferred location

### Docker

Multi-arch container images (amd64/arm64) are published to GitHub Container Registry:

```bash
docker pull ghcr.io/verge-io/vergeos-exporter:latest
```

```bash
docker run --rm -p 9888:9888 \
  ghcr.io/verge-io/vergeos-exporter:latest \
  -verge.url=https://your-vergeos-host \
  -verge.username=your-user \
  -verge.password=your-pass
```

Available tags:
- `ghcr.io/verge-io/vergeos-exporter:latest` — most recent release
- `ghcr.io/verge-io/vergeos-exporter:1.2.0` — specific version (no `v` prefix)

See `examples/docker-compose/` for a complete monitoring stack with Prometheus and Grafana.

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

- `-web.listen-address`: Address to listen on for web interface and telemetry (default: ":9888")
- `-web.telemetry-path`: Path under which to expose metrics
- `-verge.url`: VergeOS API URL (default: "https://localhost"). Also: `VERGE_URL` env var
- `-verge.username`: VergeOS API username (required). Also: `VERGE_USERNAME` env var
- `-verge.password`: VergeOS API password (required). Also: `VERGE_PASSWORD` env var
- `-scrape.timeout`: Timeout for scraping VergeOS API (default: 30s)

Environment variables are recommended over CLI flags in production to avoid exposing credentials in the process list.

### Example

```bash
./vergeos-exporter -verge.url="https://VERGEURL" -verge.username="admin" -verge.password="password"
```

### Permissions

Either a Normal or an API user can be used for the connecting user. Connecting user is required to have sufficient rights to query needed stats. Only list and read permissions to the cloud are required. MFA should be disabled. For more information on VergeOS permissions, please visit [Permissions](https://docs.verge.io/product-guide/system/permissions/)

### Connectivity

After the exporter is running, you may verify basic connectity and metrics are being exported via the VergeOS exporter HTTP endpoint by either opening a web browser to the configured port or running a curl command such as:
```bash
curl -s http://localhost:9888/metrics
```

## Metrics

See [metrics.md](metrics.md) for a complete list of exported metrics.

## Grafana Dashboard

A pre-configured Grafana dashboard is included in the `examples/grafana-dashboard.json` file. This dashboard provides a comprehensive visualization of VergeOS metrics including:

- VSAN tier performance and health metrics
- Cluster resource utilization and status
- Node health and performance indicators
- Storage metrics and drive status

To import the dashboard:

1. Open your Grafana instance
2. Click on the + icon in the side menu and select "Import"
3. Upload the `grafana-dashboard.json` file or copy its contents
4. Select your Prometheus data source
5. Click "Import" to finish


## Running as a Linux Service

To run the VergeOS Exporter as a systemd service on Linux:

1. Create a dedicated user for the exporter (optional but recommended):
```bash
sudo useradd -rs /bin/false vergeos_exporter
```

2. Copy the binary to a system location:
```bash
sudo cp vergeos-exporter /usr/local/bin/
sudo chown vergeos_exporter:vergeos_exporter /usr/local/bin/vergeos-exporter
```

3. Create a systemd service file at `/etc/systemd/system/vergeos-exporter.service`:
```ini
[Unit]
Description=VergeOS Exporter
After=network.target

[Service]
Type=simple
User=vergeos_exporter
Group=vergeos_exporter
ExecStart=/usr/local/bin/vergeos-exporter \
    -verge.url=https://VERGEURL \
    -verge.username=admin \
    -verge.password=PASSWORD

Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

4. Reload systemd and start the service:
```bash
sudo systemctl daemon-reload
sudo systemctl start vergeos-exporter
sudo systemctl enable vergeos-exporter
```

5. Check the service status:
```bash
sudo systemctl status vergeos-exporter
```

The exporter will now start automatically on system boot and restart if it crashes.

## Running as a Windows Service

The exporter has **built-in Windows service support** — no third-party wrapper is required. The `-service` flag registers, controls, and removes the service directly through the Windows Service Control Manager.

1. Create a directory for the exporter and copy the executable there:
```powershell
mkdir "C:\Program Files\vergeos-exporter"
copy vergeos-exporter.exe "C:\Program Files\vergeos-exporter"
```

2. Install the service (run **PowerShell or Command Prompt as Administrator**). Any exporter flags you pass here are stored and reused each time the service starts:
```powershell
cd "C:\Program Files\vergeos-exporter"
.\vergeos-exporter.exe -service=install `
  -verge.url="https://VERGEURL" `
  -verge.username="admin" `
  -verge.password="PASSWORD" `
  -log.file="C:\Program Files\vergeos-exporter\logs\exporter.log"
```
   The service is created with automatic startup, so it will launch on boot. Use `-log.file` to capture logs, since a Windows service has no console. Credentials can also be supplied via the `VERGE_URL`, `VERGE_USERNAME`, and `VERGE_PASSWORD` environment variables instead of flags.

3. Start the service:
```powershell
.\vergeos-exporter.exe -service=start
```

You can also manage the service using the Windows Service Manager:
- Open Services (services.msc)
- Find "VergeOS Exporter" in the list
- Right-click to Start, Stop, or Restart the service
- View service status and modify startup type

To stop or remove the service:
```powershell
.\vergeos-exporter.exe -service=stop
.\vergeos-exporter.exe -service=uninstall
```

The service will now start automatically when Windows boots. Logs can be found at the path given to `-log.file`.

## Running with Docker Compose demo/example

To quickly spin up and run the VergeOS Exporter, Prometheus, Grafana with a demo VergeOS dashboard. 

1. You can find a ready-to-run example under ```examples/docker-compose```, along with a README describing setup and usage.
This self-contained environment automatically retrieves the tagged binary release for your platform (x86_64 or arm64) and lets you explore VergeOS Exporter without a preconfigured Prometheus or Grafana instance. Docker and Docker Compose are prerequisites.

## Development

### Prerequisites

- Go 1.23 or higher
- Access to a VergeOS instance

### Building

```bash
go build
```

### Testing

Unit tests use mock servers and don't require a live VergeOS instance:

```bash
go test ./...
```

Integration tests require credentials. Set them via environment variables or a `.env` file in the repo root (gitignored):

```bash
VERGE_URL=https://your-instance.example.com
VERGE_USERNAME=your-user
VERGE_PASSWORD=your-pass
```

Then run with the `integration` build tag:

```bash
go test -tags integration ./tests -v
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
   - Build and push multi-arch Docker images to GHCR
