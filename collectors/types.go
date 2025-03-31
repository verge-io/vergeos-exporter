package collectors

// NodeDriveStats represents drive statistics for a node
type NodeDriveStats struct {
	Name           string  `json:"name"`
	ReadOps        uint64  `json:"read_ops"`
	WriteOps       uint64  `json:"write_ops"`
	ReadBytes      uint64  `json:"read_bytes"`
	WriteBytes     uint64  `json:"write_bytes"`
	Utilization    float64 `json:"utilization"`
	ReadErrors     uint64  `json:"read_errors"`
	WriteErrors    uint64  `json:"write_errors"`
	AvgLatency     float64 `json:"avg_latency"`
	MaxLatency     float64 `json:"max_latency"`
	Repairs        uint64  `json:"repairs"`
	Throttle       float64 `json:"throttle"`
	WearLevel      uint64  `json:"wear_level"`
	PowerOnHours   uint64  `json:"power_on_hours"`
	ReallocSectors uint64  `json:"realloc_sectors"`
	Temperature    float64 `json:"temperature"`
}

// DriveResponse represents the API response for drives
type DriveResponse struct {
	Name  string `json:"name"`
	Stats struct {
		ReadOps        float64 `json:"rops"`
		WriteOps       float64 `json:"wops"`
		ReadBytes      float64 `json:"read_bytes"`
		WriteBytes     float64 `json:"write_bytes"`
		Util           float64 `json:"util"`
		ReadErrors     float64 `json:"read_errors"`
		WriteErrors    float64 `json:"write_errors"`
		AvgLatency     float64 `json:"avg_latency"`
		MaxLatency     float64 `json:"max_latency"`
		Repairs        float64 `json:"repairs"`
		Throttle       float64 `json:"throttle"`
		WearLevel      float64 `json:"wear_level"`
		PowerOnHours   float64 `json:"hours"`
		ReallocSectors float64 `json:"realloc_sectors"`
		Temperature    float64 `json:"temp"`
	} `json:"stats"`
	PhysicalStatus struct {
		Serial   string `json:"serial"`
		VsanTier int    `json:"vsan_tier"`
	} `json:"physical_status"`
}

// NodeResponse represents the API response for a node
type NodeResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ID          int    `json:"id"`
	Machine     struct {
		Stats struct {
			TotalCPU float64 `json:"total_cpu"`
			RAM      int64   `json:"ram"`
		} `json:"stats"`
		Drives []DriveResponse `json:"drives"`
		Status struct {
			Status        string `json:"status"`
			StatusDisplay string `json:"status_display"`
			State         string `json:"state"`
		} `json:"status"`
		NodeStats struct {
			CPUTemp       float64 `json:"cpu_temp"`
			CPUUsage      float64 `json:"cpu_usage"`
			MemoryTotal   int64   `json:"memory_total"`
			MemoryUsed    int64   `json:"memory_used"`
			MemoryUsedPct float64 `json:"memory_used_pct"`
		} `json:"node_stats"`
		DrivesDetailed []struct {
			Name           string `json:"name"`
			PhysicalStatus struct {
				VsanTier string `json:"vsan_tier"`
				Serial   string `json:"serial"`
			} `json:"physical_status"`
			Stats struct {
				ReadOps        int     `json:"read_ops"`
				WriteOps       int     `json:"write_ops"`
				ReadBytes      int64   `json:"read_bytes"`
				WriteBytes     int64   `json:"write_bytes"`
				Util           float64 `json:"util"`
				ReadErrors     int     `json:"read_errors"`
				WriteErrors    int     `json:"write_errors"`
				AvgLatency     float64 `json:"avg_latency"`
				MaxLatency     float64 `json:"max_latency"`
				Repairs        int     `json:"repairs"`
				Throttle       float64 `json:"throttle"`
				WearLevel      int     `json:"wear_level"`
				PowerOnHours   int     `json:"power_on_hours"`
				ReallocSectors int     `json:"realloc_sectors"`
				Temperature    float64 `json:"temperature"`
			} `json:"stats"`
		} `json:"drives_detailed"`
	} `json:"machine"`
	Physical bool `json:"physical"`
}

// VSANTier represents a VSAN storage tier
type VSANTier struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Capacity    uint64  `json:"capacity"`
	Used        uint64  `json:"used"`
	Allocated   uint64  `json:"allocated"`
	DedupeRatio float64 `json:"dedupe_ratio"`
}

// VSANTierStatus represents the status of a VSAN tier
type VSANTierStatus struct {
	Tier               int    `json:"tier"`
	Status             string `json:"status"`
	State              string `json:"state"`
	Capacity           int64  `json:"capacity"`
	Used               int64  `json:"used"`
	UsedPct            int    `json:"used_pct"`
	Redundant          bool   `json:"redundant"`
	Encrypted          bool   `json:"encrypted"`
	Working            bool   `json:"working"`
	LastWalkTimeMs     int    `json:"last_walk_time_ms"`
	LastFullwalkTimeMs int    `json:"last_fullwalk_time_ms"`
	Transaction        int64  `json:"transaction"`
	Repairs            int64  `json:"repairs"`
	BadDrives          int    `json:"bad_drives"`
	Fullwalk           bool   `json:"fullwalk"`
	Progress           int    `json:"progress"`
	CurSpaceThrottleMs int    `json:"cur_space_throttle_ms"`
	StatusDisplay      string `json:"status_display"`
}

