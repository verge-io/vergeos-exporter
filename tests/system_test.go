package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vergeos-exporter/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

func TestSystemCollector(t *testing.T) {
	config := MockServerConfig{
		CloudName: "testcloud",
		Version:   "26.0.2.1",
		Hash:      "abc123def456",
	}

	mockServer := NewBaseMockServer(t, config, nil)
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewSystemCollector(client)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Verify expected metrics
	expectedMetrics := map[string]bool{
		"vergeos_system_version": false,
		"vergeos_system_info":    false,
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
	t.Run("system_version", func(t *testing.T) {
		expected := `
			# HELP vergeos_system_version Current version of the VergeOS system (always 1, version in label)
			# TYPE vergeos_system_version gauge
			vergeos_system_version{system_name="testcloud",version="26.0.2.1"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_system_version"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("system_info", func(t *testing.T) {
		expected := `
			# HELP vergeos_system_info Information about the VergeOS system
			# TYPE vergeos_system_info gauge
			vergeos_system_info{hash="abc123def456",system_name="testcloud",version="26.0.2.1"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_system_info"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestSystemCollector_StaleMetrics(t *testing.T) {
	// This test verifies that the MustNewConstMetric pattern doesn't produce stale metrics
	// when version changes between scrapes

	currentVersion := "26.0.2.1"
	currentHash := "abc123"

	// Create a custom mock server to allow dynamic version changes
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case IsVersionCheck(r):
			WriteJSONResponse(w, map[string]interface{}{
				"name":    "v4",
				"version": currentVersion,
				"hash":    currentHash,
			})
		case IsSettingsRequest(r):
			WriteJSONResponse(w, []map[string]string{
				{"key": "cloud_name", "value": "testcloud"},
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewSystemCollector(client)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// First scrape
	metrics1, err := registry.Gather()
	if err != nil {
		t.Fatalf("First gather failed: %v", err)
	}

	// Helper to count metrics with a specific version label
	countVersionMetrics := func(metrics []*dto.MetricFamily, version string) int {
		for _, mf := range metrics {
			if mf.GetName() == "vergeos_system_version" {
				for _, m := range mf.GetMetric() {
					for _, label := range m.GetLabel() {
						if label.GetName() == "version" && label.GetValue() == version {
							return 1
						}
					}
				}
			}
		}
		return 0
	}

	if count := countVersionMetrics(metrics1, "26.0.2.1"); count != 1 {
		t.Errorf("First scrape: expected metric with version 26.0.2.1, got %d", count)
	}

	// Simulate version change (e.g., after system update)
	currentVersion = "26.0.3.0"
	currentHash = "def456"

	// Second scrape - should only have new version (no stale metrics)
	metrics2, err := registry.Gather()
	if err != nil {
		t.Fatalf("Second gather failed: %v", err)
	}

	// Should have new version
	if count := countVersionMetrics(metrics2, "26.0.3.0"); count != 1 {
		t.Errorf("Second scrape: expected metric with version 26.0.3.0, got %d", count)
	}

	// Should NOT have old version (stale metric check)
	if count := countVersionMetrics(metrics2, "26.0.2.1"); count != 0 {
		t.Errorf("Second scrape: found stale metric with old version 26.0.2.1")
	}
}

func TestSystemCollector_DifferentVersion(t *testing.T) {
	// Test with a different version format
	config := MockServerConfig{
		CloudName: "production-cloud",
		Version:   "26.1.0.0",
		Hash:      "xyz789abc123def456",
	}

	mockServer := NewBaseMockServer(t, config, nil)
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewSystemCollector(client)

	t.Run("different_system_name", func(t *testing.T) {
		expected := `
			# HELP vergeos_system_version Current version of the VergeOS system (always 1, version in label)
			# TYPE vergeos_system_version gauge
			vergeos_system_version{system_name="production-cloud",version="26.1.0.0"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_system_version"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("hash_in_info", func(t *testing.T) {
		expected := `
			# HELP vergeos_system_info Information about the VergeOS system
			# TYPE vergeos_system_info gauge
			vergeos_system_info{hash="xyz789abc123def456",system_name="production-cloud",version="26.1.0.0"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_system_info"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}
