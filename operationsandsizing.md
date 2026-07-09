# Operations and Sizing Guide

This guide covers production deployment, sizing, security, and ongoing operations for the VergeOS Exporter and the Prometheus + Grafana stack it feeds. For exporter installation basics and a full metrics reference, see [README.md](README.md) and [metrics.md](metrics.md).

## Overview

Key points to orient before reading further:

- The bundled Docker Compose stack is a **reference example** to get you started — production deployments need additional security and operational work (see the callout below).
- The VergeOS Exporter publishes VSAN, cluster, node, drive, network, tenant, VM, and system metrics in Prometheus format.
- A single Docker Compose stack (exporter + Prometheus + Grafana) handles up to ~1,000 VMs from one VergeOS cloud.
- Multi-cloud monitoring is achieved by running one exporter container per cloud and pointing a shared Prometheus at all of them.
- Sizing is driven primarily by VM count and scrape interval; SSD storage matters more than extra RAM.

> ⚠️ **Reference deployment, not a turnkey production stack**
>
> The bundled Docker Compose stack is provided as an example and starting point to get you collecting metrics quickly. It is intentionally simple — single host, default ports bound to all interfaces, file-based credentials, no external TLS termination, no high availability.
>
> Production deployments will need additional work appropriate to your environment, including but not limited to:
>
> - Hardened secret management (Vault, cloud secret manager, Docker/Kubernetes secrets) instead of `.env` files
> - TLS for Grafana, Prometheus, and the exporter, typically via a reverse proxy
> - SSO or directory-backed authentication for Grafana, with the default admin disabled
> - Network segmentation, firewall rules, and host-based access controls
> - Backup, retention, and disaster-recovery procedures aligned to your RPO/RTO
> - High availability or remote-write to a long-term store (Thanos, Mimir) where uptime requirements demand it
> - Compliance, logging, and audit requirements specific to your organization
>
> Treat the sections below as a baseline. Validate every choice against your own security, compliance, and availability standards before deploying in production.

## Prerequisites

### VergeOS user

Create a dedicated monitoring user in each VergeOS cloud:

- **Permissions:** list and read access to the cloud only
- **User type:** Normal or API user
- **MFA:** must be disabled (the exporter uses basic auth)
- **Scope:** no admin or privileged permissions required

