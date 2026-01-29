package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	vergeos "github.com/verge-io/goVergeOS"
)

func TestClusterCollector(t *testing.T) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/version.json"):
			// SDK version check during client creation
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":    "v4",
				"version": "26.0.2.1",
				"hash":    "testbuild",
			})

		case strings.Contains(r.URL.Path, "/settings") && strings.Contains(r.URL.RawQuery, "cloud_name"):
			// Settings API response
			json.NewEncoder(w).Encode([]map[string]string{
				{"key": "cloud_name", "value": "testcloud"},
			})

		case strings.HasPrefix(r.URL.Path, "/api/v4/clusters/"):
			// Single cluster status response (GetStatus call)
			// The SDK requests fields=status[...] for GetStatus
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": map[string]interface{}{
					"cluster":          1,
					"status":           "online",
					"state":            "online",
					"total_nodes":      2,
					"online_nodes":     2,
					"running_machines": 5,
					"total_ram":        131072,
					"online_ram":       131072,
					"used_ram":         65536,
					"total_cores":      32,
					"online_cores":     32,
					"used_cores":       16,
					"phys_ram_used":    45000,
				},
			})

		case strings.Contains(r.URL.Path, "/clusters"):
			// Clusters list response
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"$key":           1,
					"name":           "cluster1",
					"enabled":        true,
					"ram_per_unit":   4096,
					"cores_per_unit": 1,
					"target_ram_pct": 80.0,
				},
			})

		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.String())
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create SDK client pointing to mock server
	client, err := vergeos.NewClient(
		vergeos.WithBaseURL(mockServer.URL),
		vergeos.WithCredentials("test", "test"),
		vergeos.WithInsecureTLS(true),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create collector
	collector := NewClusterCollector(client)

	// Create a new registry and register the collector
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// Collect metrics
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
	// Test with multiple clusters to verify all are processed
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/version.json"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":    "v4",
				"version": "26.0.2.1",
				"hash":    "testbuild",
			})

		case strings.Contains(r.URL.Path, "/settings"):
			json.NewEncoder(w).Encode([]map[string]string{
				{"key": "cloud_name", "value": "testcloud"},
			})

		case strings.HasPrefix(r.URL.Path, "/api/v4/clusters/1"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": map[string]interface{}{
					"cluster":          1,
					"status":           "online",
					"state":            "online",
					"total_nodes":      2,
					"online_nodes":     2,
					"running_machines": 10,
					"total_ram":        65536,
					"online_ram":       65536,
					"used_ram":         32768,
					"total_cores":      16,
					"online_cores":     16,
					"used_cores":       8,
					"phys_ram_used":    20000,
				},
			})

		case strings.HasPrefix(r.URL.Path, "/api/v4/clusters/2"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": map[string]interface{}{
					"cluster":          2,
					"status":           "online",
					"state":            "warning",
					"total_nodes":      1,
					"online_nodes":     1,
					"running_machines": 3,
					"total_ram":        32768,
					"online_ram":       32768,
					"used_ram":         16384,
					"total_cores":      8,
					"online_cores":     8,
					"used_cores":       4,
					"phys_ram_used":    10000,
				},
			})

		case strings.Contains(r.URL.Path, "/clusters"):
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"$key":           1,
					"name":           "production",
					"enabled":        true,
					"ram_per_unit":   4096,
					"cores_per_unit": 1,
					"target_ram_pct": 80.0,
				},
				{
					"$key":           2,
					"name":           "development",
					"enabled":        false,
					"ram_per_unit":   2048,
					"cores_per_unit": 2,
					"target_ram_pct": 90.0,
				},
			})

		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.String())
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	client, _ := vergeos.NewClient(
		vergeos.WithBaseURL(mockServer.URL),
		vergeos.WithCredentials("test", "test"),
		vergeos.WithInsecureTLS(true),
	)

	collector := NewClusterCollector(client)

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

	clusterCount := 2
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/version.json"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":    "v4",
				"version": "26.0.2.1",
				"hash":    "testbuild",
			})

		case strings.Contains(r.URL.Path, "/settings"):
			json.NewEncoder(w).Encode([]map[string]string{
				{"key": "cloud_name", "value": "testcloud"},
			})

		case strings.HasPrefix(r.URL.Path, "/api/v4/clusters/"):
			// Extract cluster ID and return status
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": map[string]interface{}{
					"cluster":          1,
					"status":           "online",
					"state":            "online",
					"total_nodes":      2,
					"online_nodes":     2,
					"running_machines": 5,
					"total_ram":        65536,
					"online_ram":       65536,
					"used_ram":         32768,
					"total_cores":      16,
					"online_cores":     16,
					"used_cores":       8,
					"phys_ram_used":    20000,
				},
			})

		case strings.Contains(r.URL.Path, "/clusters"):
			clusters := []map[string]interface{}{}
			for i := 1; i <= clusterCount; i++ {
				clusters = append(clusters, map[string]interface{}{
					"$key":           i,
					"name":           "cluster" + string(rune('0'+i)),
					"enabled":        true,
					"ram_per_unit":   4096,
					"cores_per_unit": 1,
					"target_ram_pct": 80.0,
				})
			}
			json.NewEncoder(w).Encode(clusters)

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	client, _ := vergeos.NewClient(
		vergeos.WithBaseURL(mockServer.URL),
		vergeos.WithCredentials("test", "test"),
		vergeos.WithInsecureTLS(true),
	)

	collector := NewClusterCollector(client)
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
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/version.json"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":    "v4",
				"version": "26.0.2.1",
				"hash":    "testbuild",
			})

		case strings.Contains(r.URL.Path, "/settings"):
			json.NewEncoder(w).Encode([]map[string]string{
				{"key": "cloud_name", "value": "testcloud"},
			})

		case strings.HasPrefix(r.URL.Path, "/api/v4/clusters/"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": map[string]interface{}{
					"cluster":          1,
					"status":           "offline",
					"state":            "offline",
					"total_nodes":      2,
					"online_nodes":     0, // No nodes online
					"running_machines": 0,
					"total_ram":        65536,
					"online_ram":       0,
					"used_ram":         0,
					"total_cores":      16,
					"online_cores":     0,
					"used_cores":       0,
					"phys_ram_used":    0,
				},
			})

		case strings.Contains(r.URL.Path, "/clusters"):
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"$key":           1,
					"name":           "offline-cluster",
					"enabled":        true,
					"ram_per_unit":   4096,
					"cores_per_unit": 1,
					"target_ram_pct": 80.0,
				},
			})

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	client, _ := vergeos.NewClient(
		vergeos.WithBaseURL(mockServer.URL),
		vergeos.WithCredentials("test", "test"),
		vergeos.WithInsecureTLS(true),
	)

	collector := NewClusterCollector(client)

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
