package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listenAddress = flag.String("web.listen-address", ":9888", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	vergeURL      = flag.String("verge.url", "http://localhost", "Base URL of the VergeOS API")
	vergeUsername = flag.String("verge.username", "", "Username for VergeOS API authentication")
	vergePassword = flag.String("verge.password", "", "Password for VergeOS API authentication")
	scrapeTimeout = flag.Duration("scrape.timeout", 30*time.Second, "Timeout for scraping VergeOS API")
)

// API response types
type TokenResponse struct {
	Location string `json:"location"`
	DBPath   string `json:"dbpath"`
	Row      int    `json:"$row"`
	Key      string `json:"$key"`
}

type Node struct {
	Name        string `json:"name"`
	ID          int    `json:"id"`
	Physical    bool   `json:"physical"`
	IPMIStatus  string `json:"ipmi_status"`
	Description string `json:"description"`
}

type NodeStats struct {
	Machine struct {
		Stats struct {
			TotalCPU      float64   `json:"total_cpu"`
			UserCPU       float64   `json:"user_cpu"`
			SystemCPU     float64   `json:"system_cpu"`
			IOWaitCPU     float64   `json:"iowait_cpu"`
			VMUsageCPU    float64   `json:"vmusage_cpu"`
			IRQCPU        float64   `json:"irq_cpu"`
			RAMUsed       int64     `json:"ram_used"`
			RAMPct        float64   `json:"ram_pct"`
			VRAMUsed      int64     `json:"vram_used"`
			CoreUsageList []float64 `json:"core_usagelist"`
			CoreTemp      float64   `json:"core_temp"`
			CoreTempTop   float64   `json:"core_temp_top"`
			CorePeak      float64   `json:"core_peak"`
			Modified      int64     `json:"modified"`
		} `json:"stats"`
		Drives []struct {
			Name  string `json:"name"`
			Stats struct {
				ReadOps    float64 `json:"rops"`
				WriteOps   float64 `json:"wops"`
				ReadBytes  float64 `json:"read_bytes"`
				WriteBytes float64 `json:"write_bytes"`
				Util       float64 `json:"util"`
			} `json:"stats"`
			PhysicalStatus struct {
				VSANTier        int     `json:"vsan_tier"`
				VSANReadErrors  int64   `json:"vsan_read_errors"`
				VSANWriteErrors int64   `json:"vsan_write_errors"`
				VSANAvgLatency  float64 `json:"vsan_avg_latency"`
				VSANMaxLatency  float64 `json:"vsan_max_latency"`
				VSANRepairing   int64   `json:"vsan_repairing"`
				VSANThrottle    float64 `json:"vsan_throttle"`
				Status          string  `json:"status"`
				WearLevel       int64   `json:"wear_level"`
				Hours           int64   `json:"hours"`
				ReallocSectors  int64   `json:"realloc_sectors"`
				Temp            int64   `json:"temp"`
			} `json:"physical_status"`
			VSANTier int `json:"vsan_tier"`
		} `json:"drives"`
		Nics []struct {
			Name  string `json:"name"`
			Stats struct {
				TxPackets float64 `json:"tx_packets"`
				RxPackets float64 `json:"rx_packets"`
				TxBytes   float64 `json:"tx_bytes"`
				RxBytes   float64 `json:"rx_bytes"`
			} `json:"stats"`
		} `json:"nics"`
	} `json:"machine"`
}

