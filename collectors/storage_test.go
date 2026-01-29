package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	vergeos "github.com/verge-io/goVergeOS"
)

// createTestSDKClient creates an SDK client configured for testing with the mock server
// Note: The mock server must handle /version.json for SDK version validation
func createTestSDKClient(t *testing.T, mockServerURL string) *vergeos.Client {
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

func TestStorageTierMetrics(t *testing.T) {
	// Create a mock server to simulate the VergeOS API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle version check first (no auth required by SDK)
		if strings.HasSuffix(r.URL.Path, "/version.json") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":    "v4",
				"version": "26.0.2.1",
				"hash":    "testbuild",
			})
			return
		}

		// Add basic auth check for API endpoints
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Handle different API endpoints
		switch {
		case strings.Contains(r.URL.Path, "/api/v4/settings"):
			// Return system name (using inline struct - SDK handles response)
			settings := []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}{
				{Key: "cloud_name", Value: "test-system"},
			}
			json.NewEncoder(w).Encode(settings)

		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			// Return tier information - SDK compatible format
			tiers := []map[string]interface{}{
				{
					"$key":         0,
					"tier":         0,
					"description":  "SSD Tier",
					"capacity":     uint64(1000000000000), // 1TB
					"used":         uint64(100000000000),  // 100GB
					"allocated":    uint64(500000000000),  // 500GB
					"dedupe_ratio": uint32(200),           // 2.0x
				},
				{
					"$key":         1,
					"tier":         1,
					"description":  "HDD Tier",
					"capacity":     uint64(5000000000000), // 5TB
					"used":         uint64(2000000000000), // 2TB
					"allocated":    uint64(3000000000000), // 3TB
					"dedupe_ratio": uint32(150),           // 1.5x
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
						"progress":              float64(0),
						"cur_space_throttle_ms": float64(0),
					},
				},
				{
					"$key":    2,
					"cluster": 1,
					"tier":    1,
					"status": map[string]interface{}{
						"tier":                  1,
						"status":                "repairing",
						"state":                 "warning",
						"transaction":           uint64(200),
						"repairs":               uint64(10),
						"working":               true,
						"bad_drives":            float64(1),
						"encrypted":             false,
						"redundant":             false,
						"last_walk_time_ms":     uint64(1500),
						"last_fullwalk_time_ms": uint64(7500),
						"fullwalk":              true,
						"progress":              float64(50),
						"cur_space_throttle_ms": float64(100),
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create a registry for testing
	registry := prometheus.NewRegistry()

	// Create a storage collector with the mock server
	sdkClient := createTestSDKClient(t, mockServer.URL)
	sc := NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	// Test tier capacity metrics
	expectedCapacity := `
# HELP vergeos_vsan_tier_capacity VSAN tier capacity in bytes
# TYPE vergeos_vsan_tier_capacity gauge
vergeos_vsan_tier_capacity{description="SSD Tier",system_name="test-system",tier="0"} 1e+12
vergeos_vsan_tier_capacity{description="HDD Tier",system_name="test-system",tier="1"} 5e+12
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedCapacity), "vergeos_vsan_tier_capacity"); err != nil {
		t.Errorf("Capacity metrics do not match expected values: %v", err)
	}

	// Test tier used metrics
	expectedUsed := `
# HELP vergeos_vsan_tier_used VSAN tier used space in bytes
# TYPE vergeos_vsan_tier_used gauge
vergeos_vsan_tier_used{description="SSD Tier",system_name="test-system",tier="0"} 1e+11
vergeos_vsan_tier_used{description="HDD Tier",system_name="test-system",tier="1"} 2e+12
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedUsed), "vergeos_vsan_tier_used"); err != nil {
		t.Errorf("Used metrics do not match expected values: %v", err)
	}

	// Test dedupe ratio
	expectedDedupe := `
# HELP vergeos_vsan_tier_dedupe_ratio VSAN tier deduplication ratio
# TYPE vergeos_vsan_tier_dedupe_ratio gauge
vergeos_vsan_tier_dedupe_ratio{description="SSD Tier",system_name="test-system",tier="0"} 2
vergeos_vsan_tier_dedupe_ratio{description="HDD Tier",system_name="test-system",tier="1"} 1.5
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedDedupe), "vergeos_vsan_tier_dedupe_ratio"); err != nil {
		t.Errorf("Dedupe ratio metrics do not match expected values: %v", err)
	}

	// Test encryption status (tier 0 = encrypted, tier 1 = not encrypted)
	expectedEncryption := `
