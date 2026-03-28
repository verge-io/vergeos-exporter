//go:build integration

package tests

import (
	"os"
	"strings"
	"testing"

	"vergeos-exporter/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	vergeos "github.com/verge-io/goVergeOS"
)

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// createIntegrationClient creates an SDK client for integration testing.
func createIntegrationClient(t *testing.T) *vergeos.Client {
	t.Helper()

	url := getEnvOrDefault("VERGE_URL", "https://vergeos.example.com")
	user := getEnvOrDefault("VERGE_USERNAME", "")
	pass := getEnvOrDefault("VERGE_PASSWORD", "REDACTED")

	client, err := vergeos.NewClient(
		vergeos.WithBaseURL(url),
		vergeos.WithCredentials(user, pass),
		vergeos.WithInsecureTLS(true),
	)
	if err != nil {
		t.Fatalf("Failed to create SDK client: %v", err)
	}
	return client
}

func TestIntegrationStorageCollector(t *testing.T) {
	client := createIntegrationClient(t)
	sc := collectors.NewStorageCollector(client)

	metrics := collectMetrics(t, sc)

	t.Run("nodes_online", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vsan_nodes_online")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_vsan_nodes_online metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "tier", "status")
		}
	})

	t.Run("drives_online", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vsan_drives_online")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_vsan_drives_online metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "tier", "status")
		}
	})
}

func TestIntegrationNodeCollector(t *testing.T) {
	client := createIntegrationClient(t)
	nc := collectors.NewNodeCollector(client)

	metrics := collectMetrics(t, nc)

	t.Run("running_cores", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_node_running_cores")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_node_running_cores metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node_name")
		}
	})

	t.Run("running_ram", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_node_running_ram")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_node_running_ram metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node_name")
		}
	})
}

func TestIntegrationTenantCollector(t *testing.T) {
	client := createIntegrationClient(t)
	tc := collectors.NewTenantCollector(client)

	metrics := collectMetrics(t, tc)

	t.Run("tenants_total", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_tenants_total")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_tenants_total metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name")
		}
	})

	t.Run("tenant_running", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_tenant_running")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_tenant_running metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "tenant_name")
		}
	})

	t.Run("tenant_status", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_tenant_status")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_tenant_status metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "tenant_name", "status")
		}
	})

	t.Run("tenant_cpu_usage", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_tenant_cpu_usage_pct")
		// May be 0 for offline tenants, but should exist for running ones
		t.Logf("Found %d tenant CPU usage metrics", len(found))
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "tenant_name")
		}
	})

	t.Run("tenant_nodes_total", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_tenant_nodes_total")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_tenant_nodes_total metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "tenant_name")
		}
	})

	t.Run("tenant_node_cpu_cores", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_tenant_node_cpu_cores")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_tenant_node_cpu_cores metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "tenant_name", "node_name")
		}
	})

	t.Run("tenant_node_ram_bytes", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_tenant_node_ram_bytes")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_tenant_node_ram_bytes metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "tenant_name", "node_name")
		}
	})

	t.Run("tenant_storage", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_tenant_storage_provisioned_bytes")
		// Storage may or may not exist depending on tenant config
		t.Logf("Found %d tenant storage provisioned metrics", len(found))
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "tenant_name", "tier")
		}
	})
}

func TestIntegrationVMCollector(t *testing.T) {
	client := createIntegrationClient(t)
	vc := collectors.NewVMCollector(client)

	metrics := collectMetrics(t, vc)

	t.Run("vm_cpu_total", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vm_cpu_total")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_vm_cpu_total metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node", "vm_name", "vm_id")
		}
		t.Logf("Found %d VM CPU total metrics", len(found))
	})

	t.Run("vm_cpu_user", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vm_cpu_user")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_vm_cpu_user metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node", "vm_name", "vm_id")
		}
	})

	t.Run("vm_cpu_system", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vm_cpu_system")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_vm_cpu_system metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node", "vm_name", "vm_id")
		}
	})

	t.Run("vm_cpu_iowait", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vm_cpu_iowait")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_vm_cpu_iowait metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node", "vm_name", "vm_id")
		}
	})

	t.Run("vm_running", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vm_running")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_vm_running metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node", "vm_name", "vm_id")
		}
	})

	t.Run("vm_enabled", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vm_enabled")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_vm_enabled metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node", "vm_name", "vm_id")
		}
	})

	t.Run("vm_cpu_cores", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vm_cpu_cores")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_vm_cpu_cores metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node", "vm_name", "vm_id")
		}
	})

	t.Run("vm_ram_bytes", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vm_ram_bytes")
		if len(found) == 0 {
			t.Fatal("Expected at least one vergeos_vm_ram_bytes metric")
		}
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node", "vm_name", "vm_id")
		}
	})

	t.Run("vm_nic_tx_bytes", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vm_nic_tx_bytes_total")
		t.Logf("Found %d VM NIC TX bytes metrics", len(found))
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node", "vm_name", "vm_id", "nic_name")
		}
	})

	t.Run("vm_nic_rx_bytes", func(t *testing.T) {
		found := filterMetrics(metrics, "vergeos_vm_nic_rx_bytes_total")
		t.Logf("Found %d VM NIC RX bytes metrics", len(found))
		for _, m := range found {
			assertHasLabels(t, m, "system_name", "cluster", "node", "vm_name", "vm_id", "nic_name")
		}
	})
}

// collectMetrics gathers all metrics from a collector as text lines.
func collectMetrics(t *testing.T, c prometheus.Collector) []string {
	t.Helper()
	registry := prometheus.NewRegistry()
	registry.MustRegister(c)

	gathered, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Convert to text format for easier inspection
	var lines []string
	for _, mf := range gathered {
		text, err := testutil.GatherAndCount(registry, mf.GetName())
		_ = err
		// Use the raw metric family for line-level checks
		for _, m := range mf.GetMetric() {
			line := mf.GetName()
			for _, lp := range m.GetLabel() {
				line += " " + lp.GetName() + "=" + lp.GetValue()
			}
			lines = append(lines, line)
		}
		t.Logf("Metric family %s: %d series", mf.GetName(), text)
	}
	return lines
}

// filterMetrics returns lines matching the given metric name prefix.
func filterMetrics(lines []string, name string) []string {
	var out []string
	for _, l := range lines {
		if strings.HasPrefix(l, name) {
			out = append(out, l)
		}
	}
	return out
}

// assertHasLabels checks that a metric line contains all expected label names.
func assertHasLabels(t *testing.T, line string, labels ...string) {
	t.Helper()
	for _, label := range labels {
		if !strings.Contains(line, label+"=") {
			t.Errorf("Metric %q missing label %q", line, label)
		}
	}
}
