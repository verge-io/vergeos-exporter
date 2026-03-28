package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	vergeos "github.com/verge-io/goVergeOS"
)

// MockServerConfig holds configuration for the mock server
type MockServerConfig struct {
	CloudName string
	Version   string
	Hash      string
}

// DefaultMockConfig returns a default mock server configuration
func DefaultMockConfig() MockServerConfig {
	return MockServerConfig{
		CloudName: "testcloud",
		Version:   "26.0.2.1",
		Hash:      "testbuild",
	}
}

// CreateTestSDKClient creates an SDK client configured for testing with the mock server
func CreateTestSDKClient(t *testing.T, mockServerURL string) *vergeos.Client {
	t.Helper()
	client, err := vergeos.NewClient(
		vergeos.WithBaseURL(mockServerURL),
		vergeos.WithCredentials("testuser", "testpass"),
		vergeos.WithInsecureTLS(true),
	)
	if err != nil {
		t.Fatalf("Failed to create SDK client: %v", err)
	}
	return client
}

// WriteJSONResponse writes a JSON response to the http.ResponseWriter
func WriteJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// HandleVersionCheck handles the SDK version check endpoint
func HandleVersionCheck(w http.ResponseWriter, config MockServerConfig) {
	WriteJSONResponse(w, map[string]interface{}{
		"name":    "v4",
		"version": config.Version,
		"hash":    config.Hash,
	})
}

// HandleSettingsCloudName handles the settings API endpoint for cloud_name
func HandleSettingsCloudName(w http.ResponseWriter, config MockServerConfig) {
	WriteJSONResponse(w, []map[string]string{
		{"key": "cloud_name", "value": config.CloudName},
	})
}

// IsVersionCheck returns true if the request is for the version endpoint
func IsVersionCheck(r *http.Request) bool {
	return strings.HasSuffix(r.URL.Path, "/version.json")
}

// IsSettingsRequest returns true if the request is for settings with cloud_name filter
func IsSettingsRequest(r *http.Request) bool {
	return strings.Contains(r.URL.Path, "/settings") &&
		(strings.Contains(r.URL.RawQuery, "cloud_name") || r.URL.RawQuery == "")
}

// CheckBasicAuth verifies basic auth credentials and returns false if auth fails
func CheckBasicAuth(w http.ResponseWriter, r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok || username != "testuser" || password != "testpass" {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

// NewBaseMockServer creates a mock server that handles version and settings endpoints
func NewBaseMockServer(t *testing.T, config MockServerConfig, additionalHandler func(w http.ResponseWriter, r *http.Request) bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle version check (no auth required)
		if IsVersionCheck(r) {
			HandleVersionCheck(w, config)
			return
		}

		// Check basic auth for other endpoints
		if !CheckBasicAuth(w, r) {
			return
		}

		// Handle settings request
		if IsSettingsRequest(r) {
			HandleSettingsCloudName(w, config)
			return
		}

		// Call additional handler if provided
		if additionalHandler != nil {
			if additionalHandler(w, r) {
				return
			}
		}

		// Default: not found
		http.Error(w, "not found", http.StatusNotFound)
	}))
}

// StorageTierMock represents a mock storage tier
type StorageTierMock struct {
	Key         int    `json:"$key"`
	Tier        int    `json:"tier"`
	Description string `json:"description"`
	Capacity    uint64 `json:"capacity"`
	Used        uint64 `json:"used"`
	UsedPct     uint32 `json:"used_pct"`
	Allocated   uint64 `json:"allocated"`
	DedupeRatio uint32 `json:"dedupe_ratio"`
}

// ClusterTierStatusMock represents a mock cluster tier status
type ClusterTierStatusMock struct {
	Tier               int     `json:"tier"`
	Status             string  `json:"status"`
	State              string  `json:"state"`
	Transaction        uint64  `json:"transaction"`
	Repairs            uint64  `json:"repairs"`
	Working            bool    `json:"working"`
	BadDrives          float64 `json:"bad_drives"`
	Encrypted          bool    `json:"encrypted"`
	Redundant          bool    `json:"redundant"`
	LastWalkTimeMs     uint64  `json:"last_walk_time_ms"`
	LastFullwalkTimeMs uint64  `json:"last_fullwalk_time_ms"`
	Fullwalk           bool    `json:"fullwalk"`
	Progress           float64 `json:"progress"`
	CurSpaceThrottleMs float64 `json:"cur_space_throttle_ms"`
}

// ClusterTierNodesOnlineMock represents mock online nodes for a tier
type ClusterTierNodesOnlineMock struct {
	Nodes []ClusterTierNodeStateMock `json:"nodes"`
}

// ClusterTierNodeStateMock represents a mock node state
type ClusterTierNodeStateMock struct {
	State string `json:"state"`
}

// ClusterTierDriveStateMock represents a mock drive state
type ClusterTierDriveStateMock struct {
	State string `json:"state"`
}

