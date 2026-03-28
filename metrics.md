# Metrics included in VergeOS Exporter

## List of Physical Nodes
- **Total Physical Nodes**: `vergeos_nodes_total` (Gauge, labeled by `system_name` and `cluster`)
- **IPMI Status per Node**: `vergeos_node_ipmi_status` (Gauge, labeled by `system_name`, `cluster`, and `node_name`)

---
## Node Details and Stats

### CPU Metrics
- **CPU Usage per Core**: `vergeos_node_cpu_core_usage` (Gauge, labeled by `system_name`, `cluster`, `node_name`, and `core_id`)
- **CPU Temperature**: `vergeos_node_core_temp` (Gauge, labeled by `system_name`, `cluster`, and `node_name`)
- **Running Cores**: `vergeos_node_running_cores` (Gauge, labeled by `system_name`, `cluster`, and `node_name`)

---
### Memory Metrics
- **RAM Used (MB)**: `vergeos_node_ram_used` (Gauge, labeled by `system_name`, `cluster`, and `node_name`)
- **RAM Usage Percentage**: `vergeos_node_ram_pct` (Gauge, labeled by `system_name`, `cluster`, and `node_name`)
- **VM RAM (MB)**: `vergeos_node_ram_allocated` (Gauge, labeled by `system_name`, `cluster`, and `node_name`)
- **Total RAM (MB)**: `vergeos_node_ram_total` (Gauge, labeled by `system_name`, `cluster`, and `node_name`)
- **Running RAM (MB)**: `vergeos_node_running_ram` (Gauge, labeled by `system_name`, `cluster`, and `node_name`)

