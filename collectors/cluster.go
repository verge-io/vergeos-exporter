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
	mutex      sync.Mutex
	systemName string

	// Metrics
	clusterStatusDesc        *prometheus.Desc
	clusterRAMTotalDesc      *prometheus.Desc
	clusterRAMUsedDesc       *prometheus.Desc
	clusterCoresTotalDesc    *prometheus.Desc
	clusterCoresUsedDesc     *prometheus.Desc
	clusterMachinesTotalDesc *prometheus.Desc
	clusterHealthDesc        *prometheus.Desc
	clustersTotalDesc        *prometheus.Desc
	clusterEnabledDesc       *prometheus.Desc
	clusterRamPerUnitDesc    *prometheus.Desc
	clusterCoresPerUnitDesc  *prometheus.Desc
	clusterTargetRamPctDesc  *prometheus.Desc
	clusterTotalNodesDesc    *prometheus.Desc
	clusterOnlineNodesDesc   *prometheus.Desc
	clusterOnlineRamDesc     *prometheus.Desc
	clusterOnlineCoresDesc   *prometheus.Desc
	clusterPhysRamUsedDesc   *prometheus.Desc
}

// NewClusterCollector creates a new ClusterCollector
func NewClusterCollector(url string, client *http.Client, username, password string) *ClusterCollector {
	cc := &ClusterCollector{
		BaseCollector: BaseCollector{
			url:        url,
			httpClient: client,
		},
		systemName: "unknown", // Will be updated in Collect
		clusterStatusDesc: prometheus.NewDesc(
			"vergeos_cluster_status",
			"Cluster status (1=online, 0=offline)",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterRAMTotalDesc: prometheus.NewDesc(
			"vergeos_cluster_total_ram",
			"Total RAM in bytes",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterRAMUsedDesc: prometheus.NewDesc(
			"vergeos_cluster_used_ram",
			"Used RAM in bytes",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterCoresTotalDesc: prometheus.NewDesc(
			"vergeos_cluster_cores_total",
			"Total number of CPU cores",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterCoresUsedDesc: prometheus.NewDesc(
			"vergeos_cluster_used_cores",
			"Number of CPU cores in use",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterMachinesTotalDesc: prometheus.NewDesc(
			"vergeos_cluster_running_machines",
			"Total number of running machines",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterHealthDesc: prometheus.NewDesc(
			"vergeos_cluster_health",
			"Cluster health status (1=healthy, 0=unhealthy)",
			[]string{"system_name", "cluster"},
			nil,
		),
		clustersTotalDesc: prometheus.NewDesc(
			"vergeos_clusters_total",
			"Total number of clusters",
			[]string{"system_name"},
			nil,
		),
		clusterEnabledDesc: prometheus.NewDesc(
			"vergeos_cluster_enabled",
			"Cluster enabled status (1=enabled, 0=disabled)",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterRamPerUnitDesc: prometheus.NewDesc(
			"vergeos_cluster_ram_per_unit",
			"RAM per unit in bytes",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterCoresPerUnitDesc: prometheus.NewDesc(
			"vergeos_cluster_cores_per_unit",
			"Cores per unit",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterTargetRamPctDesc: prometheus.NewDesc(
			"vergeos_cluster_target_ram_pct",
			"Target RAM percentage",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterTotalNodesDesc: prometheus.NewDesc(
			"vergeos_cluster_total_nodes",
			"Total number of nodes",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterOnlineNodesDesc: prometheus.NewDesc(
			"vergeos_cluster_online_nodes",
			"Number of online nodes",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterOnlineRamDesc: prometheus.NewDesc(
			"vergeos_cluster_online_ram",
			"Online RAM in bytes",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterOnlineCoresDesc: prometheus.NewDesc(
			"vergeos_cluster_online_cores",
			"Number of online cores",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterPhysRamUsedDesc: prometheus.NewDesc(
			"vergeos_cluster_phys_ram_used",
			"Physical RAM used in bytes",
			[]string{"system_name", "cluster"},
			nil,
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
	ch <- cc.clusterStatusDesc
	ch <- cc.clusterRAMTotalDesc
	ch <- cc.clusterRAMUsedDesc
	ch <- cc.clusterCoresTotalDesc
	ch <- cc.clusterCoresUsedDesc
	ch <- cc.clusterMachinesTotalDesc
	ch <- cc.clusterHealthDesc
	ch <- cc.clustersTotalDesc
	ch <- cc.clusterEnabledDesc
	ch <- cc.clusterRamPerUnitDesc
	ch <- cc.clusterCoresPerUnitDesc
	ch <- cc.clusterTargetRamPctDesc
	ch <- cc.clusterTotalNodesDesc
	ch <- cc.clusterOnlineNodesDesc
	ch <- cc.clusterOnlineRamDesc
	ch <- cc.clusterOnlineCoresDesc
	ch <- cc.clusterPhysRamUsedDesc
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

	ch <- prometheus.MustNewConstMetric(cc.clustersTotalDesc, prometheus.GaugeValue, float64(len(response)), cc.systemName)

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
						ch <- prometheus.MustNewConstMetric(cc.clusterPhysRamUsedDesc, prometheus.GaugeValue, float64(statsResp.PhysRamUsed), cc.systemName, cluster.Name)
						// cc.clusterPhysRamUsed.WithLabelValues(cc.systemName, cluster.Name).Set(float64(statsResp.PhysRamUsed))
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

		ch <- prometheus.MustNewConstMetric(cc.clusterEnabledDesc, prometheus.GaugeValue, float64(enabledValue), cc.systemName, detail.Name)
		ch <- prometheus.MustNewConstMetric(cc.clusterRamPerUnitDesc, prometheus.GaugeValue, float64(detail.RamPerUnit), cc.systemName, detail.Name)
		ch <- prometheus.MustNewConstMetric(cc.clusterCoresPerUnitDesc, prometheus.GaugeValue, float64(detail.CoresPerUnit), cc.systemName, detail.Name)
		ch <- prometheus.MustNewConstMetric(cc.clusterTargetRamPctDesc, prometheus.GaugeValue, float64(detail.TargetRamPct), cc.systemName, detail.Name)
		ch <- prometheus.MustNewConstMetric(cc.clusterTotalNodesDesc, prometheus.GaugeValue, float64(detail.Status.TotalNodes), cc.systemName, detail.Name)
		ch <- prometheus.MustNewConstMetric(cc.clusterOnlineNodesDesc, prometheus.GaugeValue, float64(detail.Status.OnlineNodes), cc.systemName, detail.Name)
		ch <- prometheus.MustNewConstMetric(cc.clusterOnlineRamDesc, prometheus.GaugeValue, float64(detail.Status.OnlineRam), cc.systemName, detail.Name)
		ch <- prometheus.MustNewConstMetric(cc.clusterOnlineCoresDesc, prometheus.GaugeValue, float64(detail.Status.OnlineCores), cc.systemName, detail.Name)

		// Note: We're no longer trying to get PhysRamUsed from the detail API since it's not present in that endpoint
		ch <- prometheus.MustNewConstMetric(cc.clusterMachinesTotalDesc, prometheus.GaugeValue, float64(detail.Status.RunningMachines), cc.systemName, detail.Name)
		ch <- prometheus.MustNewConstMetric(cc.clusterRAMTotalDesc, prometheus.GaugeValue, float64(detail.Status.TotalRam), cc.systemName, detail.Name)

		ch <- prometheus.MustNewConstMetric(cc.clusterRAMUsedDesc, prometheus.GaugeValue, float64(detail.Status.UsedRam), cc.systemName, detail.Name)
		ch <- prometheus.MustNewConstMetric(cc.clusterCoresUsedDesc, prometheus.GaugeValue, float64(detail.Status.UsedCores), cc.systemName, detail.Name)

		// Set cluster status (1=online, 0=offline)
		statusValue := 0.0
		if detail.Status.OnlineNodes > 0 {
			statusValue = 1.0
		}
		ch <- prometheus.MustNewConstMetric(cc.clusterStatusDesc, prometheus.GaugeValue, float64(statusValue), cc.systemName, detail.Name)
	}

	// cc.clusterStatus.Collect(ch)
	// cc.clusterRAMTotal.Collect(ch)
	// cc.clusterRAMUsed.Collect(ch)
	// cc.clusterCoresTotal.Collect(ch)
	// cc.clusterCoresUsed.Collect(ch)
	// cc.clusterMachinesTotal.Collect(ch)
	// cc.clusterHealth.Collect(ch)
	// cc.clustersTotal.Collect(ch)
	// cc.clusterEnabled.Collect(ch)
	// cc.clusterRamPerUnit.Collect(ch)
	// cc.clusterCoresPerUnit.Collect(ch)
	// cc.clusterTargetRamPct.Collect(ch)
	// cc.clusterTotalNodes.Collect(ch)
	// cc.clusterOnlineNodes.Collect(ch)
	// cc.clusterOnlineRam.Collect(ch)
	// cc.clusterOnlineCores.Collect(ch)
	// cc.clusterPhysRamUsed.Collect(ch)
}
