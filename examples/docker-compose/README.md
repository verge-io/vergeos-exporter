# VergeOS Prometheus & Grafana Monitoring Stack

A complete monitoring solution for VergeOS environments using Prometheus, Grafana, and the VergeOS exporter. This Docker Compose stack provides out-of-the-box monitoring with minimal configuration required.

## Overview

This stack deploys three interconnected services:

- **VergeOS Exporter** - Collects metrics from your VergeOS environment
- **Prometheus** - Time-series database for storing and querying metrics
- **Grafana** - Visualization and dashboards for monitoring data

> **Important:** This stack is designed for demonstration and testing purposes. It is not intended for direct use in production environments, which require additional security hardening, proper credential management, and other production-grade configurations. However, it may serve as a base configuration to get started.

## Prerequisites

- Docker and Docker Compose installed
  - Linux/macOS: Docker Engine or Docker Desktop
  - Windows: Docker Desktop with WSL2 backend
- Access to a VergeOS environment
- VergeOS credentials with appropriate monitoring permissions

## Quick Start

### Option 1: Using the Start Script (Recommended)

1. **Configure your environment**

   Edit the `.env` file with your VergeOS connection details:

   ```bash
   VERGE_URL=https://your-vergeos-host
   VERGE_USERNAME=your-username
   VERGE_PASSWORD=your-password
   GRAFANA_ADMIN_PASSWORD=secure-password
   ```

   > **Note:** The VergeOS user does not require admin or privileged permissions. A read-only account with access to system metrics is sufficient.

2. **Run the start script**

   **Linux/macOS:**
   ```bash
   ./start.sh
   ```

   **Windows (PowerShell):**
   ```powershell
   .\start.ps1
   ```

   This script will:
   - Pull the latest images
   - Build the VergeOS exporter
   - Start all services
   - Display access URLs for all services (both localhost and network IP)

### Option 2: Manual Docker Compose Commands

1. **Configure your environment** (same as above)

2. **Pull the latest images**

   ```bash
   docker compose pull
   ```

3. **Build the VergeOS exporter**

   ```bash
   docker compose build --no-cache --pull vergeos-exporter
   ```

4. **Start the stack**

   ```bash
   docker compose up -d
   ```

## Accessing the Services

Once running, you can access:

- **Grafana**: http://localhost:3000
  - Default username: `admin`
  - Password: Set in `.env` (default: `admin`)

- **Prometheus**: http://localhost:9090
  - Query and explore metrics directly

- **VergeOS Exporter**: http://localhost:9888/metrics
  - Raw Prometheus-formatted metrics

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VERGE_URL` | VergeOS instance URL | Required |
| `VERGE_USERNAME` | VergeOS username | Required |
| `VERGE_PASSWORD` | VergeOS password | Required |
| `EXPORTER_VERSION` | Exporter version to use | `1.1.9` |
| `EXPORTER_ARCH` | Architecture (`x86_64` or `arm64`) | `arm64` |
| `GRAFANA_ADMIN_PASSWORD` | Grafana admin password | `admin` |
| `WEB_LISTEN` | Exporter listen address | `:9888` |

### Architecture Selection

The `EXPORTER_ARCH` variable should match your host system:
- `x86_64` - For Intel/AMD processors
- `arm64` - For Apple Silicon (M1/M2/M3) or AWS Graviton

### Data Retention

Prometheus is configured with:
- **Retention time**: 15 days
- **Storage limit**: 30GB

Modify these in `docker-compose.yml` under the `prometheus` service commands if needed.

## Project Structure

```
.
├── docker-compose.yml           # Main orchestration file
├── .env                         # Environment configuration
├── prometheus/
│   └── prometheus.yml          # Prometheus scrape configuration
├── grafana/
│   └── provisioning/
│       ├── datasources/        # Pre-configured Prometheus datasource
│       └── dashboards/         # Dashboard definitions
└── vergeos-exporter/
    └── Dockerfile              # Exporter container build
```

## Managing the Stack

### View logs

```bash
docker compose logs -f
```

### View logs for a specific service

```bash
docker compose logs -f grafana
docker compose logs -f prometheus
docker compose logs -f vergeos-exporter
```

### Stop the stack

```bash
docker compose down
```

### Stop and remove volumes (clean slate)

```bash
docker compose down -v
```

### Restart a specific service

```bash
docker compose restart vergeos-exporter
```

### Rebuild after changes

```bash
docker compose up -d --build
```

## Troubleshooting

### Exporter Connection Issues

If the exporter can't connect to VergeOS:

1. Verify the `VERGE_URL` is correct and accessible
2. Check credentials in `.env`
3. Review exporter logs: `docker compose logs vergeos-exporter`

### No Data in Grafana

1. Verify Prometheus is scraping the exporter:
   - Visit http://localhost:9090/targets
   - Check that `vergeos-exporter` shows as "UP"
2. Test the exporter directly: http://localhost:9888/metrics
3. Check datasource configuration in Grafana

### Permission Issues

Ensure the VergeOS user has sufficient permissions to query system metrics.

## Upgrading the Exporter

To upgrade to a new version:

1. Update `EXPORTER_VERSION` in `.env`
2. Rebuild and restart:

   ```bash
   docker compose build --no-cache vergeos-exporter
   docker compose up -d vergeos-exporter
   ```

## Data Persistence

Persistent volumes are created for:
- `prometheus_data` - Metrics storage
- `grafana_data` - Dashboards and settings

These survive container restarts and stack restarts (unless `-v` is used with `docker compose down`).

## Security Considerations

- Change default passwords in `.env` before production use
- Consider using Docker secrets for sensitive credentials
- Restrict network access to monitoring ports as needed
- Use HTTPS for VergeOS connections

## Contributing

When making changes:
1. Test with `docker compose config` to validate YAML syntax
2. Document any new environment variables
3. Update this README with new features or configuration options

## License

See LICENSE file for details.
