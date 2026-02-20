# goVergeOS SDK Issues for vergeos-exporter

Actionable SDK requirements to restore 36 blocked metrics in the vergeos-exporter. Each issue maps to a specific VergeOS API table endpoint (NOT dashboard views).

**Context**: The vergeos-exporter was migrated to the goVergeOS SDK (`refactor/sdk-migration` branch). During migration, 36 metrics were removed because the SDK lacked services for the underlying data tables. The VergeOS API has dedicated table endpoints (`machine_stats`, `machine_nics`, `machine_nic_stats`, etc.) that provide all the data we need.

**Approach**: Use underlying API table endpoints, not dashboard views. Dashboard views were rejected by the SDK maintainer as they are DB views that cross-reference other tables.

**Total Blocked Metrics**: 36

---

## Issue 1: New `MachineStatsService` for Node CPU/RAM/Temp

**Priority**: HIGH
**Blocked Metrics**: 6
**API Endpoint**: `/api/v4/machine_stats`

### Problem

The exporter needs per-node CPU usage, RAM usage, and temperature metrics. This data lives in the `machine_stats` table, which has one row per machine with a `machine` FK. Each physical node has a `Machine` field (FK to the machines table) that links to its machine_stats row.

### API Schema

`machine_stats` has a unique constraint on `machine` (one stats row per machine).

| Field | Type | Description |
|-------|------|-------------|
| `machine` | `parent_machine` | Required FK to machines table |
| `total_cpu` | `uint8` | Total CPU usage percentage |
| `ram_used` | `uint32` | RAM used in MB |
| `ram_pct` | `uint8` | Physical RAM used percentage |
| `core_usagelist` | `json` | Per-core usage (JSON array of floats) |
| `core_temp` | `uint16` | Average core temperature (Celsius) |
| `core_temp_top` | `uint16` | Maximum core temperature (Celsius) |

### Proposed SDK Changes

**New types** (e.g., `types_machine_stats.go`):

```go
type MachineStats struct {
    Key           FlexInt   `json:"$key,omitempty"`
    Machine       int       `json:"machine,omitempty"`
    TotalCPU      uint8     `json:"total_cpu,omitempty"`
    UserCPU       uint8     `json:"user_cpu,omitempty"`
    SystemCPU     uint8     `json:"system_cpu,omitempty"`
    IOWaitCPU     uint8     `json:"iowait_cpu,omitempty"`
    RAMUsed       uint32    `json:"ram_used,omitempty"`
    RAMPct        uint8     `json:"ram_pct,omitempty"`
    CoreUsageList []float64 `json:"core_usagelist,omitempty"`
    CoreTemp      uint16    `json:"core_temp,omitempty"`
    CoreTempTop   uint16    `json:"core_temp_top,omitempty"`
}
```

**New service** (e.g., `machine_stats.go`):

```go
type MachineStatsService struct { ... }

// GetByMachine retrieves stats for a specific machine by machine ID.
// GET /api/v4/machine_stats?filter=machine eq {machineID}&fields=...
func (s *MachineStatsService) GetByMachine(ctx context.Context, machineID int) (*MachineStats, error)
```

### How the Exporter Will Use This

```go
// In NodeCollector.Collect():
nodes, _ := client.Nodes.ListPhysical(ctx)
for _, node := range nodes {
    stats, _ := client.MachineStats.GetByMachine(ctx, node.Machine)
    // Emit: vergeos_node_cpu_core_usage (per core from CoreUsageList)
    // Emit: vergeos_node_core_temp (from CoreTemp)
    // Emit: vergeos_node_ram_used (from RAMUsed)
    // Emit: vergeos_node_ram_pct (from RAMPct)
}
```

### Blocked Metrics

| Metric | Labels | Source Field |
|--------|--------|-------------|
| `vergeos_node_cpu_core_usage` | `system_name`, `cluster`, `node_name`, `core_id` | `core_usagelist` |
| `vergeos_node_core_temp` | `system_name`, `cluster`, `node_name` | `core_temp` |
| `vergeos_node_ram_used` | `system_name`, `cluster`, `node_name` | `ram_used` |
| `vergeos_node_ram_pct` | `system_name`, `cluster`, `node_name` | `ram_pct` |

### Open Question: `running_cores` / `running_ram`

