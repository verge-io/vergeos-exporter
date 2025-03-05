# Metrics included in VergeOS Exporter

## List of Physical Nodes
- **Total Physical Nodes**: `vergeos_nodes_total` (Gauge)
- **IPMI Status per Node**: `vergeos_node_ipmi_status` (Gauge, labeled by `node_name`)

---

## Node Details and Stats

### CPU Metrics
- **CPU Usage per Core**: `vergeos_node_cpu_core_usage` (Gauge, labeled by `node_name` and `core_id`)
- **CPU Temperature**: `vergeos_node_core_temp` (Gauge, labeled by `node_name`)

### Memory Metrics
- **RAM Used (MB)**: `vergeos_node_ram_used` (Gauge, labeled by `node_name`)
- **RAM Usage Percentage**: `vergeos_node_ram_pct` (Gauge, labeled by `node_name`)

### Storage Metrics
- **Drive Read Operations**: `vergeos_drive_read_ops` (Counter, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Write Operations**: `vergeos_drive_write_ops` (Counter, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Read Bytes**: `vergeos_drive_read_bytes` (Counter, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Write Bytes**: `vergeos_drive_write_bytes` (Counter, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Utilization**: `vergeos_drive_utilization` (Gauge, labeled by `node_name`, `drive_name`, and `vsan_tier`)

### Drive Health Metrics
- **Drive Read Errors**: `vergeos_drive_read_errors` (Counter, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Write Errors**: `vergeos_drive_write_errors` (Counter, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Average Latency**: `vergeos_drive_avg_latency` (Gauge, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Maximum Latency**: `vergeos_drive_max_latency` (Gauge, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Repairs**: `vergeos_drive_repairs` (Counter, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Throttle**: `vergeos_drive_throttle` (Gauge, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Wear Level**: `vergeos_drive_wear_level` (Counter, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Power On Hours**: `vergeos_drive_power_on_hours` (Counter, labeled by `node_name`, `drive_name`, and `vsan_tier`)
- **Drive Reallocated Sectors**: `vergeos_drive_reallocated_sectors` (Counter, labeled by `node_name`, `drive_name`, and `vsan_tier`)

### Network Metrics
- **NIC Transmit Packets**: `vergeos_nic_tx_packets` (Counter, labeled by `node_name` and `nic_name`)
- **NIC Receive Packets**: `vergeos_nic_rx_packets` (Counter, labeled by `node_name` and `nic_name`)
- **NIC Transmit Bytes**: `vergeos_nic_tx_bytes` (Counter, labeled by `node_name` and `nic_name`)
- **NIC Receive Bytes**: `vergeos_nic_rx_bytes` (Counter, labeled by `node_name` and `nic_name`)

---

## VSAN Tiers Overview
- **VSAN Tier Capacity**: `vergeos_vsan_tier_capacity` (Gauge, labeled by `tier_id`)
- **VSAN Tier Used Space**: `vergeos_vsan_tier_used` (Gauge, labeled by `tier_id`)
- **VSAN Tier Used Percentage**: `vergeos_vsan_tier_used_pct` (Gauge, labeled by `tier_id`)
- **VSAN Tier Allocated Space**: `vergeos_vsan_tier_allocated` (Gauge, labeled by `tier_id`)
- **VSAN Tier Dedupe Ratio**: `vergeos_vsan_tier_dedupe_ratio` (Gauge, labeled by `tier_id`)

---

## VSAN Tier Detailed Stats

### Metrics
- **VSAN Tier Transaction Count**: `vergeos_vsan_tier_transaction` (Counter, labeled by `tier_id`)
- **VSAN Tier Repairs Count**: `vergeos_vsan_tier_repairs` (Counter, labeled by `tier_id`)
- **VSAN Tier State**: `vergeos_vsan_tier_state` (Gauge, labeled by `tier_id`, `state` as a string value converted to numeric representation)
- **VSAN Bad Drives**: `vergeos_vsan_bad_drives` (Gauge, labeled by `tier_id`)
- **VSAN Encryption Status**: `vergeos_vsan_encryption_status` (Gauge, labeled by `tier_id`, `1` for encrypted, `0` for not encrypted)
- **VSAN Redundancy Status**: `vergeos_vsan_redundant` (Gauge, labeled by `tier_id`, `1` for redundant, `0` for not redundant)
- **VSAN Last Walk Time (ms)**: `vergeos_vsan_last_walk_time_ms` (Gauge, labeled by `tier_id`)
- **VSAN Last Full Walk Time (ms)**: `vergeos_vsan_last_fullwalk_time_ms` (Gauge, labeled by `tier_id`)
- **VSAN Full Walk Status**: `vergeos_vsan_fullwalk_status` (Gauge, labeled by `tier_id`, `1` for active, `0` for inactive)
- **VSAN Full Walk Progress**: `vergeos_vsan_fullwalk_progress` (Gauge, labeled by `tier_id`, percentage of full walk completion)
- **VSAN Current Space Throttle (ms)**: `vergeos_vsan_cur_space_throttle_ms` (Gauge, labeled by `tier_id`)
- **VSAN Nodes Online**: `vergeos_vsan_nodes_online` (Gauge, labeled by `tier_id`)
- **VSAN Drives Online**: `vergeos_vsan_drives_online` (Gauge, labeled by `tier_id`)
- **VSAN Drive Temperature (°C)**: `vergeos_vsan_drive_temp` (Gauge, labeled by `tier_id` and `drive_id`)
- **VSAN Drive Wear Level**: `vergeos_vsan_drive_wear_level` (Gauge, labeled by `tier_id` and `drive_id`)

---

## Cluster Overview
- **Total Clusters**: `vergeos_clusters_total` (Gauge)
- **Cluster Enabled Status**: `vergeos_cluster_enabled` (Gauge, labeled by `cluster_name`, `1` for enabled, `0` for disabled)
- **Cluster RAM Per Unit**: `vergeos_cluster_ram_per_unit` (Gauge, labeled by `cluster_name`)
- **Cluster Cores Per Unit**: `vergeos_cluster_cores_per_unit` (Gauge, labeled by `cluster_name`)
- **Cluster Target RAM Percentage**: `vergeos_cluster_target_ram_pct` (Gauge, labeled by `cluster_name`)
- **Cluster Status**: `vergeos_cluster_status` (Gauge, labeled by `cluster_name`, `1` for online, `0` for offline)

---

## Cluster Stats
- **Total Nodes in Cluster**: `vergeos_cluster_total_nodes` (Gauge, labeled by `cluster_name`)
- **Online Nodes in Cluster**: `vergeos_cluster_online_nodes` (Gauge, labeled by `cluster_name`)
- **Running Machines**: `vergeos_cluster_running_machines` (Gauge, labeled by `cluster_name`)
- **Total RAM in Cluster (MB)**: `vergeos_cluster_total_ram` (Gauge, labeled by `cluster_name`)
- **Online RAM in Cluster (MB)**: `vergeos_cluster_online_ram` (Gauge, labeled by `cluster_name`)
- **Used RAM in Cluster (MB)**: `vergeos_cluster_used_ram` (Gauge, labeled by `cluster_name`)
- **Total Cores in Cluster**: `vergeos_cluster_total_cores` (Gauge, labeled by `cluster_name`)
- **Online Cores in Cluster**: `vergeos_cluster_online_cores` (Gauge, labeled by `cluster_name`)
- **Used Cores in Cluster**: `vergeos_cluster_used_cores` (Gauge, labeled by `cluster_name`)
- **Physical RAM Used (MB)**: `vergeos_cluster_phys_ram_used` (Gauge, labeled by `cluster_name`)

---

## Aggregated from Node, VSAN, and Cluster Data
- **Total Drives Online**: `vergeos_drives_online_total` (Gauge)
- **Total NICs Online**: `vergeos_nics_online_total` (Gauge)
- **Total Memory Online**: `vergeos_memory_online_total` (Gauge)