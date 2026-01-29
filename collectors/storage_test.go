package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	vergeos "github.com/verge-io/goVergeOS"
)

// createTestSDKClient creates an SDK client configured for testing with the mock server
func createTestSDKClient(mockServerURL string) *vergeos.Client {
	client, _ := vergeos.NewClient(
		vergeos.WithBaseURL(mockServerURL),
		vergeos.WithCredentials("testuser", "testpass"),
		vergeos.WithInsecureTLS(true),
	)
	return client
}

func TestDriveStateMonitoring(t *testing.T) {
	// Create a mock server to simulate the VergeOS API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add basic auth check
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Handle different API endpoints
		switch {
		case strings.Contains(r.URL.Path, "/api/v4/settings"):
			// Return system name
			settings := []Setting{
				{
					Key:   "cloud_name",
					Value: "test-system",
				},
			}
			json.NewEncoder(w).Encode(settings)

		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			// Return tier information
			tiers := VSANResponse{
				{
					Tier:        0,
					Description: "SSD Tier",
					Capacity:    1000000000000,
					Used:        100000000000,
					Allocated:   500000000000,
					DedupeRatio: 2.0,
				},
				{
					Tier:        1,
					Description: "HDD Tier",
					Capacity:    5000000000000,
					Used:        2000000000000,
					Allocated:   3000000000000,
					DedupeRatio: 1.5,
				},
			}
			json.NewEncoder(w).Encode(tiers)

		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			// Return tier details for other metrics
			response := []struct {
				Key    int `json:"$key"`
				Status struct {
					Tier               int    `json:"tier"`
					StatusDisplay      string `json:"status_display"`
					Transaction        int64  `json:"transaction"`
					Repairs            int64  `json:"repairs"`
					Working            bool   `json:"working"`
					BadDrives          int    `json:"bad_drives"`
					Encrypted          bool   `json:"encrypted"`
					Redundant          bool   `json:"redundant"`
					LastWalkTimeMs     int64  `json:"last_walk_time_ms"`
					LastFullwalkTimeMs int64  `json:"last_fullwalk_time_ms"`
					Fullwalk           bool   `json:"fullwalk"`
					Progress           int    `json:"progress"`
					CurSpaceThrottleMs int    `json:"cur_space_throttle_ms"`
				} `json:"status"`
				NodesOnline struct {
					Nodes []struct {
						State string `json:"state"`
					} `json:"nodes"`
				} `json:"nodes_online"`
				DrivesOnline []struct {
					State string `json:"state"`
				} `json:"drives_online"`
			}{
				{
					Key: 1,
					Status: struct {
						Tier               int    `json:"tier"`
						StatusDisplay      string `json:"status_display"`
						Transaction        int64  `json:"transaction"`
						Repairs            int64  `json:"repairs"`
						Working            bool   `json:"working"`
						BadDrives          int    `json:"bad_drives"`
						Encrypted          bool   `json:"encrypted"`
						Redundant          bool   `json:"redundant"`
						LastWalkTimeMs     int64  `json:"last_walk_time_ms"`
						LastFullwalkTimeMs int64  `json:"last_fullwalk_time_ms"`
						Fullwalk           bool   `json:"fullwalk"`
						Progress           int    `json:"progress"`
						CurSpaceThrottleMs int    `json:"cur_space_throttle_ms"`
					}{
						Tier:               0,
						StatusDisplay:      "Online",
						Transaction:        100,
						Repairs:            5,
						Working:            true,
						BadDrives:          0,
						Encrypted:          true,
						Redundant:          true,
						LastWalkTimeMs:     1000,
						LastFullwalkTimeMs: 5000,
						Fullwalk:           false,
						Progress:           100,
						CurSpaceThrottleMs: 0,
					},
					NodesOnline: struct {
						Nodes []struct {
							State string `json:"state"`
						} `json:"nodes"`
					}{
						Nodes: []struct {
							State string `json:"state"`
						}{
							{State: "online"},
							{State: "online"},
						},
					},
					DrivesOnline: []struct {
						State string `json:"state"`
					}{
						{State: "online"},
						{State: "online"},
					},
				},
				{
					Key: 2,
					Status: struct {
						Tier               int    `json:"tier"`
						StatusDisplay      string `json:"status_display"`
						Transaction        int64  `json:"transaction"`
						Repairs            int64  `json:"repairs"`
						Working            bool   `json:"working"`
						BadDrives          int    `json:"bad_drives"`
						Encrypted          bool   `json:"encrypted"`
						Redundant          bool   `json:"redundant"`
						LastWalkTimeMs     int64  `json:"last_walk_time_ms"`
						LastFullwalkTimeMs int64  `json:"last_fullwalk_time_ms"`
						Fullwalk           bool   `json:"fullwalk"`
						Progress           int    `json:"progress"`
						CurSpaceThrottleMs int    `json:"cur_space_throttle_ms"`
					}{
						Tier:               1,
						StatusDisplay:      "Online",
						Transaction:        200,
						Repairs:            10,
						Working:            true,
						BadDrives:          0,
						Encrypted:          true,
						Redundant:          true,
						LastWalkTimeMs:     1500,
						LastFullwalkTimeMs: 7500,
						Fullwalk:           false,
						Progress:           100,
						CurSpaceThrottleMs: 0,
					},
					NodesOnline: struct {
						Nodes []struct {
							State string `json:"state"`
						} `json:"nodes"`
					}{
						Nodes: []struct {
							State string `json:"state"`
						}{
							{State: "online"},
							{State: "online"},
						},
					},
					DrivesOnline: []struct {
						State string `json:"state"`
					}{
						{State: "online"},
						{State: "online"},
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/api/v4/machine_drives"):
			// Return drive information using the new API endpoint format
			response := MachineDriveResponse{
				// Tier 0 drives
				{
					Key:         1,
					Name:        "sda",
					Node:        101,
					Type:        "node",
					NodeDisplay: "node1",
					StatusList:  "online",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      2,
						VsanRepairing: 0,
					},
				},
				{
					Key:         2,
					Name:        "sdb",
					Node:        101,
					Type:        "node",
					NodeDisplay: "node1",
					StatusList:  "online",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      2,
						VsanRepairing: 0,
					},
				},
				{
					Key:         3,
					Name:        "sdc",
					Node:        101,
					Type:        "node",
					NodeDisplay: "node1",
					StatusList:  "online",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      2,
						VsanRepairing: 0,
					},
				},
				{
					Key:         4,
					Name:        "sdd",
					Node:        102,
					Type:        "node",
					NodeDisplay: "node2",
					StatusList:  "offline",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      2,
						VsanRepairing: 0,
					},
				},
				{
					Key:         5,
					Name:        "sde",
					Node:        102,
					Type:        "node",
					NodeDisplay: "node2",
					StatusList:  "initializing",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      0,
						VsanRepairing: 0,
					},
				},
				{
					Key:         6,
					Name:        "sdf",
					Node:        102,
					Type:        "node",
					NodeDisplay: "node2",
					StatusList:  "verifying",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      0,
						VsanRepairing: 0,
					},
				},
				{
					Key:         7,
					Name:        "sdg",
					Node:        103,
					Type:        "node",
					NodeDisplay: "node3",
					StatusList:  "online",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      0,
						VsanRepairing: 1, // This will be marked as "repairing"
					},
				},
				// Tier 1 drives
				{
					Key:         8,
					Name:        "sdh",
					Node:        101,
					Type:        "node",
					NodeDisplay: "node1",
					StatusList:  "online",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      1,
						VsanRepairing: 0,
					},
				},
				{
					Key:         9,
					Name:        "sdi",
					Node:        101,
					Type:        "node",
					NodeDisplay: "node1",
					StatusList:  "online",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      1,
						VsanRepairing: 0,
					},
				},
				{
					Key:         10,
					Name:        "sdj",
					Node:        102,
					Type:        "node",
					NodeDisplay: "node2",
					StatusList:  "online",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      1,
						VsanRepairing: 0,
					},
				},
				{
					Key:         11,
					Name:        "sdk",
					Node:        102,
					Type:        "node",
					NodeDisplay: "node2",
					StatusList:  "online",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      1,
						VsanRepairing: 0,
					},
				},
				{
					Key:         12,
					Name:        "sdl",
					Node:        103,
					Type:        "node",
					NodeDisplay: "node3",
					StatusList:  "noredundant",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      1,
						VsanRepairing: 0,
					},
				},
				{
					Key:         13,
					Name:        "sdm",
					Node:        103,
					Type:        "node",
					NodeDisplay: "node3",
					StatusList:  "outofspace",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      1,
						VsanRepairing: 0,
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/api/v4/nodes"):
			// Return empty node list for simplicity
			nodes := []struct{}{}
			json.NewEncoder(w).Encode(nodes)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create a registry for testing
	registry := prometheus.NewRegistry()

	// Create a storage collector with the mock server
	sdkClient := createTestSDKClient(mockServer.URL)
	sc := NewStorageCollector(sdkClient, mockServer.URL, "testuser", "testpass")
	registry.MustRegister(sc)

	// Define expected metrics for both tiers
	expected := `
# HELP vergeos_vsan_drive_states Number of drives in each state (online, offline, repairing, initializing, verifying, noredundant, outofspace)
# TYPE vergeos_vsan_drive_states gauge
vergeos_vsan_drive_states{state="initializing",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_states{state="initializing",system_name="test-system",tier="2"} 1
vergeos_vsan_drive_states{state="noredundant",system_name="test-system",tier="1"} 1
vergeos_vsan_drive_states{state="noredundant",system_name="test-system",tier="2"} 0
vergeos_vsan_drive_states{state="offline",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_states{state="offline",system_name="test-system",tier="2"} 1
vergeos_vsan_drive_states{state="online",system_name="test-system",tier="1"} 4
vergeos_vsan_drive_states{state="online",system_name="test-system",tier="2"} 3
vergeos_vsan_drive_states{state="outofspace",system_name="test-system",tier="1"} 1
vergeos_vsan_drive_states{state="outofspace",system_name="test-system",tier="2"} 0
vergeos_vsan_drive_states{state="repairing",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_states{state="repairing",system_name="test-system",tier="2"} 1
vergeos_vsan_drive_states{state="verifying",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_states{state="verifying",system_name="test-system",tier="2"} 1
`

	// Gather metrics
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "vergeos_vsan_drive_states"); err != nil {
		t.Errorf("Metrics do not match expected values: %v", err)
	}
}

