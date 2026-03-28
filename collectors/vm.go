package collectors

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/goVergeOS"
)

// vmStatus holds resolved status info for a VM's machine
type vmStatus struct {
	NodeName string
	Running  bool
}

// VMCollector collects per-VM metrics
type VMCollector struct {
	BaseCollector
	mutex sync.Mutex

	// CPU metrics
	vmCPUTotal  *prometheus.Desc
	vmCPUUser   *prometheus.Desc
	vmCPUSystem *prometheus.Desc
	vmCPUIowait *prometheus.Desc

	// State metrics
	vmRunning *prometheus.Desc
	vmEnabled *prometheus.Desc

	// Config metrics
	vmCPUCores *prometheus.Desc
	vmRAMBytes *prometheus.Desc

	// NIC metrics
	vmNICTxBytes   *prometheus.Desc
	vmNICRxBytes   *prometheus.Desc
	vmNICTxPackets *prometheus.Desc
	vmNICRxPackets *prometheus.Desc
}

// NewVMCollector creates a new VMCollector
func NewVMCollector(client *vergeos.Client) *VMCollector {
	vmLabels := []string{"system_name", "cluster", "node", "vm_name", "vm_id"}
	nicLabels := append(vmLabels, "nic_name")

	return &VMCollector{
		BaseCollector: *NewBaseCollector(client),
		vmCPUTotal: prometheus.NewDesc(
			"vergeos_vm_cpu_total",
			"Total CPU usage percentage",
			vmLabels,
			nil,
		),
		vmCPUUser: prometheus.NewDesc(
			"vergeos_vm_cpu_user",
			"User CPU usage percentage",
			vmLabels,
			nil,
		),
		vmCPUSystem: prometheus.NewDesc(
			"vergeos_vm_cpu_system",
			"System CPU usage percentage",
			vmLabels,
			nil,
		),
		vmCPUIowait: prometheus.NewDesc(
			"vergeos_vm_cpu_iowait",
			"IO Wait CPU usage percentage",
			vmLabels,
			nil,
		),
		vmRunning: prometheus.NewDesc(
			"vergeos_vm_running",
			"Whether the VM is running (1=running, 0=not running)",
			vmLabels,
			nil,
		),
		vmEnabled: prometheus.NewDesc(
			"vergeos_vm_enabled",
			"Whether the VM is enabled (1=enabled, 0=disabled)",
			vmLabels,
			nil,
		),
		vmCPUCores: prometheus.NewDesc(
			"vergeos_vm_cpu_cores",
			"Number of configured CPU cores",
			vmLabels,
			nil,
		),
		vmRAMBytes: prometheus.NewDesc(
			"vergeos_vm_ram_bytes",
			"Configured RAM in bytes",
			vmLabels,
			nil,
		),
		vmNICTxBytes: prometheus.NewDesc(
			"vergeos_vm_nic_tx_bytes_total",
			"Total transmitted bytes",
			nicLabels,
			nil,
		),
		vmNICRxBytes: prometheus.NewDesc(
			"vergeos_vm_nic_rx_bytes_total",
			"Total received bytes",
			nicLabels,
			nil,
		),
		vmNICTxPackets: prometheus.NewDesc(
			"vergeos_vm_nic_tx_packets_total",
			"Total transmitted packets",
			nicLabels,
			nil,
		),
		vmNICRxPackets: prometheus.NewDesc(
			"vergeos_vm_nic_rx_packets_total",
			"Total received packets",
			nicLabels,
			nil,
		),
	}
}

// Describe implements prometheus.Collector
func (vc *VMCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- vc.vmCPUTotal
	ch <- vc.vmCPUUser
	ch <- vc.vmCPUSystem
	ch <- vc.vmCPUIowait
	ch <- vc.vmRunning
	ch <- vc.vmEnabled
	ch <- vc.vmCPUCores
	ch <- vc.vmRAMBytes
	ch <- vc.vmNICTxBytes
	ch <- vc.vmNICRxBytes
	ch <- vc.vmNICTxPackets
	ch <- vc.vmNICRxPackets
}

