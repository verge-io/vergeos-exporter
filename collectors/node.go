package collectors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Node represents a VergeOS node
type Node struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	ID             int    `json:"id"`
	Machine        int    `json:"machine"`
	Physical       bool   `json:"physical"`
	IPMIStatus     string `json:"ipmi_status"`
	IPMIStatusInfo string `json:"ipmi_status_info"`
	Cluster        int    `json:"cluster"`
}

// NodeCollector collects metrics about VergeOS nodes
type NodeCollector struct {
	BaseCollector
	mutex sync.Mutex

	// System info
	systemName string

	// Metrics
	nodesTotalDesc       *prometheus.Desc
	nodeIPMIStatusDesc   *prometheus.Desc
	nodeCPUUsageDesc     *prometheus.Desc
	nodeCoreTempDesc     *prometheus.Desc
	nodeRAMUsedDesc      *prometheus.Desc
	nodeRAMPercentDesc   *prometheus.Desc
	nodeRAMDesc          *prometheus.Desc
	nodeRAMTotalDesc     *prometheus.Desc
	nodeRunningCoresDesc *prometheus.Desc
	nodeRunningRAMDesc   *prometheus.Desc
}

// NewNodeCollector creates a new NodeCollector
func NewNodeCollector(url string, client *http.Client, username, password string) *NodeCollector {
	nc := &NodeCollector{
		BaseCollector: BaseCollector{
			url:        url,
			httpClient: client,
		},
		systemName: "unknown", // Will be updated in Collect
		nodesTotalDesc: prometheus.NewDesc(
			"vergeos_nodes_total",
			"Total number of physical nodes",
			[]string{"system_name", "cluster"},
			nil,
		),
		nodeIPMIStatusDesc: prometheus.NewDesc(
			"vergeos_node_ipmi_status",
			"IPMI status of the node (1=online, 0=offline)",
			[]string{"system_name", "cluster", "node_name"},
			nil,
		),
		nodeCPUUsageDesc: prometheus.NewDesc(
			"vergeos_node_cpu_core_usage",
			"CPU usage per core",
			[]string{"system_name", "cluster", "node_name", "core_id"},
			nil,
		),
		nodeCoreTempDesc: prometheus.NewDesc(
			"vergeos_node_core_temp",
			"Core temperature in Celsius",
			[]string{"system_name", "cluster", "node_name"},
			nil,
		),
		nodeRAMUsedDesc: prometheus.NewDesc(
			"vergeos_node_ram_used",
			"RAM used in MB",
			[]string{"system_name", "cluster", "node_name"},
			nil,
		),
		nodeRAMPercentDesc: prometheus.NewDesc(
			"vergeos_node_ram_pct",
			"RAM used percentage",
			[]string{"system_name", "cluster", "node_name"},
			nil,
		),
		nodeRAMDesc: prometheus.NewDesc(
			"vergeos_node_ram_allocated",
			"VM RAM in MB (vm_ram field)",
			[]string{"system_name", "cluster", "node_name"},
			nil,
		),
		nodeRAMTotalDesc: prometheus.NewDesc(
			"vergeos_node_ram_total",
			"Total RAM in MB (ram field)",
			[]string{"system_name", "cluster", "node_name"},
			nil,
		),
		nodeRunningCoresDesc: prometheus.NewDesc(
			"vergeos_node_running_cores",
			"Number of cores being used by workloads",
			[]string{"system_name", "cluster", "node_name"},
			nil,
		),
		nodeRunningRAMDesc: prometheus.NewDesc(
			"vergeos_node_running_ram",
			"Amount of RAM (in MB) being used by workloads",
			[]string{"system_name", "cluster", "node_name"},
			nil,
		),
	}

	// Authenticate with the API
	if err := nc.authenticate(username, password); err != nil {
		fmt.Printf("Error authenticating with VergeOS API: %v\n", err)
	}

	return nc
}

// Describe implements prometheus.Collector
func (nc *NodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nc.nodesTotalDesc
	ch <- nc.nodeIPMIStatusDesc
	ch <- nc.nodeCPUUsageDesc
	ch <- nc.nodeCoreTempDesc
	ch <- nc.nodeRAMUsedDesc
	ch <- nc.nodeRAMPercentDesc
	ch <- nc.nodeRAMDesc
	ch <- nc.nodeRAMTotalDesc
	ch <- nc.nodeRunningCoresDesc
	ch <- nc.nodeRunningRAMDesc
}