---
### Storage Metrics
- **Drive Read Operations**: `vergeos_drive_read_ops` (Counter, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Write Operations**: `vergeos_drive_write_ops` (Counter, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Read Bytes**: `vergeos_drive_read_bytes` (Counter, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Write Bytes**: `vergeos_drive_write_bytes` (Counter, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Utilization**: `vergeos_drive_util` (Gauge, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)

---
### Drive Health Metrics
- **Drive Read Errors**: `vergeos_drive_read_errors` (Counter, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Write Errors**: `vergeos_drive_write_errors` (Counter, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Repairs**: `vergeos_drive_repairs` (Counter, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Throttle**: `vergeos_drive_throttle` (Gauge, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Wear Level**: `vergeos_drive_wear_level` (Counter, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Power On Hours**: `vergeos_drive_power_on_hours` (Counter, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Reallocated Sectors**: `vergeos_drive_reallocated_sectors` (Counter, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Temperature**: `vergeos_drive_temperature` (Gauge, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`)
- **Drive Service Time**: `vergeos_drive_service_time` (Gauge, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`, in milliseconds)

### Drive Metrics
All drive metrics include the following labels:
- `system_name`: Name of the system
- `node_name`: Name of the node the drive belongs to
- `drive_name`: Name of the drive
- `tier`: VSAN tier number (0 or 1)
- `serial`: Drive serial number

---
## Network Metrics
- **NIC Transmit Packets**: `vergeos_nic_tx_packets_total` (Counter, labeled by `system_name`, `cluster`, `node_name`, and `interface`)
- **NIC Receive Packets**: `vergeos_nic_rx_packets_total` (Counter, labeled by `system_name`, `cluster`, `node_name`, and `interface`)
- **NIC Transmit Bytes**: `vergeos_nic_tx_bytes_total` (Counter, labeled by `system_name`, `cluster`, `node_name`, and `interface`)
- **NIC Receive Bytes**: `vergeos_nic_rx_bytes_total` (Counter, labeled by `system_name`, `cluster`, `node_name`, and `interface`)
- **NIC Status**: `vergeos_nic_status` (Gauge, labeled by `system_name`, `cluster`, `node_name`, and `interface`)
---
## VSAN Tiers Overview
- **VSAN Tier Capacity**: `vergeos_vsan_tier_capacity` (Gauge, labeled by `system_name`, `tier`, and `description`)
- **VSAN Tier Used Space**: `vergeos_vsan_tier_used` (Gauge, labeled by `system_name`, `tier`, and `description`)
- **VSAN Tier Used Percentage**: `vergeos_vsan_tier_used_pct` (Gauge, labeled by `system_name`, `tier`, and `description`)
- **VSAN Tier Allocated Space**: `vergeos_vsan_tier_allocated` (Gauge, labeled by `system_name`, `tier`, and `description`)
- **VSAN Tier Dedupe Ratio**: `vergeos_vsan_tier_dedupe_ratio` (Gauge, labeled by `system_name`, `tier`, and `description`)

---
## VSAN Tier Detailed Stats

### Metrics
- **VSAN Tier Transaction Count**: `vergeos_vsan_tier_transaction` (Counter, labeled by `system_name`, `tier`, and `status`)
- **VSAN Tier Repairs Count**: `vergeos_vsan_tier_repairs` (Gauge, labeled by `system_name`, `tier`, and `status`)
- **VSAN Tier State**: `vergeos_vsan_tier_state` (Gauge, labeled by `system_name`, `tier`, and `status`, `1` for working, `0` for not working)
- **VSAN Bad Drives**: `vergeos_vsan_bad_drives` (Gauge, labeled by `system_name`, `tier`, and `status`)
- **VSAN Encryption Status**: `vergeos_vsan_encryption_status` (Gauge, labeled by `system_name`, `tier`, and `status`, `1` for encrypted, `0` for not encrypted)
- **VSAN Redundancy Status**: `vergeos_vsan_redundant` (Gauge, labeled by `system_name`, `tier`, and `status`, `1` for redundant, `0` for not redundant)
- **VSAN Last Walk Time (ms)**: `vergeos_vsan_last_walk_time_ms` (Gauge, labeled by `system_name`, `tier`, and `status`)
- **VSAN Last Full Walk Time (ms)**: `vergeos_vsan_last_fullwalk_time_ms` (Gauge, labeled by `system_name`, `tier`, and `status`)
- **VSAN Full Walk Status**: `vergeos_vsan_fullwalk_status` (Gauge, labeled by `system_name`, `tier`, and `status`, `1` for active, `0` for inactive)
- **VSAN Full Walk Progress**: `vergeos_vsan_fullwalk_progress` (Gauge, labeled by `system_name`, `tier`, and `status`, percentage of full walk completion)
- **VSAN Current Space Throttle (ms)**: `vergeos_vsan_cur_space_throttle_ms` (Gauge, labeled by `system_name`, `tier`, and `status`)
- **VSAN Nodes Online**: `vergeos_vsan_nodes_online` (Gauge, labeled by `system_name`, `tier`, and `status`)
- **VSAN Drives Online**: `vergeos_vsan_drives_online` (Gauge, labeled by `system_name`, `tier`, and `status`)
- **VSAN Drive States**: `vergeos_vsan_drive_states` (Gauge, labeled by `system_name`, `tier`, and `state`, counts drives in each state: online, offline, repairing, initializing, verifying, noredundant, outofspace)

---
## Cluster Overview
- **Total Clusters**: `vergeos_clusters_total` (Gauge, labeled by `system_name`)
- **Cluster Enabled Status**: `vergeos_cluster_enabled` (Gauge, labeled by `system_name` and `cluster`, `1` for enabled, `0` for disabled)
- **Cluster RAM Per Unit**: `vergeos_cluster_ram_per_unit` (Gauge, labeled by `system_name` and `cluster`)
- **Cluster Cores Per Unit**: `vergeos_cluster_cores_per_unit` (Gauge, labeled by `system_name` and `cluster`)
- **Cluster Target RAM Percentage**: `vergeos_cluster_target_ram_pct` (Gauge, labeled by `system_name` and `cluster`)
- **Cluster Status**: `vergeos_cluster_status` (Gauge, labeled by `system_name` and `cluster`, `1` for online, `0` for offline)
- **Cluster Health**: `vergeos_cluster_health` (Gauge, labeled by `system_name` and `cluster`, `1` for healthy, `0` for unhealthy)

---
## Cluster Stats
- **Total Nodes in Cluster**: `vergeos_cluster_total_nodes` (Gauge, labeled by `system_name` and `cluster`)
- **Online Nodes in Cluster**: `vergeos_cluster_online_nodes` (Gauge, labeled by `system_name` and `cluster`)
- **Running Machines**: `vergeos_cluster_running_machines` (Gauge, labeled by `system_name` and `cluster`)
- **Total RAM in Cluster (MB)**: `vergeos_cluster_total_ram` (Gauge, labeled by `system_name` and `cluster`)
- **Online RAM in Cluster (MB)**: `vergeos_cluster_online_ram` (Gauge, labeled by `system_name` and `cluster`)
- **Used RAM in Cluster (MB)**: `vergeos_cluster_used_ram` (Gauge, labeled by `system_name` and `cluster`)
- **Total Cores in Cluster**: `vergeos_cluster_cores_total` (Gauge, labeled by `system_name` and `cluster`)
- **Online Cores in Cluster**: `vergeos_cluster_online_cores` (Gauge, labeled by `system_name` and `cluster`)
- **Used Cores in Cluster**: `vergeos_cluster_used_cores` (Gauge, labeled by `system_name` and `cluster`)
- **Physical RAM Used (MB)**: `vergeos_cluster_phys_ram_used` (Gauge, labeled by `system_name` and `cluster`)

---
## Tenant Metrics

### Tenant Overview
- **Total Tenants**: `vergeos_tenants_total` (Gauge, labeled by `system_name`)
- **Tenant Running**: `vergeos_tenant_running` (Gauge, labeled by `system_name` and `tenant_name`, 1=running, 0=not running)
- **Tenant Status**: `vergeos_tenant_status` (Gauge, labeled by `system_name`, `tenant_name`, and `status`, always 1 — info-style metric)
- **Tenant Nodes Total**: `vergeos_tenant_nodes_total` (Gauge, labeled by `system_name` and `tenant_name`)

### Tenant Resource Usage (from TenantStatsHistoryShort)
- **CPU Usage Percentage**: `vergeos_tenant_cpu_usage_pct` (Gauge, labeled by `system_name` and `tenant_name`)
- **CPU Cores**: `vergeos_tenant_cpu_cores` (Gauge, labeled by `system_name` and `tenant_name`)
- **RAM Used (bytes)**: `vergeos_tenant_ram_used_bytes` (Gauge, labeled by `system_name` and `tenant_name`)
- **RAM Allocated (bytes)**: `vergeos_tenant_ram_allocated_bytes` (Gauge, labeled by `system_name` and `tenant_name`)
- **RAM Usage Percentage**: `vergeos_tenant_ram_usage_pct` (Gauge, labeled by `system_name` and `tenant_name`)
- **IP Count**: `vergeos_tenant_ip_count` (Gauge, labeled by `system_name` and `tenant_name`)

### Tenant GPU Metrics (only emitted when GPU resources exist)
- **vGPUs Used**: `vergeos_tenant_vgpus_used` (Gauge, labeled by `system_name` and `tenant_name`)
- **vGPUs Total**: `vergeos_tenant_vgpus_total` (Gauge, labeled by `system_name` and `tenant_name`)
- **GPUs Used**: `vergeos_tenant_gpus_used` (Gauge, labeled by `system_name` and `tenant_name`)
- **GPUs Total**: `vergeos_tenant_gpus_total` (Gauge, labeled by `system_name` and `tenant_name`)

### Tenant Node Metrics
- **Node CPU Cores**: `vergeos_tenant_node_cpu_cores` (Gauge, labeled by `system_name`, `tenant_name`, and `node_name`)
- **Node RAM (bytes)**: `vergeos_tenant_node_ram_bytes` (Gauge, labeled by `system_name`, `tenant_name`, and `node_name`)
- **Node Enabled**: `vergeos_tenant_node_enabled` (Gauge, labeled by `system_name`, `tenant_name`, and `node_name`, 1=enabled, 0=disabled)
- **Node Running**: `vergeos_tenant_node_running` (Gauge, labeled by `system_name`, `tenant_name`, and `node_name`, 1=running, 0=not running)
- **Node CPU Usage Percentage**: `vergeos_tenant_node_cpu_usage_pct` (Gauge, labeled by `system_name`, `tenant_name`, and `node_name`)
- **Node RAM Used (bytes)**: `vergeos_tenant_node_ram_used_bytes` (Gauge, labeled by `system_name`, `tenant_name`, and `node_name`)
- **Node RAM Usage Percentage**: `vergeos_tenant_node_ram_usage_pct` (Gauge, labeled by `system_name`, `tenant_name`, and `node_name`)

### Tenant Storage Metrics
- **Storage Provisioned (bytes)**: `vergeos_tenant_storage_provisioned_bytes` (Gauge, labeled by `system_name`, `tenant_name`, and `tier`)
- **Storage Used (bytes)**: `vergeos_tenant_storage_used_bytes` (Gauge, labeled by `system_name`, `tenant_name`, and `tier`)
- **Storage Allocated (bytes)**: `vergeos_tenant_storage_allocated_bytes` (Gauge, labeled by `system_name`, `tenant_name`, and `tier`)
- **Storage Usage Percentage**: `vergeos_tenant_storage_used_pct` (Gauge, labeled by `system_name`, `tenant_name`, and `tier`)

### Tenant Network Metrics
- **Layer 2 Networks Total**: `vergeos_tenant_layer2_networks_total` (Gauge, labeled by `system_name` and `tenant_name`)

---
## System Version Metrics
- **System Version**: `vergeos_system_version` (Gauge, labeled by `system_name` and `version`, always 1)
- **Latest Available System Version**: `vergeos_system_version_latest` (Gauge, labeled by `system_name` and `version`, always 1)
- **System Branch**: `vergeos_system_branch` (Gauge, labeled by `system_name` and `branch`, always 1)
- **System Info**: `vergeos_system_info` (Gauge, labeled by `system_name`, `current_version`, `latest_version`, `branch`, and `hash`, always 1)