// Collect implements prometheus.Collector
func (vc *VMCollector) Collect(ch chan<- prometheus.Metric) {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()

	ctx := context.Background()

	systemName, err := vc.GetSystemName(ctx)
	if err != nil {
		log.Printf("Error getting system name: %v", err)
		return
	}

	// Build cluster ID → name mapping
	clusterMap, err := vc.buildClusterMap(ctx)
	if err != nil {
		log.Printf("Error building cluster map: %v", err)
		return
	}

	// Fetch all non-snapshot VMs
	vms, err := vc.Client().VMs.List(ctx, vergeos.WithFilter("is_snapshot eq false"))
	if err != nil {
		log.Printf("Error fetching VMs: %v", err)
		return
	}

	// Batch fetch machine stats → map[machineID]*MachineStats
	statsMap, err := vc.buildStatsMap(ctx)
	if err != nil {
		log.Printf("Error fetching machine stats: %v", err)
		return
	}

	// Batch fetch machine status with reduced fields (avoids agent_guest_info bug)
	statusMap, err := vc.buildStatusMap(ctx)
	if err != nil {
		log.Printf("Error fetching machine status: %v", err)
		return
	}

	// Batch fetch all NIC stats
	nicMap, err := vc.buildNICMap(ctx)
	if err != nil {
		log.Printf("Error fetching NIC stats: %v", err)
		// Non-fatal: continue without NIC metrics
	}

	for _, vm := range vms {
		vmID := fmt.Sprintf("%d", int(vm.ID))

		// Resolve cluster name
		clusterName := clusterMap[int(vm.Cluster)]
		if clusterName == "" {
			clusterName = fmt.Sprintf("cluster_%d", int(vm.Cluster))
		}

		// Resolve status (node name + running state)
		status := statusMap[vm.Machine]
		nodeName := status.NodeName

		labels := []string{systemName, clusterName, nodeName, vm.Name, vmID}

		// State metrics
		ch <- prometheus.MustNewConstMetric(vc.vmRunning, prometheus.GaugeValue, boolToFloat64(status.Running), labels...)
		ch <- prometheus.MustNewConstMetric(vc.vmEnabled, prometheus.GaugeValue, boolToFloat64(vm.Enabled), labels...)

		// Config metrics
		ch <- prometheus.MustNewConstMetric(vc.vmCPUCores, prometheus.GaugeValue, float64(vm.CPUCores), labels...)
		ch <- prometheus.MustNewConstMetric(vc.vmRAMBytes, prometheus.GaugeValue, float64(vm.RAM)*1048576, labels...)

		// CPU stats (0 if powered off or no stats available)
		var totalCPU, userCPU, systemCPU, iowaitCPU float64
		if stats, ok := statsMap[vm.Machine]; ok {
			totalCPU = float64(stats.TotalCPU)
			userCPU = float64(stats.UserCPU)
			systemCPU = float64(stats.SystemCPU)
			iowaitCPU = float64(stats.IOWaitCPU)
		}

		ch <- prometheus.MustNewConstMetric(vc.vmCPUTotal, prometheus.GaugeValue, totalCPU, labels...)
		ch <- prometheus.MustNewConstMetric(vc.vmCPUUser, prometheus.GaugeValue, userCPU, labels...)
		ch <- prometheus.MustNewConstMetric(vc.vmCPUSystem, prometheus.GaugeValue, systemCPU, labels...)
		ch <- prometheus.MustNewConstMetric(vc.vmCPUIowait, prometheus.GaugeValue, iowaitCPU, labels...)

		// NIC metrics (only for VMs with NICs)
		if nics, ok := nicMap[vm.Machine]; ok {
			for _, nic := range nics {
				if nic.Stats == nil {
					continue
				}
				nicLabels := append(labels, nic.Name)
				ch <- prometheus.MustNewConstMetric(vc.vmNICTxBytes, prometheus.CounterValue, float64(nic.Stats.TxBytes), nicLabels...)
				ch <- prometheus.MustNewConstMetric(vc.vmNICRxBytes, prometheus.CounterValue, float64(nic.Stats.RxBytes), nicLabels...)
				ch <- prometheus.MustNewConstMetric(vc.vmNICTxPackets, prometheus.CounterValue, float64(nic.Stats.TxPckts), nicLabels...)
				ch <- prometheus.MustNewConstMetric(vc.vmNICRxPackets, prometheus.CounterValue, float64(nic.Stats.RxPckts), nicLabels...)
			}
		}
	}
}

// buildClusterMap creates a mapping from cluster ID to cluster name
func (vc *VMCollector) buildClusterMap(ctx context.Context) (map[int]string, error) {
	clusters, err := vc.Client().Clusters.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	clusterMap := make(map[int]string)
	for _, cluster := range clusters {
		clusterMap[int(cluster.Key)] = cluster.Name
	}
	return clusterMap, nil
}

// buildStatsMap batch-fetches all machine stats and returns a map keyed by machine ID
func (vc *VMCollector) buildStatsMap(ctx context.Context) (map[int]*vergeos.MachineStats, error) {
	allStats, err := vc.Client().MachineStats.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list machine stats: %w", err)
	}

	statsMap := make(map[int]*vergeos.MachineStats)
	for i := range allStats {
		statsMap[allStats[i].Machine] = &allStats[i]
	}
	return statsMap, nil
}

// buildStatusMap batch-fetches machine statuses and returns a map of machine ID → vmStatus.
// Uses reduced fields to avoid the agent_guest_info deserialization bug in the SDK.
func (vc *VMCollector) buildStatusMap(ctx context.Context) (map[int]vmStatus, error) {
	statuses, err := vc.Client().MachineStatus.List(ctx,
		vergeos.WithFields("machine,running,node,node#name as node_name"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list machine status: %w", err)
	}

	statusMap := make(map[int]vmStatus)
	for _, s := range statuses {
		statusMap[s.Machine] = vmStatus{
			NodeName: s.NodeName,
			Running:  s.Running,
		}
	}
	return statusMap, nil
}

// buildNICMap batch-fetches all machine NICs and returns a map of machine ID → []MachineNIC.
func (vc *VMCollector) buildNICMap(ctx context.Context) (map[int][]vergeos.MachineNIC, error) {
	allNICs, err := vc.Client().MachineNICs.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list machine NICs: %w", err)
	}

	nicMap := make(map[int][]vergeos.MachineNIC)
	for _, nic := range allNICs {
		nicMap[nic.Machine] = append(nicMap[nic.Machine], nic)
	}
	return nicMap, nil
}
