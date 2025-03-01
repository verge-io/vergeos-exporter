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

- `-web.listen-address`: Address to listen on for web interface and telemetry (default: ":9888")
- `-web.telemetry-path`: Path under which to expose metrics
- `-verge.url`: VergeOS API URL (default: "http://localhost")
- `-verge.username`: VergeOS API username (required)
- `-verge.password`: VergeOS API password (required)
- `-scrape.timeout`: Timeout for scraping VergeOS API (default: 30s)

### Example

```bash
./vergeos-exporter -verge.url="http://VERGEURL" -verge.username="admin" -verge.password="password"
```

## Metrics

See [metrics.md](metrics.md) for a complete list of exported metrics.

## Running as a Linux Service

To run the VergeOS Exporter as a systemd service on Linux:

1. Create a dedicated user for the exporter (optional but recommended):
```bash
sudo useradd -rs /bin/false vergeos_exporter
```

2. Copy the binary to a system location:
```bash
sudo cp vergeos_exporter /usr/local/bin/
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

To run the VergeOS Exporter as a Windows service, we'll use NSSM (Non-Sucking Service Manager):

1. Download NSSM from the [official website](https://nssm.cc/download)

2. Extract the NSSM archive and copy the appropriate executable (nssm.exe) to a permanent location:
   - Use `nssm64.exe` for 64-bit systems (recommended)
   - Copy it to `C:\Program Files\nssm\nssm.exe`

3. Create a directory for the exporter:
```powershell
mkdir "C:\Program Files\vergeos-exporter"
```

4. Copy the vergeos-exporter executable to this directory:
```powershell
copy vergeos-exporter.exe "C:\Program Files\vergeos-exporter"
```

5. Install the service using NSSM (run Command Prompt as Administrator):
```batch
nssm install VergeOSExporter "C:\Program Files\vergeos-exporter\vergeos-exporter.exe"
nssm set VergeOSExporter AppParameters "-verge.url=http://VERGEURL -verge.username=admin -verge.password=PASSWORD"
nssm set VergeOSExporter DisplayName "VergeOS Exporter"
nssm set VergeOSExporter Description "Prometheus exporter for VergeOS metrics"
nssm set VergeOSExporter Start SERVICE_AUTO_START
nssm set VergeOSExporter ObjectName LocalSystem
nssm set VergeOSExporter AppStdout "C:\Program Files\vergeos-exporter\logs\stdout.log"
nssm set VergeOSExporter AppStderr "C:\Program Files\vergeos-exporter\logs\stderr.log"
```

6. Start the service:
```batch
nssm start VergeOSExporter
```

You can also manage the service using Windows Service Manager:
- Open Services (services.msc)
- Find "VergeOS Exporter" in the list
- Right-click to Start, Stop, or Restart the service
- View service status and modify startup type

To remove the service:
```batch
nssm remove VergeOSExporter confirm
```

The service will now start automatically when Windows boots. Logs can be found in the specified log directory.

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