// ClusterTierMock represents a mock cluster tier
type ClusterTierMock struct {
	Key          int                         `json:"$key"`
	Cluster      int                         `json:"cluster"`
	Tier         int                         `json:"tier"`
	Status       ClusterTierStatusMock       `json:"status"`
	NodesOnline  *ClusterTierNodesOnlineMock `json:"nodes_online,omitempty"`
	DrivesOnline []ClusterTierDriveStateMock `json:"drives_online,omitempty"`
}

// NodeVMStatsTotalsMock represents mock VM aggregate stats for a node
type NodeVMStatsTotalsMock struct {
	RunningCores int `json:"running_cores"`
	RunningRAM   int `json:"running_ram"`
}

// NodeMock represents a mock node
type NodeMock struct {
	ID            int                    `json:"id"`
	Name          string                 `json:"name"`
	Physical      bool                   `json:"physical"`
	Cluster       int                    `json:"cluster"`
	Machine       int                    `json:"machine"`
	IPMIStatus    string                 `json:"ipmi_status"`
	RAM           int64                  `json:"ram"`
	VMRAM         int64                  `json:"vm_ram"`
	VMStatsTotals *NodeVMStatsTotalsMock `json:"vm_stats_totals,omitempty"`
}

// ClusterMock represents a mock cluster
type ClusterMock struct {
	Key          int     `json:"$key"`
	Name         string  `json:"name"`
	Enabled      bool    `json:"enabled"`
	RAMPerUnit   int64   `json:"ram_per_unit"`
	CoresPerUnit int     `json:"cores_per_unit"`
	TargetRAMPct float64 `json:"target_ram_pct"`
}

// ClusterStatusMock represents a mock cluster status
type ClusterStatusMock struct {
	Cluster         int    `json:"cluster"`
	Status          string `json:"status"`
	State           string `json:"state"`
	TotalNodes      int    `json:"total_nodes"`
	OnlineNodes     int    `json:"online_nodes"`
	RunningMachines int    `json:"running_machines"`
	TotalRAM        int64  `json:"total_ram"`
	OnlineRAM       int64  `json:"online_ram"`
	UsedRAM         int64  `json:"used_ram"`
	TotalCores      int    `json:"total_cores"`
	OnlineCores     int    `json:"online_cores"`
	UsedCores       int    `json:"used_cores"`
	PhysRAMUsed     int64  `json:"phys_ram_used"`
}

// MachineStatsMock represents a mock machine stats record
type MachineStatsMock struct {
	Key           int             `json:"$key"`
	Machine       int             `json:"machine"`
	TotalCPU      uint8           `json:"total_cpu"`
	UserCPU       uint8           `json:"user_cpu"`
	SystemCPU     uint8           `json:"system_cpu"`
	IOWaitCPU     uint8           `json:"iowait_cpu"`
	RAMUsed       uint32          `json:"ram_used"`
	RAMPct        uint8           `json:"ram_pct"`
	CoreUsageList json.RawMessage `json:"core_usagelist"`
	CoreTemp      uint16          `json:"core_temp"`
	CoreTempTop   uint16          `json:"core_temp_top"`
}

// MachineNICStatsMock represents mock NIC traffic stats
type MachineNICStatsMock struct {
	Key     int    `json:"$key"`
	TxPckts uint64 `json:"tx_pckts"`
	RxPckts uint64 `json:"rx_pckts"`
	TxBytes uint64 `json:"tx_bytes"`
	RxBytes uint64 `json:"rx_bytes"`
}

// MachineNICStatusMock represents mock NIC link status
type MachineNICStatusMock struct {
	Key    int    `json:"$key"`
	Status string `json:"status"`
	Speed  uint32 `json:"speed"`
}

// MachineNICMock represents a mock machine NIC
type MachineNICMock struct {
	Key     int                   `json:"$key"`
	Machine int                   `json:"machine"`
	Name    string                `json:"name"`
	Stats   *MachineNICStatsMock  `json:"stats,omitempty"`
	Status  *MachineNICStatusMock `json:"status,omitempty"`
}

// MachineDrivePhysMock represents a mock physical drive
type MachineDrivePhysMock struct {
	Key             int    `json:"$key"`
	ParentDrive     int    `json:"parent_drive"`
	Path            string `json:"path"`
	Serial          string `json:"serial"`
	Temp            uint16 `json:"temp"`
	WearLevel       uint32 `json:"wear_level"`
	Hours           uint32 `json:"hours"`
	ReallocSectors  uint32 `json:"realloc_sectors"`
	VSANTier        int8   `json:"vsan_tier"`
	VSANReadErrors  uint64 `json:"vsan_read_errors"`
	VSANWriteErrors uint64 `json:"vsan_write_errors"`
	VSANRepairing   uint64 `json:"vsan_repairing"`
	VSANThrottle    uint64 `json:"vsan_throttle"`
	NodeDisplay     string `json:"node_display"`
	StatusList      string `json:"statuslist"`
}

