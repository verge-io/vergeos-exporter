package tests

import (
	"net/http"
	"strings"
	"testing"

	"vergeos-exporter/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

func TestClusterCollector(t *testing.T) {
	config := DefaultMockConfig()

	clusters := []ClusterMock{
		{Key: 1, Name: "cluster1", Enabled: true, RAMPerUnit: 4096, CoresPerUnit: 1, TargetRAMPct: 80.0},
	}

	clusterStatus := ClusterStatusMock{
		Cluster: 1, Status: "online", State: "online",
		TotalNodes: 2, OnlineNodes: 2, RunningMachines: 5,
		TotalRAM: 131072, OnlineRAM: 131072, UsedRAM: 65536,
		TotalCores: 32, OnlineCores: 32, UsedCores: 16,
		PhysRAMUsed: 45000,
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/clusters/"):
			WriteJSONResponse(w, map[string]interface{}{"status": clusterStatus})
			return true
		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewClusterCollector(client)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Verify expected metrics
	expectedMetrics := map[string]bool{
		"vergeos_clusters_total":           false,
		"vergeos_cluster_status":           false,
		"vergeos_cluster_health":           false,
		"vergeos_cluster_enabled":          false,
		"vergeos_cluster_total_ram":        false,
		"vergeos_cluster_used_ram":         false,
		"vergeos_cluster_cores_total":      false,
		"vergeos_cluster_used_cores":       false,
		"vergeos_cluster_running_machines": false,
		"vergeos_cluster_total_nodes":      false,
		"vergeos_cluster_online_nodes":     false,
		"vergeos_cluster_online_ram":       false,
		"vergeos_cluster_online_cores":     false,
		"vergeos_cluster_phys_ram_used":    false,
		"vergeos_cluster_ram_per_unit":     false,
		"vergeos_cluster_cores_per_unit":   false,
		"vergeos_cluster_target_ram_pct":   false,
	}

	for _, mf := range metrics {
		if _, ok := expectedMetrics[mf.GetName()]; ok {
			expectedMetrics[mf.GetName()] = true
		}
	}

	for metric, found := range expectedMetrics {
		if !found {
			t.Errorf("Expected metric %s not found", metric)
		}
	}

	// Verify specific metric values
	t.Run("clusters_total", func(t *testing.T) {
		expected := `
			# HELP vergeos_clusters_total Total number of clusters
			# TYPE vergeos_clusters_total gauge
			vergeos_clusters_total{system_name="testcloud"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_clusters_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("cluster_status", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_status Cluster status (1=online, 0=offline)
			# TYPE vergeos_cluster_status gauge
			vergeos_cluster_status{cluster="cluster1",system_name="testcloud"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_status"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("cluster_health", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_health Cluster health status (1=healthy, 0=unhealthy)
			# TYPE vergeos_cluster_health gauge
			vergeos_cluster_health{cluster="cluster1",system_name="testcloud"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_health"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("cluster_enabled", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_enabled Cluster enabled status (1=enabled, 0=disabled)
			# TYPE vergeos_cluster_enabled gauge
			vergeos_cluster_enabled{cluster="cluster1",system_name="testcloud"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_enabled"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("cluster_ram_metrics", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_total_ram Total RAM in MB
			# TYPE vergeos_cluster_total_ram gauge
			vergeos_cluster_total_ram{cluster="cluster1",system_name="testcloud"} 131072
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_total_ram"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("cluster_cores_metrics", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_cores_total Total number of CPU cores
			# TYPE vergeos_cluster_cores_total gauge
			vergeos_cluster_cores_total{cluster="cluster1",system_name="testcloud"} 32
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_cores_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("cluster_running_machines", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_running_machines Total number of running machines
			# TYPE vergeos_cluster_running_machines gauge
			vergeos_cluster_running_machines{cluster="cluster1",system_name="testcloud"} 5
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_running_machines"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("cluster_nodes_metrics", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_total_nodes Total number of nodes
			# TYPE vergeos_cluster_total_nodes gauge
			vergeos_cluster_total_nodes{cluster="cluster1",system_name="testcloud"} 2
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_total_nodes"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("cluster_config_metrics", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_ram_per_unit RAM per unit in MB
			# TYPE vergeos_cluster_ram_per_unit gauge
			vergeos_cluster_ram_per_unit{cluster="cluster1",system_name="testcloud"} 4096
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_ram_per_unit"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestClusterCollector_MultipleClusters(t *testing.T) {
	config := DefaultMockConfig()

	clusters := []ClusterMock{
		{Key: 1, Name: "production", Enabled: true, RAMPerUnit: 4096, CoresPerUnit: 1, TargetRAMPct: 80.0},
		{Key: 2, Name: "development", Enabled: false, RAMPerUnit: 2048, CoresPerUnit: 2, TargetRAMPct: 90.0},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/clusters/1"):
			WriteJSONResponse(w, map[string]interface{}{
				"status": ClusterStatusMock{
					Cluster: 1, Status: "online", State: "online",
					TotalNodes: 2, OnlineNodes: 2, RunningMachines: 10,
					TotalRAM: 65536, OnlineRAM: 65536, UsedRAM: 32768,
					TotalCores: 16, OnlineCores: 16, UsedCores: 8,
					PhysRAMUsed: 20000,
				},
			})
			return true
		case strings.HasPrefix(r.URL.Path, "/api/v4/clusters/2"):
			WriteJSONResponse(w, map[string]interface{}{
				"status": ClusterStatusMock{
					Cluster: 2, Status: "online", State: "warning",
					TotalNodes: 1, OnlineNodes: 1, RunningMachines: 3,
					TotalRAM: 32768, OnlineRAM: 32768, UsedRAM: 16384,
					TotalCores: 8, OnlineCores: 8, UsedCores: 4,
					PhysRAMUsed: 10000,
				},
			})
			return true
		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewClusterCollector(client)

	// Verify clusters_total shows 2
	t.Run("clusters_total", func(t *testing.T) {
		expected := `
			# HELP vergeos_clusters_total Total number of clusters
			# TYPE vergeos_clusters_total gauge
			vergeos_clusters_total{system_name="testcloud"} 2
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_clusters_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	// Verify both clusters have status metrics
	t.Run("cluster_status_multiple", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_status Cluster status (1=online, 0=offline)
			# TYPE vergeos_cluster_status gauge
			vergeos_cluster_status{cluster="production",system_name="testcloud"} 1
			vergeos_cluster_status{cluster="development",system_name="testcloud"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_status"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	// Verify health status differs based on state
	t.Run("cluster_health_multiple", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_health Cluster health status (1=healthy, 0=unhealthy)
			# TYPE vergeos_cluster_health gauge
			vergeos_cluster_health{cluster="production",system_name="testcloud"} 1
			vergeos_cluster_health{cluster="development",system_name="testcloud"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_health"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	// Verify enabled status differs
	t.Run("cluster_enabled_multiple", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_enabled Cluster enabled status (1=enabled, 0=disabled)
			# TYPE vergeos_cluster_enabled gauge
			vergeos_cluster_enabled{cluster="production",system_name="testcloud"} 1
			vergeos_cluster_enabled{cluster="development",system_name="testcloud"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_enabled"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestClusterCollector_StaleMetrics(t *testing.T) {
	// This test verifies that the MustNewConstMetric pattern doesn't produce stale metrics
	// when clusters are removed between scrapes
	config := DefaultMockConfig()

	clusterCount := 2

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/clusters/"):
			WriteJSONResponse(w, map[string]interface{}{
				"status": ClusterStatusMock{
					Cluster: 1, Status: "online", State: "online",
					TotalNodes: 2, OnlineNodes: 2, RunningMachines: 5,
					TotalRAM: 65536, OnlineRAM: 65536, UsedRAM: 32768,
					TotalCores: 16, OnlineCores: 16, UsedCores: 8,
					PhysRAMUsed: 20000,
				},
			})
			return true
		case strings.Contains(r.URL.Path, "/clusters"):
			clusters := []ClusterMock{}
			for i := 1; i <= clusterCount; i++ {
				clusters = append(clusters, ClusterMock{
					Key:          i,
					Name:         "cluster" + string(rune('0'+i)),
					Enabled:      true,
					RAMPerUnit:   4096,
					CoresPerUnit: 1,
					TargetRAMPct: 80.0,
				})
			}
			WriteJSONResponse(w, clusters)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewClusterCollector(client)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// First scrape - should have 2 clusters
	metrics1, err := registry.Gather()
	if err != nil {
		t.Fatalf("First gather failed: %v", err)
	}

	countClusterMetrics := func(metrics []*dto.MetricFamily, metricName string) int {
		for _, mf := range metrics {
			if mf.GetName() == metricName {
				return len(mf.GetMetric())
			}
		}
		return 0
	}

	if count := countClusterMetrics(metrics1, "vergeos_cluster_status"); count != 2 {
		t.Errorf("First scrape: expected 2 cluster_status metrics, got %d", count)
	}

	// Simulate cluster removal
	clusterCount = 1

	// Second scrape - should only have 1 cluster (no stale metrics)
	metrics2, err := registry.Gather()
	if err != nil {
		t.Fatalf("Second gather failed: %v", err)
	}

	if count := countClusterMetrics(metrics2, "vergeos_cluster_status"); count != 1 {
		t.Errorf("Second scrape: expected 1 cluster_status metric (no stale), got %d", count)
	}

	// Verify clusters_total is also updated
	if count := countClusterMetrics(metrics2, "vergeos_clusters_total"); count != 1 {
		t.Errorf("Second scrape: expected 1 clusters_total metric, got %d", count)
	}
}

func TestClusterCollector_OfflineCluster(t *testing.T) {
	// Test that offline clusters (0 online nodes) show status=0
	config := DefaultMockConfig()

	clusters := []ClusterMock{
		{Key: 1, Name: "offline-cluster", Enabled: true, RAMPerUnit: 4096, CoresPerUnit: 1, TargetRAMPct: 80.0},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/clusters/"):
			WriteJSONResponse(w, map[string]interface{}{
				"status": ClusterStatusMock{
					Cluster: 1, Status: "offline", State: "offline",
					TotalNodes: 2, OnlineNodes: 0, RunningMachines: 0,
					TotalRAM: 65536, OnlineRAM: 0, UsedRAM: 0,
					TotalCores: 16, OnlineCores: 0, UsedCores: 0,
					PhysRAMUsed: 0,
				},
			})
			return true
		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewClusterCollector(client)

	t.Run("offline_cluster_status", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_status Cluster status (1=online, 0=offline)
			# TYPE vergeos_cluster_status gauge
			vergeos_cluster_status{cluster="offline-cluster",system_name="testcloud"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_status"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("offline_cluster_health", func(t *testing.T) {
		expected := `
			# HELP vergeos_cluster_health Cluster health status (1=healthy, 0=unhealthy)
			# TYPE vergeos_cluster_health gauge
			vergeos_cluster_health{cluster="offline-cluster",system_name="testcloud"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_cluster_health"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}
