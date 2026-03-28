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
			NodesOnline: &ClusterTierNodesOnlineMock{
				Nodes: []ClusterTierNodeStateMock{
					{State: "online"},
					{State: "online"},
				},
			},
			DrivesOnline: []ClusterTierDriveStateMock{
				{State: "online"},
				{State: "online"},
				{State: "online"},
				{State: "online"},
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
			NodesOnline: &ClusterTierNodesOnlineMock{
				Nodes: []ClusterTierNodeStateMock{
					{State: "online"},
				},
			},
			DrivesOnline: []ClusterTierDriveStateMock{
				{State: "online"},
				{State: "online"},
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
		case strings.Contains(r.URL.Path, "/api/v4/machine_drive_phys"):
			WriteJSONResponse(w, []MachineDrivePhysMock{})
			return true
		case strings.Contains(r.URL.Path, "/api/v4/machine_drive_stats"):
			WriteJSONResponse(w, []MachineDriveStatsMock{})
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

	// Test redundancy status
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

	// Test fullwalk progress
	expectedProgress := `
# HELP vergeos_vsan_fullwalk_progress VSAN tier fullwalk progress percentage
# TYPE vergeos_vsan_fullwalk_progress gauge
vergeos_vsan_fullwalk_progress{status="online",system_name="testcloud",tier="0"} 0
vergeos_vsan_fullwalk_progress{status="repairing",system_name="testcloud",tier="1"} 50
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedProgress), "vergeos_vsan_fullwalk_progress"); err != nil {
		t.Errorf("Fullwalk progress metrics do not match expected values: %v", err)
	}

	// Test nodes online (Issue 6)
	expectedNodesOnline := `
# HELP vergeos_vsan_nodes_online Count of online nodes for VSAN tier
# TYPE vergeos_vsan_nodes_online gauge
vergeos_vsan_nodes_online{status="online",system_name="testcloud",tier="0"} 2
vergeos_vsan_nodes_online{status="repairing",system_name="testcloud",tier="1"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedNodesOnline), "vergeos_vsan_nodes_online"); err != nil {
		t.Errorf("Nodes online metrics do not match expected values: %v", err)
	}

	// Test drives online (Issue 6)
	expectedDrivesOnline := `
# HELP vergeos_vsan_drives_online Count of online drives for VSAN tier
# TYPE vergeos_vsan_drives_online gauge
vergeos_vsan_drives_online{status="online",system_name="testcloud",tier="0"} 4
vergeos_vsan_drives_online{status="repairing",system_name="testcloud",tier="1"} 2
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedDrivesOnline), "vergeos_vsan_drives_online"); err != nil {
		t.Errorf("Drives online metrics do not match expected values: %v", err)
	}
}

func TestPhantomTierFiltering(t *testing.T) {
	config := DefaultMockConfig()

	storageTiers := []StorageTierMock{
		{Key: 0, Tier: 0, Description: "Tier 0", Capacity: 1000000000000, Used: 0, Allocated: 0, DedupeRatio: 100},
		{Key: 3, Tier: 3, Description: "Tier 3", Capacity: 2000000000000, Used: 0, Allocated: 0, DedupeRatio: 100},
	}

	clusterTiers := []ClusterTierMock{
		{Key: 1, Cluster: 1, Tier: 0, Status: ClusterTierStatusMock{Tier: 0, Status: "online", State: "online", Transaction: 100, Working: true, Encrypted: true, Redundant: true, LastWalkTimeMs: 1000, LastFullwalkTimeMs: 5000}},
		{Key: 2, Cluster: 1, Tier: 1, Status: ClusterTierStatusMock{Tier: 1, Status: "phantom", State: "error", Transaction: 999, Repairs: 999, BadDrives: 999}},
		{Key: 3, Cluster: 1, Tier: 2, Status: ClusterTierStatusMock{Tier: 2, Status: "phantom", State: "error", Transaction: 999, Repairs: 999, BadDrives: 999}},
		{Key: 4, Cluster: 1, Tier: 3, Status: ClusterTierStatusMock{Tier: 3, Status: "online", State: "online", Transaction: 200, Working: true, Encrypted: true, Redundant: true, LastWalkTimeMs: 2000, LastFullwalkTimeMs: 10000}},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			WriteJSONResponse(w, storageTiers)
			return true
		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			WriteJSONResponse(w, clusterTiers)
			return true
		case strings.Contains(r.URL.Path, "/api/v4/machine_drive_phys"):
			WriteJSONResponse(w, []MachineDrivePhysMock{})
			return true
		case strings.Contains(r.URL.Path, "/api/v4/machine_drive_stats"):
			WriteJSONResponse(w, []MachineDriveStatsMock{})
			return true
		}
		return false
	})
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := CreateTestSDKClient(t, mockServer.URL)
	sc := collectors.NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	expectedCapacity := `
# HELP vergeos_vsan_tier_capacity VSAN tier capacity in bytes
# TYPE vergeos_vsan_tier_capacity gauge
vergeos_vsan_tier_capacity{description="Tier 0",system_name="testcloud",tier="0"} 1e+12
vergeos_vsan_tier_capacity{description="Tier 3",system_name="testcloud",tier="3"} 2e+12
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedCapacity), "vergeos_vsan_tier_capacity"); err != nil {
		t.Errorf("Bug #27: Phantom tier filtering failed: %v", err)
	}

	expectedRedundant := `
# HELP vergeos_vsan_redundant VSAN tier redundancy status (1=redundant, 0=not redundant)
# TYPE vergeos_vsan_redundant gauge
vergeos_vsan_redundant{status="online",system_name="testcloud",tier="0"} 1
vergeos_vsan_redundant{status="online",system_name="testcloud",tier="3"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedRedundant), "vergeos_vsan_redundant"); err != nil {
		t.Errorf("Bug #27: Phantom tier filtering failed: %v", err)
	}

	expectedBadDrives := `