The original exporter got `running_cores` and `running_ram` from the node dashboard's `vm_stats_totals` object. These represent cores/RAM allocated to running VMs. Please verify:
- Are these available in `machine_stats` under a different name?
- Or do they need a separate source (e.g., `vm_stats_totals` table)?
- The exporter needs 2 additional metrics: `vergeos_node_running_cores` and `vergeos_node_running_ram`

---

## Issue 2: New `MachineNICService` for Network Traffic

**Priority**: HIGH
**Blocked Metrics**: 7
**API Endpoints**: `/api/v4/machine_nics`, `/api/v4/machine_nic_stats`, `/api/v4/machine_nic_status`

### Problem

The exporter needs per-NIC traffic counters and link status for physical nodes. This data is spread across three related tables: `machine_nics` (NIC list per machine), `machine_nic_stats` (traffic counters), and `machine_nic_status` (link state).

### API Schema

**`machine_nics`** — parent: `machine`, unique: `[machine, name]`

| Field | Type | Description |
|-------|------|-------------|
| `machine` | `parent_machine` | FK to machines table |
| `name` | `text` | Interface name (e.g., `eno1`) |
| `stats` | `row` | FK to `machine_nic_stats` (readonly) |
| `status` | `row` | FK to `machine_nic_status` (readonly) |

**`machine_nic_stats`** — parent: `parent_nic`, unique: `parent_nic`

| Field | Type | Description |
|-------|------|-------------|
| `parent_nic` | `row` | FK to `machine_nics` |
| `tx_pckts` | `uint64` | Total TX packets |
| `rx_pckts` | `uint64` | Total RX packets |
| `tx_bytes` | `uint64` | Total TX bytes |
| `rx_bytes` | `uint64` | Total RX bytes |

Note: `tx_errors`/`rx_errors` are not in `machine_nic_stats`. The original exporter got these from the dashboard view. Please verify if error counters are available elsewhere or if these 2 metrics should be dropped.

**`machine_nic_status`** — parent: `parent_nic`, unique: `parent_nic`

| Field | Type | Description |
|-------|------|-------------|
| `parent_nic` | `row` | FK to `machine_nics` |
| `status` | `text` | `up`, `down`, `unknown`, `lowerlayerdown` |
| `speed` | `uint32` | Link speed in Mbps |

### Proposed SDK Changes

**New types**:

```go
type MachineNICStats struct {
    Key     FlexInt `json:"$key,omitempty"`
    TxPckts uint64  `json:"tx_pckts,omitempty"`
    RxPckts uint64  `json:"rx_pckts,omitempty"`
    TxBytes uint64  `json:"tx_bytes,omitempty"`
    RxBytes uint64  `json:"rx_bytes,omitempty"`
}

type MachineNICStatus struct {
    Key    FlexInt `json:"$key,omitempty"`
    Status string  `json:"status,omitempty"` // up, down, unknown, lowerlayerdown
    Speed  uint32  `json:"speed,omitempty"`
}

type MachineNIC struct {
    Key     FlexInt           `json:"$key,omitempty"`
    Machine int               `json:"machine,omitempty"`
    Name    string            `json:"name,omitempty"`
    Stats   *MachineNICStats  `json:"stats,omitempty"`
    Status  *MachineNICStatus `json:"status,omitempty"`
}
```

**New service**:

```go
type MachineNICService struct { ... }

// ListByMachine returns all NICs for a given machine, with stats and status expanded.
// GET /api/v4/machine_nics?filter=machine eq {machineID}&fields=$key,name,stats[all],status[all]
func (s *MachineNICService) ListByMachine(ctx context.Context, machineID int) ([]MachineNIC, error)
```

The `stats[all]` and `status[all]` syntax expands the FK rows inline, avoiding separate API calls per NIC.

### How the Exporter Will Use This

```go
// In NetworkCollector.Collect():
nodes, _ := client.Nodes.ListPhysical(ctx)
for _, node := range nodes {
    nics, _ := client.MachineNICs.ListByMachine(ctx, node.Machine)
    for _, nic := range nics {
        // Labels: system_name, cluster, node.Name, nic.Name
        // Emit: vergeos_nic_tx_packets_total (from nic.Stats.TxPckts)
        // Emit: vergeos_nic_rx_packets_total (from nic.Stats.RxPckts)
        // Emit: vergeos_nic_tx_bytes_total (from nic.Stats.TxBytes)
        // Emit: vergeos_nic_rx_bytes_total (from nic.Stats.RxBytes)
        // Emit: vergeos_nic_status (1 if nic.Status.Status == "up", else 0)
    }
}
```

### Blocked Metrics