// Collect implements prometheus.Collector
func (nc *NodeCollector) Collect(ch chan<- prometheus.Metric) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	// Get system name
	req, err := nc.makeRequest("GET", "/api/v4/settings?fields=most&filter=key%20eq%20%22cloud_name%22")
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	resp, err := nc.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var settings []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	fmt.Println(resp.Body)
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		fmt.Printf("Error decoding settings response: %v\n", err)
		return
	}

	for _, setting := range settings {
		if setting.Key == "cloud_name" {
			nc.systemName = setting.Value
			break
		}
	}

	// Get physical nodes
	req, err = nc.makeRequest("GET", "/api/v4/nodes?filter=physical%20eq%20true")
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	resp, err = nc.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var nodes []Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return
	}

	// Create a map to count nodes per cluster
	clusterNodeCounts := make(map[string]int)

	// Keep the original metric with "all" for backward compatibility
	ch <- prometheus.MustNewConstMetric(nc.nodesTotalDesc, prometheus.GaugeValue, float64(len(nodes)), nc.systemName, "all")

	// Process each node
	for _, node := range nodes {
		// Calculate IPMI status value (will be set after we get cluster info)
		ipmiStatus := 0.0
		if node.IPMIStatus == "ok" {
			ipmiStatus = 1.0
		}

		// Get detailed node stats
		req, err := nc.makeRequest("GET", fmt.Sprintf("/api/v4/nodes/%d?fields=dashboard", node.ID))
		if err != nil {
			fmt.Printf("Error creating request for node %s: %v\n", node.Name, err)
			continue
		}

		resp, err := nc.httpClient.Do(req)
		if err != nil {
			fmt.Printf("Error executing request for node %s: %v\n", node.Name, err)
			continue
		}

		// Define a struct to capture the relevant fields from the node detail response
		var nodeData struct {
			Machine struct {
				Stats struct {
					CoreUsageList []float64 `json:"core_usagelist"` // Changed from core_usage_list to core_usagelist
					CoreTemp      float64   `json:"core_temp"`
					RAMUsed       int64     `json:"ram_used"`
					RAMPct        float64   `json:"ram_pct"`
				} `json:"stats"`
			} `json:"machine"`
			VMStatsTotals struct {
				RunningCores int64 `json:"running_cores"`
				RunningRAM   int64 `json:"running_ram"`
			} `json:"vm_stats_totals"`
			ClusterDisplay string `json:"cluster_display"`
			VMRAM          int64  `json:"vm_ram"`
			RAM            int64  `json:"ram"`
		}

		// Decode the response into the nodeData struct
		if err := json.NewDecoder(resp.Body).Decode(&nodeData); err != nil {
			fmt.Printf("Error decoding response for node %s: %v\n", node.Name, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// Set metrics
		// Set per-core CPU usage using the Stats field
		for i, usage := range nodeData.Machine.Stats.CoreUsageList {
			ch <- prometheus.MustNewConstMetric(nc.nodeCPUUsageDesc, prometheus.GaugeValue, usage, nc.systemName, nodeData.ClusterDisplay, node.Name, fmt.Sprintf("%d", i))
		}
		// Set core temperature using the Stats field
		if nodeData.Machine.Stats.CoreTemp > 0 {
			ch <- prometheus.MustNewConstMetric(nc.nodeCoreTempDesc, prometheus.GaugeValue, nodeData.Machine.Stats.CoreTemp, nc.systemName, nodeData.ClusterDisplay, node.Name)
		}

		// Set IPMI status with system_name and cluster labels

		ch <- prometheus.MustNewConstMetric(nc.nodeIPMIStatusDesc, prometheus.GaugeValue, ipmiStatus, nc.systemName, nodeData.ClusterDisplay, node.Name)
		// Set RAM metrics using the Stats field
		ch <- prometheus.MustNewConstMetric(nc.nodeRAMUsedDesc, prometheus.GaugeValue, float64(nodeData.Machine.Stats.RAMUsed), nc.systemName, nodeData.ClusterDisplay, node.Name)
		ch <- prometheus.MustNewConstMetric(nc.nodeRAMPercentDesc, prometheus.GaugeValue, nodeData.Machine.Stats.RAMPct, nc.systemName, nodeData.ClusterDisplay, node.Name)

		// Set the new RAM metrics from the root level fields
		ch <- prometheus.MustNewConstMetric(nc.nodeRAMDesc, prometheus.GaugeValue, float64(nodeData.VMRAM), nc.systemName, nodeData.ClusterDisplay, node.Name)
		ch <- prometheus.MustNewConstMetric(nc.nodeRAMTotalDesc, prometheus.GaugeValue, float64(nodeData.RAM), nc.systemName, nodeData.ClusterDisplay, node.Name)

		// Set the new metrics for running cores and RAM
		ch <- prometheus.MustNewConstMetric(nc.nodeRunningCoresDesc, prometheus.GaugeValue, float64(nodeData.VMStatsTotals.RunningCores), nc.systemName, nodeData.ClusterDisplay, node.Name)
		ch <- prometheus.MustNewConstMetric(nc.nodeRunningRAMDesc, prometheus.GaugeValue, float64(nodeData.VMStatsTotals.RunningRAM), nc.systemName, nodeData.ClusterDisplay, node.Name)

		// Count nodes per cluster
		clusterNodeCounts[nodeData.ClusterDisplay]++
	}

	// Set the nodes total metric per cluster
	for clusterName, count := range clusterNodeCounts {
		ch <- prometheus.MustNewConstMetric(nc.nodesTotalDesc, prometheus.GaugeValue, float64(count), nc.systemName, clusterName)
	}
}