type VSANTier struct {
	Key         int    `json:"$key"`
	Tier        int    `json:"tier"`
	Description string `json:"description"`
	Status      struct {
		Key                int     `json:"$key"`
		Tier               int     `json:"tier"`
		Status             string  `json:"status"`
		State              string  `json:"state"`
		Capacity           int64   `json:"capacity"`
		Used               int64   `json:"used"`
		UsedPct            int     `json:"used_pct"`
		Redundant          bool    `json:"redundant"`
		Encrypted          bool    `json:"encrypted"`
		Working            bool    `json:"working"`
		LastWalkTimeMs     int64   `json:"last_walk_time_ms"`
		LastFullwalkTimeMs int64   `json:"last_fullwalk_time_ms"`
		Transaction        int64   `json:"transaction"`
		Repairs            int64   `json:"repairs"`
		BadDrives          int64   `json:"bad_drives"`
		Fullwalk           bool    `json:"fullwalk"`
		Progress           float64 `json:"progress"`
		CurSpaceThrottleMs int64   `json:"cur_space_throttle_ms"`
	} `json:"status"`
	Stats struct {
		ReadOps    float64 `json:"rops"`
		WriteOps   float64 `json:"wops"`
		ReadBytes  float64 `json:"rbps"`
		WriteBytes float64 `json:"wbps"`
	} `json:"stats"`
	DrivesCount int `json:"drives_count"`
	NodesCount  int `json:"nodes_count"`
	VSANDrives  []struct {
		Key                      int    `json:"$key"`
		Temp                     int    `json:"temp"`
		Encrypted                bool   `json:"encrypted"`
		TempWarn                 bool   `json:"temp_warn"`
		LocateStatus             string `json:"locate_status"`
		ReallocSectorsWarn       bool   `json:"realloc_sectors_warn"`
		WearLevelWarn            bool   `json:"wear_level_warn"`
		HoursWarn                bool   `json:"hours_warn"`
		CurrentPendingSectorWarn bool   `json:"current_pending_sector_warn"`
		OfflineUncorrectableWarn bool   `json:"offline_uncorrectable_warn"`
		VSANOnlineSince          int64  `json:"vsan_online_since"`
	} `json:"vsan_drives"`
	DrivesOnline []struct {
		State string `json:"state"`
	} `json:"drives_online"`
	NodesOnline struct {
		Nodes []struct {
			State string `json:"state"`
		} `json:"nodes"`
	} `json:"nodes_online"`
}

type VSANTierStatus struct {
	Tier               int     `json:"tier"`
	Status             string  `json:"status"`
	State              string  `json:"state"`
	Transaction        int64   `json:"transaction"`
	Repairs            int64   `json:"repairs"`
	BadDrives          int64   `json:"bad_drives"`
	Encrypted          bool    `json:"encrypted"`
	Redundant          bool    `json:"redundant"`
	LastWalkTimeMs     int64   `json:"last_walk_time_ms"`
	LastFullwalkTimeMs int64   `json:"last_fullwalk_time_ms"`
	Fullwalk           bool    `json:"fullwalk"`
	Progress           float64 `json:"progress"`
	CurSpaceThrottleMs int64   `json:"cur_space_throttle_ms"`
}

type ClusterInfo struct {
	Key          int     `json:"$key"`
	Name         string  `json:"name"`
	Enabled      bool    `json:"enabled"`
	RamPerUnit   int64   `json:"ram_per_unit"`
	CoresPerUnit int     `json:"cores_per_unit"`
	TargetRamPct float64 `json:"target_ram_pct"`
	Status       int     `json:"status"`
}

type ClusterStats struct {
	TotalNodes      int   `json:"total_nodes"`
	OnlineNodes     int   `json:"online_nodes"`
	RunningMachines int   `json:"running_machines"`
	TotalRam        int64 `json:"total_ram"`
	OnlineRam       int64 `json:"online_ram"`
	UsedRam         int64 `json:"used_ram"`
	TotalCores      int   `json:"total_cores"`
	OnlineCores     int   `json:"online_cores"`
	UsedCores       int   `json:"used_cores"`
	PhysRamUsed     int64 `json:"phys_ram_used"`
}