| Metric | Labels | Source Field |
|--------|--------|-------------|
| `vergeos_nic_tx_packets_total` | `system_name`, `cluster`, `node_name`, `interface` | `machine_nic_stats.tx_pckts` |
| `vergeos_nic_rx_packets_total` | `system_name`, `cluster`, `node_name`, `interface` | `machine_nic_stats.rx_pckts` |
| `vergeos_nic_tx_bytes_total` | `system_name`, `cluster`, `node_name`, `interface` | `machine_nic_stats.tx_bytes` |
| `vergeos_nic_rx_bytes_total` | `system_name`, `cluster`, `node_name`, `interface` | `machine_nic_stats.rx_bytes` |
| `vergeos_nic_tx_errors_total` | `system_name`, `cluster`, `node_name`, `interface` | **TBD** — see note above |
| `vergeos_nic_rx_errors_total` | `system_name`, `cluster`, `node_name`, `interface` | **TBD** — see note above |
| `vergeos_nic_status` | `system_name`, `cluster`, `node_name`, `interface` | `machine_nic_status.status` |

---

## Issue 3: New `MachineDriveStatsService` for Drive I/O

**Priority**: MEDIUM
**Blocked Metrics**: 6
**API Endpoint**: `/api/v4/machine_drive_stats`

### Problem

The exporter needs per-drive I/O metrics (reads, writes, bytes, utilization, service time). This data is in the `machine_drive_stats` table, with one row per drive linked via `parent_drive` FK to `machine_drives`.

### API Schema

`machine_drive_stats` — parent: `parent_drive`, unique: `parent_drive`

| Field | Type | Description |
|-------|------|-------------|
| `parent_drive` | `row` | FK to `machine_drives` |
| `reads` | `uint64` | Total read operations |
| `writes` | `uint64` | Total write operations |
| `read_bytes` | `uint64` | Total bytes read |
| `write_bytes` | `uint64` | Total bytes written |
| `rops` | `uint64` | Read operations per second |
| `wops` | `uint64` | Write operations per second |
| `rbps` | `uint64` | Read bytes per second |
| `wbps` | `uint64` | Write bytes per second |
| `service_time` | `num` | Average I/O service time (ms) |
| `util` | `num` | I/O utilization percentage |
| `physical` | `bool` | True if physical drive stats |

### Proposed SDK Changes

**New types**:

```go
type MachineDriveStats struct {
    Key         FlexInt `json:"$key,omitempty"`
    ParentDrive int     `json:"parent_drive,omitempty"`
    Reads       uint64  `json:"reads,omitempty"`
    Writes      uint64  `json:"writes,omitempty"`
    ReadBytes   uint64  `json:"read_bytes,omitempty"`
    WriteBytes  uint64  `json:"write_bytes,omitempty"`
    Rops        uint64  `json:"rops,omitempty"`
    Wops        uint64  `json:"wops,omitempty"`
    Rbps        uint64  `json:"rbps,omitempty"`
    Wbps        uint64  `json:"wbps,omitempty"`
    ServiceTime float64 `json:"service_time,omitempty"`
    Util        float64 `json:"util,omitempty"`
    Physical    bool    `json:"physical,omitempty"`
}
```

**New service**:

```go
type MachineDriveStatsService struct { ... }

// List returns all drive stats (optionally filtered to physical drives).
// GET /api/v4/machine_drive_stats?filter=physical eq true&fields=...
func (s *MachineDriveStatsService) List(ctx context.Context) ([]MachineDriveStats, error)

// GetByDrive retrieves stats for a specific drive.
// GET /api/v4/machine_drive_stats?filter=parent_drive eq {driveID}&fields=...
func (s *MachineDriveStatsService) GetByDrive(ctx context.Context, driveID int) (*MachineDriveStats, error)
```

### How the Exporter Will Use This

```go
// In StorageCollector.Collect():
drives, _ := client.MachineDrivePhys.List(ctx)     // includes NodeDisplay, Serial, etc.
allStats, _ := client.MachineDriveStats.List(ctx)   // bulk fetch
statsMap := buildStatsMap(allStats)                  // map[parentDrive]MachineDriveStats

for _, drive := range drives {
    stats := statsMap[drive.ParentDrive]
    // Labels: system_name, drive.NodeDisplay, drive.Name, tier, drive.Serial
    // Emit: vergeos_drive_read_ops (from stats.Reads)
    // Emit: vergeos_drive_write_ops (from stats.Writes)
    // Emit: vergeos_drive_read_bytes (from stats.ReadBytes)
    // Emit: vergeos_drive_write_bytes (from stats.WriteBytes)
    // Emit: vergeos_drive_util (from stats.Util)
    // Emit: vergeos_drive_service_time (from stats.ServiceTime)
}
```

