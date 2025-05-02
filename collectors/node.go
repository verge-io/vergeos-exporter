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
	nodesTotal       *prometheus.GaugeVec
	nodeIPMIStatus   *prometheus.GaugeVec
	nodeCPUUsage     *prometheus.GaugeVec
	nodeCoreTemp     *prometheus.GaugeVec
	nodeRAMUsed      *prometheus.GaugeVec
	nodeRAMPercent   *prometheus.GaugeVec
	nodeRAM          *prometheus.GaugeVec // VM RAM allocated (vm_ram field)
	nodeRAMTotal     *prometheus.GaugeVec // Total RAM (ram field)
	nodeRunningCores *prometheus.GaugeVec // Running cores used by workloads
	nodeRunningRAM   *prometheus.GaugeVec // Running RAM used by workloads
}

// NewNodeCollector creates a new NodeCollector
func NewNodeCollector(url string, client *http.Client, username, password string) *NodeCollector {
	nc := &NodeCollector{
		BaseCollector: BaseCollector{
			url:        url,
			httpClient: client,
		},
		systemName: "unknown", // Will be updated in Collect
		nodesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_nodes_total",
			Help: "Total number of physical nodes",
		}, []string{"system_name", "cluster"}),
		nodeIPMIStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_ipmi_status",
			Help: "IPMI status of the node (1=online, 0=offline)",
		}, []string{"system_name", "cluster", "node_name"}),
		nodeCPUUsage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_cpu_core_usage",
			Help: "CPU usage per core",
		}, []string{"system_name", "cluster", "node_name", "core_id"}),
		nodeCoreTemp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_core_temp",
			Help: "Core temperature in Celsius",
		}, []string{"system_name", "cluster", "node_name"}),
		nodeRAMUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_ram_used",
			Help: "RAM used in MB",
		}, []string{"system_name", "cluster", "node_name"}),
		nodeRAMPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_ram_pct",
			Help: "RAM used percentage",
		}, []string{"system_name", "cluster", "node_name"}),
		nodeRAM: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_ram_allocated",
			Help: "VM RAM in MB (vm_ram field)",
		}, []string{"system_name", "cluster", "node_name"}),
		nodeRAMTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_ram_total",
			Help: "Total RAM in MB (ram field)",
		}, []string{"system_name", "cluster", "node_name"}),
		nodeRunningCores: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_running_cores",
			Help: "Number of cores being used by workloads",
		}, []string{"system_name", "cluster", "node_name"}),
		nodeRunningRAM: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_running_ram",
			Help: "Amount of RAM (in MB) being used by workloads",
		}, []string{"system_name", "cluster", "node_name"}),
	}

	// Authenticate with the API
	if err := nc.authenticate(username, password); err != nil {
		fmt.Printf("Error authenticating with VergeOS API: %v\n", err)
	}

	return nc
}

// Describe implements prometheus.Collector
func (nc *NodeCollector) Describe(ch chan<- *prometheus.Desc) {
	nc.nodesTotal.Describe(ch)
	nc.nodeIPMIStatus.Describe(ch)
	nc.nodeCPUUsage.Describe(ch)
	nc.nodeCoreTemp.Describe(ch)
	nc.nodeRAMUsed.Describe(ch)
	nc.nodeRAMPercent.Describe(ch)
	nc.nodeRAM.Describe(ch)
	nc.nodeRAMTotal.Describe(ch)
	nc.nodeRunningCores.Describe(ch)
	nc.nodeRunningRAM.Describe(ch)
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
	nc.nodesTotal.WithLabelValues(nc.systemName, "all").Set(float64(len(nodes)))

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
			nc.nodeCPUUsage.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name, fmt.Sprintf("%d", i)).Set(usage)
		}
		// Set core temperature using the Stats field
		if nodeData.Machine.Stats.CoreTemp > 0 {
			nc.nodeCoreTemp.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name).Set(nodeData.Machine.Stats.CoreTemp)
		}

		// Set IPMI status with system_name and cluster labels
		nc.nodeIPMIStatus.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name).Set(ipmiStatus)
		// Set RAM metrics using the Stats field
		nc.nodeRAMUsed.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name).Set(float64(nodeData.Machine.Stats.RAMUsed))
		nc.nodeRAMPercent.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name).Set(nodeData.Machine.Stats.RAMPct)

		// Set the new RAM metrics from the root level fields
		nc.nodeRAM.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name).Set(float64(nodeData.VMRAM))
		nc.nodeRAMTotal.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name).Set(float64(nodeData.RAM))

		// Set the new metrics for running cores and RAM
		nc.nodeRunningCores.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name).Set(float64(nodeData.VMStatsTotals.RunningCores))
		nc.nodeRunningRAM.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name).Set(float64(nodeData.VMStatsTotals.RunningRAM))

		// Count nodes per cluster
		clusterNodeCounts[nodeData.ClusterDisplay]++
	}

	// Collect all metrics
	nc.nodesTotal.Collect(ch)
	nc.nodeIPMIStatus.Collect(ch)
	nc.nodeCPUUsage.Collect(ch)
	nc.nodeCoreTemp.Collect(ch)
	nc.nodeRAMUsed.Collect(ch)
	nc.nodeRAMPercent.Collect(ch)
	nc.nodeRAM.Collect(ch)
	nc.nodeRAMTotal.Collect(ch)
	nc.nodeRunningCores.Collect(ch)
	nc.nodeRunningRAM.Collect(ch)

	// Set the nodes total metric per cluster
	for clusterName, count := range clusterNodeCounts {
		nc.nodesTotal.WithLabelValues(nc.systemName, clusterName).Set(float64(count))
	}
}
