package collectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestVMCollector(t *testing.T) {
	// Create a mock server to simulate the VergeOS API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add basic auth check
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Handle different API endpoints
		switch {
		case strings.Contains(r.URL.Path, "/api/v4/settings"):
			// Return system name
			settings := []Setting{
				{
					Key:   "cloud_name",
					Value: "test-system",
				},
			}
			json.NewEncoder(w).Encode(settings)

		case strings.Contains(r.URL.Path, "/api/v4/machines/1"):
			// Return detailed VM 1 (running)
			response := map[string]interface{}{
				"$key":    1,
				"name":    "test-vm-1",
				"cluster": "test-cluster",
				"status": map[string]interface{}{
					"running": true,
					"node":    10,
					"status":  "running",
				},
				"stats": map[string]interface{}{
					"total_cpu":  25.5,
					"user_cpu":   15.0,
					"system_cpu": 8.5,
					"iowait_cpu": 2.0,
				},
			}
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/api/v4/machines/2"):
			// Return detailed VM 2 (powered off)
			response := map[string]interface{}{
				"$key":    2,
				"name":    "test-vm-2",
				"cluster": "test-cluster",
				"status": map[string]interface{}{
					"running": false,
					"node":    0,
					"status":  "stopped",
				},
				"stats": map[string]interface{}{
					"total_cpu":  0,
					"user_cpu":   0,
					"system_cpu": 0,
					"iowait_cpu": 0,
				},
			}
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/api/v4/machines"):
			// Return list of VMs (excluding snapshots)
			vms := []map[string]interface{}{
				{
					"$key":        1,
					"name":        "test-vm-1",
					"type":        "vm",
					"is_snapshot": false,
				},
				{
					"$key":        2,
					"name":        "test-vm-2",
					"type":        "vm",
					"is_snapshot": false,
				},
			}
			json.NewEncoder(w).Encode(vms)

		case strings.Contains(r.URL.Path, "/api/v4/nodes/10"):
			// Return node name
			response := map[string]interface{}{
				"name": "node1",
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create VM collector with mock server
	collector := NewVMCollector(mockServer.URL, mockServer.Client(), "testuser", "testpass")

	// Register collector
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// Expected metrics
	expected := `
# HELP vergeos_vm_cpu_iowait IO Wait CPU usage percentage for the VM
# TYPE vergeos_vm_cpu_iowait gauge
vergeos_vm_cpu_iowait{cluster="test-cluster",node="",system_name="test-system",vm_id="2",vm_name="test-vm-2"} 0
vergeos_vm_cpu_iowait{cluster="test-cluster",node="node1",system_name="test-system",vm_id="1",vm_name="test-vm-1"} 2
# HELP vergeos_vm_cpu_system System CPU usage percentage for the VM
# TYPE vergeos_vm_cpu_system gauge
vergeos_vm_cpu_system{cluster="test-cluster",node="",system_name="test-system",vm_id="2",vm_name="test-vm-2"} 0
vergeos_vm_cpu_system{cluster="test-cluster",node="node1",system_name="test-system",vm_id="1",vm_name="test-vm-1"} 8.5
# HELP vergeos_vm_cpu_total Total CPU usage percentage for the VM
# TYPE vergeos_vm_cpu_total gauge
vergeos_vm_cpu_total{cluster="test-cluster",node="",system_name="test-system",vm_id="2",vm_name="test-vm-2"} 0
vergeos_vm_cpu_total{cluster="test-cluster",node="node1",system_name="test-system",vm_id="1",vm_name="test-vm-1"} 25.5
# HELP vergeos_vm_cpu_user User CPU usage percentage for the VM
# TYPE vergeos_vm_cpu_user gauge
vergeos_vm_cpu_user{cluster="test-cluster",node="",system_name="test-system",vm_id="2",vm_name="test-vm-2"} 0
vergeos_vm_cpu_user{cluster="test-cluster",node="node1",system_name="test-system",vm_id="1",vm_name="test-vm-1"} 15
`

	// Test metrics
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Errorf("Metrics do not match expected values: %v", err)
	}
}

func TestVMCollectorNoVMs(t *testing.T) {
	// Create a mock server that returns no VMs
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case strings.Contains(r.URL.Path, "/api/v4/settings"):
			settings := []Setting{
				{
					Key:   "cloud_name",
					Value: "test-system",
				},
			}
			json.NewEncoder(w).Encode(settings)

		case strings.Contains(r.URL.Path, "/api/v4/machines"):
			// Return empty list
			json.NewEncoder(w).Encode([]map[string]interface{}{})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	collector := NewVMCollector(mockServer.URL, mockServer.Client(), "testuser", "testpass")

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// With no VMs, we should get empty metrics (just the type/help headers)
	expected := `
# HELP vergeos_vm_cpu_iowait IO Wait CPU usage percentage for the VM
# TYPE vergeos_vm_cpu_iowait gauge
# HELP vergeos_vm_cpu_system System CPU usage percentage for the VM
# TYPE vergeos_vm_cpu_system gauge
# HELP vergeos_vm_cpu_total Total CPU usage percentage for the VM
# TYPE vergeos_vm_cpu_total gauge
# HELP vergeos_vm_cpu_user User CPU usage percentage for the VM
# TYPE vergeos_vm_cpu_user gauge
`

	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Errorf("Metrics do not match expected values: %v", err)
	}
}
