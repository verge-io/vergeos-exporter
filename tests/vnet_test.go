package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vergeos-exporter/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// newVNetMockServer builds a mock server serving /vnets and
// /vnet_monitor_stats_history_short (filtered by "vnet eq <id>").
func newVNetMockServer(t *testing.T, config MockServerConfig, clusters []ClusterMock, vnets []VNetMock, stats []VNetMonitorStatsMock) *httptest.Server {
	return NewBaseMockServer(t, config, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case strings.Contains(r.URL.Path, "/clusters"):
			WriteJSONResponse(w, clusters)
			return true
		case strings.Contains(r.URL.Path, "/vnet_monitor_stats_history_short"):
			filter := r.URL.Query().Get("filter")
			matched := []VNetMonitorStatsMock{}
			for _, s := range stats {
				if filter == fmt.Sprintf("vnet eq %d", s.VNet) {
					matched = append(matched, s)
				}
			}
			WriteJSONResponse(w, matched)
			return true
		case strings.Contains(r.URL.Path, "/vnets"):
			WriteJSONResponse(w, vnets)
			return true
		}
		return false
	})
}

func TestVNetCollector_Metrics(t *testing.T) {
	config := DefaultMockConfig()
	config.CloudName = "test-cloud"

	clusters := []ClusterMock{
		{Key: 1, Name: "cluster1", Enabled: true},
	}

	vnets := []VNetMock{
		// External network with gateway monitoring and stats available
		{Key: 10, Name: "external", Enabled: true, PowerState: true, Cluster: 1,
			Type: "external", Layer2Type: "", MonitorGateway: true},
		// Internal network, no monitoring
		{Key: 11, Name: "internal", Enabled: true, PowerState: false, Cluster: 1,
			Type: "internal", Layer2Type: "vxlan", MonitorGateway: false},
		// Monitoring enabled but no stats row yet
		{Key: 12, Name: "newnet", Enabled: false, PowerState: false, Cluster: 1,
			Type: "internal", Layer2Type: "vlan", MonitorGateway: true},
	}

	stats := []VNetMonitorStatsMock{
		{Key: 1, VNet: 10, Sent: 100, Quality: 98, DroppedPct: 2,
			LatencyUSAvg: 1500, LatencyUSPeak: 9000, Duplicates: 1, Truncated: 2,
			Dropped: 3, BadChecksums: 4, BadData: 5, Timestamp: 1700000000},
	}

	mockServer := newVNetMockServer(t, config, clusters, vnets, stats)
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewVNetCollector(client, TestScrapeTimeout)

	t.Run("enabled", func(t *testing.T) {
		expected := `
			# HELP vergeos_vnet_enabled Whether the virtual network is enabled (1=enabled, 0=disabled)
			# TYPE vergeos_vnet_enabled gauge
			vergeos_vnet_enabled{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 1
			vergeos_vnet_enabled{cluster="cluster1",layer2_type="vxlan",system_name="test-cloud",type="internal",vnet_id="11",vnet_name="internal"} 1
			vergeos_vnet_enabled{cluster="cluster1",layer2_type="vlan",system_name="test-cloud",type="internal",vnet_id="12",vnet_name="newnet"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vnet_enabled"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("powerstate", func(t *testing.T) {
		expected := `
			# HELP vergeos_vnet_powerstate Whether the virtual network is running (1=running, 0=stopped)
			# TYPE vergeos_vnet_powerstate gauge
			vergeos_vnet_powerstate{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 1
			vergeos_vnet_powerstate{cluster="cluster1",layer2_type="vxlan",system_name="test-cloud",type="internal",vnet_id="11",vnet_name="internal"} 0
			vergeos_vnet_powerstate{cluster="cluster1",layer2_type="vlan",system_name="test-cloud",type="internal",vnet_id="12",vnet_name="newnet"} 0
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vnet_powerstate"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("monitor_gateway", func(t *testing.T) {
		expected := `
			# HELP vergeos_vnet_monitor_gateway Whether gateway monitoring is enabled for the virtual network (1=enabled, 0=disabled)
			# TYPE vergeos_vnet_monitor_gateway gauge
			vergeos_vnet_monitor_gateway{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 1
			vergeos_vnet_monitor_gateway{cluster="cluster1",layer2_type="vxlan",system_name="test-cloud",type="internal",vnet_id="11",vnet_name="internal"} 0
			vergeos_vnet_monitor_gateway{cluster="cluster1",layer2_type="vlan",system_name="test-cloud",type="internal",vnet_id="12",vnet_name="newnet"} 1
		`
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), "vergeos_vnet_monitor_gateway"); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})

	t.Run("monitor_stats", func(t *testing.T) {
		// Only vnet 10 has a stats row; vnet 12 has monitoring enabled but no
		// stats yet and must not emit monitor series or fail the scrape.
		expected := `
			# HELP vergeos_vnet_monitor_quality Gateway network quality percentage (100 - dropped_pct) from the latest sample
			# TYPE vergeos_vnet_monitor_quality gauge
			vergeos_vnet_monitor_quality{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 98
			# HELP vergeos_vnet_monitor_latency_usec_avg Average gateway latency in microseconds from the latest sample
			# TYPE vergeos_vnet_monitor_latency_usec_avg gauge
			vergeos_vnet_monitor_latency_usec_avg{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 1500
			# HELP vergeos_vnet_monitor_latency_usec_peak Peak gateway latency in microseconds from the latest sample
			# TYPE vergeos_vnet_monitor_latency_usec_peak gauge
			vergeos_vnet_monitor_latency_usec_peak{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 9000
			# HELP vergeos_vnet_monitor_dropped_pct Percentage of dropped monitoring packets in the latest sample
			# TYPE vergeos_vnet_monitor_dropped_pct gauge
			vergeos_vnet_monitor_dropped_pct{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 2
			# HELP vergeos_vnet_monitor_sent Monitoring packets sent in the latest gateway-monitoring sample
			# TYPE vergeos_vnet_monitor_sent gauge
			vergeos_vnet_monitor_sent{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 100
			# HELP vergeos_vnet_monitor_duplicates Duplicate monitoring packets in the latest sample
			# TYPE vergeos_vnet_monitor_duplicates gauge
			vergeos_vnet_monitor_duplicates{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 1
			# HELP vergeos_vnet_monitor_truncated Truncated monitoring packets in the latest sample
			# TYPE vergeos_vnet_monitor_truncated gauge
			vergeos_vnet_monitor_truncated{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 2
			# HELP vergeos_vnet_monitor_dropped Dropped monitoring packets in the latest sample
			# TYPE vergeos_vnet_monitor_dropped gauge
			vergeos_vnet_monitor_dropped{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 3
			# HELP vergeos_vnet_monitor_bad_checksums Monitoring packets with bad checksums in the latest sample
			# TYPE vergeos_vnet_monitor_bad_checksums gauge
			vergeos_vnet_monitor_bad_checksums{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 4
			# HELP vergeos_vnet_monitor_bad_data Monitoring packets with bad data in the latest sample
			# TYPE vergeos_vnet_monitor_bad_data gauge
			vergeos_vnet_monitor_bad_data{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 5
			# HELP vergeos_vnet_monitor_timestamp_seconds Unix timestamp of the latest gateway-monitoring sample
			# TYPE vergeos_vnet_monitor_timestamp_seconds gauge
			vergeos_vnet_monitor_timestamp_seconds{cluster="cluster1",layer2_type="",system_name="test-cloud",type="external",vnet_id="10",vnet_name="external"} 1.7e+09
		`
		metricNames := []string{
			"vergeos_vnet_monitor_sent",
			"vergeos_vnet_monitor_quality",
			"vergeos_vnet_monitor_dropped_pct",
			"vergeos_vnet_monitor_latency_usec_avg",
			"vergeos_vnet_monitor_latency_usec_peak",
			"vergeos_vnet_monitor_duplicates",
			"vergeos_vnet_monitor_truncated",
			"vergeos_vnet_monitor_dropped",
			"vergeos_vnet_monitor_bad_checksums",
			"vergeos_vnet_monitor_bad_data",
			"vergeos_vnet_monitor_timestamp_seconds",
		}
		if err := testutil.CollectAndCompare(collector, strings.NewReader(expected), metricNames...); err != nil {
			t.Errorf("Unexpected metric values: %v", err)
		}
	})
}

func TestVNetCollector_Describe(t *testing.T) {
	config := DefaultMockConfig()

	mockServer := NewBaseMockServer(t, config, nil)
	defer mockServer.Close()

	client := CreateTestSDKClient(t, mockServer.URL)
	collector := collectors.NewVNetCollector(client, TestScrapeTimeout)

	ch := make(chan *prometheus.Desc, 20)
	collector.Describe(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}

	if count != 14 {
		t.Errorf("Expected 14 descriptors, got %d", count)
	}
}
