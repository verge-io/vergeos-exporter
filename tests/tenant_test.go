package tests

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"vergeos-exporter/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

func TestTenantCollector(t *testing.T) {
	config := DefaultMockConfig()

	tenants := []TenantMock{
		{Key: 1, Name: "tenant-alpha", IsSnapshot: false},
		{Key: 2, Name: "tenant-beta", IsSnapshot: false},
		{Key: 99, Name: "tenant-snap", IsSnapshot: true}, // should be filtered
	}

	tenantStatuses := []TenantStatusMock{
		{Key: 1, Tenant: 1, Running: true, Status: "online", State: "online"},
		{Key: 2, Tenant: 2, Running: false, Status: "offline", State: "offline"},
		{Key: 3, Tenant: 99, Running: false, Status: "offline", State: "offline"}, // snapshot tenant
	}

	tenantStatsHistory := map[int][]TenantStatsHistoryShortMock{
		1: {{Key: 10, Tenant: 1, Timestamp: 1000, TotalCPU: 45, CoreCount: 8, RAMUsed: 12288, RAMAllocated: 16384, RAMPct: 75, IPCount: 5}},
		2: {{Key: 20, Tenant: 2, Timestamp: 1000, TotalCPU: 10, CoreCount: 4, RAMUsed: 2048, RAMAllocated: 8192, RAMPct: 25, IPCount: 2}},
	}

	tenantNodes := []TenantNodeMock{
		{Key: 1, Tenant: 1, NodeID: 1, Name: "alpha-node1", Enabled: true, Machine: 201, CPUCores: 4, RAM: 8192},
		{Key: 2, Tenant: 1, NodeID: 2, Name: "alpha-node2", Enabled: true, Machine: 202, CPUCores: 4, RAM: 8192},
		{Key: 3, Tenant: 2, NodeID: 1, Name: "beta-node1", Enabled: true, Machine: 203, CPUCores: 4, RAM: 8192},
		{Key: 4, Tenant: 99, NodeID: 1, Name: "snap-node1", Enabled: true, Machine: 204, CPUCores: 2, RAM: 4096, IsSnapshot: true},
	}

	tenantStorage := []TenantStorageMock{
		{Key: 1, Tenant: 1, Tier: 0, Provisioned: 1099511627776, Used: 549755813888, Allocated: 824633720832, UsedPct: 50},
		{Key: 2, Tenant: 2, Tier: 0, Provisioned: 549755813888, Used: 109951162778, Allocated: 274877906944, UsedPct: 20},
	}

	tenantL2Networks := []TenantLayer2NetworkMock{
		{Key: 1, Tenant: 1, VNet: 10, Enabled: true},
		{Key: 2, Tenant: 1, VNet: 11, Enabled: true},
		{Key: 3, Tenant: 2, VNet: 10, Enabled: true},
	}

	machineStatuses := map[int]MachineStatusMock{
		201: {Key: 1, Machine: 201, Running: true, Status: "running", State: "online"},
		202: {Key: 2, Machine: 202, Running: true, Status: "running", State: "online"},
		203: {Key: 3, Machine: 203, Running: false, Status: "stopped", State: "offline"},
	}

	machineStats := map[int]MachineStatsMock{
		201: {Key: 1, Machine: 201, TotalCPU: 60, RAMUsed: 6000, RAMPct: 73},
		202: {Key: 2, Machine: 202, TotalCPU: 30, RAMUsed: 3000, RAMPct: 37},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/tenant_status"):
			WriteJSONResponse(w, tenantStatuses)
			return true

		case strings.Contains(r.URL.Path, "/tenant_stats_history_short"):
			filter := r.URL.Query().Get("filter")
			for tid, stats := range tenantStatsHistory {
				if strings.Contains(filter, strconv.Itoa(tid)) {
					WriteJSONResponse(w, stats)
					return true
				}
			}
			WriteJSONResponse(w, []TenantStatsHistoryShortMock{})
			return true

		case strings.Contains(r.URL.Path, "/tenant_nodes"):
			WriteJSONResponse(w, tenantNodes)
			return true

		case strings.Contains(r.URL.Path, "/tenant_storage"):
			WriteJSONResponse(w, tenantStorage)
			return true

		case strings.Contains(r.URL.Path, "/tenant_layer2_vnets"):
			WriteJSONResponse(w, tenantL2Networks)
			return true

		case strings.Contains(r.URL.Path, "/tenants"):
			WriteJSONResponse(w, tenants)
			return true

		case strings.Contains(r.URL.Path, "/machine_status"):
			var all []MachineStatusMock
			for _, s := range machineStatuses {
				all = append(all, s)
			}
			WriteJSONResponse(w, all)
			return true

		case strings.Contains(r.URL.Path, "/machine_stats"):
			var all []MachineStatsMock
			for _, s := range machineStats {
				all = append(all, s)
			}
			WriteJSONResponse(w, all)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewTenantCollector(client, TestScrapeTimeout)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Verify all expected metrics exist
	expectedMetrics := map[string]bool{
		"vergeos_tenants_total":                    false,
		"vergeos_tenant_running":                   false,
		"vergeos_tenant_status":                    false,
		"vergeos_tenant_cpu_usage_pct":             false,
		"vergeos_tenant_cpu_cores":                 false,
		"vergeos_tenant_ram_used_bytes":            false,
		"vergeos_tenant_ram_allocated_bytes":       false,
		"vergeos_tenant_ram_usage_pct":             false,
		"vergeos_tenant_ip_count":                  false,
		"vergeos_tenant_nodes_total":               false,
		"vergeos_tenant_node_cpu_cores":            false,
		"vergeos_tenant_node_ram_bytes":            false,
		"vergeos_tenant_node_enabled":              false,
		"vergeos_tenant_node_running":              false,
		"vergeos_tenant_node_cpu_usage_pct":        false,
		"vergeos_tenant_node_ram_used_bytes":       false,
		"vergeos_tenant_node_ram_usage_pct":        false,
		"vergeos_tenant_storage_provisioned_bytes": false,
		"vergeos_tenant_storage_used_bytes":        false,
		"vergeos_tenant_storage_allocated_bytes":   false,
		"vergeos_tenant_storage_used_pct":          false,
		"vergeos_tenant_layer2_networks_total":     false,
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
	t.Run("tenants_total", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenants_total Total number of tenants
			# TYPE vergeos_tenants_total gauge
			vergeos_tenants_total{system_name="testcloud"} 2
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenants_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("tenant_running", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenant_running Whether the tenant is running (1=running, 0=not running)
			# TYPE vergeos_tenant_running gauge
			vergeos_tenant_running{system_name="testcloud",tenant_name="tenant-alpha"} 1
			vergeos_tenant_running{system_name="testcloud",tenant_name="tenant-beta"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenant_running"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("tenant_status", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenant_status Tenant status (value is always 1, status in label)
			# TYPE vergeos_tenant_status gauge
			vergeos_tenant_status{status="online",system_name="testcloud",tenant_name="tenant-alpha"} 1
			vergeos_tenant_status{status="offline",system_name="testcloud",tenant_name="tenant-beta"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenant_status"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("tenant_cpu", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenant_cpu_usage_pct Tenant total CPU usage percentage
			# TYPE vergeos_tenant_cpu_usage_pct gauge
			vergeos_tenant_cpu_usage_pct{system_name="testcloud",tenant_name="tenant-alpha"} 45
			vergeos_tenant_cpu_usage_pct{system_name="testcloud",tenant_name="tenant-beta"} 10
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenant_cpu_usage_pct"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("tenant_ram", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenant_ram_used_bytes Tenant RAM used in bytes
			# TYPE vergeos_tenant_ram_used_bytes gauge
			vergeos_tenant_ram_used_bytes{system_name="testcloud",tenant_name="tenant-alpha"} 1.2884901888e+10
			vergeos_tenant_ram_used_bytes{system_name="testcloud",tenant_name="tenant-beta"} 2.147483648e+09
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenant_ram_used_bytes"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("tenant_nodes_total", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenant_nodes_total Number of nodes per tenant
			# TYPE vergeos_tenant_nodes_total gauge
			vergeos_tenant_nodes_total{system_name="testcloud",tenant_name="tenant-alpha"} 2
			vergeos_tenant_nodes_total{system_name="testcloud",tenant_name="tenant-beta"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenant_nodes_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("tenant_node_running", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenant_node_running Whether the tenant node is running (1=running, 0=not running)
			# TYPE vergeos_tenant_node_running gauge
			vergeos_tenant_node_running{node_name="alpha-node1",system_name="testcloud",tenant_name="tenant-alpha"} 1
			vergeos_tenant_node_running{node_name="alpha-node2",system_name="testcloud",tenant_name="tenant-alpha"} 1
			vergeos_tenant_node_running{node_name="beta-node1",system_name="testcloud",tenant_name="tenant-beta"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenant_node_running"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("tenant_node_cpu_usage", func(t *testing.T) {
		// beta-node1 (machine 203) has no MachineStats, so no cpu usage emitted for it
		expected := `
			# HELP vergeos_tenant_node_cpu_usage_pct Tenant node CPU usage percentage
			# TYPE vergeos_tenant_node_cpu_usage_pct gauge
			vergeos_tenant_node_cpu_usage_pct{node_name="alpha-node1",system_name="testcloud",tenant_name="tenant-alpha"} 60
			vergeos_tenant_node_cpu_usage_pct{node_name="alpha-node2",system_name="testcloud",tenant_name="tenant-alpha"} 30
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenant_node_cpu_usage_pct"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("tenant_storage", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenant_storage_provisioned_bytes Tenant storage provisioned in bytes
			# TYPE vergeos_tenant_storage_provisioned_bytes gauge
			vergeos_tenant_storage_provisioned_bytes{system_name="testcloud",tenant_name="tenant-alpha",tier="0"} 1.099511627776e+12
			vergeos_tenant_storage_provisioned_bytes{system_name="testcloud",tenant_name="tenant-beta",tier="0"} 5.49755813888e+11
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenant_storage_provisioned_bytes"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("tenant_l2_networks", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenant_layer2_networks_total Number of layer 2 networks assigned to tenant
			# TYPE vergeos_tenant_layer2_networks_total gauge
			vergeos_tenant_layer2_networks_total{system_name="testcloud",tenant_name="tenant-alpha"} 2
			vergeos_tenant_layer2_networks_total{system_name="testcloud",tenant_name="tenant-beta"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenant_layer2_networks_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestTenantCollector_SnapshotFiltering(t *testing.T) {
	config := DefaultMockConfig()

	tenants := []TenantMock{
		{Key: 1, Name: "real-tenant", IsSnapshot: false},
		{Key: 2, Name: "snap-tenant", IsSnapshot: true},
	}

	tenantNodes := []TenantNodeMock{
		{Key: 1, Tenant: 1, Name: "real-node", Enabled: true, Machine: 301, CPUCores: 4, RAM: 8192},
		{Key: 2, Tenant: 1, Name: "snap-node", Enabled: true, Machine: 302, CPUCores: 2, RAM: 4096, IsSnapshot: true},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/tenant_status"):
			WriteJSONResponse(w, []TenantStatusMock{
				{Key: 1, Tenant: 1, Running: true, Status: "online", State: "online"},
			})
			return true
		case strings.Contains(r.URL.Path, "/tenant_stats_history_short"):
			filter := r.URL.Query().Get("filter")
			if strings.Contains(filter, "1") {
				WriteJSONResponse(w, []TenantStatsHistoryShortMock{
					{Key: 1, Tenant: 1, Timestamp: 1000, TotalCPU: 50, CoreCount: 4, RAMUsed: 4096, RAMAllocated: 8192, RAMPct: 50, IPCount: 3},
				})
				return true
			}
			WriteJSONResponse(w, []TenantStatsHistoryShortMock{})
			return true
		case strings.Contains(r.URL.Path, "/tenant_nodes"):
			WriteJSONResponse(w, tenantNodes)
			return true
		case strings.Contains(r.URL.Path, "/tenant_storage"):
			WriteJSONResponse(w, []TenantStorageMock{})
			return true
		case strings.Contains(r.URL.Path, "/tenant_layer2_vnets"):
			WriteJSONResponse(w, []TenantLayer2NetworkMock{})
			return true
		case strings.Contains(r.URL.Path, "/tenants"):
			WriteJSONResponse(w, tenants)
			return true
		case strings.Contains(r.URL.Path, "/machine_status"):
			WriteJSONResponse(w, []MachineStatusMock{
				{Key: 1, Machine: 301, Running: true, Status: "running", State: "online"},
			})
			return true
		case strings.Contains(r.URL.Path, "/machine_stats"):
			WriteJSONResponse(w, []MachineStatsMock{
				{Key: 1, Machine: 301, TotalCPU: 50, RAMUsed: 4000, RAMPct: 50},
			})
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewTenantCollector(client, TestScrapeTimeout)

	// Verify only 1 tenant counted (snapshot excluded)
	t.Run("total_excludes_snapshots", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenants_total Total number of tenants
			# TYPE vergeos_tenants_total gauge
			vergeos_tenants_total{system_name="testcloud"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenants_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	// Verify only 1 node (snapshot node excluded)
	t.Run("nodes_exclude_snapshots", func(t *testing.T) {
		expected := `
			# HELP vergeos_tenant_nodes_total Number of nodes per tenant
			# TYPE vergeos_tenant_nodes_total gauge
			vergeos_tenant_nodes_total{system_name="testcloud",tenant_name="real-tenant"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_tenant_nodes_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestTenantCollector_StaleMetrics(t *testing.T) {
	config := DefaultMockConfig()
	tenantCount := 2

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/tenant_status"):
			statuses := []TenantStatusMock{}
			for i := 1; i <= tenantCount; i++ {
				statuses = append(statuses, TenantStatusMock{
					Key: i, Tenant: i, Running: true, Status: "online", State: "online",
				})
			}
			WriteJSONResponse(w, statuses)
			return true
		case strings.Contains(r.URL.Path, "/tenant_stats_history_short"):
			filter := r.URL.Query().Get("filter")
			for i := 1; i <= tenantCount; i++ {
				if strings.Contains(filter, strconv.Itoa(i)) {
					WriteJSONResponse(w, []TenantStatsHistoryShortMock{
						{Key: i, Tenant: i, Timestamp: 1000, TotalCPU: 50, CoreCount: 4, RAMUsed: 4096, RAMAllocated: 8192, RAMPct: 50, IPCount: 1},
					})
					return true
				}
			}
			WriteJSONResponse(w, []TenantStatsHistoryShortMock{})
			return true
		case strings.Contains(r.URL.Path, "/tenant_nodes"):
			WriteJSONResponse(w, []TenantNodeMock{})
			return true
		case strings.Contains(r.URL.Path, "/tenant_storage"):
			WriteJSONResponse(w, []TenantStorageMock{})
			return true
		case strings.Contains(r.URL.Path, "/tenant_layer2_vnets"):
			WriteJSONResponse(w, []TenantLayer2NetworkMock{})
			return true
		case strings.Contains(r.URL.Path, "/tenants"):
			tenants := []TenantMock{}
			for i := 1; i <= tenantCount; i++ {
				tenants = append(tenants, TenantMock{
					Key: i, Name: "tenant-" + strconv.Itoa(i),
				})
			}
			WriteJSONResponse(w, tenants)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewTenantCollector(client, TestScrapeTimeout)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// First scrape - 2 tenants
	metrics1, err := registry.Gather()
	if err != nil {
		t.Fatalf("First gather failed: %v", err)
	}

	countRunningMetrics := func(metrics []*dto.MetricFamily) int {
		for _, mf := range metrics {
			if mf.GetName() == "vergeos_tenant_running" {
				return len(mf.GetMetric())
			}
		}
		return 0
	}

	if count := countRunningMetrics(metrics1); count != 2 {
		t.Errorf("First scrape: expected 2 running metrics, got %d", count)
	}

	// Remove a tenant
	tenantCount = 1

	// Second scrape - only 1 tenant (no stale metrics)
	metrics2, err := registry.Gather()
	if err != nil {
		t.Fatalf("Second gather failed: %v", err)
	}

	if count := countRunningMetrics(metrics2); count != 1 {
		t.Errorf("Second scrape: expected 1 running metric (no stale), got %d", count)
	}
}
