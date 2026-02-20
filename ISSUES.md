# goVergeOS SDK Issues for vergeos-exporter

Status tracker for SDK requirements to restore metrics removed during the SDK migration.

**Context**: The vergeos-exporter was migrated to the goVergeOS SDK (`refactor/sdk-migration` branch). During migration, 36 metrics were removed because the SDK lacked services for the underlying data tables.

---

## Issue 1: `MachineStatsService` for Node CPU/RAM/Temp — RESOLVED

**Status**: Resolved in goVergeOS PR #20
**Metrics Restored**: 4

| Metric | Status |
|--------|--------|
| `vergeos_node_cpu_core_usage` | Implemented |
| `vergeos_node_core_temp` | Implemented |
| `vergeos_node_ram_used` | Implemented |
| `vergeos_node_ram_pct` | Implemented |

**Remaining**: `vergeos_node_running_cores` and `vergeos_node_running_ram` are not available in `machine_stats`. These come from the node dashboard's `vm_stats_totals` object. API source TBD.

---

## Issue 2: `MachineNICService` for Network Traffic — RESOLVED

**Status**: Resolved in goVergeOS PR #20
**Metrics Restored**: 5

| Metric | Status |
|--------|--------|
| `vergeos_nic_tx_packets_total` | Implemented |
| `vergeos_nic_rx_packets_total` | Implemented |
| `vergeos_nic_tx_bytes_total` | Implemented |
| `vergeos_nic_rx_bytes_total` | Implemented |
| `vergeos_nic_status` | Implemented |

**Dropped**: `vergeos_nic_tx_errors_total` and `vergeos_nic_rx_errors_total` — not available in `machine_nic_stats` table. The original exporter got these from dashboard views which are not supported by the SDK.

---

## Issue 3: `MachineDriveStatsService` for Drive I/O — RESOLVED

**Status**: Resolved in goVergeOS PR #20
**Metrics Restored**: 6

| Metric | Status |
|--------|--------|
| `vergeos_drive_read_ops` | Implemented |
| `vergeos_drive_write_ops` | Implemented |
| `vergeos_drive_read_bytes` | Implemented |
| `vergeos_drive_write_bytes` | Implemented |
| `vergeos_drive_util` | Implemented |
| `vergeos_drive_service_time` | Implemented |

---

## Issue 4: `NodeDisplay` and `StatusList` on `MachineDrivePhys` — RESOLVED

**Status**: Resolved in goVergeOS PR #20
**Metrics Restored**: 9 (8 drive hardware + 1 drive state counting)

| Metric | Status |
|--------|--------|
| `vergeos_drive_temperature` | Implemented |
| `vergeos_drive_wear_level` | Implemented |
| `vergeos_drive_power_on_hours` | Implemented |
| `vergeos_drive_reallocated_sectors` | Implemented |
| `vergeos_drive_read_errors` | Implemented |
| `vergeos_drive_write_errors` | Implemented |
| `vergeos_drive_repairs` | Implemented |
| `vergeos_drive_throttle` | Implemented |
| `vergeos_vsan_drive_states` | Implemented |

---

## Issue 5: `UpdateSettings` and `UpdateSourcePackages` Services — RESOLVED

**Status**: Resolved in goVergeOS PR #20
**Metrics Restored**: 2

| Metric | Status |
|--------|--------|
| `vergeos_system_branch` | Implemented |
| `vergeos_system_version_latest` | Implemented |

Additionally, `vergeos_system_info` labels updated to include `branch`, `latest_version`, and `current_version`.

---

## Issue 6: `NodesOnline` / `DrivesOnline` on `ClusterTier` — BLOCKED

**Status**: Blocked by API limitation
**Metrics Blocked**: 2

The `NodesOnline` and `DrivesOnline` fields are present in the SDK (added in PR #20), but using `fields=all` to fetch them causes the API to expand FK fields (like `Cluster`) from integers to objects, breaking JSON deserialization. A targeted field list that includes `nodes_online` and `drives_online` without `fields=all` is needed.

| Metric | Status |
|--------|--------|
| `vergeos_vsan_nodes_online` | Blocked — API limitation |
| `vergeos_vsan_drives_online` | Blocked — API limitation |

---

## Summary

| Issue | Status | Metrics Restored |
|-------|--------|-----------------|
| 1 — MachineStats | Resolved | 4 |
| 2 — MachineNICs | Resolved | 5 |
| 3 — MachineDriveStats | Resolved | 6 |
| 4 — MachineDrivePhys | Resolved | 9 |
| 5 — UpdateSettings | Resolved | 2 |
| 6 — ClusterTier nodes/drives | Blocked | 0 |
| **Total** | | **26 restored** |

**Remaining unimplemented metrics** (4):
- `vergeos_vsan_nodes_online` — Issue 6 (API limitation)
- `vergeos_vsan_drives_online` — Issue 6 (API limitation)
- `vergeos_node_running_cores` — API source TBD
- `vergeos_node_running_ram` — API source TBD

**Dropped metrics** (4):
- `vergeos_nic_tx_errors_total` — not in `machine_nic_stats` table
- `vergeos_nic_rx_errors_total` — not in `machine_nic_stats` table
- `vergeos_network_collector_info` — placeholder removed (replaced by real NIC metrics)