# HELP vergeos_vsan_bad_drives Number of bad drives in VSAN tier
# TYPE vergeos_vsan_bad_drives gauge
vergeos_vsan_bad_drives{status="online",system_name="testcloud",tier="0"} 0
vergeos_vsan_bad_drives{status="online",system_name="testcloud",tier="3"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedBadDrives), "vergeos_vsan_bad_drives"); err != nil {
		t.Errorf("Bug #27: Phantom tier bad_drives leaked: %v", err)
	}
}

func TestStaleMetricsFix(t *testing.T) {
	config := DefaultMockConfig()

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
				{Key: 1, Cluster: 1, Tier: 0, Status: ClusterTierStatusMock{Tier: 0, Status: status, State: status, Transaction: 100, Working: true, Encrypted: true, Redundant: true, LastWalkTimeMs: 1000, LastFullwalkTimeMs: 5000}},
			}
			WriteJSONResponse(w, clusterTiers)
			return true
		case strings.Contains(r.URL.Path, "/api/v4/machine_drive_phys"):
			WriteJSONResponse(w, []MachineDrivePhysMock{})
			return true
		case strings.Contains(r.URL.Path, "/api/v4/machine_drive_stats"):
			WriteJSONResponse(w, []MachineDriveStatsMock{})
			return true
		}
		return false
	})
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := CreateTestSDKClient(t, mockServer.URL)
	sc := collectors.NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	expectedOnline := `
# HELP vergeos_vsan_redundant VSAN tier redundancy status (1=redundant, 0=not redundant)
# TYPE vergeos_vsan_redundant gauge
vergeos_vsan_redundant{status="online",system_name="testcloud",tier="0"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedOnline), "vergeos_vsan_redundant"); err != nil {
		t.Errorf("Bug #28 test step 1 failed: %v", err)
	}

	statusMutex.Lock()
	currentStatus = "repairing"
	statusMutex.Unlock()

	expectedRepairing := `
