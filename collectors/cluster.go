package collectors

import (
	"context"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/goVergeOS"
)

// ClusterCollector collects metrics about VergeOS clusters
type ClusterCollector struct {
	BaseCollector
	mutex sync.Mutex

	// Metric descriptors (using MustNewConstMetric pattern to avoid stale metrics)
	clusterStatus        *prometheus.Desc
	clusterRAMTotal      *prometheus.Desc
	clusterRAMUsed       *prometheus.Desc
	clusterCoresTotal    *prometheus.Desc
	clusterCoresUsed     *prometheus.Desc
	clusterMachinesTotal *prometheus.Desc
	clusterHealth        *prometheus.Desc
	clustersTotal        *prometheus.Desc
	clusterEnabled       *prometheus.Desc
	clusterRamPerUnit    *prometheus.Desc
	clusterCoresPerUnit  *prometheus.Desc
	clusterTargetRamPct  *prometheus.Desc
	clusterTotalNodes    *prometheus.Desc
	clusterOnlineNodes   *prometheus.Desc
	clusterOnlineRam     *prometheus.Desc
	clusterOnlineCores   *prometheus.Desc
	clusterPhysRamUsed   *prometheus.Desc
}

// NewClusterCollector creates a new ClusterCollector
func NewClusterCollector(client *vergeos.Client) *ClusterCollector {
	return &ClusterCollector{
		BaseCollector: *NewBaseCollector(client),
		clusterStatus: prometheus.NewDesc(
			"vergeos_cluster_status",
			"Cluster status (1=online, 0=offline)",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterRAMTotal: prometheus.NewDesc(
			"vergeos_cluster_total_ram",
			"Total RAM in MB",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterRAMUsed: prometheus.NewDesc(
			"vergeos_cluster_used_ram",
			"Used RAM in MB",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterCoresTotal: prometheus.NewDesc(
			"vergeos_cluster_cores_total",
			"Total number of CPU cores",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterCoresUsed: prometheus.NewDesc(
			"vergeos_cluster_used_cores",
			"Number of CPU cores in use",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterMachinesTotal: prometheus.NewDesc(
			"vergeos_cluster_running_machines",
			"Total number of running machines",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterHealth: prometheus.NewDesc(
			"vergeos_cluster_health",
			"Cluster health status (1=healthy, 0=unhealthy)",
			[]string{"system_name", "cluster"},
			nil,
		),
		clustersTotal: prometheus.NewDesc(
			"vergeos_clusters_total",
			"Total number of clusters",
			[]string{"system_name"},
			nil,
		),
		clusterEnabled: prometheus.NewDesc(
			"vergeos_cluster_enabled",
			"Cluster enabled status (1=enabled, 0=disabled)",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterRamPerUnit: prometheus.NewDesc(
			"vergeos_cluster_ram_per_unit",
			"RAM per unit in MB",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterCoresPerUnit: prometheus.NewDesc(
			"vergeos_cluster_cores_per_unit",
			"Cores per unit",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterTargetRamPct: prometheus.NewDesc(
			"vergeos_cluster_target_ram_pct",
			"Target RAM percentage",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterTotalNodes: prometheus.NewDesc(
			"vergeos_cluster_total_nodes",
			"Total number of nodes",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterOnlineNodes: prometheus.NewDesc(
			"vergeos_cluster_online_nodes",
			"Number of online nodes",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterOnlineRam: prometheus.NewDesc(
			"vergeos_cluster_online_ram",
			"Online RAM in MB",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterOnlineCores: prometheus.NewDesc(
			"vergeos_cluster_online_cores",
			"Number of online cores",
			[]string{"system_name", "cluster"},
			nil,
		),
		clusterPhysRamUsed: prometheus.NewDesc(
			"vergeos_cluster_phys_ram_used",
			"Physical RAM used in bytes",
			[]string{"system_name", "cluster"},
			nil,
		),
	}
}

// Describe implements prometheus.Collector
func (cc *ClusterCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cc.clusterStatus
	ch <- cc.clusterRAMTotal
	ch <- cc.clusterRAMUsed
	ch <- cc.clusterCoresTotal
	ch <- cc.clusterCoresUsed
	ch <- cc.clusterMachinesTotal
	ch <- cc.clusterHealth
	ch <- cc.clustersTotal
	ch <- cc.clusterEnabled
	ch <- cc.clusterRamPerUnit
	ch <- cc.clusterCoresPerUnit
	ch <- cc.clusterTargetRamPct
	ch <- cc.clusterTotalNodes
	ch <- cc.clusterOnlineNodes
	ch <- cc.clusterOnlineRam
	ch <- cc.clusterOnlineCores
	ch <- cc.clusterPhysRamUsed
}

// Collect implements prometheus.Collector
func (cc *ClusterCollector) Collect(ch chan<- prometheus.Metric) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	ctx := context.Background()

	// Get system name using SDK
	systemName, err := cc.GetSystemName(ctx)
	if err != nil {
		log.Printf("Error getting system name: %v", err)
		return
	}

	// Get cluster list using SDK
	clusters, err := cc.client.Clusters.List(ctx)
	if err != nil {
		log.Printf("Error fetching clusters: %v", err)
		return
	}

	// Emit total clusters metric
	ch <- prometheus.MustNewConstMetric(
		cc.clustersTotal,
		prometheus.GaugeValue,
		float64(len(clusters)),
		systemName,
	)

	// Process each cluster
	for _, cluster := range clusters {
		clusterName := cluster.Name

		// Get cluster status using SDK
		status, err := cc.client.Clusters.GetStatus(ctx, cluster.Key.Int())
		if err != nil {
			log.Printf("Error fetching cluster %d (%s) status: %v", cluster.Key, clusterName, err)
			continue
		}

		// Enabled status (1=enabled, 0=disabled)
		enabledValue := 0.0
		if cluster.Enabled {
			enabledValue = 1.0
		}
		ch <- prometheus.MustNewConstMetric(
			cc.clusterEnabled,
			prometheus.GaugeValue,
			enabledValue,
			systemName, clusterName,
		)

		// Cluster configuration metrics
		ch <- prometheus.MustNewConstMetric(
			cc.clusterRamPerUnit,
			prometheus.GaugeValue,
			float64(cluster.RAMPerUnit),
			systemName, clusterName,
		)
		ch <- prometheus.MustNewConstMetric(
			cc.clusterCoresPerUnit,
			prometheus.GaugeValue,
			float64(cluster.CoresPerUnit),
			systemName, clusterName,
		)
		ch <- prometheus.MustNewConstMetric(
			cc.clusterTargetRamPct,
			prometheus.GaugeValue,
			cluster.TargetRAMPct,
			systemName, clusterName,
		)

		// Node metrics
		ch <- prometheus.MustNewConstMetric(
			cc.clusterTotalNodes,
			prometheus.GaugeValue,
			float64(status.TotalNodes),
			systemName, clusterName,
		)
		ch <- prometheus.MustNewConstMetric(
			cc.clusterOnlineNodes,
			prometheus.GaugeValue,
			float64(status.OnlineNodes),
			systemName, clusterName,
		)

		// RAM metrics (SDK returns MB for virtual RAM)
		ch <- prometheus.MustNewConstMetric(
			cc.clusterRAMTotal,
			prometheus.GaugeValue,
			float64(status.TotalRAM),
			systemName, clusterName,
		)
		ch <- prometheus.MustNewConstMetric(
			cc.clusterRAMUsed,
			prometheus.GaugeValue,
			float64(status.UsedRAM),
			systemName, clusterName,
		)
		ch <- prometheus.MustNewConstMetric(
			cc.clusterOnlineRam,
			prometheus.GaugeValue,
			float64(status.OnlineRAM),
			systemName, clusterName,
		)

		// Physical RAM used (SDK returns bytes)
		ch <- prometheus.MustNewConstMetric(
			cc.clusterPhysRamUsed,
			prometheus.GaugeValue,
			float64(status.PhysRAMUsed),
			systemName, clusterName,
		)

		// Core metrics
		ch <- prometheus.MustNewConstMetric(
			cc.clusterCoresTotal,
			prometheus.GaugeValue,
			float64(status.TotalCores),
			systemName, clusterName,
		)
		ch <- prometheus.MustNewConstMetric(
			cc.clusterCoresUsed,
			prometheus.GaugeValue,
			float64(status.UsedCores),
			systemName, clusterName,
		)
		ch <- prometheus.MustNewConstMetric(
			cc.clusterOnlineCores,
			prometheus.GaugeValue,
			float64(status.OnlineCores),
			systemName, clusterName,
		)

		// Running machines
		ch <- prometheus.MustNewConstMetric(
			cc.clusterMachinesTotal,
			prometheus.GaugeValue,
			float64(status.RunningMachines),
			systemName, clusterName,
		)

		// Cluster status (1=online if has online nodes, 0=offline)
		statusValue := 0.0
		if status.OnlineNodes > 0 {
			statusValue = 1.0
		}
		ch <- prometheus.MustNewConstMetric(
			cc.clusterStatus,
			prometheus.GaugeValue,
			statusValue,
			systemName, clusterName,
		)

		// Cluster health (1=healthy if state is online, 0=unhealthy)
		healthValue := 0.0
		if status.State == "online" {
			healthValue = 1.0
		}
		ch <- prometheus.MustNewConstMetric(
			cc.clusterHealth,
			prometheus.GaugeValue,
			healthValue,
			systemName, clusterName,
		)
	}
}