# HELP vergeos_vsan_encryption_status VSAN tier encryption status (1=encrypted, 0=not encrypted)
# TYPE vergeos_vsan_encryption_status gauge
vergeos_vsan_encryption_status{status="online",system_name="test-system",tier="0"} 1
vergeos_vsan_encryption_status{status="repairing",system_name="test-system",tier="1"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedEncryption), "vergeos_vsan_encryption_status"); err != nil {
		t.Errorf("Encryption status metrics do not match expected values: %v", err)
	}

	// Test redundancy status (tier 0 = redundant, tier 1 = not redundant)
	expectedRedundant := `
# HELP vergeos_vsan_redundant VSAN tier redundancy status (1=redundant, 0=not redundant)
# TYPE vergeos_vsan_redundant gauge
vergeos_vsan_redundant{status="online",system_name="test-system",tier="0"} 1
vergeos_vsan_redundant{status="repairing",system_name="test-system",tier="1"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedRedundant), "vergeos_vsan_redundant"); err != nil {
		t.Errorf("Redundancy status metrics do not match expected values: %v", err)
	}

	// Test bad drives
	expectedBadDrives := `
# HELP vergeos_vsan_bad_drives Number of bad drives in VSAN tier
# TYPE vergeos_vsan_bad_drives gauge
vergeos_vsan_bad_drives{status="online",system_name="test-system",tier="0"} 0
vergeos_vsan_bad_drives{status="repairing",system_name="test-system",tier="1"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedBadDrives), "vergeos_vsan_bad_drives"); err != nil {
		t.Errorf("Bad drives metrics do not match expected values: %v", err)
	}

	// Test fullwalk progress (tier 1 is at 50%)
	expectedProgress := `
# HELP vergeos_vsan_fullwalk_progress VSAN tier fullwalk progress percentage
# TYPE vergeos_vsan_fullwalk_progress gauge
vergeos_vsan_fullwalk_progress{status="online",system_name="test-system",tier="0"} 0
vergeos_vsan_fullwalk_progress{status="repairing",system_name="test-system",tier="1"} 50
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedProgress), "vergeos_vsan_fullwalk_progress"); err != nil {
		t.Errorf("Fullwalk progress metrics do not match expected values: %v", err)
	}
}

// TestPhantomTierFiltering tests Bug #27 - phantom tier filtering
// When cluster_tiers returns a tier that doesn't exist in storage_tiers,
// it should be skipped (not reported as a metric)
func TestPhantomTierFiltering(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle version check first
		if strings.HasSuffix(r.URL.Path, "/version.json") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":    "v4",
				"version": "26.0.2.1",
				"hash":    "testbuild",
			})
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case strings.Contains(r.URL.Path, "/api/v4/settings"):
			json.NewEncoder(w).Encode([]map[string]string{{"key": "cloud_name", "value": "test-system"}})

		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			// Only tier 0 and 3 configured (non-contiguous)
			tiers := []map[string]interface{}{
				{"$key": 0, "tier": 0, "description": "Tier 0", "capacity": uint64(1000000000000), "used": uint64(0), "allocated": uint64(0), "dedupe_ratio": uint32(100)},
				{"$key": 3, "tier": 3, "description": "Tier 3", "capacity": uint64(2000000000000), "used": uint64(0), "allocated": uint64(0), "dedupe_ratio": uint32(100)},
			}
			json.NewEncoder(w).Encode(tiers)

		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			// cluster_tiers returns tier 0, 1, 2, 3 - but 1 and 2 are phantom tiers
			response := []map[string]interface{}{
				{
					"$key": 1, "cluster": 1, "tier": 0,
					"status": map[string]interface{}{
						"tier": 0, "status": "online", "state": "online", "transaction": uint64(100),
						"repairs": uint64(0), "working": true, "bad_drives": float64(0),
						"encrypted": true, "redundant": true, "last_walk_time_ms": uint64(1000),
						"last_fullwalk_time_ms": uint64(5000), "fullwalk": false,
						"progress": float64(0), "cur_space_throttle_ms": float64(0),
					},
				},
				{
					// Phantom tier 1 - should be skipped
					"$key": 2, "cluster": 1, "tier": 1,
					"status": map[string]interface{}{
						"tier": 1, "status": "phantom", "state": "error", "transaction": uint64(999),
						"repairs": uint64(999), "working": false, "bad_drives": float64(999),
						"encrypted": false, "redundant": false, "last_walk_time_ms": uint64(999),
						"last_fullwalk_time_ms": uint64(999), "fullwalk": false,
						"progress": float64(0), "cur_space_throttle_ms": float64(0),
					},
				},
				{
					// Phantom tier 2 - should be skipped
					"$key": 3, "cluster": 1, "tier": 2,
					"status": map[string]interface{}{
						"tier": 2, "status": "phantom", "state": "error", "transaction": uint64(999),
						"repairs": uint64(999), "working": false, "bad_drives": float64(999),
						"encrypted": false, "redundant": false, "last_walk_time_ms": uint64(999),
						"last_fullwalk_time_ms": uint64(999), "fullwalk": false,
						"progress": float64(0), "cur_space_throttle_ms": float64(0),
					},
				},
				{
					"$key": 4, "cluster": 1, "tier": 3,
					"status": map[string]interface{}{
						"tier": 3, "status": "online", "state": "online", "transaction": uint64(200),
						"repairs": uint64(0), "working": true, "bad_drives": float64(0),
						"encrypted": true, "redundant": true, "last_walk_time_ms": uint64(2000),
						"last_fullwalk_time_ms": uint64(10000), "fullwalk": false,
						"progress": float64(0), "cur_space_throttle_ms": float64(0),
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := createTestSDKClient(t, mockServer.URL)
	sc := NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	// Only tier 0 and 3 should be reported (not phantom tiers 1 and 2)
	expectedCapacity := `
