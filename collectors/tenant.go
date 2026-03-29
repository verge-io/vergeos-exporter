package collectors

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/goVergeOS"
)

// TenantCollector collects metrics about VergeOS tenants, tenant nodes,
// tenant storage, and tenant networks.
//
// All metrics use MustNewConstMetric pattern to avoid stale label issues (Bug #28).
type TenantCollector struct {
	BaseCollector
	mutex sync.Mutex

	// Tenant-level metrics (labels: system_name, tenant_name)
	tenantsTotal        *prometheus.Desc
	tenantRunning       *prometheus.Desc
	tenantStatus        *prometheus.Desc
	tenantCPUUsagePct   *prometheus.Desc
	tenantCPUCores      *prometheus.Desc
	tenantRAMUsedBytes  *prometheus.Desc
	tenantRAMAllocBytes *prometheus.Desc
	tenantRAMUsagePct   *prometheus.Desc
	tenantIPCount       *prometheus.Desc
	tenantNodesTotal    *prometheus.Desc
	tenantVGPUsUsed     *prometheus.Desc
	tenantVGPUsTotal    *prometheus.Desc
	tenantGPUsUsed      *prometheus.Desc
	tenantGPUsTotal     *prometheus.Desc

	// Tenant node metrics (labels: system_name, tenant_name, node_name)
	tenantNodeCPUCores     *prometheus.Desc
	tenantNodeRAMBytes     *prometheus.Desc
	tenantNodeEnabled      *prometheus.Desc
	tenantNodeRunning      *prometheus.Desc
	tenantNodeCPUUsagePct  *prometheus.Desc
	tenantNodeRAMUsedBytes *prometheus.Desc
	tenantNodeRAMUsagePct  *prometheus.Desc

	// Tenant storage metrics (labels: system_name, tenant_name, tier)
	tenantStorageProvisioned *prometheus.Desc
	tenantStorageUsed        *prometheus.Desc
	tenantStorageAllocated   *prometheus.Desc
	tenantStorageUsedPct     *prometheus.Desc

	// Tenant network metrics (labels: system_name, tenant_name)
	tenantL2NetworksTotal *prometheus.Desc
}

