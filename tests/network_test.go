package tests

import (
	"net/http"
	"strings"
	"testing"

	"vergeos-exporter/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNetworkCollector_NICMetrics(t *testing.T) {
	config := DefaultMockConfig()
	config.CloudName = "test-cloud"

	nodes := []NodeMock{
		{ID: 1, Name: "node1", Physical: true, Cluster: 1, Machine: 101, IPMIStatus: "ok", RAM: 65536},
	}

	clusters := []ClusterMock{
		{Key: 1, Name: "cluster1", Enabled: true},
	}

	nics := []MachineNICMock{
		{
			Key: 1, Machine: 101, Name: "eno1",
			Stats:  &MachineNICStatsMock{Key: 1, TxPckts: 1000, RxPckts: 2000, TxBytes: 100000, RxBytes: 200000},
			Status: &MachineNICStatusMock{Key: 1, Status: "up", Speed: 10000},
		},
		{
			Key: 2, Machine: 101, Name: "eno2",
			Stats:  &MachineNICStatsMock{Key: 2, TxPckts: 500, RxPckts: 600, TxBytes: 50000, RxBytes: 60000},
			Status: &MachineNICStatusMock{Key: 2, Status: "down", Speed: 0},
		},
	}

	mockServer := NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		case strings.Contains(r.URL.Path, "/nodes") && strings.Contains(r.URL.RawQuery, "physical"):
			WriteJSONResponse(w, nodes)
			return true
		case strings.Contains(r.URL.Path, "/machine_nics"):
			WriteJSONResponse(w, nics)
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewNetworkCollector(client, TestScrapeTimeout)

	t.Run("tx_packets", func(t *testing.T) {
		expected := `
			# HELP vergeos_nic_tx_packets_total Total transmitted packets
			# TYPE vergeos_nic_tx_packets_total counter
			vergeos_nic_tx_packets_total{cluster="cluster1",interface="eno1",node_name="node1",system_name="test-cloud"} 1000
			vergeos_nic_tx_packets_total{cluster="cluster1",interface="eno2",node_name="node1",system_name="test-cloud"} 500
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_nic_tx_packets_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("rx_packets", func(t *testing.T) {
		expected := `
			# HELP vergeos_nic_rx_packets_total Total received packets
			# TYPE vergeos_nic_rx_packets_total counter
			vergeos_nic_rx_packets_total{cluster="cluster1",interface="eno1",node_name="node1",system_name="test-cloud"} 2000
			vergeos_nic_rx_packets_total{cluster="cluster1",interface="eno2",node_name="node1",system_name="test-cloud"} 600
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_nic_rx_packets_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("tx_bytes", func(t *testing.T) {
		expected := `
			# HELP vergeos_nic_tx_bytes_total Total transmitted bytes
			# TYPE vergeos_nic_tx_bytes_total counter
			vergeos_nic_tx_bytes_total{cluster="cluster1",interface="eno1",node_name="node1",system_name="test-cloud"} 100000
			vergeos_nic_tx_bytes_total{cluster="cluster1",interface="eno2",node_name="node1",system_name="test-cloud"} 50000
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_nic_tx_bytes_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("rx_bytes", func(t *testing.T) {
		expected := `
			# HELP vergeos_nic_rx_bytes_total Total received bytes
			# TYPE vergeos_nic_rx_bytes_total counter
			vergeos_nic_rx_bytes_total{cluster="cluster1",interface="eno1",node_name="node1",system_name="test-cloud"} 200000
			vergeos_nic_rx_bytes_total{cluster="cluster1",interface="eno2",node_name="node1",system_name="test-cloud"} 60000
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_nic_rx_bytes_total"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("nic_status", func(t *testing.T) {
		expected := `
			# HELP vergeos_nic_status NIC link status (1=up, 0=other)
			# TYPE vergeos_nic_status gauge
			vergeos_nic_status{cluster="cluster1",interface="eno1",node_name="node1",system_name="test-cloud"} 1
			vergeos_nic_status{cluster="cluster1",interface="eno2",node_name="node1",system_name="test-cloud"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_nic_status"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestNetworkCollector_Describe(t *testing.T) {
	config := DefaultMockConfig()

	mockServer := NewBaseMockServer(t, config, nil)
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewNetworkCollector(client, TestScrapeTimeout)

	// Verify Describe sends exactly 5 descriptors
	ch := make(chan *prometheus.Desc, 10)
	collector.Describe(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}

	if count != 5 {
		t.Errorf("Expected 5 descriptors, got %d", count)
	}
}

func TestNetworkCollector_MultipleNodes(t *testing.T) {
	config := DefaultMockConfig()

	nodes := []NodeMock{
		{ID: 1, Name: "node1", Physical: true, Cluster: 1, Machine: 101},
		{ID: 2, Name: "node2", Physical: true, Cluster: 1, Machine: 102},
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
		case strings.Contains(r.URL.Path, "/machine_nics"):
			WriteJSONResponse(w, []MachineNICMock{
				{Key: 1, Machine: 101, Name: "eno1",
					Stats:  &MachineNICStatsMock{Key: 1, TxPckts: 100, RxPckts: 200, TxBytes: 1000, RxBytes: 2000},
					Status: &MachineNICStatusMock{Key: 1, Status: "up", Speed: 10000}},
				{Key: 2, Machine: 102, Name: "eno1",
					Stats:  &MachineNICStatsMock{Key: 2, TxPckts: 300, RxPckts: 400, TxBytes: 3000, RxBytes: 4000},
					Status: &MachineNICStatusMock{Key: 2, Status: "up", Speed: 10000}},
			})
			return true
		}
		return false
	})
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewNetworkCollector(client, TestScrapeTimeout)

	expected := `
		# HELP vergeos_nic_tx_packets_total Total transmitted packets
		# TYPE vergeos_nic_tx_packets_total counter
		vergeos_nic_tx_packets_total{cluster="cluster1",interface="eno1",node_name="node1",system_name="testcloud"} 100
		vergeos_nic_tx_packets_total{cluster="cluster1",interface="eno1",node_name="node2",system_name="testcloud"} 300
	`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_nic_tx_packets_total"); err != nil {
		t.Errorf("Unexpected metric values: %v", err)
	}
}