See [VergeOS Permissions](https://docs.verge.io/product-guide/system/permissions/) for details.

### Connectivity

| From | To | Port | Protocol | Purpose |
|------|------|------|----------|---------|
| Exporter host | VergeOS cloud | 443 | HTTPS | API scraping |
| Prometheus | Exporter | 9888 | HTTP | Metrics scrape |
| Grafana | Prometheus | 9090 | HTTP | Queries |
| Users | Grafana | 3000 | HTTP/HTTPS | Dashboard access |

### Topology

| Pattern | When to use |
|---------|-------------|
| Single-host stack | One VergeOS cloud, up to ~1000 VMs |
| Single Prometheus, multiple exporters | Multiple VergeOS clouds, centralized metrics |
| Federated Prometheus | Multiple sites with local Prometheus, aggregated centrally |
| Remote write (Thanos/Mimir) | Long-term retention beyond 15 days |

## Deployment Walkthrough

The fastest path to a working stack is the bundled Docker Compose example under `examples/docker-compose/` — see that directory's README for a quickstart. The steps below assume that example as the starting point.

### Clone and configure

```bash
git clone https://github.com/verge-io/vergeos-exporter.git
cd vergeos-exporter/examples/docker-compose
```

Edit `.env` with values for your environment:

| Variable | Description | Example |
|----------|-------------|---------|
| `VERGE_URL` | VergeOS cloud URL | `https://vergeos.example.com` |
| `VERGE_USERNAME` | Monitoring user | `prom-monitor` |
| `VERGE_PASSWORD` | Monitoring password | strong password |
| `INSECURE` | Skip TLS verify (self-signed certs) | `true` or omit |
| `EXPORTER_VERSION` | Exporter image tag | `latest` or `1.2.0` |
| `EXPORTER_ARCH` | `x86_64` or `arm64` | `arm64` |
| `GRAFANA_ADMIN_PASSWORD` | Grafana admin password | strong password |

### Start the stack

```bash
docker compose pull
docker compose up -d
```

Or use the convenience scripts:

```bash
./start.sh    # Linux/macOS
.\start.ps1   # Windows
```

### Verify

```bash
# Exporter authenticated and running
docker compose logs vergeos-exporter | grep "Successfully connected"

# Metrics endpoint responds with vergeos_* series
curl -s http://localhost:9888/metrics | grep -c "^vergeos_"

# Prometheus is scraping the exporter
curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.labels.job=="vergeos") | {instance, health, lastError}'

# System name shows up as a Prometheus label
curl -s http://localhost:9090/api/v1/label/system_name/values
```

> 💡 **End-to-end health check:** the `system_name` label is populated automatically from the VergeOS cloud name. If `/api/v1/label/system_name/values` returns your cloud name, the entire path (cloud → exporter → Prometheus) is healthy.

## Sizing and Resource Planning

### CPU and RAM

| VM count | Prometheus | Grafana | Exporter | Host minimum |
|----------|-----------|---------|----------|--------------|
| < 100 | 1 core / 1 GB | 1 core / 512 MB | 1 core / 256 MB | 2 cores / 4 GB |
| 100–500 | 1 core / 2 GB | 1 core / 512 MB | 1 core / 256 MB | 2 cores / 4 GB |
| 500–1000 | 2 cores / 4 GB | 1 core / 512 MB | 1 core / 512 MB | 4 cores / 8 GB |
| 1000+ | 4 cores / 8 GB | 2 cores / 1 GB | 2 cores / 1 GB | 8 cores / 16 GB |

SSD storage is more important than extra RAM — Prometheus does a lot of random reads during queries.

### Storage

Daily data generated depends on series count:

| VM count per cloud | Series | MB/day | 15-day retention |
|--------------------|--------|--------|------------------|
| < 100 | ~1,000 | ~2 MB | ~30 MB |
| ~500 | ~15,000 | ~30 MB | ~450 MB |
| ~1000 | ~36,000 | ~75 MB | ~1.1 GB |

For multi-cloud deployments, sum the per-cloud estimates.

### Retention

Set retention in `docker-compose.yml` under the `prometheus` service:

```yaml
command:
  - --storage.tsdb.retention.time=15d
  - --storage.tsdb.retention.size=30GB
```

Whichever limit is hit first triggers pruning. Prometheus does not downsample — it keeps full resolution and deletes old blocks.

### Scrape interval

| Interval | Resolution | API load | Use case |
|----------|-----------|----------|----------|
| 15s | High | High | Small environments, real-time ops |
| 30s | Good | Medium | Standard deployments |
| 60s | Acceptable | Low | Recommended for 500+ VMs |
| 120s | Coarse | Very low | Long-term trending, very large clouds |

Set scrape timeout greater than or equal to scrape interval. For 1000+ VM environments, `scrape_timeout: 60s` is recommended — exporter scrapes can spike to 30s during heavy collection cycles.

In `prometheus.yml`:

```yaml
global:
  scrape_interval: 60s
scrape_configs:
  - job_name: vergeos
    scrape_timeout: 60s
    static_configs:
      - targets: ['vergeos-exporter:9888']
```

## Security Hardening

### Credentials

- Store credentials in `.env` with filesystem permissions `600`
- Never commit `.env` to version control — add to `.gitignore`
- For production, consider Docker secrets or an external secret manager (Vault, AWS Secrets Manager)
- Rotate monitoring passwords periodically
- Use separate credentials per environment (dev/staging/prod)

### Least privilege

- The monitoring user requires only list and read permissions to the cloud
- Do not use admin or root accounts for the exporter
- Create a dedicated `prom-monitor` or `verge-monitoring` user per VergeOS cloud

### Network exposure

By default, the stack binds all services to `0.0.0.0`. For production:

- Restrict Grafana (port 3000) to trusted networks via firewall
- Put Grafana behind a reverse proxy with TLS (nginx, Caddy, Traefik)
- Keep Prometheus (9090) and the exporter (9888) internal-only
- Change the default Grafana admin password before first login
- Disable Grafana anonymous access (default behavior; verify)

### Image pinning

Replace `:latest` tags with specific versions for reproducibility:

```yaml
prometheus:
  image: prom/prometheus:v2.54.1
grafana:
  image: grafana/grafana-oss:11.3.0
vergeos-exporter:
  image: ghcr.io/verge-io/vergeos-exporter:1.2.0
```

Review and update pins quarterly or when CVEs are published.

## Multi-Tenant and Multi-Site Patterns

For monitoring multiple VergeOS clouds, run one exporter container per cloud and point Prometheus at all of them. Grafana dashboards filter by the `system_name` label (populated by the exporter from the VergeOS cloud name), so all sites appear in the same dashboard with a site selector.

### Additional exporter service

```yaml
vergeos-exporter-site2:
  image: ghcr.io/verge-io/vergeos-exporter:${EXPORTER_VERSION:-latest}
  container_name: vergeos-exporter-site2
  restart: unless-stopped
  command:
    - "-verge.url=https://cloud2.example.com"
    - "-verge.username=${SITE2_USERNAME}"
    - "-verge.password=${SITE2_PASSWORD}"
    - "-web.listen-address=:9888"
    - "-insecure=${SITE2_INSECURE:-false}"
  ports:
    - "9889:9888"
  networks:
    - monitoring
```

### Multi-target scrape config

```yaml
scrape_configs:
  - job_name: vergeos
    scrape_interval: 60s
    scrape_timeout: 60s
    static_configs:
      - targets:
          - vergeos-exporter:9888
          - vergeos-exporter-site2:9888
          - vergeos-exporter-site3:9888
```

## Alerting Baseline

Recommended alert rules — configure in Grafana under Alerting, or in a Prometheus rules file:

| Alert | Condition | Severity | Notes |
|-------|-----------|----------|-------|
| VSAN fill critical | `vergeos_vsan_tier_used_pct > 90` | Critical | Data loss risk |
| VSAN fill warning | `vergeos_vsan_tier_used_pct > 75` | Warning | Plan expansion |
| VSAN not redundant | `vergeos_vsan_redundant == 0` | Critical | No fault tolerance |
| Bad drives detected | `vergeos_vsan_bad_drives > 0` | Warning | Schedule replacement |
| Cluster offline | `vergeos_cluster_status == 0` | Critical | Immediate action |
| Node offline | `vergeos_cluster_online_nodes < vergeos_cluster_total_nodes` | Warning | Investigate node |
| CPU overcommit | `(vergeos_cluster_used_cores / vergeos_cluster_online_cores) * 100 > 300` | Warning | Performance impact |
| RAM N+1 violation | `vergeos_cluster_used_ram / vergeos_cluster_total_ram > 0.85` | Warning | Losing a node would be tight |
| Scrape failing | `up{job="vergeos"} == 0` | Critical | Monitoring blind spot |
| Scrape duration high | `scrape_duration_seconds{job="vergeos"} > 30` | Warning | API slow or sized wrong |
| Drive temperature high | `vergeos_drive_temperature > 55` | Warning | Cooling issue |
| Drive repairs active | `vergeos_vsan_tier_repairs > 0` | Info | Rebuild in progress |

Grafana's built-in alerting handles rule evaluation, routing, and notifications via email, Slack, PagerDuty, webhook, and others — no separate Alertmanager required.

## Troubleshooting

### Useful commands

```bash
# Service status
docker compose ps

# Logs
docker compose logs -f vergeos-exporter
docker compose logs -f prometheus
docker compose logs -f grafana

# Exporter metrics endpoint
curl -s http://localhost:9888/metrics | grep -c "^vergeos_"

# Prometheus targets and errors
curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {instance, health, lastError, lastScrapeDuration}'

# Which systems are reporting
curl -s http://localhost:9090/api/v1/label/system_name/values

# Scrape duration stats
curl -s 'http://localhost:9090/api/v1/query?query=scrape_duration_seconds{job="vergeos"}'

# Reload Prometheus config (requires --web.enable-lifecycle)
curl -X POST http://localhost:9090/-/reload

# Restart a single service
docker compose restart vergeos-exporter
```

### Common issues

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| Exporter logs "Authentication failed" | Bad credentials or MFA enabled | Verify `.env`, disable MFA on monitoring user |
| Exporter logs TLS certificate verification error | Self-signed certificate | Set `INSECURE=true` in `.env` |
| Prometheus target shows `health=down` | Network, DNS, firewall | Check `docker compose logs prometheus`, verify exporter URL |
| Grafana shows "Datasource not found" | Datasource uid mismatch | Verify datasource uid matches dashboard references |
| Grafana panels empty | Wrong time range or system variable | Check dashboard variables default to "All" |
| Scrape timeouts or gaps in data | Large environment, slow API | Increase `scrape_interval` and `scrape_timeout` to 60s |
| Dashboard panels show literal `${DS_PROMETHEUS}` | Imported without substitution | Use the entrypoint sed pattern or manual import |
| Volume not persisting after `down -v` | Expected — `-v` removes volumes | Omit `-v` unless you want a clean slate |

## Upgrade Procedure

### Routine upgrade

```bash
docker compose pull
docker compose up -d
```

Grafana and Prometheus handle in-place upgrades cleanly. The exporter will restart and reconnect.

### Major version upgrade

1. Take a backup (see Backup and Restore below)
2. Review release notes for breaking changes
3. Pin the new version in `.env` or `docker-compose.yml`:
   ```bash
   EXPORTER_VERSION=2.0.0
   ```
4. Pull and apply:
   ```bash
   docker compose pull
   docker compose up -d
   ```
5. Verify:
   ```bash
   docker compose logs -f vergeos-exporter
   curl -s http://localhost:9888/metrics | head
   ```

### Rollback

```bash
# Pin previous version
EXPORTER_VERSION=1.2.0
docker compose pull vergeos-exporter
docker compose up -d vergeos-exporter
```

Prometheus TSDB format is backward-compatible within the same major version but not forward-compatible — rolling back Prometheus itself may require restoring from backup.

## Backup and Restore

### What to back up

- **Prometheus data** — `/prometheus` (TSDB blocks, WAL, head chunks)
- **Grafana data** — `/var/lib/grafana` (custom dashboards, users, settings)
- **Configuration** — `.env`, `docker-compose.yml`, `prometheus/`, `grafana/provisioning/` (commit to version control)

### Full backup (recommended)

Stops the stack briefly but captures a consistent state including the WAL:

```bash
docker compose stop prometheus grafana

docker cp prometheus:/prometheus ./backup/prometheus-$(date +%Y%m%d)
docker cp grafana:/var/lib/grafana ./backup/grafana-$(date +%Y%m%d)

docker compose start prometheus grafana

tar -czf backup-$(date +%Y%m%d).tar.gz backup/
```

### Snapshot backup (zero downtime, compacted blocks only)

Requires `--web.enable-admin-api` on Prometheus (add to the `command:` list in `docker-compose.yml`). Captures blocks but not the WAL — may miss the last ~2 hours of data.

```bash
SNAP=$(curl -s -X POST http://localhost:9090/api/v1/admin/tsdb/snapshot | jq -r '.data.name')
docker cp prometheus:/prometheus/snapshots/$SNAP ./backup/$SNAP
```

### Restore

```bash
# Stop services — copying to a running Prometheus risks corruption
docker compose stop prometheus grafana

# Optional: clear existing data first
docker compose down -v
docker compose up -d --no-start prometheus grafana

# Copy data back in
docker cp ./backup/prometheus-20260410/. prometheus:/prometheus/
docker cp ./backup/grafana-20260410/. grafana:/var/lib/grafana/

docker compose start prometheus grafana

# Verify
curl -s http://localhost:9090/api/v1/label/system_name/values
```

### Migrating to a new host

1. Full backup on the source host
2. Copy the backup archive and configuration files (`.env`, `docker-compose.yml`, provisioning dirs) to the target
3. Install Docker and Docker Compose on the target
4. Restore data per the Restore section
5. Update DNS or external references to point to the new host
6. Verify scraping and dashboards before decommissioning the source

### Backup schedule

| Frequency | Method | Retention |
|-----------|--------|-----------|
| Daily | Full backup during maintenance window | 7 days |
| Weekly | Full backup, archive offsite | 4 weeks |
| Monthly | Full backup, archive offsite | 12 months |
| Before upgrade | Full backup | Until upgrade validated |

## Additional Resources

- [README.md](README.md) — exporter installation, flags, and basic usage
- [metrics.md](metrics.md) — full list of exported `vergeos_*` metrics
- [examples/docker-compose/README.md](examples/docker-compose/README.md) — bundled stack quickstart
- [VergeOS Permissions](https://docs.verge.io/product-guide/system/permissions/) — creating the dedicated monitoring user
- [Prometheus documentation](https://prometheus.io/docs/) — upstream reference for retention, alerting, and remote-write
- [Grafana documentation](https://grafana.com/docs/grafana/latest/) — upstream reference for dashboards, alerting, and authentication