# HELP vergeos_vsan_redundant VSAN tier redundancy status (1=redundant, 0=not redundant)
# TYPE vergeos_vsan_redundant gauge
vergeos_vsan_redundant{status="repairing",system_name="testcloud",tier="0"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expectedRepairing), "vergeos_vsan_redundant"); err != nil {
		t.Errorf("Bug #28 test step 2 failed - expected only 'repairing' status (no stale 'online'): %v", err)
	}

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
		case strings.Contains(r.URL.Path, "/api/v4/machine_drive_phys"):
			WriteJSONResponse(w, []MachineDrivePhysMock{})
			return true
		case strings.Contains(r.URL.Path, "/api/v4/machine_drive_stats"):
			WriteJSONResponse(w, []MachineDriveStatsMock{})
			return true
		}
		return false
	})
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := CreateTestSDKClient(t, mockServer.URL)
	sc := collectors.NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	for _, mf := range metrics {
		if *mf.Name == "vergeos_vsan_tier_capacity" {
			if len(mf.Metric) > 0 {
				t.Errorf("Expected no capacity metrics when no tiers configured, but got %d", len(mf.Metric))
			}
		}
	}
}

func TestDriveMetrics(t *testing.T) {
	config := DefaultMockConfig()

	storageTiers := []StorageTierMock{
		{Key: 0, Tier: 0, Description: "SSD", Capacity: 1000000000000, Used: 0, DedupeRatio: 100},
	}

	clusterTiers := []ClusterTierMock{
		{Key: 1, Cluster: 1, Tier: 0, Status: ClusterTierStatusMock{Tier: 0, Status: "online", State: "online", Working: true, Redundant: true, Encrypted: true}},
	}

	drives := []MachineDrivePhysMock{
		{Key: 1, ParentDrive: 10, Path: "/dev/sda", Serial: "WD-001", Temp: 38, WearLevel: 5, Hours: 10000, ReallocSectors: 0, VSANTier: 0, VSANReadErrors: 2, VSANWriteErrors: 1, VSANRepairing: 0, VSANThrottle: 0, NodeDisplay: "node1", StatusList: "online"},
		{Key: 2, ParentDrive: 20, Path: "/dev/sdb", Serial: "WD-002", Temp: 42, WearLevel: 10, Hours: 20000, ReallocSectors: 3, VSANTier: 0, VSANReadErrors: 0, VSANWriteErrors: 0, VSANRepairing: 100, VSANThrottle: 5000, NodeDisplay: "node1", StatusList: "repairing"},
	}

	driveStats := []MachineDriveStatsMock{
		{Key: 1, ParentDrive: 10, Reads: 50000, Writes: 30000, ReadBytes: 500000000, WriteBytes: 300000000, ServiceTime: 0.5, Util: 25.0, Physical: true},
		{Key: 2, ParentDrive: 20, Reads: 10000, Writes: 5000, ReadBytes: 100000000, WriteBytes: 50000000, ServiceTime: 1.2, Util: 60.0, Physical: true},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/api/v4/storage_tiers"):
			WriteJSONResponse(w, storageTiers)
			return true
		case strings.Contains(r.URL.Path, "/api/v4/cluster_tiers"):
			WriteJSONResponse(w, clusterTiers)
			return true
		case strings.Contains(r.URL.Path, "/api/v4/machine_drive_phys"):
			WriteJSONResponse(w, drives)
			return true
		case strings.Contains(r.URL.Path, "/api/v4/machine_drive_stats"):
			WriteJSONResponse(w, driveStats)
			return true
		}
		return false
	})
	defer mockServer.Close()

	registry := prometheus.NewRegistry()
	sdkClient := CreateTestSDKClient(t, mockServer.URL)
	sc := collectors.NewStorageCollector(sdkClient)
	registry.MustRegister(sc)

	t.Run("drive_temperature", func(t *testing.T) {
		expected := `
# HELP vergeos_drive_temperature Drive temperature in Celsius
# TYPE vergeos_drive_temperature gauge
vergeos_drive_temperature{drive_name="/dev/sda",node_name="node1",serial="WD-001",system_name="testcloud",tier="0"} 38
vergeos_drive_temperature{drive_name="/dev/sdb",node_name="node1",serial="WD-002",system_name="testcloud",tier="0"} 42
`
		if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "vergeos_drive_temperature"); err != nil {
			t.Errorf("Drive temperature metrics mismatch: %v", err)
		}
	})

	t.Run("drive_wear_level", func(t *testing.T) {
		expected := `
# HELP vergeos_drive_wear_level Drive wear level percentage
# TYPE vergeos_drive_wear_level counter
vergeos_drive_wear_level{drive_name="/dev/sda",node_name="node1",serial="WD-001",system_name="testcloud",tier="0"} 5
vergeos_drive_wear_level{drive_name="/dev/sdb",node_name="node1",serial="WD-002",system_name="testcloud",tier="0"} 10
`
		if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "vergeos_drive_wear_level"); err != nil {
			t.Errorf("Drive wear level metrics mismatch: %v", err)
		}
	})

	t.Run("drive_read_ops", func(t *testing.T) {
		expected := `
# HELP vergeos_drive_read_ops Total drive read operations
# TYPE vergeos_drive_read_ops counter
vergeos_drive_read_ops{drive_name="/dev/sda",node_name="node1",serial="WD-001",system_name="testcloud",tier="0"} 50000
vergeos_drive_read_ops{drive_name="/dev/sdb",node_name="node1",serial="WD-002",system_name="testcloud",tier="0"} 10000
`
		if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "vergeos_drive_read_ops"); err != nil {
			t.Errorf("Drive read ops metrics mismatch: %v", err)
		}
	})

	t.Run("drive_util", func(t *testing.T) {
		expected := `
# HELP vergeos_drive_util Drive I/O utilization percentage
# TYPE vergeos_drive_util gauge
vergeos_drive_util{drive_name="/dev/sda",node_name="node1",serial="WD-001",system_name="testcloud",tier="0"} 25
vergeos_drive_util{drive_name="/dev/sdb",node_name="node1",serial="WD-002",system_name="testcloud",tier="0"} 60
`
		if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "vergeos_drive_util"); err != nil {
			t.Errorf("Drive util metrics mismatch: %v", err)
		}
	})

	t.Run("drive_service_time", func(t *testing.T) {
		expected := `
# HELP vergeos_drive_service_time Drive average I/O service time in milliseconds
# TYPE vergeos_drive_service_time gauge
vergeos_drive_service_time{drive_name="/dev/sda",node_name="node1",serial="WD-001",system_name="testcloud",tier="0"} 0.5
vergeos_drive_service_time{drive_name="/dev/sdb",node_name="node1",serial="WD-002",system_name="testcloud",tier="0"} 1.2
`
		if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "vergeos_drive_service_time"); err != nil {
			t.Errorf("Drive service time metrics mismatch: %v", err)
		}
	})

	t.Run("drive_states", func(t *testing.T) {
		expected := `
# HELP vergeos_vsan_drive_states Count of drives in each state per tier
# TYPE vergeos_vsan_drive_states gauge
vergeos_vsan_drive_states{state="online",system_name="testcloud",tier="0"} 1
vergeos_vsan_drive_states{state="repairing",system_name="testcloud",tier="0"} 1
`
		if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "vergeos_vsan_drive_states"); err != nil {
			t.Errorf("Drive states metrics mismatch: %v", err)
		}
	})

	t.Run("drive_read_errors", func(t *testing.T) {
		expected := `
# HELP vergeos_drive_read_errors VSAN drive read error count
# TYPE vergeos_drive_read_errors counter
vergeos_drive_read_errors{drive_name="/dev/sda",node_name="node1",serial="WD-001",system_name="testcloud",tier="0"} 2
vergeos_drive_read_errors{drive_name="/dev/sdb",node_name="node1",serial="WD-002",system_name="testcloud",tier="0"} 0
`
		if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "vergeos_drive_read_errors"); err != nil {
			t.Errorf("Drive read errors metrics mismatch: %v", err)
		}
	})

	t.Run("drive_repairs", func(t *testing.T) {
		expected := `
# HELP vergeos_drive_repairs VSAN drive blocks being repaired
# TYPE vergeos_drive_repairs counter
vergeos_drive_repairs{drive_name="/dev/sda",node_name="node1",serial="WD-001",system_name="testcloud",tier="0"} 0
vergeos_drive_repairs{drive_name="/dev/sdb",node_name="node1",serial="WD-002",system_name="testcloud",tier="0"} 100
`
		if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "vergeos_drive_repairs"); err != nil {
			t.Errorf("Drive repairs metrics mismatch: %v", err)
		}
	})
}
