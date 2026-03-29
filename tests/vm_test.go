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

func TestVMCollector(t *testing.T) {
	config := DefaultMockConfig()

	vms := []VMMock{
		{Key: 1, Name: "web-server", Machine: 101, Cluster: 1, IsSnapshot: false, PowerState: true, Enabled: true, CPUCores: 4, RAM: 8192},
		{Key: 2, Name: "db-server", Machine: 102, Cluster: 1, IsSnapshot: false, PowerState: false, Enabled: true, CPUCores: 8, RAM: 16384},
		{Key: 99, Name: "snap-vm", Machine: 199, Cluster: 1, IsSnapshot: true, PowerState: false, Enabled: false, CPUCores: 2, RAM: 4096},
	}

	clusters := []ClusterMock{
		{Key: 1, Name: "compute-cluster", Enabled: true},
	}

	allMachineStats := []MachineStatsMock{
		{Key: 1, Machine: 101, TotalCPU: 75, UserCPU: 50, SystemCPU: 20, IOWaitCPU: 5},
	}

	allMachineStatuses := []MachineStatusMock{
		{Key: 1, Machine: 101, Running: true, Status: "running", State: "online", Node: 1, NodeName: "node1"},
		{Key: 2, Machine: 102, Running: false, Status: "stopped", State: "offline"},
	}

	allNICs := []MachineNICMock{
		{Key: 1, Machine: 101, Name: "net0", Stats: &MachineNICStatsMock{Key: 1, TxPckts: 1000, RxPckts: 2000, TxBytes: 500000, RxBytes: 1000000}},
		{Key: 2, Machine: 101, Name: "net1", Stats: &MachineNICStatsMock{Key: 2, TxPckts: 300, RxPckts: 400, TxBytes: 150000, RxBytes: 200000}},
	}

	allDrives := []VMDriveMock{
		{Key: 10, Machine: 101, Name: "drive0", Interface: "virtio-scsi", Media: "disk", SizeBytes: 107374182400, UsedBytes: 53687091200, Enabled: true},
		{Key: 11, Machine: 101, Name: "drive1", Interface: "virtio-scsi", Media: "disk", SizeBytes: 214748364800, UsedBytes: 10737418240, Enabled: true},
		{Key: 12, Machine: 102, Name: "drive0", Interface: "virtio-scsi", Media: "disk", SizeBytes: 53687091200, UsedBytes: 21474836480, Enabled: true},
	}

	allDriveStats := []MachineDriveStatsMock{
		{Key: 1, ParentDrive: 10, Reads: 100000, Writes: 50000, ReadBytes: 409600000, WriteBytes: 204800000, ServiceTime: 0.5, Util: 15.2, Physical: false},
		{Key: 2, ParentDrive: 11, Reads: 5000, Writes: 2000, ReadBytes: 20480000, WriteBytes: 8192000, ServiceTime: 0.8, Util: 3.1, Physical: false},
		{Key: 3, ParentDrive: 12, Reads: 200, Writes: 100, ReadBytes: 819200, WriteBytes: 409600, ServiceTime: 0.3, Util: 1.0, Physical: false},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/vms"):
			filter := r.URL.Query().Get("filter")
			if strings.Contains(filter, "is_snapshot eq false") {
				var filtered []VMMock
				for _, vm := range vms {
					if !vm.IsSnapshot {
						filtered = append(filtered, vm)
					}
				}
				WriteJSONResponse(w, filtered)
			} else {
				WriteJSONResponse(w, vms)
			}
			return true

		case strings.Contains(r.URL.Path, "/machine_drive_stats"):
			WriteJSONResponse(w, allDriveStats)
			return true

		case strings.Contains(r.URL.Path, "/machine_drives"):
			WriteJSONResponse(w, allDrives)
			return true

		case strings.Contains(r.URL.Path, "/machine_stats"):
			WriteJSONResponse(w, allMachineStats)
			return true

		case strings.Contains(r.URL.Path, "/machine_status"):
			WriteJSONResponse(w, allMachineStatuses)
			return true

		case strings.Contains(r.URL.Path, "/machine_nics"):
			WriteJSONResponse(w, allNICs)
			return true

		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewVMCollector(client, TestScrapeTimeout)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Verify all expected metrics exist
	expectedMetrics := map[string]bool{
		"vergeos_vm_cpu_total":              false,
		"vergeos_vm_cpu_user":               false,
		"vergeos_vm_cpu_system":             false,
		"vergeos_vm_cpu_iowait":             false,
		"vergeos_vm_running":                false,
		"vergeos_vm_enabled":                false,
		"vergeos_vm_cpu_cores":              false,
		"vergeos_vm_ram_bytes":              false,
		"vergeos_vm_nic_tx_bytes_total":     false,
		"vergeos_vm_nic_rx_bytes_total":     false,
		"vergeos_vm_nic_tx_packets_total":   false,
		"vergeos_vm_nic_rx_packets_total":   false,
		"vergeos_vm_disk_size_bytes":        false,
		"vergeos_vm_disk_used_bytes":        false,
		"vergeos_vm_disk_read_ops_total":    false,
		"vergeos_vm_disk_write_ops_total":   false,
		"vergeos_vm_disk_read_bytes_total":  false,
		"vergeos_vm_disk_write_bytes_total": false,
		"vergeos_vm_disk_util":              false,
		"vergeos_vm_disk_service_time":      false,
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

	t.Run("cpu_total", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_cpu_total Total CPU usage percentage
			# TYPE vergeos_vm_cpu_total gauge
			vergeos_vm_cpu_total{cluster="compute-cluster",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 75
			vergeos_vm_cpu_total{cluster="compute-cluster",node="",system_name="testcloud",vm_id="2",vm_name="db-server"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_cpu_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("running", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_running Whether the VM is running (1=running, 0=not running)
			# TYPE vergeos_vm_running gauge
			vergeos_vm_running{cluster="compute-cluster",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 1
			vergeos_vm_running{cluster="compute-cluster",node="",system_name="testcloud",vm_id="2",vm_name="db-server"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_running"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("enabled", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_enabled Whether the VM is enabled (1=enabled, 0=disabled)
			# TYPE vergeos_vm_enabled gauge
			vergeos_vm_enabled{cluster="compute-cluster",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 1
			vergeos_vm_enabled{cluster="compute-cluster",node="",system_name="testcloud",vm_id="2",vm_name="db-server"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_enabled"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("cpu_cores", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_cpu_cores Number of configured CPU cores
			# TYPE vergeos_vm_cpu_cores gauge
			vergeos_vm_cpu_cores{cluster="compute-cluster",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 4
			vergeos_vm_cpu_cores{cluster="compute-cluster",node="",system_name="testcloud",vm_id="2",vm_name="db-server"} 8
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_cpu_cores"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("ram_bytes", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_ram_bytes Configured RAM in bytes
			# TYPE vergeos_vm_ram_bytes gauge
			vergeos_vm_ram_bytes{cluster="compute-cluster",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 8.589934592e+09
			vergeos_vm_ram_bytes{cluster="compute-cluster",node="",system_name="testcloud",vm_id="2",vm_name="db-server"} 1.7179869184e+10
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_ram_bytes"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("nic_tx_bytes", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_nic_tx_bytes_total Total transmitted bytes
			# TYPE vergeos_vm_nic_tx_bytes_total counter
			vergeos_vm_nic_tx_bytes_total{cluster="compute-cluster",nic_name="net0",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 500000
			vergeos_vm_nic_tx_bytes_total{cluster="compute-cluster",nic_name="net1",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 150000
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_nic_tx_bytes_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("nic_rx_bytes", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_nic_rx_bytes_total Total received bytes
			# TYPE vergeos_vm_nic_rx_bytes_total counter
			vergeos_vm_nic_rx_bytes_total{cluster="compute-cluster",nic_name="net0",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 1e+06
			vergeos_vm_nic_rx_bytes_total{cluster="compute-cluster",nic_name="net1",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 200000
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_nic_rx_bytes_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("nic_tx_packets", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_nic_tx_packets_total Total transmitted packets
			# TYPE vergeos_vm_nic_tx_packets_total counter
			vergeos_vm_nic_tx_packets_total{cluster="compute-cluster",nic_name="net0",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 1000
			vergeos_vm_nic_tx_packets_total{cluster="compute-cluster",nic_name="net1",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 300
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_nic_tx_packets_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("disk_size_bytes", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_disk_size_bytes Configured disk size in bytes
			# TYPE vergeos_vm_disk_size_bytes gauge
			vergeos_vm_disk_size_bytes{cluster="compute-cluster",disk_name="drive0",interface="virtio-scsi",media="disk",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 1.073741824e+11
			vergeos_vm_disk_size_bytes{cluster="compute-cluster",disk_name="drive1",interface="virtio-scsi",media="disk",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 2.147483648e+11
			vergeos_vm_disk_size_bytes{cluster="compute-cluster",disk_name="drive0",interface="virtio-scsi",media="disk",node="",system_name="testcloud",vm_id="2",vm_name="db-server"} 5.36870912e+10
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_disk_size_bytes"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("disk_read_ops", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_disk_read_ops_total Total disk read operations
			# TYPE vergeos_vm_disk_read_ops_total counter
			vergeos_vm_disk_read_ops_total{cluster="compute-cluster",disk_name="drive0",interface="virtio-scsi",media="disk",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 100000
			vergeos_vm_disk_read_ops_total{cluster="compute-cluster",disk_name="drive1",interface="virtio-scsi",media="disk",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 5000
			vergeos_vm_disk_read_ops_total{cluster="compute-cluster",disk_name="drive0",interface="virtio-scsi",media="disk",node="",system_name="testcloud",vm_id="2",vm_name="db-server"} 200
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_disk_read_ops_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("disk_util", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_disk_util Disk I/O utilization percentage
			# TYPE vergeos_vm_disk_util gauge
			vergeos_vm_disk_util{cluster="compute-cluster",disk_name="drive0",interface="virtio-scsi",media="disk",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 15.2
			vergeos_vm_disk_util{cluster="compute-cluster",disk_name="drive1",interface="virtio-scsi",media="disk",node="node1",system_name="testcloud",vm_id="1",vm_name="web-server"} 3.1
			vergeos_vm_disk_util{cluster="compute-cluster",disk_name="drive0",interface="virtio-scsi",media="disk",node="",system_name="testcloud",vm_id="2",vm_name="db-server"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_disk_util"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestVMCollector_SnapshotFiltering(t *testing.T) {
	config := DefaultMockConfig()

	vms := []VMMock{
		{Key: 1, Name: "real-vm", Machine: 101, Cluster: 1, IsSnapshot: false, PowerState: true, Enabled: true, CPUCores: 2, RAM: 4096},
		{Key: 2, Name: "snapshot-vm", Machine: 102, Cluster: 1, IsSnapshot: true, PowerState: false, Enabled: false, CPUCores: 1, RAM: 2048},
	}

	clusters := []ClusterMock{
		{Key: 1, Name: "cluster1", Enabled: true},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/vms"):
			filter := r.URL.Query().Get("filter")
			if strings.Contains(filter, "is_snapshot eq false") {
				var filtered []VMMock
				for _, vm := range vms {
					if !vm.IsSnapshot {
						filtered = append(filtered, vm)
					}
				}
				WriteJSONResponse(w, filtered)
			} else {
				WriteJSONResponse(w, vms)
			}
			return true

		case strings.Contains(r.URL.Path, "/machine_drive_stats"):
			WriteJSONResponse(w, []MachineDriveStatsMock{})
			return true

		case strings.Contains(r.URL.Path, "/machine_drives"):
			WriteJSONResponse(w, []VMDriveMock{})
			return true

		case strings.Contains(r.URL.Path, "/machine_stats"):
			WriteJSONResponse(w, []MachineStatsMock{
				{Key: 1, Machine: 101, TotalCPU: 30, UserCPU: 20, SystemCPU: 8, IOWaitCPU: 2},
			})
			return true

		case strings.Contains(r.URL.Path, "/machine_status"):
			WriteJSONResponse(w, []MachineStatusMock{
				{Key: 1, Machine: 101, Running: true, Status: "running", State: "online", Node: 1, NodeName: "node1"},
			})
			return true

		case strings.Contains(r.URL.Path, "/machine_nics"):
			WriteJSONResponse(w, []MachineNICMock{})
			return true

		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewVMCollector(client, TestScrapeTimeout)

	t.Run("excludes_snapshots", func(t *testing.T) {
		expected := `
			# HELP vergeos_vm_cpu_total Total CPU usage percentage
			# TYPE vergeos_vm_cpu_total gauge
			vergeos_vm_cpu_total{cluster="cluster1",node="node1",system_name="testcloud",vm_id="1",vm_name="real-vm"} 30
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vm_cpu_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestVMCollector_StaleMetrics(t *testing.T) {
	config := DefaultMockConfig()
	vmCount := 2

	clusters := []ClusterMock{
		{Key: 1, Name: "cluster1", Enabled: true},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/vms"):
			var vms []VMMock
			for i := 1; i <= vmCount; i++ {
				vms = append(vms, VMMock{
					Key: i, Name: "vm-" + intToStr(i), Machine: 100 + i, Cluster: 1,
					PowerState: true, Enabled: true, CPUCores: 2, RAM: 4096,
				})
			}
			WriteJSONResponse(w, vms)
			return true

		case strings.Contains(r.URL.Path, "/machine_drive_stats"):
			WriteJSONResponse(w, []MachineDriveStatsMock{})
			return true

		case strings.Contains(r.URL.Path, "/machine_drives"):
			WriteJSONResponse(w, []VMDriveMock{})
			return true

		case strings.Contains(r.URL.Path, "/machine_stats"):
			var stats []MachineStatsMock
			for i := 1; i <= vmCount; i++ {
				stats = append(stats, MachineStatsMock{
					Key: i, Machine: 100 + i, TotalCPU: 50, UserCPU: 30, SystemCPU: 15, IOWaitCPU: 5,
				})
			}
			WriteJSONResponse(w, stats)
			return true

		case strings.Contains(r.URL.Path, "/machine_status"):
			var statuses []MachineStatusMock
			for i := 1; i <= vmCount; i++ {
				statuses = append(statuses, MachineStatusMock{
					Key: i, Machine: 100 + i, Running: true, Status: "running", State: "online", Node: 1, NodeName: "node1",
				})
			}
			WriteJSONResponse(w, statuses)
			return true

		case strings.Contains(r.URL.Path, "/machine_nics"):
			WriteJSONResponse(w, []MachineNICMock{})
			return true

		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewVMCollector(client, TestScrapeTimeout)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// First scrape — 2 VMs
	metrics1, err := registry.Gather()
	if err != nil {
		t.Fatalf("First gather failed: %v", err)
	}

	countCPUMetrics := func(metrics []*dto.MetricFamily) int {
		for _, mf := range metrics {
			if mf.GetName() == "vergeos_vm_cpu_total" {
				return len(mf.GetMetric())
			}
		}
		return 0
	}

	if count := countCPUMetrics(metrics1); count != 2 {
		t.Errorf("First scrape: expected 2 cpu_total metrics, got %d", count)
	}

	// Remove a VM
	vmCount = 1

	// Second scrape — only 1 VM (no stale metrics)
	metrics2, err := registry.Gather()
	if err != nil {
		t.Fatalf("Second gather failed: %v", err)
	}

	if count := countCPUMetrics(metrics2); count != 1 {
		t.Errorf("Second scrape: expected 1 cpu_total metric (no stale), got %d", count)
	}
}
