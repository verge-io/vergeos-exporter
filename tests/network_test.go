package tests

import (
	"strings"
	"testing"

	"vergeos-exporter/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNetworkCollector_InfoMetric(t *testing.T) {
	config := DefaultMockConfig()
	config.CloudName = "test-cloud"

	mockServer := NewBaseMockServer(t, config, nil)
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewNetworkCollector(client)

	// Test that info metric is emitted
	expected := `
		# HELP vergeos_network_collector_info NetworkCollector status (1=active, metrics pending SDK support for node dashboard NICs)
		# TYPE vergeos_network_collector_info gauge
		vergeos_network_collector_info{system_name="test-cloud"} 1
	`

	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Errorf("Unexpected metrics: %v", err)
	}
}

func TestNetworkCollector_Describe(t *testing.T) {
	config := DefaultMockConfig()

	mockServer := NewBaseMockServer(t, config, nil)
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewNetworkCollector(client)

	// Verify Describe sends exactly 1 descriptor (info metric only)
	ch := make(chan *prometheus.Desc, 10)
	collector.Describe(ch)
	close(ch)

	count := 0
	for desc := range ch {
		count++
		if !strings.Contains(desc.String(), "vergeos_network_collector_info") {
			t.Errorf("Expected info metric descriptor, got: %s", desc.String())
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 descriptor (info metric only due to SDK gaps), got %d", count)
	}
}

func TestNetworkCollector_NoNICMetrics(t *testing.T) {
	// This test verifies that NIC metrics are NOT emitted (due to SDK gaps)
	config := DefaultMockConfig()
	config.CloudName = "test-cloud"

	mockServer := NewBaseMockServer(t, config, nil)
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewNetworkCollector(client)

	// Collect all metrics
	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	// Verify we only get the info metric, no NIC metrics
	metricCount := 0
	for metric := range ch {
		metricCount++
		desc := metric.Desc().String()
		// Ensure no NIC metrics are emitted
		nicMetrics := []string{
			"vergeos_nic_tx_packets_total",
			"vergeos_nic_rx_packets_total",
			"vergeos_nic_tx_bytes_total",
			"vergeos_nic_rx_bytes_total",
			"vergeos_nic_tx_errors_total",
			"vergeos_nic_rx_errors_total",
			"vergeos_nic_status",
		}
		for _, nicMetric := range nicMetrics {
			if strings.Contains(desc, nicMetric) {
				t.Errorf("NIC metric should not be emitted (SDK gaps): %s", nicMetric)
			}
		}
	}

	// Should only have the info metric
	if metricCount != 1 {
		t.Errorf("Expected 1 metric (info only), got %d", metricCount)
	}
}
