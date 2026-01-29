package collectors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// VMCollector collects metrics about VergeOS virtual machines
type VMCollector struct {
	BaseCollector
	mutex sync.Mutex

	// Metrics
	vmTotalCPU  *prometheus.GaugeVec
	vmUserCPU   *prometheus.GaugeVec
	vmSystemCPU *prometheus.GaugeVec
	vmIOWaitCPU *prometheus.GaugeVec
}

// vmListResponse represents a VM from the list endpoint
type vmListResponse struct {
	Key        int    `json:"$key"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsSnapshot bool   `json:"is_snapshot"`
}

// vmDetailResponse represents the detailed VM response with stats
type vmDetailResponse struct {
	Key     int    `json:"$key"`
	Name    string `json:"name"`
	Cluster string `json:"cluster"`
	Status  struct {
		Running bool   `json:"running"`
		Node    int    `json:"node"`
		Status  string `json:"status"`
	} `json:"status"`
	Stats struct {
		TotalCPU  float64 `json:"total_cpu"`
		UserCPU   float64 `json:"user_cpu"`
		SystemCPU float64 `json:"system_cpu"`
		IOWaitCPU float64 `json:"iowait_cpu"`
	} `json:"stats"`
}

// nodeNameResponse represents the node name lookup response
type nodeNameResponse struct {
	Name string `json:"name"`
}

// NewVMCollector creates a new VMCollector
func NewVMCollector(url string, client *http.Client, username, password string) *VMCollector {
	vc := &VMCollector{
		BaseCollector: BaseCollector{
			url:        url,
			httpClient: client,
		},
		vmTotalCPU: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vm_cpu_total",
			Help: "Total CPU usage percentage for the VM",
		}, []string{"system_name", "cluster", "node", "vm_name", "vm_id"}),
		vmUserCPU: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vm_cpu_user",
			Help: "User CPU usage percentage for the VM",
		}, []string{"system_name", "cluster", "node", "vm_name", "vm_id"}),
		vmSystemCPU: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vm_cpu_system",
			Help: "System CPU usage percentage for the VM",
		}, []string{"system_name", "cluster", "node", "vm_name", "vm_id"}),
		vmIOWaitCPU: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vm_cpu_iowait",
			Help: "IO Wait CPU usage percentage for the VM",
		}, []string{"system_name", "cluster", "node", "vm_name", "vm_id"}),
	}

	// Authenticate with the API
	if err := vc.authenticate(username, password); err != nil {
		fmt.Printf("Error authenticating with VergeOS API: %v\n", err)
	}

	return vc
}

// Describe implements prometheus.Collector
func (vc *VMCollector) Describe(ch chan<- *prometheus.Desc) {
	vc.vmTotalCPU.Describe(ch)
	vc.vmUserCPU.Describe(ch)
	vc.vmSystemCPU.Describe(ch)
	vc.vmIOWaitCPU.Describe(ch)
}

// Collect implements prometheus.Collector
func (vc *VMCollector) Collect(ch chan<- prometheus.Metric) {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()

	// Reset metrics to clear stale VM entries
	vc.vmTotalCPU.Reset()
	vc.vmUserCPU.Reset()
	vc.vmSystemCPU.Reset()
	vc.vmIOWaitCPU.Reset()

	// Get system name
	systemName, err := vc.getSystemName()
	if err != nil {
		fmt.Printf("Error getting system name: %v\n", err)
		return
	}

	// Get list of VMs (excluding snapshots)
	req, err := vc.makeRequest("GET", "/api/v4/machines?filter=type%20eq%20'vm'%20and%20is_snapshot%20eq%20false")
	if err != nil {
		fmt.Printf("Error creating VM list request: %v\n", err)
		return
	}

	resp, err := vc.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing VM list request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var vms []vmListResponse
	if err := json.NewDecoder(resp.Body).Decode(&vms); err != nil {
		fmt.Printf("Error decoding VM list response: %v\n", err)
		return
	}

	// Cache for node ID to name lookups
	nodeNameCache := make(map[int]string)

	// Process each VM
	for _, vm := range vms {
		// Get detailed VM stats
		req, err := vc.makeRequest("GET", fmt.Sprintf("/api/v4/machines/%d?fields=dashboard", vm.Key))
		if err != nil {
			fmt.Printf("Error creating request for VM %s: %v\n", vm.Name, err)
			continue
		}

		resp, err := vc.httpClient.Do(req)
		if err != nil {
			fmt.Printf("Error executing request for VM %s: %v\n", vm.Name, err)
			continue
		}

		var vmDetail vmDetailResponse
		if err := json.NewDecoder(resp.Body).Decode(&vmDetail); err != nil {
			fmt.Printf("Error decoding response for VM %s: %v\n", vm.Name, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// Get node name from cache or API
		nodeName := ""
		if vmDetail.Status.Node > 0 {
			if cachedName, ok := nodeNameCache[vmDetail.Status.Node]; ok {
				nodeName = cachedName
			} else {
				nodeName = vc.getNodeName(vmDetail.Status.Node)
				nodeNameCache[vmDetail.Status.Node] = nodeName
			}
		}

		// Get cluster name (use "unknown" if not set)
		clusterName := vmDetail.Cluster
		if clusterName == "" {
			clusterName = "unknown"
		}

		// Set metrics
		vmID := fmt.Sprintf("%d", vm.Key)
		vc.vmTotalCPU.WithLabelValues(systemName, clusterName, nodeName, vm.Name, vmID).Set(vmDetail.Stats.TotalCPU)
		vc.vmUserCPU.WithLabelValues(systemName, clusterName, nodeName, vm.Name, vmID).Set(vmDetail.Stats.UserCPU)
		vc.vmSystemCPU.WithLabelValues(systemName, clusterName, nodeName, vm.Name, vmID).Set(vmDetail.Stats.SystemCPU)
		vc.vmIOWaitCPU.WithLabelValues(systemName, clusterName, nodeName, vm.Name, vmID).Set(vmDetail.Stats.IOWaitCPU)
	}

	// Collect all metrics
	vc.vmTotalCPU.Collect(ch)
	vc.vmUserCPU.Collect(ch)
	vc.vmSystemCPU.Collect(ch)
	vc.vmIOWaitCPU.Collect(ch)
}

// getNodeName retrieves the node name for a given node ID
func (vc *VMCollector) getNodeName(nodeID int) string {
	req, err := vc.makeRequest("GET", fmt.Sprintf("/api/v4/nodes/%d", nodeID))
	if err != nil {
		return ""
	}

	resp, err := vc.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var node nodeNameResponse
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return ""
	}

	return node.Name
}