# HELP vergeos_vsan_tier_capacity VSAN tier capacity in bytes
# TYPE vergeos_vsan_tier_capacity gauge
vergeos_vsan_tier_capacity{description="Tier 0",system_name="test-system",tier="0"} 1e+12
vergeos_vsan_tier_capacity{description="Tier 3",system_name="test-system",tier="3"} 2e+12
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedCapacity), "vergeos_vsan_tier_capacity"); err != nil {
		t.Errorf("Bug #27: Phantom tier filtering failed. Capacity metrics do not match expected values: %v", err)
	}

	// Verify tier status metrics only include valid tiers (0 and 3)
	expectedRedundant := `
# HELP vergeos_vsan_redundant VSAN tier redundancy status (1=redundant, 0=not redundant)
# TYPE vergeos_vsan_redundant gauge
vergeos_vsan_redundant{status="online",system_name="test-system",tier="0"} 1
vergeos_vsan_redundant{status="online",system_name="test-system",tier="3"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedRedundant), "vergeos_vsan_redundant"); err != nil {
		t.Errorf("Bug #27: Phantom tier filtering failed. Redundancy metrics do not match expected values: %v", err)
	}

	// Verify bad_drives from phantom tiers is NOT reported
	expectedBadDrives := `
# HELP vergeos_vsan_bad_drives Number of bad drives in VSAN tier
# TYPE vergeos_vsan_bad_drives gauge
vergeos_vsan_bad_drives{status="online",system_name="test-system",tier="0"} 0
vergeos_vsan_bad_drives{status="online",system_name="test-system",tier="3"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedBadDrives), "vergeos_vsan_bad_drives"); err != nil {
		t.Errorf("Bug #27: Phantom tier bad_drives leaked. Metrics do not match expected values: %v", err)
	}
}