### Blocked Metrics

| Metric | Labels | Source Field |
|--------|--------|-------------|
| `vergeos_drive_read_ops` | `system_name`, `node_name`, `drive_name`, `tier`, `serial` | `reads` |
| `vergeos_drive_write_ops` | `system_name`, `node_name`, `drive_name`, `tier`, `serial` | `writes` |
| `vergeos_drive_read_bytes` | `system_name`, `node_name`, `drive_name`, `tier`, `serial` | `read_bytes` |
| `vergeos_drive_write_bytes` | `system_name`, `node_name`, `drive_name`, `tier`, `serial` | `write_bytes` |
| `vergeos_drive_util` | `system_name`, `node_name`, `drive_name`, `tier`, `serial` | `util` |
| `vergeos_drive_service_time` | `system_name`, `node_name`, `drive_name`, `tier`, `serial` | `service_time` |

---

## Issue 4: Add `NodeDisplay` and `StatusList` to `MachineDrivePhys`

**Priority**: MEDIUM
**Blocked Metrics**: 15 (9 drive hardware + 6 drive states)
**API Endpoint**: `/api/v4/machine_drive_phys` (computed fields)

### Problem

The existing `MachineDrivePhys` struct lacks two computed fields needed to identify which node a drive belongs to and its operational status. Without `NodeDisplay`, per-drive metrics can't include a `node_name` label. Without `StatusList`, drive state counting is impossible.

### API Verification

The VergeOS API supports computed field aliases via `#` notation:

```
GET /api/v4/machine_drive_phys?fields=...,parent_drive#machine#name as node_display,parent_drive#status#status as statuslist
```

Returns:
```json
{
  "$key": 1,
  "node_display": "node1",
  "statuslist": "online",
  "vsan_tier": 0,
  "serial": "WD-12345"
}
```

### Proposed SDK Changes

This is the simplest change with the highest leverage.

**Update existing `MachineDrivePhys` struct** (in `types_vsan.go`):

```go
type MachineDrivePhys struct {
    // ... existing fields ...

    // NodeDisplay is the name of the physical node containing this drive.
    // Populated via computed field: parent_drive#machine#name
    NodeDisplay string `json:"node_display,omitempty"`

    // StatusList is the drive status (online, offline, repairing, initializing, verifying, noredundant, outofspace).
    // Populated via computed field: parent_drive#status#status
    StatusList string `json:"statuslist,omitempty"`
}
```

**Update `machineDrivePhysListFields` constant** (in `machine_drive_phys.go`):

```go
const machineDrivePhysListFields = "...,parent_drive#machine#name as node_display,parent_drive#status#status as statuslist"
```

### How the Exporter Will Use This

**Per-drive hardware metrics** (already have all data in MachineDrivePhys except node_name):

```go
drives, _ := client.MachineDrivePhys.List(ctx)
for _, drive := range drives {
    // Labels: system_name, drive.NodeDisplay, drive.Name, tier, drive.Serial
    // Emit: vergeos_drive_temperature (from drive.Temp)
    // Emit: vergeos_drive_wear_level (from drive.WearLevel)
    // Emit: vergeos_drive_power_on_hours (from drive.Hours)
    // Emit: vergeos_drive_reallocated_sectors (from drive.ReallocSectors)
    // Emit: vergeos_drive_read_errors (from drive.VSANReadErrors)
    // Emit: vergeos_drive_write_errors (from drive.VSANWriteErrors)
    // Emit: vergeos_drive_repairs (from drive.VSANRepairing)
    // Emit: vergeos_drive_throttle (from drive.VSANThrottle)
}
```

Note: `vergeos_drive_service_time` comes from Issue 3 (`machine_drive_stats`), not this struct.

**Drive state counting**:

```go
// Group drives by node+tier+status, emit counts
stateCounts := map[stateKey]int{}
for _, drive := range drives {
    key := stateKey{drive.NodeDisplay, drive.VSANTier, drive.StatusList}
    stateCounts[key]++
}
for key, count := range stateCounts {
    // Emit: vergeos_vsan_drive_states (gauge)
    // Labels: system_name, tier, state (e.g., "online", "repairing")
}
```

### Blocked Metrics

