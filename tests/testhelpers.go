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

// ClusterTierMock represents a mock cluster tier
type ClusterTierMock struct {
	Key     int                   `json:"$key"`
	Cluster int                   `json:"cluster"`
	Tier    int                   `json:"tier"`
	Status  ClusterTierStatusMock `json:"status"`
}

// NodeMock represents a mock node
type NodeMock struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Physical   bool   `json:"physical"`
	Cluster    int    `json:"cluster"`
	IPMIStatus string `json:"ipmi_status"`
	RAM        int64  `json:"ram"`
	VMRAM      int64  `json:"vm_ram"`
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
