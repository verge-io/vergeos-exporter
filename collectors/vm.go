package collectors

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/govergeos"
)

var _ prometheus.Collector = (*VMCollector)(nil)

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

	// Disk config metrics
	vmDiskSizeBytes *prometheus.Desc
	vmDiskUsedBytes *prometheus.Desc

	// Disk I/O metrics
	vmDiskReadOps     *prometheus.Desc
	vmDiskWriteOps    *prometheus.Desc
	vmDiskReadBytes   *prometheus.Desc
	vmDiskWriteBytes  *prometheus.Desc
	vmDiskUtil        *prometheus.Desc
	vmDiskServiceTime *prometheus.Desc
}

// NewVMCollector creates a new VMCollector
func NewVMCollector(client *vergeos.Client, scrapeTimeout time.Duration) *VMCollector {
	vmLabels := []string{"system_name", "cluster", "node", "vm_name", "vm_id"}
	nicLabels := []string{"system_name", "cluster", "node", "vm_name", "vm_id", "nic_name"}
	diskLabels := []string{"system_name", "cluster", "node", "vm_name", "vm_id", "disk_name", "interface", "media"}

	return &VMCollector{
		BaseCollector: *NewBaseCollector(client, scrapeTimeout),
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
		vmDiskSizeBytes: prometheus.NewDesc(
			"vergeos_vm_disk_size_bytes",
			"Configured disk size in bytes",
			diskLabels,
			nil,
		),
		vmDiskUsedBytes: prometheus.NewDesc(
			"vergeos_vm_disk_used_bytes",
			"Actual disk used space in bytes",
			diskLabels,
			nil,
		),
		vmDiskReadOps: prometheus.NewDesc(
			"vergeos_vm_disk_read_ops_total",
			"Total disk read operations",
			diskLabels,
			nil,
		),
		vmDiskWriteOps: prometheus.NewDesc(
			"vergeos_vm_disk_write_ops_total",
			"Total disk write operations",
			diskLabels,
			nil,
		),
		vmDiskReadBytes: prometheus.NewDesc(
			"vergeos_vm_disk_read_bytes_total",
			"Total disk bytes read",
			diskLabels,
			nil,
		),
		vmDiskWriteBytes: prometheus.NewDesc(
			"vergeos_vm_disk_write_bytes_total",
			"Total disk bytes written",
			diskLabels,
			nil,
		),
		vmDiskUtil: prometheus.NewDesc(
			"vergeos_vm_disk_util",
			"Disk I/O utilization percentage",
			diskLabels,
			nil,
		),
		vmDiskServiceTime: prometheus.NewDesc(
			"vergeos_vm_disk_service_time",
			"Disk average I/O service time in milliseconds",
			diskLabels,
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
	ch <- vc.vmDiskSizeBytes
	ch <- vc.vmDiskUsedBytes
	ch <- vc.vmDiskReadOps
	ch <- vc.vmDiskWriteOps
	ch <- vc.vmDiskReadBytes
	ch <- vc.vmDiskWriteBytes
	ch <- vc.vmDiskUtil
	ch <- vc.vmDiskServiceTime
}

// Collect implements prometheus.Collector
func (vc *VMCollector) Collect(ch chan<- prometheus.Metric) {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()

	ctx, cancel := vc.ScrapeContext()
	defer cancel()

	systemName, err := vc.GetSystemName(ctx)
	if err != nil {
		log.Printf("Error getting system name: %v", err)
		return
	}

	// Build cluster ID → name mapping
	clusterMap, err := vc.BuildClusterMap(ctx)
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

	// Batch fetch machine status
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

	// Batch fetch all VM drives and virtual drive stats
	diskMap, err := vc.buildDiskMap(ctx)
	if err != nil {
		log.Printf("Error fetching VM drives: %v", err)
		// Non-fatal: continue without disk metrics
	}

	diskStatsMap, err := vc.buildDiskStatsMap(ctx)
	if err != nil {
		log.Printf("Error fetching VM disk stats: %v", err)
		// Non-fatal: continue without disk I/O metrics
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

		// Disk metrics (only for VMs with drives)
		if disks, ok := diskMap[vm.Machine]; ok {
			for _, disk := range disks {
				diskLabels := append(labels, disk.Name, disk.Interface, disk.Media)

				// Config: size and used
				ch <- prometheus.MustNewConstMetric(vc.vmDiskSizeBytes, prometheus.GaugeValue, float64(disk.SizeBytes), diskLabels...)
				ch <- prometheus.MustNewConstMetric(vc.vmDiskUsedBytes, prometheus.GaugeValue, float64(disk.UsedBytes), diskLabels...)

				// I/O stats (if available)
				if dStats, ok := diskStatsMap[int(disk.ID)]; ok {
					ch <- prometheus.MustNewConstMetric(vc.vmDiskReadOps, prometheus.CounterValue, float64(dStats.Reads), diskLabels...)
					ch <- prometheus.MustNewConstMetric(vc.vmDiskWriteOps, prometheus.CounterValue, float64(dStats.Writes), diskLabels...)
					ch <- prometheus.MustNewConstMetric(vc.vmDiskReadBytes, prometheus.CounterValue, float64(dStats.ReadBytes), diskLabels...)
					ch <- prometheus.MustNewConstMetric(vc.vmDiskWriteBytes, prometheus.CounterValue, float64(dStats.WriteBytes), diskLabels...)
					ch <- prometheus.MustNewConstMetric(vc.vmDiskUtil, prometheus.GaugeValue, dStats.Util, diskLabels...)
					ch <- prometheus.MustNewConstMetric(vc.vmDiskServiceTime, prometheus.GaugeValue, dStats.ServiceTime, diskLabels...)
				}
			}
		}
	}
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
func (vc *VMCollector) buildStatusMap(ctx context.Context) (map[int]vmStatus, error) {
	statuses, err := vc.Client().MachineStatus.List(ctx)
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

// buildDiskMap batch-fetches all VM drives and returns a map of machine ID → []VMDrive.
func (vc *VMCollector) buildDiskMap(ctx context.Context) (map[int][]vergeos.VMDrive, error) {
	allDrives, err := vc.Client().VMDrives.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list VM drives: %w", err)
	}

	diskMap := make(map[int][]vergeos.VMDrive)
	for _, drive := range allDrives {
		diskMap[drive.Machine] = append(diskMap[drive.Machine], drive)
	}
	return diskMap, nil
}

// buildDiskStatsMap batch-fetches virtual drive stats and returns a map of parent drive ID → MachineDriveStats.
func (vc *VMCollector) buildDiskStatsMap(ctx context.Context) (map[int]*vergeos.MachineDriveStats, error) {
	allStats, err := vc.Client().MachineDriveStats.List(ctx,
		vergeos.WithFilter("physical eq false"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list VM disk stats: %w", err)
	}

	statsMap := make(map[int]*vergeos.MachineDriveStats)
	for i := range allStats {
		statsMap[allStats[i].ParentDrive] = &allStats[i]
	}
	return statsMap, nil
}