// Prometheus exporter type
type Exporter struct {
	url        string
	username   string
	password   string
	token      string
	httpClient *http.Client

	// Node metrics
	nodesTotal     prometheus.Gauge
	nodeIPMIStatus *prometheus.GaugeVec

	// CPU metrics
	nodeCPUCoreUsage *prometheus.GaugeVec
	nodeCoreTemp     *prometheus.GaugeVec

	// Memory metrics
	nodeRAMUsed    *prometheus.GaugeVec
	nodeRAMPercent *prometheus.GaugeVec

	// Drive metrics
	driveReadOps     *prometheus.CounterVec
	driveWriteOps    *prometheus.CounterVec
	driveReadBytes   *prometheus.CounterVec
	driveWriteBytes  *prometheus.CounterVec
	driveUtil        *prometheus.GaugeVec
	driveReadErrors  *prometheus.CounterVec
	driveWriteErrors *prometheus.CounterVec
	driveAvgLatency  *prometheus.GaugeVec
	driveMaxLatency  *prometheus.GaugeVec
	driveRepairs     *prometheus.CounterVec
	driveThrottle    *prometheus.GaugeVec
	driveWearLevel   *prometheus.CounterVec
	drivePowerOnHours *prometheus.CounterVec
	driveReallocSectors *prometheus.CounterVec
	driveTemperature    *prometheus.GaugeVec

	// NIC metrics
	nicTxPackets *prometheus.CounterVec
	nicRxPackets *prometheus.CounterVec
	nicTxBytes   *prometheus.CounterVec
	nicRxBytes   *prometheus.CounterVec

	// VSAN metrics
	vsanTierCapacity    *prometheus.GaugeVec
	vsanTierUsed        *prometheus.GaugeVec
	vsanTierUsedPct     *prometheus.GaugeVec
	vsanTierAllocated   *prometheus.GaugeVec
	vsanTierDedupeRatio *prometheus.GaugeVec

	// VSAN detailed metrics
	vsanTierTransaction    *prometheus.CounterVec
	vsanTierRepairs        *prometheus.CounterVec
	vsanTierState          *prometheus.GaugeVec
	vsanBadDrives          *prometheus.GaugeVec
	vsanEncryptionStatus   *prometheus.GaugeVec
	vsanRedundant          *prometheus.GaugeVec
	vsanLastWalkTimeMs     *prometheus.GaugeVec
	vsanLastFullwalkTimeMs *prometheus.GaugeVec
	vsanFullwalkStatus     *prometheus.GaugeVec
	vsanFullwalkProgress   *prometheus.GaugeVec
	vsanCurSpaceThrottleMs *prometheus.GaugeVec
	vsanNodesOnline        *prometheus.GaugeVec
	vsanDrivesOnline       *prometheus.GaugeVec
	vsanDriveWearLevel     *prometheus.GaugeVec

	// Cluster metrics
	clustersTotal       prometheus.Gauge
	clusterEnabled      *prometheus.GaugeVec
	clusterRamPerUnit   *prometheus.GaugeVec
	clusterCoresPerUnit *prometheus.GaugeVec
	clusterTargetRamPct *prometheus.GaugeVec
	clusterStatus       *prometheus.GaugeVec

	// Cluster stats metrics
	clusterTotalNodes      *prometheus.GaugeVec
	clusterOnlineNodes     *prometheus.GaugeVec
	clusterRunningMachines *prometheus.GaugeVec
	clusterTotalRam        *prometheus.GaugeVec
	clusterOnlineRam       *prometheus.GaugeVec
	clusterUsedRam         *prometheus.GaugeVec
	clusterTotalCores      *prometheus.GaugeVec
	clusterOnlineCores     *prometheus.GaugeVec
	clusterUsedCores       *prometheus.GaugeVec
	clusterPhysRamUsed     *prometheus.GaugeVec
}

