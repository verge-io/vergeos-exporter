package tests

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"vergeos-exporter/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestStorageTierMetrics(t *testing.T) {
	config := DefaultMockConfig()

	storageTiers := []StorageTierMock{
		{Key: 0, Tier: 0, Description: "SSD Tier", Capacity: 1000000000000, Used: 100000000000, Allocated: 500000000000, DedupeRatio: 200},
		{Key: 1, Tier: 1, Description: "HDD Tier", Capacity: 5000000000000, Used: 2000000000000, Allocated: 3000000000000, DedupeRatio: 150},
	}

	clusterTiers := []ClusterTierMock{
		{
			Key: 1, Cluster: 1, Tier: 0,
			Status: ClusterTierStatusMock{
				Tier: 0, Status: "online", State: "online", Transaction: 100, Repairs: 5,
				Working: true, BadDrives: 0, Encrypted: true, Redundant: true,
				LastWalkTimeMs: 1000, LastFullwalkTimeMs: 5000, Fullwalk: false,
				Progress: 0, CurSpaceThrottleMs: 0,
			},
		},
		{
			Key: 2, Cluster: 1, Tier: 1,
			Status: ClusterTierStatusMock{
				Tier: 1, Status: "repairing", State: "warning", Transaction: 200, Repairs: 10,
				Working: true, BadDrives: 1, Encrypted: false, Redundant: false,
				LastWalkTimeMs: 1500, LastFullwalkTimeMs: 7500, Fullwalk: true,
				Progress: 50, CurSpaceThrottleMs: 100,
			},
		},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			WriteJSONResponse(w, storageTiers)
			return true
		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			WriteJSONResponse(w, clusterTiers)
			return true
		}
		return false
	})
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := CreateTestSDKClient(t, mockServer.URL)
	sc := collectors.NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	// Test tier capacity metrics
	expectedCapacity := `
# HELP vergeos_vsan_tier_capacity VSAN tier capacity in bytes
# TYPE vergeos_vsan_tier_capacity gauge
vergeos_vsan_tier_capacity{description="SSD Tier",system_name="testcloud",tier="0"} 1e+12
vergeos_vsan_tier_capacity{description="HDD Tier",system_name="testcloud",tier="1"} 5e+12
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedCapacity), "vergeos_vsan_tier_capacity"); err != nil {
		t.Errorf("Capacity metrics do not match expected values: %v", err)
	}

	// Test tier used metrics
	expectedUsed := `
# HELP vergeos_vsan_tier_used VSAN tier used space in bytes
# TYPE vergeos_vsan_tier_used gauge
vergeos_vsan_tier_used{description="SSD Tier",system_name="testcloud",tier="0"} 1e+11
vergeos_vsan_tier_used{description="HDD Tier",system_name="testcloud",tier="1"} 2e+12
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedUsed), "vergeos_vsan_tier_used"); err != nil {
		t.Errorf("Used metrics do not match expected values: %v", err)
	}

	// Test dedupe ratio
	expectedDedupe := `
# HELP vergeos_vsan_tier_dedupe_ratio VSAN tier deduplication ratio
# TYPE vergeos_vsan_tier_dedupe_ratio gauge
vergeos_vsan_tier_dedupe_ratio{description="SSD Tier",system_name="testcloud",tier="0"} 2
vergeos_vsan_tier_dedupe_ratio{description="HDD Tier",system_name="testcloud",tier="1"} 1.5
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedDedupe), "vergeos_vsan_tier_dedupe_ratio"); err != nil {
		t.Errorf("Dedupe ratio metrics do not match expected values: %v", err)
	}

	// Test encryption status (tier 0 = encrypted, tier 1 = not encrypted)
	expectedEncryption := `
# HELP vergeos_vsan_encryption_status VSAN tier encryption status (1=encrypted, 0=not encrypted)
# TYPE vergeos_vsan_encryption_status gauge
vergeos_vsan_encryption_status{status="online",system_name="testcloud",tier="0"} 1
vergeos_vsan_encryption_status{status="repairing",system_name="testcloud",tier="1"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedEncryption), "vergeos_vsan_encryption_status"); err != nil {
		t.Errorf("Encryption status metrics do not match expected values: %v", err)
	}

	// Test redundancy status (tier 0 = redundant, tier 1 = not redundant)
	expectedRedundant := `
# HELP vergeos_vsan_redundant VSAN tier redundancy status (1=redundant, 0=not redundant)
# TYPE vergeos_vsan_redundant gauge
vergeos_vsan_redundant{status="online",system_name="testcloud",tier="0"} 1
vergeos_vsan_redundant{status="repairing",system_name="testcloud",tier="1"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedRedundant), "vergeos_vsan_redundant"); err != nil {
		t.Errorf("Redundancy status metrics do not match expected values: %v", err)
	}

	// Test bad drives
	expectedBadDrives := `