**Per-drive hardware metrics (9)** — all need `node_name` from `NodeDisplay`:

| Metric | Source Field |
|--------|-------------|
| `vergeos_drive_temperature` | `temp` |
| `vergeos_drive_wear_level` | `wear_level` |
| `vergeos_drive_power_on_hours` | `hours` |
| `vergeos_drive_reallocated_sectors` | `realloc_sectors` |
| `vergeos_drive_read_errors` | `vsan_read_errors` |
| `vergeos_drive_write_errors` | `vsan_write_errors` |
| `vergeos_drive_repairs` | `vsan_repairing` |
| `vergeos_drive_throttle` | `vsan_throttle` |

Note: `vergeos_drive_temperature` requires `node_name` label per metrics.md. The data itself (`temp`) is already in the SDK struct but can't be emitted without the node mapping.

**Drive state count metric (1 metric, multiple label values)**:

| Metric | Labels |
|--------|--------|
| `vergeos_vsan_drive_states` | `system_name`, `tier`, `state` |

States: `online`, `offline`, `repairing`, `initializing`, `verifying`, `noredundant`, `outofspace`

---

## Issue 5: System Branch and Latest Version

**Priority**: LOW
**Blocked Metrics**: 2
**API Endpoint**: `/api/v4/system` (preferred) or `/api/v4/update_dashboard`

### Problem

The exporter needs the update branch (stable/preview) and latest available version. The SDK's `System.GetInfo()` uses `/version.json` which only provides current version, not branch or available updates.

### API Options

**Option A — `/api/v4/system`** (preferred, simpler):

The `system` table is a single-row table with computed fields:

| Field | Type | Description |
|-------|------|-------------|
| `cloud_name` | `text` | System name (computed from settings) |
| `yb_version` | `text` | Current VergeOS version (computed from nodes/1) |
| `branch` | `text` | Update branch (computed from update_settings/1) |

```
GET /api/v4/system?fields=cloud_name,yb_version,branch
```

**Option B — `/api/v4/update_dashboard`** (for latest available version):

The `update_dashboard` is a composite view. The exporter needs:

```json
{
  "packages": [{
    "name": "ybos",
    "version": "4.12.0",
    "branch": "stable",
    "source_packages": [{ "version": "4.12.1" }]
  }]
}
```

### Proposed SDK Changes

**Option A** (add `GetSystem()` method):

```go
type SystemInfo struct {
    CloudName string `json:"cloud_name,omitempty"`
    YBVersion string `json:"yb_version,omitempty"`
    Branch    string `json:"branch,omitempty"`
}

// GetSystem retrieves system info from /api/v4/system.
func (s *SystemService) GetSystem(ctx context.Context) (*SystemInfo, error)
```

**Option B** (add `GetUpdateDashboard()` method — for latest version):

```go
type UpdateSourcePackage struct {
    Version     string `json:"version,omitempty"`
    Description string `json:"description,omitempty"`
}

type UpdatePackage struct {
    Name           string                `json:"name,omitempty"`
    Version        string                `json:"version,omitempty"`
    Branch         string                `json:"branch,omitempty"`
    SourcePackages []UpdateSourcePackage `json:"source_packages,omitempty"`
}

type UpdateDashboard struct {
    Packages []UpdatePackage `json:"packages,omitempty"`
}

// GetUpdateDashboard retrieves update availability information.
func (s *SystemService) GetUpdateDashboard(ctx context.Context) (*UpdateDashboard, error)
```

Both methods may be needed — Option A for `branch`, Option B for `latest_version`.

### How the Exporter Will Use This

```go
// In SystemCollector.Collect():
sysInfo, _ := client.System.GetSystem(ctx)
// Emit: vergeos_system_branch (labels: system_name, branch=sysInfo.Branch)

updateInfo, _ := client.System.GetUpdateDashboard(ctx)
for _, pkg := range updateInfo.Packages {
    if pkg.Name == "ybos" && len(pkg.SourcePackages) > 0 {
        // Emit: vergeos_system_version_latest (labels: system_name, version=pkg.SourcePackages[0].Version)
    }
}

// Update vergeos_system_info labels to include latest_version and branch
```

### Blocked Metrics

| Metric | Labels | Source |
|--------|--------|-------|
| `vergeos_system_branch` | `system_name`, `branch` | `system.branch` or `update_dashboard.packages[ybos].branch` |
| `vergeos_system_version_latest` | `system_name`, `version` | `update_dashboard.packages[ybos].source_packages[0].version` |

