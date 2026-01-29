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
			// Return tier information - SDK compatible format
			// Using tiers 0 and 1 (valid in test environments)
			tiers := []map[string]interface{}{
				{
					"$key":         0,
					"tier":         0,
					"description":  "SSD Tier",
					"capacity":     uint64(1000000000000),
					"used":         uint64(100000000000),
					"allocated":    uint64(500000000000),
					"dedupe_ratio": uint32(200),
				},
				{
					"$key":         1,
					"tier":         1,
					"description":  "HDD Tier",
					"capacity":     uint64(5000000000000),
					"used":         uint64(2000000000000),
					"allocated":    uint64(3000000000000),
					"dedupe_ratio": uint32(150),
				},
			}
			json.NewEncoder(w).Encode(tiers)

		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			// Return tier details - SDK compatible format
			response := []map[string]interface{}{
				{
					"$key":    1,
					"cluster": 1,
					"tier":    0,
					"status": map[string]interface{}{
						"tier":                  0,
						"status":                "online",
						"state":                 "online",
						"transaction":           uint64(100),
						"repairs":               uint64(5),
						"working":               true,
						"bad_drives":            float64(0),
						"encrypted":             true,
						"redundant":             true,
						"last_walk_time_ms":     uint64(1000),
						"last_fullwalk_time_ms": uint64(5000),
						"fullwalk":              false,
						"progress":              float64(100),
						"cur_space_throttle_ms": float64(0),
					},
				},
				{
					"$key":    2,
					"cluster": 1,
					"tier":    1,
					"status": map[string]interface{}{
						"tier":                  1,
						"status":                "online",
						"state":                 "online",
						"transaction":           uint64(200),
						"repairs":               uint64(10),
						"working":               true,
						"bad_drives":            float64(0),
						"encrypted":             true,
						"redundant":             true,
						"last_walk_time_ms":     uint64(1500),
						"last_fullwalk_time_ms": uint64(7500),
						"fullwalk":              false,
						"progress":              float64(100),
						"cur_space_throttle_ms": float64(0),
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/api/v4/machine_drives"):
			// Return drive information - use JSON maps with correct field names
			// collectDriveStateMetrics uses: vsan_tier, vsan_repairing, statuslist, node_display
			// Tier 0: node1 (3 online), node2 (1 offline, 1 initializing, 1 verifying), node3 (1 repairing)
			// Tier 1: node1 (2 online), node2 (2 online), node3 (1 noredundant, 1 outofspace)
			response := []map[string]interface{}{
				// Tier 0 drives
				{"$key": 1, "name": "sda", "node": 101, "node_display": "node1", "statuslist": "online", "vsan_tier": 0},
				{"$key": 2, "name": "sdb", "node": 101, "node_display": "node1", "statuslist": "online", "vsan_tier": 0},
				{"$key": 3, "name": "sdc", "node": 101, "node_display": "node1", "statuslist": "online", "vsan_tier": 0},
				{"$key": 4, "name": "sdd", "node": 102, "node_display": "node2", "statuslist": "offline", "vsan_tier": 0},
				{"$key": 5, "name": "sde", "node": 102, "node_display": "node2", "statuslist": "initializing", "vsan_tier": 0},
				{"$key": 6, "name": "sdf", "node": 102, "node_display": "node2", "statuslist": "verifying", "vsan_tier": 0},
				{"$key": 7, "name": "sdg", "node": 103, "node_display": "node3", "statuslist": "online", "vsan_tier": 0, "vsan_repairing": 1}, // repairing overrides online
				// Tier 1 drives
				{"$key": 8, "name": "sdh", "node": 101, "node_display": "node1", "statuslist": "online", "vsan_tier": 1},
				{"$key": 9, "name": "sdi", "node": 101, "node_display": "node1", "statuslist": "online", "vsan_tier": 1},
				{"$key": 10, "name": "sdj", "node": 102, "node_display": "node2", "statuslist": "online", "vsan_tier": 1},
				{"$key": 11, "name": "sdk", "node": 102, "node_display": "node2", "statuslist": "online", "vsan_tier": 1},
				{"$key": 12, "name": "sdl", "node": 103, "node_display": "node3", "statuslist": "noredundant", "vsan_tier": 1},
				{"$key": 13, "name": "sdm", "node": 103, "node_display": "node3", "statuslist": "outofspace", "vsan_tier": 1},
			}
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/api/v4/nodes"):
			// Return empty node list - we're testing collectDriveStateMetrics, not drive metrics from nodes
			json.NewEncoder(w).Encode([]struct{}{})

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

	// Test online counts
	// Tier 0: node1=3, node2=0, node3=0 (node3 has 1 repairing which overrides online)
	// Tier 1: node1=2, node2=2, node3=0
	expectedOnline := `
# HELP vergeos_vsan_drive_online_count Number of drives in the 'online' state per node and tier
# TYPE vergeos_vsan_drive_online_count gauge
vergeos_vsan_drive_online_count{node_name="node1",system_name="test-system",tier="0"} 3
vergeos_vsan_drive_online_count{node_name="node1",system_name="test-system",tier="1"} 2
vergeos_vsan_drive_online_count{node_name="node2",system_name="test-system",tier="0"} 0
vergeos_vsan_drive_online_count{node_name="node2",system_name="test-system",tier="1"} 2
vergeos_vsan_drive_online_count{node_name="node3",system_name="test-system",tier="0"} 0
vergeos_vsan_drive_online_count{node_name="node3",system_name="test-system",tier="1"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedOnline), "vergeos_vsan_drive_online_count"); err != nil {
		t.Errorf("Online count metrics do not match expected values: %v", err)
	}

	// Test offline counts
	// Tier 0: node1=0, node2=1, node3=0
	// Tier 1: all 0
	expectedOffline := `
# HELP vergeos_vsan_drive_offline_count Number of drives in the 'offline' state per node and tier
# TYPE vergeos_vsan_drive_offline_count gauge
vergeos_vsan_drive_offline_count{node_name="node1",system_name="test-system",tier="0"} 0
vergeos_vsan_drive_offline_count{node_name="node1",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_offline_count{node_name="node2",system_name="test-system",tier="0"} 1
vergeos_vsan_drive_offline_count{node_name="node2",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_offline_count{node_name="node3",system_name="test-system",tier="0"} 0
vergeos_vsan_drive_offline_count{node_name="node3",system_name="test-system",tier="1"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedOffline), "vergeos_vsan_drive_offline_count"); err != nil {
		t.Errorf("Offline count metrics do not match expected values: %v", err)
	}

	// Test repairing counts
	// Tier 0: node3=1 (vsan_repairing overrides statuslist)
	expectedRepairing := `
# HELP vergeos_vsan_drive_repairing_count Number of drives in the 'repairing' state per node and tier
# TYPE vergeos_vsan_drive_repairing_count gauge
vergeos_vsan_drive_repairing_count{node_name="node1",system_name="test-system",tier="0"} 0
vergeos_vsan_drive_repairing_count{node_name="node1",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_repairing_count{node_name="node2",system_name="test-system",tier="0"} 0
vergeos_vsan_drive_repairing_count{node_name="node2",system_name="test-system",tier="1"} 0
vergeos_vsan_drive_repairing_count{node_name="node3",system_name="test-system",tier="0"} 1
vergeos_vsan_drive_repairing_count{node_name="node3",system_name="test-system",tier="1"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedRepairing), "vergeos_vsan_drive_repairing_count"); err != nil {
		t.Errorf("Repairing count metrics do not match expected values: %v", err)
	}
}

func TestDriveStateEdgeCases(t *testing.T) {
	// Create a mock server to simulate edge cases
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case strings.Contains(r.URL.Path, "/api/v4/settings"):
			settings := []Setting{{Key: "cloud_name", Value: "test-system"}}
			json.NewEncoder(w).Encode(settings)

		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			// Only tier 1 configured
			tiers := []map[string]interface{}{
				{"$key": 1, "tier": 1, "description": "Test Tier", "capacity": uint64(1000000000000), "used": uint64(0), "allocated": uint64(0), "dedupe_ratio": uint32(100)},
			}
			json.NewEncoder(w).Encode(tiers)

		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			response := []map[string]interface{}{
				{
					"$key": 1, "cluster": 1, "tier": 1,
					"status": map[string]interface{}{
						"tier": 1, "status": "online", "state": "online", "transaction": uint64(100),
						"repairs": uint64(0), "working": true, "bad_drives": float64(0),
						"encrypted": true, "redundant": true, "last_walk_time_ms": uint64(1000),
						"last_fullwalk_time_ms": uint64(5000), "fullwalk": false,
						"progress": float64(100), "cur_space_throttle_ms": float64(0),
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/api/v4/machine_drives"):
			// Test edge cases: one online drive, two with unknown states (should be ignored)
			response := []map[string]interface{}{
				{"$key": 1, "name": "sda", "node": 101, "node_display": "node1", "statuslist": "online", "vsan_tier": 1},
				{"$key": 2, "name": "sdb", "node": 101, "node_display": "node1", "statuslist": "unknown_state", "vsan_tier": 1},
				{"$key": 3, "name": "sdc", "node": 102, "node_display": "node2", "statuslist": "another_unknown", "vsan_tier": 1},
			}
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/api/v4/nodes"):
			json.NewEncoder(w).Encode([]struct{}{})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := createTestSDKClient(mockServer.URL)
	sc := NewStorageCollector(sdkClient, mockServer.URL, "testuser", "testpass")
	registry.MustRegister(sc)

	// With unknown states, only node1 should have 1 online drive
	// Unknown states don't increment any counter
	expectedOnline := `
# HELP vergeos_vsan_drive_online_count Number of drives in the 'online' state per node and tier
# TYPE vergeos_vsan_drive_online_count gauge
vergeos_vsan_drive_online_count{node_name="node1",system_name="test-system",tier="1"} 1
vergeos_vsan_drive_online_count{node_name="node2",system_name="test-system",tier="1"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedOnline), "vergeos_vsan_drive_online_count"); err != nil {
		t.Errorf("Edge case metrics do not match expected values: %v", err)
	}
}