# HELP vergeos_vsan_bad_drives Number of bad drives in VSAN tier
# TYPE vergeos_vsan_bad_drives gauge
vergeos_vsan_bad_drives{status="online",system_name="testcloud",tier="0"} 0
vergeos_vsan_bad_drives{status="repairing",system_name="testcloud",tier="1"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedBadDrives), "vergeos_vsan_bad_drives"); err != nil {
		t.Errorf("Bad drives metrics do not match expected values: %v", err)
	}

	// Test fullwalk progress (tier 1 is at 50%)
	expectedProgress := `
# HELP vergeos_vsan_fullwalk_progress VSAN tier fullwalk progress percentage
# TYPE vergeos_vsan_fullwalk_progress gauge
vergeos_vsan_fullwalk_progress{status="online",system_name="testcloud",tier="0"} 0
vergeos_vsan_fullwalk_progress{status="repairing",system_name="testcloud",tier="1"} 50
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedProgress), "vergeos_vsan_fullwalk_progress"); err != nil {
		t.Errorf("Fullwalk progress metrics do not match expected values: %v", err)
	}
}

// TestPhantomTierFiltering tests Bug #27 - phantom tier filtering
// When cluster_tiers returns a tier that doesn't exist in storage_tiers,
// it should be skipped (not reported as a metric)
func TestPhantomTierFiltering(t *testing.T) {
	config := DefaultMockConfig()

	// Only tier 0 and 3 configured (non-contiguous)
	storageTiers := []StorageTierMock{
		{Key: 0, Tier: 0, Description: "Tier 0", Capacity: 1000000000000, Used: 0, Allocated: 0, DedupeRatio: 100},
		{Key: 3, Tier: 3, Description: "Tier 3", Capacity: 2000000000000, Used: 0, Allocated: 0, DedupeRatio: 100},
	}

	// cluster_tiers returns tier 0, 1, 2, 3 - but 1 and 2 are phantom tiers
	clusterTiers := []ClusterTierMock{
		{
			Key: 1, Cluster: 1, Tier: 0,
			Status: ClusterTierStatusMock{
				Tier: 0, Status: "online", State: "online", Transaction: 100,
				Repairs: 0, Working: true, BadDrives: 0, Encrypted: true, Redundant: true,
				LastWalkTimeMs: 1000, LastFullwalkTimeMs: 5000, Fullwalk: false, Progress: 0, CurSpaceThrottleMs: 0,
			},
		},
		{
			// Phantom tier 1 - should be skipped
			Key: 2, Cluster: 1, Tier: 1,
			Status: ClusterTierStatusMock{
				Tier: 1, Status: "phantom", State: "error", Transaction: 999,
				Repairs: 999, Working: false, BadDrives: 999, Encrypted: false, Redundant: false,
				LastWalkTimeMs: 999, LastFullwalkTimeMs: 999, Fullwalk: false, Progress: 0, CurSpaceThrottleMs: 0,
			},
		},
		{
			// Phantom tier 2 - should be skipped
			Key: 3, Cluster: 1, Tier: 2,
			Status: ClusterTierStatusMock{
				Tier: 2, Status: "phantom", State: "error", Transaction: 999,
				Repairs: 999, Working: false, BadDrives: 999, Encrypted: false, Redundant: false,
				LastWalkTimeMs: 999, LastFullwalkTimeMs: 999, Fullwalk: false, Progress: 0, CurSpaceThrottleMs: 0,
			},
		},
		{
			Key: 4, Cluster: 1, Tier: 3,
			Status: ClusterTierStatusMock{
				Tier: 3, Status: "online", State: "online", Transaction: 200,
				Repairs: 0, Working: true, BadDrives: 0, Encrypted: true, Redundant: true,
				LastWalkTimeMs: 2000, LastFullwalkTimeMs: 10000, Fullwalk: false, Progress: 0, CurSpaceThrottleMs: 0,
			},
		},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			WriteJSONResponse(w, storageTiers)
			return true
		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			WriteJSONResponse(w, clusterTiers)
			return true
		}
		return false
	})
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := CreateTestSDKClient(t, mockServer.URL)
	sc := collectors.NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	// Only tier 0 and 3 should be reported (not phantom tiers 1 and 2)
	expectedCapacity := `
