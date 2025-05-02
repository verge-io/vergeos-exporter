package collectors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// ClusterResponse represents the API response for clusters
type ClusterResponse []ClusterInfo

// ClusterCollector collects metrics about VergeOS clusters
type ClusterCollector struct {
	BaseCollector
	mutex sync.Mutex

	// Metrics
	clusterStatus        *prometheus.GaugeVec
	clusterRAMTotal      *prometheus.GaugeVec
	clusterRAMUsed       *prometheus.GaugeVec
	clusterCoresTotal    *prometheus.GaugeVec
	clusterCoresUsed     *prometheus.GaugeVec
	clusterMachinesTotal *prometheus.GaugeVec
	clusterHealth        *prometheus.GaugeVec
	clustersTotal        *prometheus.GaugeVec
	clusterEnabled       *prometheus.GaugeVec
	clusterRamPerUnit    *prometheus.GaugeVec
	clusterCoresPerUnit  *prometheus.GaugeVec
	clusterTargetRamPct  *prometheus.GaugeVec
	clusterTotalNodes    *prometheus.GaugeVec
	clusterOnlineNodes   *prometheus.GaugeVec
	clusterOnlineRam     *prometheus.GaugeVec
	clusterOnlineCores   *prometheus.GaugeVec
	clusterPhysRamUsed   *prometheus.GaugeVec
	systemName           string
}

// NewClusterCollector creates a new ClusterCollector
func NewClusterCollector(url string, client *http.Client, username, password string) *ClusterCollector {
	cc := &ClusterCollector{
		BaseCollector: BaseCollector{
			url:        url,
			httpClient: client,
		},
		systemName: "unknown", // Will be updated in Collect
		clusterStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_status",
				Help: "Cluster status (1=online, 0=offline)",
			},
			[]string{"system_name", "cluster"},
		),
		clusterRAMTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_total_ram",
				Help: "Total RAM in bytes",
			},
			[]string{"system_name", "cluster"},
		),
		clusterRAMUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_used_ram",
				Help: "Used RAM in bytes",
			},
			[]string{"system_name", "cluster"},
		),
		clusterCoresTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_cores_total",
				Help: "Total number of CPU cores",
			},
			[]string{"system_name", "cluster"},
		),
		clusterCoresUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_used_cores",
				Help: "Number of CPU cores in use",
			},
			[]string{"system_name", "cluster"},
		),
		clusterMachinesTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_running_machines",
				Help: "Total number of running machines",
			},
			[]string{"system_name", "cluster"},
		),
		clusterHealth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_health",
				Help: "Cluster health status (1=healthy, 0=unhealthy)",
			},
			[]string{"system_name", "cluster"},
		),
		clustersTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_clusters_total",
				Help: "Total number of clusters",
			},
			[]string{"system_name"},
		),
		clusterEnabled: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_enabled",
				Help: "Cluster enabled status (1=enabled, 0=disabled)",
			},
			[]string{"system_name", "cluster"},
		),
		clusterRamPerUnit: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_ram_per_unit",
				Help: "RAM per unit in bytes",
			},
			[]string{"system_name", "cluster"},
		),
		clusterCoresPerUnit: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_cores_per_unit",
				Help: "Cores per unit",
			},
			[]string{"system_name", "cluster"},
		),
		clusterTargetRamPct: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_target_ram_pct",
				Help: "Target RAM percentage",
			},
			[]string{"system_name", "cluster"},
		),
		clusterTotalNodes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_total_nodes",
				Help: "Total number of nodes",
			},
			[]string{"system_name", "cluster"},
		),
		clusterOnlineNodes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_online_nodes",
				Help: "Number of online nodes",
			},
			[]string{"system_name", "cluster"},
		),
		clusterOnlineRam: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_online_ram",
				Help: "Online RAM in bytes",
			},
			[]string{"system_name", "cluster"},
		),
		clusterOnlineCores: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_online_cores",
				Help: "Number of online cores",
			},
			[]string{"system_name", "cluster"},
		),
		clusterPhysRamUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_phys_ram_used",
				Help: "Physical RAM used in bytes",
			},
			[]string{"system_name", "cluster"},
		),
	}

	// Authenticate with the API
	if err := cc.authenticate(username, password); err != nil {
		fmt.Printf("Error authenticating with VergeOS API: %v\n", err)
	}

	return cc
}

// Describe implements prometheus.Collector
func (cc *ClusterCollector) Describe(ch chan<- *prometheus.Desc) {
	cc.clusterStatus.Describe(ch)
	cc.clusterRAMTotal.Describe(ch)
	cc.clusterRAMUsed.Describe(ch)
	cc.clusterCoresTotal.Describe(ch)
	cc.clusterCoresUsed.Describe(ch)
	cc.clusterMachinesTotal.Describe(ch)
	cc.clusterHealth.Describe(ch)
	cc.clustersTotal.Describe(ch)
	cc.clusterEnabled.Describe(ch)
	cc.clusterRamPerUnit.Describe(ch)
	cc.clusterCoresPerUnit.Describe(ch)
	cc.clusterTargetRamPct.Describe(ch)
	cc.clusterTotalNodes.Describe(ch)
	cc.clusterOnlineNodes.Describe(ch)
	cc.clusterOnlineRam.Describe(ch)
	cc.clusterOnlineCores.Describe(ch)
	cc.clusterPhysRamUsed.Describe(ch)
}

