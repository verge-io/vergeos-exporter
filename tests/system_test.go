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

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/update_settings/1"):
			WriteJSONResponse(w, UpdateSettingsMock{Key: 1, Source: 3, Branch: 35, BranchName: "stable-4.13"})
			return true
		case strings.Contains(r.URL.Path, "/update_source_packages"):
			WriteJSONResponse(w, []UpdateSourcePackageMock{
				{Key: 1, Name: "ybos", Branch: 35, Source: 3, Version: "4.13.1"},
			})
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewSystemCollector(client, TestScrapeTimeout)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Verify expected metrics
	expectedMetrics := map[string]bool{
		"vergeos_system_version":        false,
		"vergeos_system_info":           false,
		"vergeos_system_branch":         false,
		"vergeos_system_version_latest": false,
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
			vergeos_system_info{branch="stable-4.13",current_version="26.0.2.1",hash="abc123def456",latest_version="4.13.1",system_name="testcloud"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_system_info"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("system_branch", func(t *testing.T) {
		expected := `
			# HELP vergeos_system_branch Update branch of the VergeOS system (always 1, branch in label)
			# TYPE vergeos_system_branch gauge
			vergeos_system_branch{branch="stable-4.13",system_name="testcloud"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_system_branch"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("system_version_latest", func(t *testing.T) {
		expected := `
			# HELP vergeos_system_version_latest Latest available version of the VergeOS system (always 1, version in label)
			# TYPE vergeos_system_version_latest gauge
			vergeos_system_version_latest{system_name="testcloud",version="4.13.1"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_system_version_latest"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestSystemCollector_StaleMetrics(t *testing.T) {
	currentVersion := "26.0.2.1"
	currentHash := "abc123"

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
		case strings.Contains(r.URL.Path, "/update_settings/1"):
			WriteJSONResponse(w, UpdateSettingsMock{Key: 1, Source: 3, Branch: 35, BranchName: "stable"})
		case strings.Contains(r.URL.Path, "/update_source_packages"):
			WriteJSONResponse(w, []UpdateSourcePackageMock{
				{Key: 1, Name: "ybos", Branch: 35, Source: 3, Version: "4.13.0"},
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewSystemCollector(client, TestScrapeTimeout)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// First scrape
	metrics1, err := registry.Gather()
	if err != nil {
		t.Fatalf("First gather failed: %v", err)
	}

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

	// Simulate version change
	currentVersion = "26.0.3.0"
	currentHash = "def456"

	metrics2, err := registry.Gather()
	if err != nil {
		t.Fatalf("Second gather failed: %v", err)
	}

	if count := countVersionMetrics(metrics2, "26.0.3.0"); count != 1 {
		t.Errorf("Second scrape: expected metric with version 26.0.3.0, got %d", count)
	}

	if count := countVersionMetrics(metrics2, "26.0.2.1"); count != 0 {
		t.Errorf("Second scrape: found stale metric with old version 26.0.2.1")
	}
}

func TestSystemCollector_DifferentVersion(t *testing.T) {
	config := MockServerConfig{
		CloudName: "production-cloud",
		Version:   "26.1.0.0",
		Hash:      "xyz789abc123def456",
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/update_settings/1"):
			WriteJSONResponse(w, UpdateSettingsMock{Key: 1, Source: 1, Branch: 10, BranchName: "stable-4.14"})
			return true
		case strings.Contains(r.URL.Path, "/update_source_packages"):
			WriteJSONResponse(w, []UpdateSourcePackageMock{
				{Key: 1, Name: "ybos", Branch: 10, Source: 1, Version: "4.14.0"},
			})
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewSystemCollector(client, TestScrapeTimeout)

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

	t.Run("info_with_branch_and_latest", func(t *testing.T) {
		expected := `
			# HELP vergeos_system_info Information about the VergeOS system
			# TYPE vergeos_system_info gauge
			vergeos_system_info{branch="stable-4.14",current_version="26.1.0.0",hash="xyz789abc123def456",latest_version="4.14.0",system_name="production-cloud"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_system_info"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}