---

## Issue 6: Add `NodesOnline` / `DrivesOnline` to `ClusterTier`

**Priority**: LOW
**Blocked Metrics**: 2
**API Endpoint**: `/api/v4/cluster_tiers` (additional fields)

### Problem

The existing `ClusterTier` struct doesn't include `nodes_online` and `drives_online` fields. These are nested objects in the API response when using `fields=all`.

### API Verification

```
GET /api/v4/cluster_tiers?fields=all
```

Response includes:
```json
{
  "$key": 1,
  "tier": 0,
  "nodes_online": {
    "nodes": [
      {"state": "online"},
      {"state": "online"},
      {"state": "offline"}
    ]
  },
  "drives_online": [
    {"state": "online"},
    {"state": "online"},
    {"state": "repairing"}
  ]
}
```

### Proposed SDK Changes

**Add to existing `ClusterTier` struct** (in `types_vsan.go`):

```go
type ClusterTierNodeState struct {
    State string `json:"state,omitempty"`
}

type ClusterTierNodesOnline struct {
    Nodes []ClusterTierNodeState `json:"nodes,omitempty"`
}

type ClusterTierDriveState struct {
    State string `json:"state,omitempty"`
}

type ClusterTier struct {
    // ... existing fields ...

    NodesOnline  *ClusterTierNodesOnline `json:"nodes_online,omitempty"`
    DrivesOnline []ClusterTierDriveState `json:"drives_online,omitempty"`
}

// CountOnlineNodes returns the number of nodes with state "online".
func (ct *ClusterTier) CountOnlineNodes() int { ... }

// CountOnlineDrives returns the number of drives with state "online".
func (ct *ClusterTier) CountOnlineDrives() int { ... }
```

**Update field list** in `cluster_tiers.go` to include `nodes_online` and `drives_online` (or use `fields=all`).

### How the Exporter Will Use This

```go
// In StorageCollector.Collect() (existing cluster_tiers loop):
for _, tier := range clusterTiers {
    // Emit: vergeos_vsan_nodes_online (from tier.CountOnlineNodes())
    // Emit: vergeos_vsan_drives_online (from tier.CountOnlineDrives())
}
```

### Blocked Metrics

| Metric | Labels | Source |
|--------|--------|-------|
| `vergeos_vsan_nodes_online` | `system_name`, `tier`, `status` | `cluster_tiers.nodes_online` |
| `vergeos_vsan_drives_online` | `system_name`, `tier`, `status` | `cluster_tiers.drives_online` |

---

## Summary

| Issue | API Endpoint | Priority | Blocked Metrics | SDK Work |
|-------|-------------|----------|-----------------|----------|
| 1 | `/api/v4/machine_stats` | HIGH | 4 (+2 TBD) | New `MachineStatsService` + types |
| 2 | `/api/v4/machine_nics` + `nic_stats` + `nic_status` | HIGH | 7 | New `MachineNICService` + types |
| 3 | `/api/v4/machine_drive_stats` | MEDIUM | 6 | New `MachineDriveStatsService` + types |
| 4 | `/api/v4/machine_drive_phys` (computed fields) | MEDIUM | 9 (+drive states) | Add `NodeDisplay`/`StatusList` to existing struct |
| 5 | `/api/v4/system` or `update_dashboard` | LOW | 2 | New `GetSystem()` / `GetUpdateDashboard()` |
| 6 | `/api/v4/cluster_tiers` (extra fields) | LOW | 2 | Add `NodesOnline`/`DrivesOnline` to existing struct |

**Total**: ~36 blocked metrics

**Dependencies**: Issues 3 and 4 are both needed for full drive metric restoration (Issue 4 provides `node_name` labels, Issue 3 provides I/O stats).

---

## Notes for SDK Maintainer

1. **All data is available via the API** — these are new service/type additions, not API limitations
2. **Issues 1+2 are highest priority** — they block 11+ metrics for node monitoring
3. **Issue 4 is the simplest change** — just 2 computed fields added to an existing struct + field list update
4. **FK expansion syntax** (`stats[all]`, `status[all]`) lets the SDK avoid N+1 queries per NIC
5. **Computed field syntax** (`parent_drive#machine#name as node_display`) is a VergeOS API feature for traversing FK chains
6. **Backward compatible** — all new fields use `omitempty`
7. **vergeos-exporter repo**: https://github.com/verge-io/vergeos-exporter
8. **Branch**: `refactor/sdk-migration`
