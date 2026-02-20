package collectors

import (
	"context"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/goVergeOS"
)

// NetworkCollector collects metrics about physical node network interfaces.
// Currently emits a placeholder info metric. NIC traffic metrics are pending
// SDK support for machine_nics/machine_nic_stats/machine_nic_status tables.
// See ISSUES.md Issue 2 for details.
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

}