// Collect implements prometheus.Collector
func (cc *ClusterCollector) Collect(ch chan<- prometheus.Metric) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	// Get system name
	req, err := cc.makeRequest("GET", "/api/v4/settings?fields=most&filter=key%20eq%20%22cloud_name%22")
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	resp, err := cc.httpClient.Do(req)
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
			cc.systemName = setting.Value
			break
		}
	}

	// Get cluster information
	req, err = cc.makeRequest("GET", "/api/v4/clusters?fields=all")
	if err != nil {
		fmt.Printf("Error creating request for clusters: %v\n", err)
		return
	}

	resp, err = cc.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing request for clusters: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var response ClusterListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("Error decoding clusters response: %v\n", err)
		return
	}

	cc.clustersTotal.WithLabelValues(cc.systemName).Set(float64(len(response)))

	// Get detailed information for each cluster
	for _, cluster := range response {
		// First get the physical RAM usage from the cluster_stats_history_short endpoint
		// This is the only endpoint that provides the phys_ram_used field
		statsURL := fmt.Sprintf("/api/v4/cluster_stats_history_short/%d?fields=all", cluster.Key)
		req, err = cc.makeRequest("GET", statsURL)
		if err != nil {
			fmt.Printf("Error creating request for cluster stats: %v\n", err)
		} else {
			resp, err = cc.httpClient.Do(req)
			if err != nil {
				fmt.Printf("Error executing request for cluster stats: %v\n", err)
			} else {
				// Read the response body
				statsBytes, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					fmt.Printf("Error reading cluster stats response body: %v\n", err)
				} else {
					// Parse the stats response
					var statsResp struct {
						Key         int   `json:"$key"`
						Cluster     int   `json:"cluster"`
						PhysRamUsed int64 `json:"phys_ram_used"`
						Timestamp   int64 `json:"timestamp"`
					}
					if err := json.NewDecoder(bytes.NewReader(statsBytes)).Decode(&statsResp); err != nil {
						fmt.Printf("Error decoding cluster stats response: %v\n", err)
					} else {
						cc.clusterPhysRamUsed.WithLabelValues(cc.systemName, cluster.Name).Set(float64(statsResp.PhysRamUsed))
					}
				}
			}
		}

		// Now get the other cluster details from the regular endpoint
		detailURL := fmt.Sprintf("/api/v4/clusters/%d?fields=dashboard", cluster.Key)
		req, err = cc.makeRequest("GET", detailURL)
		if err != nil {
			fmt.Printf("Error creating request for cluster details: %v\n", err)
			continue
		}

		resp, err = cc.httpClient.Do(req)
		if err != nil {
			fmt.Printf("Error executing request for cluster details: %v\n", err)
			continue
		}

		var detail ClusterDetailResponse
		if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
			fmt.Printf("Error decoding cluster details response: %v\n", err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		enabledValue := 0.0
		if detail.Enabled {
			enabledValue = 1.0
		}

		cc.clusterEnabled.WithLabelValues(cc.systemName, detail.Name).Set(enabledValue)
		cc.clusterRamPerUnit.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.RamPerUnit))
		cc.clusterCoresPerUnit.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.CoresPerUnit))
		cc.clusterTargetRamPct.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.TargetRamPct))
		cc.clusterTotalNodes.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.Status.TotalNodes))
		cc.clusterOnlineNodes.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.Status.OnlineNodes))
		cc.clusterOnlineRam.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.Status.OnlineRam))
		cc.clusterOnlineCores.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.Status.OnlineCores))

		// Note: We're no longer trying to get PhysRamUsed from the detail API since it's not present in that endpoint
		cc.clusterMachinesTotal.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.Status.RunningMachines))
		cc.clusterRAMTotal.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.Status.TotalRam))
		cc.clusterRAMUsed.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.Status.UsedRam))
		cc.clusterCoresUsed.WithLabelValues(cc.systemName, detail.Name).Set(float64(detail.Status.UsedCores))

		// Set cluster status (1=online, 0=offline)
		statusValue := 0.0
		if detail.Status.OnlineNodes > 0 {
			statusValue = 1.0
		}
		cc.clusterStatus.WithLabelValues(cc.systemName, detail.Name).Set(statusValue)
	}

	cc.clusterStatus.Collect(ch)
	cc.clusterRAMTotal.Collect(ch)
	cc.clusterRAMUsed.Collect(ch)
	cc.clusterCoresTotal.Collect(ch)
	cc.clusterCoresUsed.Collect(ch)
	cc.clusterMachinesTotal.Collect(ch)
	cc.clusterHealth.Collect(ch)
	cc.clustersTotal.Collect(ch)
	cc.clusterEnabled.Collect(ch)
	cc.clusterRamPerUnit.Collect(ch)
	cc.clusterCoresPerUnit.Collect(ch)
	cc.clusterTargetRamPct.Collect(ch)
	cc.clusterTotalNodes.Collect(ch)
	cc.clusterOnlineNodes.Collect(ch)
	cc.clusterOnlineRam.Collect(ch)
	cc.clusterOnlineCores.Collect(ch)
	cc.clusterPhysRamUsed.Collect(ch)
}