func TestDriveStateEdgeCases(t *testing.T) {
	// Create a mock server to simulate the VergeOS API with edge cases
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add basic auth check
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Handle different API endpoints
		switch {
		case strings.Contains(r.URL.Path, "/api/v4/settings"):
			// Return system name
			settings := []Setting{
				{
					Key:   "cloud_name",
					Value: "test-system",
				},
			}
			json.NewEncoder(w).Encode(settings)

		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			// Return tier information
			tiers := VSANResponse{
				{
					Tier:        0,
					Description: "Empty Tier",
					Capacity:    1000000000000,
					Used:        0,
					Allocated:   0,
					DedupeRatio: 1.0,
				},
				{
					Tier:        1,
					Description: "Unknown States Tier",
					Capacity:    5000000000000,
					Used:        2000000000000,
					Allocated:   3000000000000,
					DedupeRatio: 1.5,
				},
			}
			json.NewEncoder(w).Encode(tiers)

		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			// Return tier details for other metrics
			clusterResponse := []struct {
				Key    int `json:"$key"`
				Status struct {
					Tier               int    `json:"tier"`
					StatusDisplay      string `json:"status_display"`
					Transaction        int64  `json:"transaction"`
					Repairs            int64  `json:"repairs"`
					Working            bool   `json:"working"`
					BadDrives          int    `json:"bad_drives"`
					Encrypted          bool   `json:"encrypted"`
					Redundant          bool   `json:"redundant"`
					LastWalkTimeMs     int64  `json:"last_walk_time_ms"`
					LastFullwalkTimeMs int64  `json:"last_fullwalk_time_ms"`
					Fullwalk           bool   `json:"fullwalk"`
					Progress           int    `json:"progress"`
					CurSpaceThrottleMs int    `json:"cur_space_throttle_ms"`
				} `json:"status"`
				NodesOnline struct {
					Nodes []struct {
						State string `json:"state"`
					} `json:"nodes"`
				} `json:"nodes_online"`
				DrivesOnline []struct {
					State string `json:"state"`
				} `json:"drives_online"`
			}{
				{
					Key: 1,
					Status: struct {
						Tier               int    `json:"tier"`
						StatusDisplay      string `json:"status_display"`
						Transaction        int64  `json:"transaction"`
						Repairs            int64  `json:"repairs"`
						Working            bool   `json:"working"`
						BadDrives          int    `json:"bad_drives"`
						Encrypted          bool   `json:"encrypted"`
						Redundant          bool   `json:"redundant"`
						LastWalkTimeMs     int64  `json:"last_walk_time_ms"`
						LastFullwalkTimeMs int64  `json:"last_fullwalk_time_ms"`
						Fullwalk           bool   `json:"fullwalk"`
						Progress           int    `json:"progress"`
						CurSpaceThrottleMs int    `json:"cur_space_throttle_ms"`
					}{
						Tier:               0,
						StatusDisplay:      "Online",
						Transaction:        100,
						Repairs:            5,
						Working:            true,
						BadDrives:          0,
						Encrypted:          true,
						Redundant:          true,
						LastWalkTimeMs:     1000,
						LastFullwalkTimeMs: 5000,
						Fullwalk:           false,
						Progress:           100,
						CurSpaceThrottleMs: 0,
					},
					NodesOnline: struct {
						Nodes []struct {
							State string `json:"state"`
						} `json:"nodes"`
					}{
						Nodes: []struct {
							State string `json:"state"`
						}{
							{State: "online"},
						},
					},
					DrivesOnline: []struct {
						State string `json:"state"`
					}{},
				},
				{
					Key: 2,
					Status: struct {
						Tier               int    `json:"tier"`
						StatusDisplay      string `json:"status_display"`
						Transaction        int64  `json:"transaction"`
						Repairs            int64  `json:"repairs"`
						Working            bool   `json:"working"`
						BadDrives          int    `json:"bad_drives"`
						Encrypted          bool   `json:"encrypted"`
						Redundant          bool   `json:"redundant"`
						LastWalkTimeMs     int64  `json:"last_walk_time_ms"`
						LastFullwalkTimeMs int64  `json:"last_fullwalk_time_ms"`
						Fullwalk           bool   `json:"fullwalk"`
						Progress           int    `json:"progress"`
						CurSpaceThrottleMs int    `json:"cur_space_throttle_ms"`
					}{
						Tier:               1,
						StatusDisplay:      "Online",
						Transaction:        200,
						Repairs:            10,
						Working:            true,
						BadDrives:          0,
						Encrypted:          true,
						Redundant:          true,
						LastWalkTimeMs:     1500,
						LastFullwalkTimeMs: 7500,
						Fullwalk:           false,
						Progress:           100,
						CurSpaceThrottleMs: 0,
					},
					NodesOnline: struct {
						Nodes []struct {
							State string `json:"state"`
						} `json:"nodes"`
					}{
						Nodes: []struct {
							State string `json:"state"`
						}{
							{State: "online"},
						},
					},
					DrivesOnline: []struct {
						State string `json:"state"`
					}{
						{State: "online"},
					},
				},
			}
			json.NewEncoder(w).Encode(clusterResponse)

		case strings.Contains(r.URL.Path, "/api/v4/machine_drives"):
			// Return edge cases: empty tier and unknown states
			response := MachineDriveResponse{
				// Tier 1 - One online drive and two with unknown states
				{
					Key:         1,
					Name:        "sda",
					Node:        101,
					Type:        "node",
					NodeDisplay: "node1",
					StatusList:  "online",
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      1, // Keep tier 1 for this drive
						VsanRepairing: 0,
					},
				},
				{
					Key:         2,
					Name:        "sdb",
					Node:        101,
					Type:        "node",
					NodeDisplay: "node1",
					StatusList:  "unknown_state", // Unknown state
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      1, // Keep tier 1 for this drive
						VsanRepairing: 0,
					},
				},
				{
					Key:         3,
					Name:        "sdc",
					Node:        102,
					Type:        "node",
					NodeDisplay: "node2",
					StatusList:  "another_unknown", // Another unknown state
					PhysicalStatus: struct {
						Bus             string  `json:"bus"`
						Model           string  `json:"model"`
						DriveSize       int64   `json:"drive_size"`
						Fw              string  `json:"fw"`
						Path            string  `json:"path"`
						Serial          string  `json:"phys_serial"`
						VsanTier        int     `json:"vsan_tier"`
						VsanPath        string  `json:"vsan_path"`
						VsanDriveID     int     `json:"vsan_driveid"`
						LocateStatus    string  `json:"locate_status"`
						VsanRepairing   int     `json:"vsan_repairing"`
						VsanReadErrors  int     `json:"vsan_read_errors"`
						VsanWriteErrors int     `json:"vsan_write_errors"`
						Temp            float64 `json:"temp"`
						Location        string  `json:"location"`
						Hours           int     `json:"hours"`
						ReallocSectors  int     `json:"realloc_sectors"`
						WearLevel       int     `json:"wear_level"`
					}{
						VsanTier:      1, // Keep tier 1 for this drive
						VsanRepairing: 0,
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/api/v4/nodes"):
			// Return empty node list for simplicity
			nodes := []struct{}{}
			json.NewEncoder(w).Encode(nodes)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create a registry for testing
	registry := prometheus.NewRegistry()

	// Create a storage collector with the mock server
	sdkClient := createTestSDKClient(mockServer.URL)
	sc := NewStorageCollector(sdkClient, mockServer.URL, "testuser", "testpass")
	registry.MustRegister(sc)

	// Define expected metrics for both tiers
	expected := `
# HELP vergeos_vsan_drive_states Number of drives in each state (online, offline, repairing, initializing, verifying, noredundant, outofspace)
# TYPE vergeos_vsan_drive_states gauge
vergeos_vsan_drive_states{state="initializing",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_states{state="initializing",system_name="test-system",tier="2"} 0
vergeos_vsan_drive_states{state="noredundant",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_states{state="noredundant",system_name="test-system",tier="2"} 0
vergeos_vsan_drive_states{state="offline",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_states{state="offline",system_name="test-system",tier="2"} 0
vergeos_vsan_drive_states{state="online",system_name="test-system",tier="1"} 1
vergeos_vsan_drive_states{state="online",system_name="test-system",tier="2"} 0
vergeos_vsan_drive_states{state="outofspace",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_states{state="outofspace",system_name="test-system",tier="2"} 0
vergeos_vsan_drive_states{state="repairing",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_states{state="repairing",system_name="test-system",tier="2"} 0
vergeos_vsan_drive_states{state="verifying",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_states{state="verifying",system_name="test-system",tier="2"} 0
`

	// Gather metrics
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "vergeos_vsan_drive_states"); err != nil {
		t.Errorf("Metrics do not match expected values: %v", err)
	}
}
