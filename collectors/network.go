package collectors

import (
	"context"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/goVergeOS"
)

// NetworkCollector collects metrics about physical node network interfaces.
//
// NOTE: All physical NIC metrics have been removed due to SDK gaps.
// The SDK's Node struct does not capture the dashboard fields needed for NIC metrics:
// - machine.nics[].name - NIC name
// - machine.nics[].status - NIC status (up/down)
// - machine.nics[].stats.tx_packets, rx_packets, tx_bytes, rx_bytes, tx_errors, rx_errors
//
// See .claude/GAPS.md for details. Once the SDK adds Node dashboard support,
// NIC metrics can be restored.
//
// Metrics removed:
// - vergeos_nic_tx_packets_total
// - vergeos_nic_rx_packets_total
// - vergeos_nic_tx_bytes_total
// - vergeos_nic_rx_bytes_total
// - vergeos_nic_tx_errors_total
// - vergeos_nic_rx_errors_total
// - vergeos_nic_status
type NetworkCollector struct {
	BaseCollector
	mutex sync.Mutex

	// Placeholder metric to indicate collector is registered but has no data
	// This helps users understand the collector exists but SDK support is pending
	collectorInfo *prometheus.Desc
}

// NewNetworkCollector creates a new NetworkCollector.
func NewNetworkCollector(client *vergeos.Client) *NetworkCollector {
	return &NetworkCollector{
		BaseCollector: *NewBaseCollector(client),
		collectorInfo: prometheus.NewDesc(
			"vergeos_network_collector_info",
			"NetworkCollector status (1=active, metrics pending SDK support for node dashboard NICs)",
			[]string{"system_name"},
			nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (nc *NetworkCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nc.collectorInfo
}

// Collect implements prometheus.Collector.
func (nc *NetworkCollector) Collect(ch chan<- prometheus.Metric) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	ctx := context.Background()

	// Get system name for labeling
	systemName, err := nc.GetSystemName(ctx)
	if err != nil {
		log.Printf("NetworkCollector: Error getting system name: %v", err)
		return
	}

	// Emit info metric to indicate collector is active
	ch <- prometheus.MustNewConstMetric(
		nc.collectorInfo,
		prometheus.GaugeValue,
		1.0,
		systemName,
	)

	// NOTE: Physical NIC metrics cannot be collected because the SDK's Node struct
	// does not capture dashboard fields (machine.nics). See .claude/GAPS.md.
	//
	// When SDK adds Node.Machine.NICs support, restore metrics:
	// - vergeos_nic_tx_packets_total
	// - vergeos_nic_rx_packets_total
	// - vergeos_nic_tx_bytes_total
	// - vergeos_nic_rx_bytes_total
	// - vergeos_nic_tx_errors_total
	// - vergeos_nic_rx_errors_total
	// - vergeos_nic_status
}
