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
	Name          string `json:"name"`
	Description   string `json:"description"`
	ID            int    `json:"id"`
	Machine       int    `json:"machine"`
	Physical      bool   `json:"physical"`
	IPMIStatus    string `json:"ipmi_status"`
	IPMIStatusInfo string `json:"ipmi_status_info"`
	Cluster       int    `json:"cluster"`
}

// NodeStats represents node statistics
type NodeStats struct {
	TotalCPU     float64   `json:"total_cpu"`
	RAM          int64     `json:"ram"`
	CoreUsageList []float64 `json:"core_usagelist"`
	CoreTemp     float64   `json:"core_temp"`
	CoreTempTop  float64   `json:"core_temp_top"`
}

// NodeCollector collects metrics about VergeOS nodes
type NodeCollector struct {
	BaseCollector
	mutex sync.Mutex

	// System info
	systemName string

	// Metrics
	nodesTotal      prometheus.Gauge
	nodeIPMIStatus  *prometheus.GaugeVec
	nodeCPUUsage    *prometheus.GaugeVec
	nodeCoreTemp    *prometheus.GaugeVec
	nodeRAMUsed     *prometheus.GaugeVec
	nodeRAMPercent  *prometheus.GaugeVec
}

// NewNodeCollector creates a new NodeCollector
func NewNodeCollector(url string, client *http.Client, username, password string) *NodeCollector {
	nc := &NodeCollector{
		BaseCollector: BaseCollector{
			url:        url,
			httpClient: client,
		},
		systemName: "unknown", // Will be updated in Collect
		nodesTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "vergeos_physical_nodes_total",
			Help: "Total number of physical nodes",
		}),
		nodeIPMIStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_ipmi_status",
			Help: "IPMI status of the node (1=online, 0=offline)",
		}, []string{"node_name"}),
		nodeCPUUsage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_cpu_core_usage",
			Help: "CPU usage per core",
		}, []string{"system_name", "cluster", "node_name", "core_id"}),
		nodeCoreTemp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_core_temp",
			Help: "Core temperature in Celsius",
		}, []string{"system_name", "cluster", "node_name"}),
		nodeRAMUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_ram_used_mb",
			Help: "RAM used in MB",
		}, []string{"node_name"}),
		nodeRAMPercent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_node_ram_used_percent",
			Help: "RAM used percentage",
		}, []string{"node_name"}),
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

	nc.nodesTotal.Set(float64(len(nodes)))

	// Process each node
	for _, node := range nodes {
		// Set IPMI status
		ipmiStatus := 0.0
		if node.IPMIStatus == "ok" {
			ipmiStatus = 1.0
		}
		nc.nodeIPMIStatus.WithLabelValues(node.Name).Set(ipmiStatus)

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

		var nodeStats struct {
			Machine struct {
				Stats NodeStats `json:"stats"`
			} `json:"machine"`
			ClusterDisplay string `json:"cluster_display"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&nodeStats); err != nil {
			fmt.Printf("Error decoding response for node %s: %v\n", node.Name, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// Set metrics
		// Set per-core CPU usage
		for i, usage := range nodeStats.Machine.Stats.CoreUsageList {
			nc.nodeCPUUsage.WithLabelValues(nc.systemName, nodeStats.ClusterDisplay, node.Name, fmt.Sprintf("%d", i)).Set(usage)
		}
		// Set core temperature
		if nodeStats.Machine.Stats.CoreTemp > 0 {
			nc.nodeCoreTemp.WithLabelValues(nc.systemName, nodeStats.ClusterDisplay, node.Name).Set(nodeStats.Machine.Stats.CoreTemp)
		}
		nc.nodeRAMUsed.WithLabelValues(node.Name).Set(float64(nodeStats.Machine.Stats.RAM))
		nc.nodeRAMPercent.WithLabelValues(node.Name).Set(nodeStats.Machine.Stats.TotalCPU)
	}

	// Collect all metrics
	nc.nodesTotal.Collect(ch)
	nc.nodeIPMIStatus.Collect(ch)
	nc.nodeCPUUsage.Collect(ch)
	nc.nodeCoreTemp.Collect(ch)
	nc.nodeRAMUsed.Collect(ch)
	nc.nodeRAMPercent.Collect(ch)
}
