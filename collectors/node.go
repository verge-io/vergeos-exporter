package collectors

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/goVergeOS"
)

// NodeCollector collects metrics about VergeOS physical nodes
type NodeCollector struct {
	BaseCollector
	mutex sync.Mutex

	// Metric descriptors (using MustNewConstMetric pattern to avoid stale metrics)
	nodesTotal     *prometheus.Desc
	nodeIPMIStatus *prometheus.Desc
	nodeRAMTotal   *prometheus.Desc
	nodeRAM        *prometheus.Desc

	// MachineStats metrics (Issue 1)
	nodeCPUCoreUsage *prometheus.Desc
	nodeCoreTemp     *prometheus.Desc
	nodeRAMUsed      *prometheus.Desc
	nodeRAMPct       *prometheus.Desc

	// VM aggregate metrics (Issue 7)
	nodeRunningCores *prometheus.Desc
	nodeRunningRAM   *prometheus.Desc
}

// NewNodeCollector creates a new NodeCollector
func NewNodeCollector(client *vergeos.Client, scrapeTimeout time.Duration) *NodeCollector {
	nodeLabels := []string{"system_name", "cluster", "node_name"}

	nc := &NodeCollector{
		BaseCollector: *NewBaseCollector(client, scrapeTimeout),
		nodesTotal: prometheus.NewDesc(
			"vergeos_nodes_total",
			"Total number of physical nodes",
			[]string{"system_name", "cluster"},
			nil,
		),
		nodeIPMIStatus: prometheus.NewDesc(
			"vergeos_node_ipmi_status",
			"IPMI status of the node (1=ok, 0=other)",
			nodeLabels,
			nil,
		),
		nodeRAMTotal: prometheus.NewDesc(
			"vergeos_node_ram_total",
			"Total RAM in MB",
			nodeLabels,
			nil,
		),
		nodeRAM: prometheus.NewDesc(
			"vergeos_node_ram_allocated",
			"VM RAM in MB (vm_ram field)",
			nodeLabels,
			nil,
		),
		nodeCPUCoreUsage: prometheus.NewDesc(
			"vergeos_node_cpu_core_usage",
			"CPU usage percentage per core",
			append(nodeLabels, "core_id"),
			nil,
		),
		nodeCoreTemp: prometheus.NewDesc(
			"vergeos_node_core_temp",
			"Average CPU core temperature in Celsius",
			nodeLabels,
			nil,
		),
		nodeRAMUsed: prometheus.NewDesc(
			"vergeos_node_ram_used",
			"Physical RAM used in MB",
			nodeLabels,
			nil,
		),
		nodeRAMPct: prometheus.NewDesc(
			"vergeos_node_ram_pct",
			"Physical RAM used percentage",
			nodeLabels,
			nil,
		),
		nodeRunningCores: prometheus.NewDesc(
			"vergeos_node_running_cores",
			"Total CPU cores allocated to running VMs",
			nodeLabels,
			nil,
		),
		nodeRunningRAM: prometheus.NewDesc(
			"vergeos_node_running_ram",
			"Total RAM in MB allocated to running VMs",
			nodeLabels,
			nil,
		),
	}

	return nc
}

// Describe implements prometheus.Collector
func (nc *NodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nc.nodesTotal
	ch <- nc.nodeIPMIStatus
	ch <- nc.nodeRAMTotal
	ch <- nc.nodeRAM
	ch <- nc.nodeCPUCoreUsage
	ch <- nc.nodeCoreTemp
	ch <- nc.nodeRAMUsed
	ch <- nc.nodeRAMPct
	ch <- nc.nodeRunningCores
	ch <- nc.nodeRunningRAM
}