// NewTenantCollector creates a new TenantCollector.
func NewTenantCollector(client *vergeos.Client, scrapeTimeout time.Duration) *TenantCollector {
	tenantLabels := []string{"system_name", "tenant_name"}
	tenantNodeLabels := []string{"system_name", "tenant_name", "node_name"}
	tenantStorageLabels := []string{"system_name", "tenant_name", "tier"}
	tenantStatusLabels := []string{"system_name", "tenant_name", "status"}

	return &TenantCollector{
		BaseCollector: *NewBaseCollector(client, scrapeTimeout),

		// Tenant-level metrics
		tenantsTotal: prometheus.NewDesc(
			"vergeos_tenants_total",
			"Total number of tenants",
			[]string{"system_name"}, nil,
		),
		tenantRunning: prometheus.NewDesc(
			"vergeos_tenant_running",
			"Whether the tenant is running (1=running, 0=not running)",
			tenantLabels, nil,
		),
		tenantStatus: prometheus.NewDesc(
			"vergeos_tenant_status",
			"Tenant status (value is always 1, status in label)",
			tenantStatusLabels, nil,
		),
		tenantCPUUsagePct: prometheus.NewDesc(
			"vergeos_tenant_cpu_usage_pct",
			"Tenant total CPU usage percentage",
			tenantLabels, nil,
		),
		tenantCPUCores: prometheus.NewDesc(
			"vergeos_tenant_cpu_cores",
			"Tenant CPU core count",
			tenantLabels, nil,
		),
		tenantRAMUsedBytes: prometheus.NewDesc(
			"vergeos_tenant_ram_used_bytes",
			"Tenant RAM used in bytes",
			tenantLabels, nil,
		),
		tenantRAMAllocBytes: prometheus.NewDesc(
			"vergeos_tenant_ram_allocated_bytes",
			"Tenant RAM allocated in bytes",
			tenantLabels, nil,
		),
		tenantRAMUsagePct: prometheus.NewDesc(
			"vergeos_tenant_ram_usage_pct",
			"Tenant RAM usage percentage",
			tenantLabels, nil,
		),
		tenantIPCount: prometheus.NewDesc(
			"vergeos_tenant_ip_count",
			"Tenant IP address count",
			tenantLabels, nil,
		),
		tenantNodesTotal: prometheus.NewDesc(
			"vergeos_tenant_nodes_total",
			"Number of nodes per tenant",
			tenantLabels, nil,
		),
		tenantVGPUsUsed: prometheus.NewDesc(
			"vergeos_tenant_vgpus_used",
			"Tenant virtual GPUs in use",
			tenantLabels, nil,
		),
		tenantVGPUsTotal: prometheus.NewDesc(
			"vergeos_tenant_vgpus_total",
			"Tenant total virtual GPUs available",
			tenantLabels, nil,
		),
		tenantGPUsUsed: prometheus.NewDesc(
			"vergeos_tenant_gpus_used",
			"Tenant physical GPUs in use",
			tenantLabels, nil,
		),
		tenantGPUsTotal: prometheus.NewDesc(
			"vergeos_tenant_gpus_total",
			"Tenant total physical GPUs available",
			tenantLabels, nil,
		),

		// Tenant node metrics
		tenantNodeCPUCores: prometheus.NewDesc(
			"vergeos_tenant_node_cpu_cores",
			"CPU cores allocated to tenant node",
			tenantNodeLabels, nil,
		),
		tenantNodeRAMBytes: prometheus.NewDesc(
			"vergeos_tenant_node_ram_bytes",
			"RAM allocated to tenant node in bytes",
			tenantNodeLabels, nil,
		),
		tenantNodeEnabled: prometheus.NewDesc(
			"vergeos_tenant_node_enabled",
			"Whether the tenant node is enabled (1=enabled, 0=disabled)",
			tenantNodeLabels, nil,
		),
		tenantNodeRunning: prometheus.NewDesc(
			"vergeos_tenant_node_running",
			"Whether the tenant node is running (1=running, 0=not running)",
			tenantNodeLabels, nil,
		),
		tenantNodeCPUUsagePct: prometheus.NewDesc(
			"vergeos_tenant_node_cpu_usage_pct",
			"Tenant node CPU usage percentage",
			tenantNodeLabels, nil,
		),
		tenantNodeRAMUsedBytes: prometheus.NewDesc(
			"vergeos_tenant_node_ram_used_bytes",
			"Tenant node RAM used in bytes",
			tenantNodeLabels, nil,
		),
		tenantNodeRAMUsagePct: prometheus.NewDesc(
			"vergeos_tenant_node_ram_usage_pct",
			"Tenant node RAM usage percentage",
			tenantNodeLabels, nil,
		),

		// Tenant storage metrics
		tenantStorageProvisioned: prometheus.NewDesc(
			"vergeos_tenant_storage_provisioned_bytes",
			"Tenant storage provisioned in bytes",
			tenantStorageLabels, nil,
		),
		tenantStorageUsed: prometheus.NewDesc(
			"vergeos_tenant_storage_used_bytes",
			"Tenant storage used in bytes",
			tenantStorageLabels, nil,
		),
		tenantStorageAllocated: prometheus.NewDesc(
			"vergeos_tenant_storage_allocated_bytes",
			"Tenant storage allocated in bytes",
			tenantStorageLabels, nil,
		),
		tenantStorageUsedPct: prometheus.NewDesc(
			"vergeos_tenant_storage_used_pct",
			"Tenant storage usage percentage",
			tenantStorageLabels, nil,
		),

		// Tenant network metrics
		tenantL2NetworksTotal: prometheus.NewDesc(
			"vergeos_tenant_layer2_networks_total",
			"Number of layer 2 networks assigned to tenant",
			tenantLabels, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (tc *TenantCollector) Describe(ch chan<- *prometheus.Desc) {
	// Tenant-level
	ch <- tc.tenantsTotal
	ch <- tc.tenantRunning
	ch <- tc.tenantStatus
	ch <- tc.tenantCPUUsagePct
	ch <- tc.tenantCPUCores
	ch <- tc.tenantRAMUsedBytes
	ch <- tc.tenantRAMAllocBytes
	ch <- tc.tenantRAMUsagePct
	ch <- tc.tenantIPCount
	ch <- tc.tenantNodesTotal
	ch <- tc.tenantVGPUsUsed
	ch <- tc.tenantVGPUsTotal
	ch <- tc.tenantGPUsUsed
	ch <- tc.tenantGPUsTotal

	// Tenant node
	ch <- tc.tenantNodeCPUCores
	ch <- tc.tenantNodeRAMBytes
	ch <- tc.tenantNodeEnabled
	ch <- tc.tenantNodeRunning
	ch <- tc.tenantNodeCPUUsagePct
	ch <- tc.tenantNodeRAMUsedBytes
	ch <- tc.tenantNodeRAMUsagePct

	// Tenant storage
	ch <- tc.tenantStorageProvisioned
	ch <- tc.tenantStorageUsed
	ch <- tc.tenantStorageAllocated
	ch <- tc.tenantStorageUsedPct

	// Tenant network
	ch <- tc.tenantL2NetworksTotal
}

// Collect implements prometheus.Collector.
func (tc *TenantCollector) Collect(ch chan<- prometheus.Metric) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	ctx, cancel := tc.ScrapeContext()
	defer cancel()

	systemName, err := tc.GetSystemName(ctx)
	if err != nil {
		log.Printf("TenantCollector: Error getting system name: %v", err)
		return
	}

	// Build tenant map (id -> name) and emit total count
	tenantMap, tenantCount := tc.buildTenantMap(ctx)
	if tenantMap == nil {
		return
	}

	ch <- prometheus.MustNewConstMetric(
		tc.tenantsTotal, prometheus.GaugeValue,
		float64(tenantCount),
		systemName,
	)

	// Collect tenant status metrics
	tc.collectTenantStatusMetrics(ctx, ch, systemName, tenantMap)

	// Collect tenant aggregate stats (CPU, RAM, IP, GPU from TenantStatsHistoryShort)
	tc.collectTenantStatsMetrics(ctx, ch, systemName, tenantMap)

	// Collect tenant node metrics
	tc.collectTenantNodeMetrics(ctx, ch, systemName, tenantMap)

	// Collect tenant storage metrics
	tc.collectTenantStorageMetrics(ctx, ch, systemName, tenantMap)

	// Collect tenant network metrics
	tc.collectTenantNetworkMetrics(ctx, ch, systemName, tenantMap)
}

// buildTenantMap fetches tenants and builds a map of tenant ID to name.
// Returns nil map on error. Filters out snapshot tenants.
func (tc *TenantCollector) buildTenantMap(ctx context.Context) (map[int]string, int) {
	tenants, err := tc.Client().Tenants.List(ctx)
	if err != nil {
		log.Printf("TenantCollector: Error fetching tenants: %v", err)
		return nil, 0
	}

	tenantMap := make(map[int]string)
	count := 0
	for _, t := range tenants {
		if t.IsSnapshot {
			continue
		}
		tenantMap[int(t.Key)] = t.Name
		count++
	}

	return tenantMap, count
}

// tenantName resolves a tenant ID to its name using the map, with fallback.
func tenantName(tenantMap map[int]string, tenantID int) string {
	if name, ok := tenantMap[tenantID]; ok {
		return name
	}
	return fmt.Sprintf("tenant_%d", tenantID)
}

// collectTenantStatusMetrics emits running and status metrics per tenant.
func (tc *TenantCollector) collectTenantStatusMetrics(ctx context.Context, ch chan<- prometheus.Metric, systemName string, tenantMap map[int]string) {
	statuses, err := tc.Client().TenantStatus.List(ctx)
	if err != nil {
		log.Printf("TenantCollector: Error fetching tenant statuses: %v", err)
		return
	}

	for _, s := range statuses {
		name := tenantName(tenantMap, s.Tenant)
		// Skip tenants not in our map (e.g. snapshots)
		if _, ok := tenantMap[s.Tenant]; !ok {
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			tc.tenantRunning, prometheus.GaugeValue,
			boolToFloat64(s.Running),
			systemName, name,
		)

		// Info-style metric with status as label
		if s.Status != "" {
			ch <- prometheus.MustNewConstMetric(
				tc.tenantStatus, prometheus.GaugeValue,
				1.0,
				systemName, name, s.Status,
			)
		}
	}
}

// collectTenantStatsMetrics emits aggregate CPU/RAM/IP/GPU metrics per tenant
// using TenantStatsHistoryShort.GetLatest().
func (tc *TenantCollector) collectTenantStatsMetrics(ctx context.Context, ch chan<- prometheus.Metric, systemName string, tenantMap map[int]string) {
	for tenantID, name := range tenantMap {
		stats, err := tc.Client().TenantStatsHistoryShort.GetLatest(ctx, tenantID)
		if err != nil {
			if vergeos.IsNotFoundError(err) {
				// Offline tenants may not have stats
				continue
			}
			log.Printf("TenantCollector: Error fetching stats for tenant %s: %v", name, err)
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			tc.tenantCPUUsagePct, prometheus.GaugeValue,
			float64(stats.TotalCPU),
			systemName, name,
		)
		ch <- prometheus.MustNewConstMetric(
			tc.tenantCPUCores, prometheus.GaugeValue,
			float64(stats.CoreCount),
			systemName, name,
		)
		ch <- prometheus.MustNewConstMetric(
			tc.tenantRAMUsedBytes, prometheus.GaugeValue,
			float64(stats.RAMUsed)*1048576,
			systemName, name,
		)
		ch <- prometheus.MustNewConstMetric(
			tc.tenantRAMAllocBytes, prometheus.GaugeValue,
			float64(stats.RAMAllocated)*1048576,
			systemName, name,
		)
		ch <- prometheus.MustNewConstMetric(
			tc.tenantRAMUsagePct, prometheus.GaugeValue,
			float64(stats.RAMPct),
			systemName, name,
		)
		ch <- prometheus.MustNewConstMetric(
			tc.tenantIPCount, prometheus.GaugeValue,
			float64(stats.IPCount),
			systemName, name,
		)

		// GPU metrics — only emit when GPU resources exist
		if stats.VGPUsTotal > 0 || stats.GPUsTotal > 0 {
			ch <- prometheus.MustNewConstMetric(
				tc.tenantVGPUsUsed, prometheus.GaugeValue,
				float64(stats.VGPUsUsed),
				systemName, name,
			)
			ch <- prometheus.MustNewConstMetric(
				tc.tenantVGPUsTotal, prometheus.GaugeValue,
				float64(stats.VGPUsTotal),
				systemName, name,
			)
			ch <- prometheus.MustNewConstMetric(
				tc.tenantGPUsUsed, prometheus.GaugeValue,
				float64(stats.GPUsUsed),
				systemName, name,
			)
			ch <- prometheus.MustNewConstMetric(
				tc.tenantGPUsTotal, prometheus.GaugeValue,
				float64(stats.GPUsTotal),
				systemName, name,
			)
		}
	}
}

// collectTenantNodeMetrics emits per-node allocation and runtime metrics.
func (tc *TenantCollector) collectTenantNodeMetrics(ctx context.Context, ch chan<- prometheus.Metric, systemName string, tenantMap map[int]string) {
	nodes, err := tc.Client().TenantNodes.List(ctx)
	if err != nil {
		log.Printf("TenantCollector: Error fetching tenant nodes: %v", err)
		return
	}

	// Batch-fetch machine statuses and stats (avoids N+1 per-node API calls)
	allStatuses, err := tc.Client().MachineStatus.List(ctx)
	if err != nil {
		log.Printf("TenantCollector: Error batch-fetching machine statuses: %v", err)
	}
	statusMap := make(map[int]*vergeos.MachineStatus)
	for i := range allStatuses {
		statusMap[allStatuses[i].Machine] = &allStatuses[i]
	}

	allStats, err := tc.Client().MachineStats.List(ctx)
	if err != nil {
		log.Printf("TenantCollector: Error batch-fetching machine stats: %v", err)
	}
	statsMap := make(map[int]*vergeos.MachineStats)
	for i := range allStats {
		statsMap[allStats[i].Machine] = &allStats[i]
	}

	// Count nodes per tenant
	nodeCounts := make(map[int]int)

	for _, node := range nodes {
		if node.IsSnapshot {
			continue
		}

		tid := int(node.Tenant)
		// Skip nodes for tenants not in our map (snapshots)
		if _, ok := tenantMap[tid]; !ok {
			continue
		}

		tName := tenantName(tenantMap, tid)
		nodeCounts[tid]++

		// Allocation metrics
		ch <- prometheus.MustNewConstMetric(
			tc.tenantNodeCPUCores, prometheus.GaugeValue,
			float64(node.CPUCores),
			systemName, tName, node.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			tc.tenantNodeRAMBytes, prometheus.GaugeValue,
			float64(node.RAM)*1048576,
			systemName, tName, node.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			tc.tenantNodeEnabled, prometheus.GaugeValue,
			boolToFloat64(node.Enabled),
			systemName, tName, node.Name,
		)

		// Runtime metrics from pre-fetched machine status/stats
		machineID := int(node.Machine)
		if machineID > 0 {
			if status, ok := statusMap[machineID]; ok {
				ch <- prometheus.MustNewConstMetric(
					tc.tenantNodeRunning, prometheus.GaugeValue,
					boolToFloat64(status.Running),
					systemName, tName, node.Name,
				)
			}

			if stats, ok := statsMap[machineID]; ok {
				ch <- prometheus.MustNewConstMetric(
					tc.tenantNodeCPUUsagePct, prometheus.GaugeValue,
					float64(stats.TotalCPU),
					systemName, tName, node.Name,
				)
				ch <- prometheus.MustNewConstMetric(
					tc.tenantNodeRAMUsedBytes, prometheus.GaugeValue,
					float64(stats.RAMUsed)*1048576,
					systemName, tName, node.Name,
				)
				ch <- prometheus.MustNewConstMetric(
					tc.tenantNodeRAMUsagePct, prometheus.GaugeValue,
					float64(stats.RAMPct),
					systemName, tName, node.Name,
				)
			}
		}
	}

	// Emit node counts per tenant
	for tid, count := range nodeCounts {
		ch <- prometheus.MustNewConstMetric(
			tc.tenantNodesTotal, prometheus.GaugeValue,
			float64(count),
			systemName, tenantName(tenantMap, tid),
		)
	}
}

// collectTenantStorageMetrics emits per-tier storage allocation metrics.
func (tc *TenantCollector) collectTenantStorageMetrics(ctx context.Context, ch chan<- prometheus.Metric, systemName string, tenantMap map[int]string) {
	storage, err := tc.Client().TenantStorage.List(ctx)
	if err != nil {
		log.Printf("TenantCollector: Error fetching tenant storage: %v", err)
		return
	}

	for _, s := range storage {
		tid := int(s.Tenant)
		if _, ok := tenantMap[tid]; !ok {
			continue
		}

		tName := tenantName(tenantMap, tid)
		tierStr := fmt.Sprintf("%d", int(s.Tier))

		ch <- prometheus.MustNewConstMetric(
			tc.tenantStorageProvisioned, prometheus.GaugeValue,
			float64(s.Provisioned),
			systemName, tName, tierStr,
		)
		ch <- prometheus.MustNewConstMetric(
			tc.tenantStorageUsed, prometheus.GaugeValue,
			float64(s.Used),
			systemName, tName, tierStr,
		)
		ch <- prometheus.MustNewConstMetric(
			tc.tenantStorageAllocated, prometheus.GaugeValue,
			float64(s.Allocated),
			systemName, tName, tierStr,
		)
		ch <- prometheus.MustNewConstMetric(
			tc.tenantStorageUsedPct, prometheus.GaugeValue,
			float64(s.UsedPct),
			systemName, tName, tierStr,
		)
	}
}

// collectTenantNetworkMetrics emits L2 network count per tenant.
func (tc *TenantCollector) collectTenantNetworkMetrics(ctx context.Context, ch chan<- prometheus.Metric, systemName string, tenantMap map[int]string) {
	networks, err := tc.Client().TenantLayer2Networks.List(ctx)
	if err != nil {
		log.Printf("TenantCollector: Error fetching tenant L2 networks: %v", err)
		return
	}

	// Count networks per tenant
	netCounts := make(map[int]int)
	for _, n := range networks {
		tid := int(n.Tenant)
		if _, ok := tenantMap[tid]; !ok {
			continue
		}
		netCounts[tid]++
	}

	for tid, count := range netCounts {
		ch <- prometheus.MustNewConstMetric(
			tc.tenantL2NetworksTotal, prometheus.GaugeValue,
			float64(count),
			systemName, tenantName(tenantMap, tid),
		)
	}
}
