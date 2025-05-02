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
- **Drive Service Time**: `vergeos_drive_service_time` (Gauge, labeled by `system_name`, `node_name`, `drive_name`, `tier`, and `serial`, in seconds)

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
- **NIC Transmit Errors**: `vergeos_nic_tx_errors_total` (Counter, labeled by `system_name`, `cluster`, `node_name`, and `interface`)
- **NIC Receive Errors**: `vergeos_nic_rx_errors_total` (Counter, labeled by `system_name`, `cluster`, `node_name`, and `interface`)
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