// Collect implements prometheus.Collector
func (nc *NodeCollector) Collect(ch chan<- prometheus.Metric) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	ctx, cancel := nc.ScrapeContext()
	defer cancel()

	// Get system name using SDK
	systemName, err := nc.GetSystemName(ctx)
	if err != nil {
		log.Printf("Error getting system name: %v", err)
		return
	}

	// Build cluster ID -> name mapping
	clusterMap, err := nc.BuildClusterMap(ctx)
	if err != nil {
		log.Printf("Error building cluster map: %v", err)
		return
	}

	// Get physical nodes using SDK
	nodes, err := nc.Client().Nodes.ListPhysical(ctx)
	if err != nil {
		log.Printf("Error fetching physical nodes: %v", err)
		return
	}

	// Count nodes per cluster
	clusterNodeCounts := make(map[string]int)

	// Process each node
	for _, node := range nodes {
		// Get cluster name from mapping
		clusterName := clusterMap[node.Cluster]
		if clusterName == "" {
			clusterName = fmt.Sprintf("cluster_%d", node.Cluster)
		}

		// Count nodes per cluster
		clusterNodeCounts[clusterName]++

		// IPMI status (1=ok, 0=other)
		ipmiStatus := 0.0
		if node.IPMIStatus == "ok" {
			ipmiStatus = 1.0
		}

		ch <- prometheus.MustNewConstMetric(
			nc.nodeIPMIStatus,
			prometheus.GaugeValue,
			ipmiStatus,
			systemName, clusterName, node.Name,
		)

		// RAM total
		ch <- prometheus.MustNewConstMetric(
			nc.nodeRAMTotal,
			prometheus.GaugeValue,
			float64(node.RAM),
			systemName, clusterName, node.Name,
		)

		// VM RAM allocated
		ch <- prometheus.MustNewConstMetric(
			nc.nodeRAM,
			prometheus.GaugeValue,
			float64(node.VMRAM),
			systemName, clusterName, node.Name,
		)

		// VM aggregate metrics (Issue 7)
		if node.VMStatsTotals != nil {
			ch <- prometheus.MustNewConstMetric(
				nc.nodeRunningCores,
				prometheus.GaugeValue,
				float64(node.VMStatsTotals.RunningCores),
				systemName, clusterName, node.Name,
			)
			ch <- prometheus.MustNewConstMetric(
				nc.nodeRunningRAM,
				prometheus.GaugeValue,
				float64(node.VMStatsTotals.RunningRAM),
				systemName, clusterName, node.Name,
			)
		}

		// Fetch MachineStats for this node
		stats, err := nc.Client().MachineStats.GetByMachine(ctx, node.Machine)
		if err != nil {
			log.Printf("Error fetching machine stats for node %s (machine %d): %v", node.Name, node.Machine, err)
			continue
		}

		// Per-core CPU usage
		coreUsages, err := stats.GetCoreUsages()
		if err != nil {
			log.Printf("Error parsing core usages for node %s: %v", node.Name, err)
		} else {
			for i, usage := range coreUsages {
				ch <- prometheus.MustNewConstMetric(
					nc.nodeCPUCoreUsage,
					prometheus.GaugeValue,
					usage,
					systemName, clusterName, node.Name, fmt.Sprintf("%d", i),
				)
			}
		}

		// Core temperature
		ch <- prometheus.MustNewConstMetric(
			nc.nodeCoreTemp,
			prometheus.GaugeValue,
			float64(stats.CoreTemp),
			systemName, clusterName, node.Name,
		)

		// RAM used (MB)
		ch <- prometheus.MustNewConstMetric(
			nc.nodeRAMUsed,
			prometheus.GaugeValue,
			float64(stats.RAMUsed),
			systemName, clusterName, node.Name,
		)

		// RAM percentage
		ch <- prometheus.MustNewConstMetric(
			nc.nodeRAMPct,
			prometheus.GaugeValue,
			float64(stats.RAMPct),
			systemName, clusterName, node.Name,
		)
	}

	// Emit total nodes metric with "all" label for overall count
	ch <- prometheus.MustNewConstMetric(
		nc.nodesTotal,
		prometheus.GaugeValue,
		float64(len(nodes)),
		systemName, "all",
	)

	// Emit nodes per cluster
	for clusterName, count := range clusterNodeCounts {
		ch <- prometheus.MustNewConstMetric(
			nc.nodesTotal,
			prometheus.GaugeValue,
			float64(count),
			systemName, clusterName,
		)
	}
}
