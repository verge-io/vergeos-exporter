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

	url := getEnvOrDefault("VERGE_URL", "https://midgard.subether.me")
	user := getEnvOrDefault("VERGE_USERNAME", "admin")
	pass := getEnvOrDefault("VERGE_PASSWORD", "jenifer8")

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
