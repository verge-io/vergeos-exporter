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

func TestNodeCollector(t *testing.T) {
	config := DefaultMockConfig()

	nodes := []NodeMock{
		{ID: 1, Name: "node1", Physical: true, Cluster: 1, IPMIStatus: "ok", RAM: 65536, VMRAM: 32768},
		{ID: 2, Name: "node2", Physical: true, Cluster: 1, IPMIStatus: "offline", RAM: 65536, VMRAM: 16384},
	}

	clusters := []ClusterMock{
		{Key: 1, Name: "cluster1", Enabled: true},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		case strings.Contains(r.URL.Path, "/nodes") && strings.Contains(r.URL.RawQuery, "physical"):
			WriteJSONResponse(w, nodes)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewNodeCollector(client)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

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
	config := DefaultMockConfig()

	nodeCount := 2
	clusters := []ClusterMock{
		{Key: 1, Name: "cluster1", Enabled: true},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		case strings.Contains(r.URL.Path, "/nodes"):
			nodes := []NodeMock{}
			for i := 1; i <= nodeCount; i++ {
				nodes = append(nodes, NodeMock{
					ID:         i,
					Name:       "node" + string(rune('0'+i)),
					Physical:   true,
					Cluster:    1,
					IPMIStatus: "ok",
					RAM:        65536,
					VMRAM:      32768,
				})
			}
			WriteJSONResponse(w, nodes)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewNodeCollector(client)

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

func TestNodeCollector_MultipleClusters(t *testing.T) {
	config := DefaultMockConfig()

	nodes := []NodeMock{
		{ID: 1, Name: "prod-node1", Physical: true, Cluster: 1, IPMIStatus: "ok", RAM: 65536, VMRAM: 32768},
		{ID: 2, Name: "prod-node2", Physical: true, Cluster: 1, IPMIStatus: "ok", RAM: 65536, VMRAM: 32768},
		{ID: 3, Name: "dev-node1", Physical: true, Cluster: 2, IPMIStatus: "ok", RAM: 32768, VMRAM: 16384},
	}

	clusters := []ClusterMock{
		{Key: 1, Name: "production", Enabled: true},
		{Key: 2, Name: "development", Enabled: true},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		case strings.Contains(r.URL.Path, "/nodes"):
			WriteJSONResponse(w, nodes)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewNodeCollector(client)

	t.Run("nodes_total_per_cluster", func(t *testing.T) {
		expected := `
			# HELP vergeos_nodes_total Total number of physical nodes
			# TYPE vergeos_nodes_total gauge
			vergeos_nodes_total{cluster="all",system_name="testcloud"} 3
			vergeos_nodes_total{cluster="production",system_name="testcloud"} 2
			vergeos_nodes_total{cluster="development",system_name="testcloud"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_nodes_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}
