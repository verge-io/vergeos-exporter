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

func TestNodeCollector(t *testing.T) {
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

		case strings.Contains(r.URL.Path, "/clusters"):
			// Clusters list response
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"$key": 1, "name": "cluster1", "enabled": true},
			})

		case strings.Contains(r.URL.Path, "/nodes") && strings.Contains(r.URL.RawQuery, "physical"):
			// Physical nodes list response
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":          1,
					"name":        "node1",
					"physical":    true,
					"cluster":     1,
					"ipmi_status": "ok",
					"ram":         65536,
					"vm_ram":      32768,
				},
				{
					"id":          2,
					"name":        "node2",
					"physical":    true,
					"cluster":     1,
					"ipmi_status": "offline",
					"ram":         65536,
					"vm_ram":      16384,
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
	collector := NewNodeCollector(client)

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
		"vergeos_nodes_total":        false,
		"vergeos_node_ipmi_status":   false,
		"vergeos_node_ram_total":     false,
		"vergeos_node_ram_allocated": false,
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
	t.Run("nodes_total", func(t *testing.T) {
		expected := `
			# HELP vergeos_nodes_total Total number of physical nodes
			# TYPE vergeos_nodes_total gauge
			vergeos_nodes_total{cluster="all",system_name="testcloud"} 2
			vergeos_nodes_total{cluster="cluster1",system_name="testcloud"} 2
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_nodes_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("ipmi_status", func(t *testing.T) {
		expected := `
			# HELP vergeos_node_ipmi_status IPMI status of the node (1=ok, 0=other)
			# TYPE vergeos_node_ipmi_status gauge
			vergeos_node_ipmi_status{cluster="cluster1",node_name="node1",system_name="testcloud"} 1
			vergeos_node_ipmi_status{cluster="cluster1",node_name="node2",system_name="testcloud"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_node_ipmi_status"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("ram_total", func(t *testing.T) {
		expected := `
			# HELP vergeos_node_ram_total Total RAM in MB
			# TYPE vergeos_node_ram_total gauge
			vergeos_node_ram_total{cluster="cluster1",node_name="node1",system_name="testcloud"} 65536
			vergeos_node_ram_total{cluster="cluster1",node_name="node2",system_name="testcloud"} 65536
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_node_ram_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("ram_allocated", func(t *testing.T) {
		expected := `
			# HELP vergeos_node_ram_allocated VM RAM in MB (vm_ram field)
			# TYPE vergeos_node_ram_allocated gauge
			vergeos_node_ram_allocated{cluster="cluster1",node_name="node1",system_name="testcloud"} 32768
			vergeos_node_ram_allocated{cluster="cluster1",node_name="node2",system_name="testcloud"} 16384
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_node_ram_allocated"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestNodeCollector_StaleMetrics(t *testing.T) {
	// This test verifies that the MustNewConstMetric pattern doesn't produce stale metrics
	// when nodes are removed between scrapes

	nodeCount := 2
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

		case strings.Contains(r.URL.Path, "/clusters"):
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"$key": 1, "name": "cluster1", "enabled": true},
			})

		case strings.Contains(r.URL.Path, "/nodes"):
			nodes := []map[string]interface{}{}
			for i := 1; i <= nodeCount; i++ {
				nodes = append(nodes, map[string]interface{}{
					"id":          i,
					"name":        "node" + string(rune('0'+i)),
					"physical":    true,
					"cluster":     1,
					"ipmi_status": "ok",
					"ram":         65536,
					"vm_ram":      32768,
				})
			}
			json.NewEncoder(w).Encode(nodes)

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	client, err := vergeos.NewClient(
		vergeos.WithBaseURL(mockServer.URL),
		vergeos.WithCredentials("test", "test"),
		vergeos.WithInsecureTLS(true),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	collector := NewNodeCollector(client)
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// First scrape - should have 2 nodes
	metrics1, err := registry.Gather()
	if err != nil {
		t.Fatalf("First gather failed: %v", err)
	}

	countIpmiMetrics := func(metrics []*dto.MetricFamily) int {
		for _, mf := range metrics {
			if mf.GetName() == "vergeos_node_ipmi_status" {
				return len(mf.GetMetric())
			}
		}
		return 0
	}

	if count := countIpmiMetrics(metrics1); count != 2 {
		t.Errorf("First scrape: expected 2 IPMI metrics, got %d", count)
	}

	// Simulate node removal
	nodeCount = 1

	// Second scrape - should only have 1 node (no stale metrics)
	metrics2, err := registry.Gather()
	if err != nil {
		t.Fatalf("Second gather failed: %v", err)
	}

	if count := countIpmiMetrics(metrics2); count != 1 {
		t.Errorf("Second scrape: expected 1 IPMI metric (no stale), got %d", count)
	}
}