// VSANTierResponse represents the API response for VSAN tier status
type VSANTierResponse []struct {
	Key            int            `json:"$key"`
	Tier           int            `json:"tier"`
	Description    string         `json:"description"`
	ClusterDisplay string         `json:"cluster_display"`
	Status         VSANTierStatus `json:"status"`
	DrivesOnline   []struct {
		State string `json:"state"`
	} `json:"drives_online"`
	NodesOnline struct {
		Nodes []struct {
			State string `json:"state"`
		} `json:"nodes"`
	} `json:"nodes_online"`
	DrivesCount int `json:"drives_count"`
	NodesCount  int `json:"nodes_count"`
}

// ClusterInfo represents information about a cluster
type ClusterInfo struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	TotalRAM    int64  `json:"total_ram"`
	UsedRAM     int64  `json:"used_ram"`
	TotalCores  int    `json:"total_cores"`
	UsedCores   int    `json:"used_cores"`
	RunningVMs  int    `json:"running_machines"`
	TotalNodes  int    `json:"total_nodes"`
	OnlineNodes int    `json:"online_nodes"`
	OnlineRAM   int64  `json:"online_ram"`
	OnlineCores int    `json:"online_cores"`
	PhysRAMUsed int64  `json:"phys_ram_used"`
}

// ClusterListResponse represents the list of clusters
type ClusterListResponse []struct {
	Key  int    `json:"$key"`
	Name string `json:"name"`
}

// ClusterDetailResponse represents the detailed information about a cluster
type ClusterDetailResponse struct {
	Key          int    `json:"$key"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	RamPerUnit   int    `json:"ram_per_unit"`
	CoresPerUnit int    `json:"cores_per_unit"`
	TargetRamPct int    `json:"target_ram_pct"`
	Status       struct {
		TotalNodes      int   `json:"total_nodes"`
		OnlineNodes     int   `json:"online_nodes"`
		OnlineRam       int64 `json:"online_ram"`
		OnlineCores     int   `json:"online_cores"`
		PhysRamUsed     int64 `json:"phys_ram_used"`
		RunningMachines int   `json:"running_machines"`
		TotalRam        int64 `json:"total_ram"`
		UsedRam         int64 `json:"used_ram"`
		UsedCores       int   `json:"used_cores"`
	} `json:"status"`
}

// Setting represents a system setting
type Setting struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	DefaultValue string `json:"default_value"`
	Description  string `json:"description"`
}

// StorageTier represents a VSAN storage tier
type StorageTier struct {
	Tier         int     `json:"tier"`
	Description  string  `json:"description"`
	Capacity     int64   `json:"capacity"`
	Used         int64   `json:"used"`
	Allocated    int64   `json:"allocated"`
	Stats        int     `json:"stats"`
	Modified     int64   `json:"modified"`
	UsedPct      int     `json:"used_pct"`
	UsedInflated int64   `json:"used_inflated"`
	DedupeRatio  float64 `json:"dedupe_ratio"`
}

// VSANResponse represents the API response for VSAN tiers
type VSANResponse []StorageTier

// ClusterTierResponse represents the API response for cluster tier details
type ClusterTierResponse []struct {
	Key     int `json:"$key"`
	Cluster struct {
		Key   int `json:"$key"`
		Nodes []struct {
			Name    string `json:"name"`
			Key     int    `json:"$key"`
			Machine struct {
				Status struct {
					Status        string `json:"status"`
					StatusDisplay string `json:"status_display"`
					State         string `json:"state"`
				} `json:"status"`
				Stats struct {
					IowaitCPU float64 `json:"iowait_cpu"`
				} `json:"stats"`
				DrivesCount int `json:"drives_count"`
				Drives      []struct {
					PhysicalStatus struct {
						VsanUsed      int64 `json:"vsan_used"`
						VsanMax       int64 `json:"vsan_max"`
						VsanRepairing int64 `json:"vsan_repairing"`
						VsanTier      int   `json:"vsan_tier"`
					} `json:"physical_status"`
				} `json:"drives"`
			} `json:"machine"`
		} `json:"nodes"`
	} `json:"cluster"`
	Status struct {
		Key                int    `json:"$key"`
		Tier               int    `json:"tier"`
		Status             string `json:"status"`
		State              string `json:"state"`
		Capacity           int64  `json:"capacity"`
		Used               int64  `json:"used"`
		UsedPct            int    `json:"used_pct"`
		Redundant          bool   `json:"redundant"`
		Encrypted          bool   `json:"encrypted"`
		Working            bool   `json:"working"`
		LastWalkTimeMs     int64  `json:"last_walk_time_ms"`
		LastFullwalkTimeMs int64  `json:"last_fullwalk_time_ms"`
		Transaction        int64  `json:"transaction"`
		Repairs            int64  `json:"repairs"`
		BadDrives          int    `json:"bad_drives"`
		Fullwalk           bool   `json:"fullwalk"`
		Progress           int    `json:"progress"`
		CurSpaceThrottleMs int    `json:"cur_space_throttle_ms"`
		StatusDisplay      string `json:"status_display"`
	} `json:"status"`
}

// PhysicalNodeResponse represents the API response for physical nodes
type PhysicalNodeResponse []struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	ID             int    `json:"id"`
	Machine        int    `json:"machine"`
	Physical       bool   `json:"physical"`
	IPMIStatus     string `json:"ipmi_status"`
	IPMIStatusInfo string `json:"ipmi_status_info"`
}

// SystemNameResponse represents the API response for system name settings
type SystemNameResponse []struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