func NewExporter(url, username, password string) *Exporter {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   *scrapeTimeout,
	}

	return &Exporter{
		url:        url,
		username:   username,
		password:   password,
		httpClient: client,
		nodesTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "vergeos_nodes_total",
			Help: "Total number of physical nodes",
		}),
		nodeIPMIStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_node_ipmi_status",
				Help: "IPMI status of the node (1 for online, 0 for offline)",
			},
			[]string{"node_name"},
		),
		nodeCPUCoreUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_node_cpu_core_usage",
				Help: "CPU usage percentage per core",
			},
			[]string{"node_name", "core_id"},
		),
		nodeCoreTemp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_node_core_temp",
				Help: "CPU core temperature",
			},
			[]string{"node_name"},
		),
		nodeRAMUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_node_ram_used",
				Help: "RAM used in bytes",
			},
			[]string{"node_name"},
		),
		nodeRAMPercent: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_node_ram_pct",
				Help: "RAM usage percentage",
			},
			[]string{"node_name"},
		),
		driveReadOps: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_drive_read_ops",
				Help: "Number of read operations",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),
		driveWriteOps: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_drive_write_ops",
				Help: "Number of write operations",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),
		driveReadBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_drive_read_bytes",
				Help: "Number of bytes read",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),
		driveWriteBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_drive_write_bytes",
				Help: "Number of bytes written",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),
		driveUtil: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_drive_utilization",
				Help: "Drive utilization",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),

		// Drive VSAN metrics
		driveReadErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_drive_read_errors",
				Help: "Number of drive read errors",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),

		driveWriteErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_drive_write_errors",
				Help: "Number of drive write errors",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),

		driveAvgLatency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_drive_avg_latency",
				Help: "Drive average latency",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),

		driveMaxLatency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_drive_max_latency",
				Help: "Drive maximum latency",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),

		driveRepairs: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_drive_repairs",
				Help: "Number of drive repairs",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),

		driveThrottle: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_drive_throttle",
				Help: "Drive throttle value",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),

		driveWearLevel: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_drive_wear_level",
				Help: "Drive wear level",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),

		drivePowerOnHours: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_drive_power_on_hours",
				Help: "Drive power on hours",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),

		driveReallocSectors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_drive_reallocated_sectors",
				Help: "Number of reallocated sectors on the drive",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),
		driveTemperature: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_drive_temperature",
				Help: "Temperature of the drive in Celsius",
			},
			[]string{"node_name", "drive_name", "vsan_tier"},
		),

		nicTxPackets: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_nic_tx_packets",
				Help: "Number of transmitted packets",
			},
			[]string{"node_name", "nic_name"},
		),
		nicRxPackets: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_nic_rx_packets",
				Help: "Number of received packets",
			},
			[]string{"node_name", "nic_name"},
		),
		nicTxBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_nic_tx_bytes",
				Help: "Number of transmitted bytes",
			},
			[]string{"node_name", "nic_name"},
		),
		nicRxBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_nic_rx_bytes",
				Help: "Number of received bytes",
			},
			[]string{"node_name", "nic_name"},
		),
		vsanTierCapacity: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_tier_capacity",
				Help: "VSAN tier total capacity in bytes",
			},
			[]string{"tier_id"},
		),
		vsanTierUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_tier_used",
				Help: "VSAN tier used space in bytes",
			},
			[]string{"tier_id"},
		),
		vsanTierUsedPct: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_tier_used_pct",
				Help: "VSAN tier used space percentage",
			},
			[]string{"tier_id"},
		),
		vsanTierAllocated: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_tier_allocated",
				Help: "VSAN tier allocated space in bytes",
			},
			[]string{"tier_id"},
		),
		vsanTierDedupeRatio: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_tier_dedupe_ratio",
				Help: "VSAN tier deduplication ratio",
			},
			[]string{"tier_id"},
		),
		vsanTierTransaction: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_vsan_tier_transaction",
				Help: "VSAN tier transaction count",
			},
			[]string{"tier_id"},
		),
		vsanTierRepairs: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_vsan_tier_repairs",
				Help: "VSAN tier repairs count",
			},
			[]string{"tier_id"},
		),
		vsanTierState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_tier_state",
				Help: "VSAN tier state (1 for online, 0 for offline)",
			},
			[]string{"tier_id", "state"},
		),
		vsanBadDrives: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_bad_drives",
				Help: "Number of bad drives in VSAN tier",
			},
			[]string{"tier_id"},
		),
		vsanEncryptionStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_encryption_status",
				Help: "VSAN tier encryption status (1 for encrypted, 0 for not encrypted)",
			},
			[]string{"tier_id"},
		),
		vsanRedundant: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_redundant",
				Help: "VSAN tier redundancy status (1 for redundant, 0 for not redundant)",
			},
			[]string{"tier_id"},
		),
		vsanLastWalkTimeMs: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_last_walk_time_ms",
				Help: "VSAN tier last walk time in milliseconds",
			},
			[]string{"tier_id"},
		),
		vsanLastFullwalkTimeMs: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_last_fullwalk_time_ms",
				Help: "VSAN tier last full walk time in milliseconds",
			},
			[]string{"tier_id"},
		),
		vsanFullwalkStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_fullwalk_status",
				Help: "VSAN tier full walk status (1 for active, 0 for inactive)",
			},
			[]string{"tier_id"},
		),
		vsanFullwalkProgress: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_fullwalk_progress",
				Help: "VSAN tier full walk progress percentage",
			},
			[]string{"tier_id"},
		),
		vsanCurSpaceThrottleMs: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_cur_space_throttle_ms",
				Help: "VSAN tier current space throttle in milliseconds",
			},
			[]string{"tier_id"},
		),
		vsanNodesOnline: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_nodes_online",
				Help: "Number of online nodes in VSAN tier",
			},
			[]string{"tier_id"},
		),
		vsanDrivesOnline: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_drives_online",
				Help: "Number of drives online per VSAN tier",
			},
			[]string{"tier_id"},
		),
		vsanDriveWearLevel: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_vsan_drive_wear_level",
				Help: "Drive wear level per VSAN tier and drive",
			},
			[]string{"tier_id", "drive_id"},
		),

		// Cluster metrics
		clustersTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "vergeos_clusters_total",
				Help: "Total number of clusters",
			},
		),
		clusterEnabled: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_enabled",
				Help: "Cluster enabled status (1 for enabled, 0 for disabled)",
			},
			[]string{"cluster_name"},
		),
		clusterRamPerUnit: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_ram_per_unit",
				Help: "RAM per unit for the cluster",
			},
			[]string{"cluster_name"},
		),
		clusterCoresPerUnit: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_cores_per_unit",
				Help: "Cores per unit for the cluster",
			},
			[]string{"cluster_name"},
		),
		clusterTargetRamPct: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_target_ram_pct",
				Help: "Target RAM percentage for the cluster",
			},
			[]string{"cluster_name"},
		),
		clusterStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_status",
				Help: "Cluster status (1 for online, 0 for offline)",
			},
			[]string{"cluster_name"},
		),

		clusterTotalNodes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_total_nodes",
				Help: "Total number of nodes in the cluster",
			},
			[]string{"cluster_name"},
		),
		clusterOnlineNodes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_online_nodes",
				Help: "Number of online nodes in the cluster",
			},
			[]string{"cluster_name"},
		),
		clusterRunningMachines: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_running_machines",
				Help: "Number of running machines in the cluster",
			},
			[]string{"cluster_name"},
		),
		clusterTotalRam: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_total_ram",
				Help: "Total RAM in the cluster (MB)",
			},
			[]string{"cluster_name"},
		),
		clusterOnlineRam: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_online_ram",
				Help: "Online RAM in the cluster (MB)",
			},
			[]string{"cluster_name"},
		),
		clusterUsedRam: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_used_ram",
				Help: "Used RAM in the cluster (MB)",
			},
			[]string{"cluster_name"},
		),
		clusterTotalCores: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_total_cores",
				Help: "Total cores in the cluster",
			},
			[]string{"cluster_name"},
		),
		clusterOnlineCores: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_online_cores",
				Help: "Online cores in the cluster",
			},
			[]string{"cluster_name"},
		),
		clusterUsedCores: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_used_cores",
				Help: "Used cores in the cluster",
			},
			[]string{"cluster_name"},
		),
		clusterPhysRamUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_cluster_phys_ram_used",
				Help: "Physical RAM used in the cluster (MB)",
			},
			[]string{"cluster_name"},
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.nodesTotal.Describe(ch)
	e.nodeIPMIStatus.Describe(ch)
	e.nodeCPUCoreUsage.Describe(ch)
	e.nodeCoreTemp.Describe(ch)
	e.nodeRAMUsed.Describe(ch)
	e.nodeRAMPercent.Describe(ch)
	e.driveReadOps.Describe(ch)
	e.driveWriteOps.Describe(ch)
	e.driveReadBytes.Describe(ch)
	e.driveWriteBytes.Describe(ch)
	e.driveUtil.Describe(ch)
	e.driveReadErrors.Describe(ch)
	e.driveWriteErrors.Describe(ch)
	e.driveAvgLatency.Describe(ch)
	e.driveMaxLatency.Describe(ch)
	e.driveRepairs.Describe(ch)
	e.driveThrottle.Describe(ch)
	e.driveWearLevel.Describe(ch)
	e.drivePowerOnHours.Describe(ch)
	e.driveReallocSectors.Describe(ch)
	e.driveTemperature.Describe(ch)
	e.nicTxPackets.Describe(ch)
	e.nicRxPackets.Describe(ch)
	e.nicTxBytes.Describe(ch)
	e.nicRxBytes.Describe(ch)
	e.vsanTierCapacity.Describe(ch)
	e.vsanTierUsed.Describe(ch)
	e.vsanTierUsedPct.Describe(ch)
	e.vsanTierAllocated.Describe(ch)
	e.vsanTierDedupeRatio.Describe(ch)
	e.vsanTierTransaction.Describe(ch)
	e.vsanTierRepairs.Describe(ch)
	e.vsanTierState.Describe(ch)
	e.vsanBadDrives.Describe(ch)
	e.vsanEncryptionStatus.Describe(ch)
	e.vsanRedundant.Describe(ch)
	e.vsanLastWalkTimeMs.Describe(ch)
	e.vsanLastFullwalkTimeMs.Describe(ch)
	e.vsanFullwalkStatus.Describe(ch)
	e.vsanFullwalkProgress.Describe(ch)
	e.vsanCurSpaceThrottleMs.Describe(ch)
	e.vsanNodesOnline.Describe(ch)
	e.vsanDrivesOnline.Describe(ch)
	e.vsanDriveWearLevel.Describe(ch)
	e.clustersTotal.Describe(ch)
	e.clusterEnabled.Describe(ch)
	e.clusterRamPerUnit.Describe(ch)
	e.clusterCoresPerUnit.Describe(ch)
	e.clusterTargetRamPct.Describe(ch)
	e.clusterStatus.Describe(ch)
	e.clusterTotalNodes.Describe(ch)
	e.clusterOnlineNodes.Describe(ch)
	e.clusterRunningMachines.Describe(ch)
	e.clusterTotalRam.Describe(ch)
	e.clusterOnlineRam.Describe(ch)
	e.clusterUsedRam.Describe(ch)
	e.clusterTotalCores.Describe(ch)
	e.clusterOnlineCores.Describe(ch)
	e.clusterUsedCores.Describe(ch)
	e.clusterPhysRamUsed.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.collectNodeMetrics(ch)
	e.collectVSANMetrics(ch)
	e.collectClusterMetrics(ch)

	e.nodesTotal.Collect(ch)
	e.nodeIPMIStatus.Collect(ch)
	e.nodeCPUCoreUsage.Collect(ch)
	e.nodeCoreTemp.Collect(ch)
	e.nodeRAMUsed.Collect(ch)
	e.nodeRAMPercent.Collect(ch)
	e.driveReadOps.Collect(ch)
	e.driveWriteOps.Collect(ch)
	e.driveReadBytes.Collect(ch)
	e.driveWriteBytes.Collect(ch)
	e.driveUtil.Collect(ch)
	e.driveReadErrors.Collect(ch)
	e.driveWriteErrors.Collect(ch)
	e.driveAvgLatency.Collect(ch)
	e.driveMaxLatency.Collect(ch)
	e.driveRepairs.Collect(ch)
	e.driveThrottle.Collect(ch)
	e.driveWearLevel.Collect(ch)
	e.drivePowerOnHours.Collect(ch)
	e.driveReallocSectors.Collect(ch)
	e.driveTemperature.Collect(ch)
	e.nicTxPackets.Collect(ch)
	e.nicRxPackets.Collect(ch)
	e.nicTxBytes.Collect(ch)
	e.nicRxBytes.Collect(ch)
	e.vsanTierCapacity.Collect(ch)
	e.vsanTierUsed.Collect(ch)
	e.vsanTierUsedPct.Collect(ch)
	e.vsanTierAllocated.Collect(ch)
	e.vsanTierDedupeRatio.Collect(ch)
	e.vsanTierTransaction.Collect(ch)
	e.vsanTierRepairs.Collect(ch)
	e.vsanTierState.Collect(ch)
	e.vsanBadDrives.Collect(ch)
	e.vsanEncryptionStatus.Collect(ch)
	e.vsanRedundant.Collect(ch)
	e.vsanLastWalkTimeMs.Collect(ch)
	e.vsanLastFullwalkTimeMs.Collect(ch)
	e.vsanFullwalkStatus.Collect(ch)
	e.vsanFullwalkProgress.Collect(ch)
	e.vsanCurSpaceThrottleMs.Collect(ch)
	e.vsanNodesOnline.Collect(ch)
	e.vsanDrivesOnline.Collect(ch)
	e.vsanDriveWearLevel.Collect(ch)
	e.clustersTotal.Collect(ch)
	e.clusterEnabled.Collect(ch)
	e.clusterRamPerUnit.Collect(ch)
	e.clusterCoresPerUnit.Collect(ch)
	e.clusterTargetRamPct.Collect(ch)
	e.clusterStatus.Collect(ch)
	e.clusterTotalNodes.Collect(ch)
	e.clusterOnlineNodes.Collect(ch)
	e.clusterRunningMachines.Collect(ch)
	e.clusterTotalRam.Collect(ch)
	e.clusterOnlineRam.Collect(ch)
	e.clusterUsedRam.Collect(ch)
	e.clusterTotalCores.Collect(ch)
	e.clusterOnlineCores.Collect(ch)
	e.clusterUsedCores.Collect(ch)
	e.clusterPhysRamUsed.Collect(ch)
}

func main() {
	flag.Parse()

	exporter := NewExporter(*vergeURL, *vergeUsername, *vergePassword)
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>VergeOS Exporter</title></head>
			<body>
			<h1>VergeOS Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Printf("Starting VergeOS exporter on %s", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
