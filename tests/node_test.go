package tests

import (
	"encoding/json"
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
		{ID: 1, Name: "node1", Physical: true, Cluster: 1, Machine: 101, IPMIStatus: "ok", RAM: 65536, VMRAM: 32768, VMStatsTotals: &NodeVMStatsTotalsMock{RunningCores: 24, RunningRAM: 49152}},
		{ID: 2, Name: "node2", Physical: true, Cluster: 1, Machine: 102, IPMIStatus: "offline", RAM: 65536, VMRAM: 16384, VMStatsTotals: &NodeVMStatsTotalsMock{RunningCores: 8, RunningRAM: 16384}},
	}

	clusters := []ClusterMock{
		{Key: 1, Name: "cluster1", Enabled: true},
	}

	machineStats := map[int]MachineStatsMock{
		101: {Key: 1, Machine: 101, TotalCPU: 45, RAMUsed: 48000, RAMPct: 73, CoreUsageList: json.RawMessage(`[10.5, 20.3, 30.1, 40.0]`), CoreTemp: 55, CoreTempTop: 62},
		102: {Key: 2, Machine: 102, TotalCPU: 30, RAMUsed: 32000, RAMPct: 49, CoreUsageList: json.RawMessage(`[5.0, 15.0]`), CoreTemp: 42, CoreTempTop: 48},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		case strings.Contains(r.URL.Path, "/nodes") && strings.Contains(r.URL.RawQuery, "physical"):
			WriteJSONResponse(w, nodes)
			return true
		case strings.Contains(r.URL.Path, "/machine_stats"):
			// Parse machine filter from query
			filter := r.URL.Query().Get("filter")
			for machineID, stats := range machineStats {
				if strings.Contains(filter, string(rune('0'+machineID))) || strings.Contains(filter, "101") || strings.Contains(filter, "102") {
					// Match by checking if the filter contains the machine ID
					for mid, s := range machineStats {
						if strings.Contains(filter, intToStr(mid)) {
							WriteJSONResponse(w, []MachineStatsMock{s})
							return true
						}
						_ = s
					}
				}
				_ = stats
			}
			// Fallback: try to match by extracting machine ID from filter
			for mid, s := range machineStats {
				if strings.Contains(filter, intToStr(mid)) {
					WriteJSONResponse(w, []MachineStatsMock{s})
					return true
				}
			}
			WriteJSONResponse(w, []MachineStatsMock{})
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
		"vergeos_nodes_total":         false,
		"vergeos_node_ipmi_status":    false,
		"vergeos_node_ram_total":      false,
		"vergeos_node_ram_allocated":  false,
		"vergeos_node_cpu_core_usage": false,
		"vergeos_node_core_temp":      false,
		"vergeos_node_ram_used":       false,
		"vergeos_node_ram_pct":        false,
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

	t.Run("core_temp", func(t *testing.T) {
		expected := `
			# HELP vergeos_node_core_temp Average CPU core temperature in Celsius
			# TYPE vergeos_node_core_temp gauge
			vergeos_node_core_temp{cluster="cluster1",node_name="node1",system_name="testcloud"} 55
			vergeos_node_core_temp{cluster="cluster1",node_name="node2",system_name="testcloud"} 42
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_node_core_temp"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("ram_used", func(t *testing.T) {
		expected := `
			# HELP vergeos_node_ram_used Physical RAM used in MB
			# TYPE vergeos_node_ram_used gauge
			vergeos_node_ram_used{cluster="cluster1",node_name="node1",system_name="testcloud"} 48000
			vergeos_node_ram_used{cluster="cluster1",node_name="node2",system_name="testcloud"} 32000
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_node_ram_used"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("ram_pct", func(t *testing.T) {
		expected := `
			# HELP vergeos_node_ram_pct Physical RAM used percentage
			# TYPE vergeos_node_ram_pct gauge
			vergeos_node_ram_pct{cluster="cluster1",node_name="node1",system_name="testcloud"} 73
			vergeos_node_ram_pct{cluster="cluster1",node_name="node2",system_name="testcloud"} 49
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_node_ram_pct"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("cpu_core_usage", func(t *testing.T) {
		// Node1 has 4 cores, Node2 has 2 cores
		expected := `
			# HELP vergeos_node_cpu_core_usage CPU usage percentage per core
			# TYPE vergeos_node_cpu_core_usage gauge
			vergeos_node_cpu_core_usage{cluster="cluster1",core_id="0",node_name="node1",system_name="testcloud"} 10.5
			vergeos_node_cpu_core_usage{cluster="cluster1",core_id="1",node_name="node1",system_name="testcloud"} 20.3
			vergeos_node_cpu_core_usage{cluster="cluster1",core_id="2",node_name="node1",system_name="testcloud"} 30.1
			vergeos_node_cpu_core_usage{cluster="cluster1",core_id="3",node_name="node1",system_name="testcloud"} 40
			vergeos_node_cpu_core_usage{cluster="cluster1",core_id="0",node_name="node2",system_name="testcloud"} 5
			vergeos_node_cpu_core_usage{cluster="cluster1",core_id="1",node_name="node2",system_name="testcloud"} 15
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_node_cpu_core_usage"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("running_cores", func(t *testing.T) {
		expected := `
			# HELP vergeos_node_running_cores Total CPU cores allocated to running VMs
			# TYPE vergeos_node_running_cores gauge
			vergeos_node_running_cores{cluster="cluster1",node_name="node1",system_name="testcloud"} 24
			vergeos_node_running_cores{cluster="cluster1",node_name="node2",system_name="testcloud"} 8
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_node_running_cores"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("running_ram", func(t *testing.T) {
		expected := `
			# HELP vergeos_node_running_ram Total RAM in MB allocated to running VMs
			# TYPE vergeos_node_running_ram gauge
			vergeos_node_running_ram{cluster="cluster1",node_name="node1",system_name="testcloud"} 49152
			vergeos_node_running_ram{cluster="cluster1",node_name="node2",system_name="testcloud"} 16384
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_node_running_ram"); err != nil {
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
		case strings.Contains(r.URL.Path, "/machine_stats"):
			// Return stats for any machine
			filter := r.URL.Query().Get("filter")
			for i := 1; i <= nodeCount; i++ {
				if strings.Contains(filter, intToStr(100+i)) {
					WriteJSONResponse(w, []MachineStatsMock{
						{Key: i, Machine: 100 + i, TotalCPU: 50, RAMUsed: 32000, RAMPct: 50, CoreUsageList: json.RawMessage(`[50.0]`), CoreTemp: 50},
					})
					return true
				}
			}
			WriteJSONResponse(w, []MachineStatsMock{})
			return true
		case strings.Contains(r.URL.Path, "/nodes"):
			nodes := []NodeMock{}
			for i := 1; i <= nodeCount; i++ {
				nodes = append(nodes, NodeMock{
					ID:         i,
					Name:       "node" + string(rune('0'+i)),
					Physical:   true,
					Cluster:    1,
					Machine:    100 + i,
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
		{ID: 1, Name: "prod-node1", Physical: true, Cluster: 1, Machine: 101, IPMIStatus: "ok", RAM: 65536, VMRAM: 32768},
		{ID: 2, Name: "prod-node2", Physical: true, Cluster: 1, Machine: 102, IPMIStatus: "ok", RAM: 65536, VMRAM: 32768},
		{ID: 3, Name: "dev-node1", Physical: true, Cluster: 2, Machine: 103, IPMIStatus: "ok", RAM: 32768, VMRAM: 16384},
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
		case strings.Contains(r.URL.Path, "/machine_stats"):
			filter := r.URL.Query().Get("filter")
			for _, mid := range []int{101, 102, 103} {
				if strings.Contains(filter, intToStr(mid)) {
					WriteJSONResponse(w, []MachineStatsMock{
						{Key: mid, Machine: mid, TotalCPU: 50, RAMUsed: 32000, RAMPct: 50, CoreUsageList: json.RawMessage(`[50.0]`), CoreTemp: 50},
					})
					return true
				}
			}
			WriteJSONResponse(w, []MachineStatsMock{})
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

// intToStr converts an int to its string representation
func intToStr(n int) string {
	s := ""
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