// MachineDriveStatsMock represents mock drive I/O stats
type MachineDriveStatsMock struct {
	Key         int     `json:"$key"`
	ParentDrive int     `json:"parent_drive"`
	Reads       uint64  `json:"reads"`
	Writes      uint64  `json:"writes"`
	ReadBytes   uint64  `json:"read_bytes"`
	WriteBytes  uint64  `json:"write_bytes"`
	ServiceTime float64 `json:"service_time"`
	Util        float64 `json:"util"`
	Physical    bool    `json:"physical"`
}

// TenantMock represents a mock tenant
type TenantMock struct {
	Key         int    `json:"$key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	UUID        string `json:"uuid,omitempty"`
	IsSnapshot  bool   `json:"is_snapshot"`
	Isolate     bool   `json:"isolate"`
}

// TenantNodeMock represents a mock tenant node
type TenantNodeMock struct {
	Key        int    `json:"$key"`
	Tenant     int    `json:"tenant"`
	NodeID     int    `json:"nodeid"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	Machine    int    `json:"machine"`
	IsSnapshot bool   `json:"is_snapshot"`
	CPUCores   int    `json:"cpu_cores"`
	RAM        int    `json:"ram"`
}

// TenantStatusMock represents a mock tenant status
type TenantStatusMock struct {
	Key       int    `json:"$key"`
	Tenant    int    `json:"tenant"`
	Running   bool   `json:"running"`
	Starting  bool   `json:"starting"`
	Stopping  bool   `json:"stopping"`
	Migrating bool   `json:"migrating"`
	Status    string `json:"status"`
	State     string `json:"state"`
}

// TenantStatsHistoryShortMock represents a mock tenant stats history record
type TenantStatsHistoryShortMock struct {
	Key          int    `json:"$key"`
	Tenant       int    `json:"tenant"`
	Timestamp    uint32 `json:"timestamp"`
	TotalCPU     uint32 `json:"total_cpu"`
	CoreCount    uint32 `json:"core_count"`
	RAMUsed      uint32 `json:"ram_used"`
	RAMAllocated uint32 `json:"ram_allocated"`
	RAMPct       uint32 `json:"ram_pct"`
	IPCount      uint32 `json:"ip_count"`
	VGPUsUsed    uint16 `json:"vgpus_used"`
	VGPUsTotal   uint16 `json:"vgpus_total"`
	GPUsUsed     uint16 `json:"gpus_used"`
	GPUsTotal    uint16 `json:"gpus_total"`
}

// TenantStorageMock represents a mock tenant storage allocation
type TenantStorageMock struct {
	Key         int   `json:"$key"`
	Tenant      int   `json:"tenant"`
	Tier        int   `json:"tier"`
	Provisioned int64 `json:"provisioned"`
	Used        int64 `json:"used"`
	Allocated   int64 `json:"allocated"`
	UsedPct     int   `json:"used_pct"`
}

// TenantLayer2NetworkMock represents a mock tenant L2 network assignment
type TenantLayer2NetworkMock struct {
	Key     int  `json:"$key"`
	Tenant  int  `json:"tenant"`
	VNet    int  `json:"vnet"`
	Enabled bool `json:"enabled"`
}

// MachineStatusMock represents a mock machine status
type MachineStatusMock struct {
	Key          int    `json:"$key"`
	Machine      int    `json:"machine"`
	Running      bool   `json:"running"`
	Status       string `json:"status"`
	State        string `json:"state"`
	Node         int    `json:"node,omitempty"`
	NodeName     string `json:"node_name,omitempty"`
	RunningCores int    `json:"running_cores"`
	RunningRAM   int    `json:"running_ram"`
}

// VMMock represents a mock VM
type VMMock struct {
	Key        int    `json:"$key"`
	Name       string `json:"name"`
	Machine    int    `json:"machine"`
	Cluster    int    `json:"cluster"`
	IsSnapshot bool   `json:"is_snapshot"`
	PowerState bool   `json:"powerstate"`
	Enabled    bool   `json:"enabled"`
	CPUCores   int    `json:"cpu_cores"`
	RAM        int    `json:"ram"`
}

// VMDriveMock represents a mock VM drive
type VMDriveMock struct {
	Key           int    `json:"$key"`
	Machine       int    `json:"machine"`
	Name          string `json:"name"`
	Interface     string `json:"interface"`
	Media         string `json:"media"`
	SizeBytes     int64  `json:"disksize"`
	UsedBytes     int64  `json:"used_bytes"`
	PreferredTier string `json:"preferred_tier,omitempty"`
	Enabled       bool   `json:"enabled"`
}

// UpdateSettingsMock represents mock update settings
type UpdateSettingsMock struct {
	Key        int    `json:"$key"`
	Source     int    `json:"source"`
	Branch     int    `json:"branch"`
	BranchName string `json:"branch_name"`
}

// UpdateSourcePackageMock represents a mock update source package
type UpdateSourcePackageMock struct {
	Key     int    `json:"$key"`
	Name    string `json:"name"`
	Branch  int    `json:"branch"`
	Source  int    `json:"source"`
	Version string `json:"version"`
}