# HELP vergeos_vsan_tier_capacity VSAN tier capacity in bytes
# TYPE vergeos_vsan_tier_capacity gauge
vergeos_vsan_tier_capacity{description="Tier 0",system_name="testcloud",tier="0"} 1e+12
vergeos_vsan_tier_capacity{description="Tier 3",system_name="testcloud",tier="3"} 2e+12
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedCapacity), "vergeos_vsan_tier_capacity"); err != nil {
		t.Errorf("Bug #27: Phantom tier filtering failed. Capacity metrics do not match expected values: %v", err)
	}

	// Verify tier status metrics only include valid tiers (0 and 3)
	expectedRedundant := `
# HELP vergeos_vsan_redundant VSAN tier redundancy status (1=redundant, 0=not redundant)
# TYPE vergeos_vsan_redundant gauge
vergeos_vsan_redundant{status="online",system_name="testcloud",tier="0"} 1
vergeos_vsan_redundant{status="online",system_name="testcloud",tier="3"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedRedundant), "vergeos_vsan_redundant"); err != nil {
		t.Errorf("Bug #27: Phantom tier filtering failed. Redundancy metrics do not match expected values: %v", err)
	}

	// Verify bad_drives from phantom tiers is NOT reported
	expectedBadDrives := `
# HELP vergeos_vsan_bad_drives Number of bad drives in VSAN tier
# TYPE vergeos_vsan_bad_drives gauge
vergeos_vsan_bad_drives{status="online",system_name="testcloud",tier="0"} 0
vergeos_vsan_bad_drives{status="online",system_name="testcloud",tier="3"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedBadDrives), "vergeos_vsan_bad_drives"); err != nil {
		t.Errorf("Bug #27: Phantom tier bad_drives leaked. Metrics do not match expected values: %v", err)
	}
}

// TestStaleMetricsFix tests Bug #28 - stale metrics when tier status changes
// With GaugeVec, old label combinations would persist after status changes.
// With MustNewConstMetric pattern, only current state is emitted each scrape.
func TestStaleMetricsFix(t *testing.T) {
	config := DefaultMockConfig()

	// Use mutex to dynamically change tier status during test
	currentStatus := "online"
	statusMutex := &sync.Mutex{}

	storageTiers := []StorageTierMock{
		{Key: 0, Tier: 0, Description: "Test Tier", Capacity: 1000000000000, Used: 0, Allocated: 0, DedupeRatio: 100},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			WriteJSONResponse(w, storageTiers)
			return true
		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			statusMutex.Lock()
			status := currentStatus
			statusMutex.Unlock()

			clusterTiers := []ClusterTierMock{
				{
					Key: 1, Cluster: 1, Tier: 0,
					Status: ClusterTierStatusMock{
						Tier: 0, Status: status, State: status, Transaction: 100,
						Repairs: 0, Working: true, BadDrives: 0, Encrypted: true, Redundant: true,
						LastWalkTimeMs: 1000, LastFullwalkTimeMs: 5000, Fullwalk: false, Progress: 0, CurSpaceThrottleMs: 0,
					},
				},
			}
			WriteJSONResponse(w, clusterTiers)
			return true
		}
		return false
	})
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := CreateTestSDKClient(t, mockServer.URL)
	sc := collectors.NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	// First gather - status is "online"
	expectedOnline := `
# HELP vergeos_vsan_redundant VSAN tier redundancy status (1=redundant, 0=not redundant)
# TYPE vergeos_vsan_redundant gauge
vergeos_vsan_redundant{status="online",system_name="testcloud",tier="0"} 1
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
vergeos_vsan_redundant{status="repairing",system_name="testcloud",tier="0"} 1
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
	config := DefaultMockConfig()

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			WriteJSONResponse(w, []StorageTierMock{})
			return true
		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			WriteJSONResponse(w, []ClusterTierMock{})
			return true
		}
		return false
	})
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := CreateTestSDKClient(t, mockServer.URL)
	sc := collectors.NewStorageCollector(sdkClient)
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