// TestStaleMetricsFix tests Bug #28 - stale metrics when tier status changes
// With GaugeVec, old label combinations would persist after status changes.
// With MustNewConstMetric pattern, only current state is emitted each scrape.
func TestStaleMetricsFix(t *testing.T) {
	// Use atomic value to dynamically change tier status during test
	currentStatus := "online"
	statusMutex := &sync.Mutex{}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle version check first
		if strings.HasSuffix(r.URL.Path, "/version.json") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":    "v4",
				"version": "26.0.2.1",
				"hash":    "testbuild",
			})
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case strings.Contains(r.URL.Path, "/api/v4/settings"):
			json.NewEncoder(w).Encode([]map[string]string{{"key": "cloud_name", "value": "test-system"}})

		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			tiers := []map[string]interface{}{
				{"$key": 0, "tier": 0, "description": "Test Tier", "capacity": uint64(1000000000000), "used": uint64(0), "allocated": uint64(0), "dedupe_ratio": uint32(100)},
			}
			json.NewEncoder(w).Encode(tiers)

		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			statusMutex.Lock()
			status := currentStatus
			statusMutex.Unlock()

			response := []map[string]interface{}{
				{
					"$key": 1, "cluster": 1, "tier": 0,
					"status": map[string]interface{}{
						"tier": 0, "status": status, "state": status, "transaction": uint64(100),
						"repairs": uint64(0), "working": true, "bad_drives": float64(0),
						"encrypted": true, "redundant": true, "last_walk_time_ms": uint64(1000),
						"last_fullwalk_time_ms": uint64(5000), "fullwalk": false,
						"progress": float64(0), "cur_space_throttle_ms": float64(0),
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := createTestSDKClient(t, mockServer.URL)
	sc := NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	// First gather - status is "online"
	expectedOnline := `
# HELP vergeos_vsan_redundant VSAN tier redundancy status (1=redundant, 0=not redundant)
# TYPE vergeos_vsan_redundant gauge
vergeos_vsan_redundant{status="online",system_name="test-system",tier="0"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedOnline), "vergeos_vsan_redundant"); err != nil {
		t.Errorf("Bug #28 test step 1 failed - expected online status: %v", err)
	}

	// Change status to "repairing"
	statusMutex.Lock()
	currentStatus = "repairing"
	statusMutex.Unlock()

	// Second gather - status should now be ONLY "repairing", NOT "online"
	// With the old GaugeVec approach, this would fail because "online" would still be present
	expectedRepairing := `
# HELP vergeos_vsan_redundant VSAN tier redundancy status (1=redundant, 0=not redundant)
# TYPE vergeos_vsan_redundant gauge
vergeos_vsan_redundant{status="repairing",system_name="test-system",tier="0"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedRepairing), "vergeos_vsan_redundant"); err != nil {
		t.Errorf("Bug #28 test step 2 failed - expected only 'repairing' status (no stale 'online'): %v", err)
	}

	// Verify "online" is completely gone by checking gathered metrics directly
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	for _, mf := range metrics {
		if *mf.Name == "vergeos_vsan_redundant" {
			for _, m := range mf.Metric {
				for _, lp := range m.Label {
					if *lp.Name == "status" && *lp.Value == "online" {
						t.Errorf("Bug #28: Stale metric found! 'online' status should not persist after changing to 'repairing'")
					}
				}
			}
		}
	}
}

// TestNoTiersConfigured tests behavior when no storage tiers are configured
func TestNoTiersConfigured(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle version check first
		if strings.HasSuffix(r.URL.Path, "/version.json") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":    "v4",
				"version": "26.0.2.1",
				"hash":    "testbuild",
			})
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case strings.Contains(r.URL.Path, "/api/v4/settings"):
			json.NewEncoder(w).Encode([]map[string]string{{"key": "cloud_name", "value": "test-system"}})

		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			// No tiers configured
			json.NewEncoder(w).Encode([]map[string]interface{}{})

		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			// No cluster tiers
			json.NewEncoder(w).Encode([]map[string]interface{}{})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := createTestSDKClient(t, mockServer.URL)
	sc := NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	// No metrics should be emitted for capacity when no tiers exist
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check that no capacity metrics are present
	for _, mf := range metrics {
		if *mf.Name == "vergeos_vsan_tier_capacity" {
			if len(mf.Metric) > 0 {
				t.Errorf("Expected no capacity metrics when no tiers configured, but got %d", len(mf.Metric))
			}
		}
	}
}
