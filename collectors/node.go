package collectors

import (
	"context"
	"fmt"
	"log"
	"sync"

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
}

// NewNodeCollector creates a new NodeCollector
func NewNodeCollector(client *vergeos.Client) *NodeCollector {
	nc := &NodeCollector{
		BaseCollector: *NewBaseCollector(client),
		nodesTotal: prometheus.NewDesc(
			"vergeos_nodes_total",
			"Total number of physical nodes",
			[]string{"system_name", "cluster"},
			nil,
		),
		nodeIPMIStatus: prometheus.NewDesc(
			"vergeos_node_ipmi_status",
			"IPMI status of the node (1=ok, 0=other)",
			[]string{"system_name", "cluster", "node_name"},
			nil,
		),
		nodeRAMTotal: prometheus.NewDesc(
			"vergeos_node_ram_total",
			"Total RAM in MB",
			[]string{"system_name", "cluster", "node_name"},
			nil,
		),
		nodeRAM: prometheus.NewDesc(
			"vergeos_node_ram_allocated",
			"VM RAM in MB (vm_ram field)",
			[]string{"system_name", "cluster", "node_name"},
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
}

// Collect implements prometheus.Collector
func (nc *NodeCollector) Collect(ch chan<- prometheus.Metric) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	ctx := context.Background()

	// Get system name using SDK
	systemName, err := nc.GetSystemName(ctx)
	if err != nil {
		log.Printf("Error getting system name: %v", err)
		return
	}

	// Build cluster ID -> name mapping
	clusterMap, err := nc.buildClusterMap(ctx)
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

// buildClusterMap creates a mapping from cluster ID to cluster name
func (nc *NodeCollector) buildClusterMap(ctx context.Context) (map[int]string, error) {
	clusters, err := nc.Client().Clusters.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	clusterMap := make(map[int]string)
	for _, cluster := range clusters {
		clusterMap[int(cluster.Key)] = cluster.Name
	}

	return clusterMap, nil
}
