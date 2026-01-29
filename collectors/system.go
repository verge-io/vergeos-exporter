package collectors

import (
	"context"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/goVergeOS"
)

// SystemCollector collects metrics about VergeOS system versions
type SystemCollector struct {
	BaseCollector
	mutex sync.Mutex

	// Metric descriptors (using MustNewConstMetric pattern to avoid stale metrics)
	systemVersion *prometheus.Desc
	systemInfo    *prometheus.Desc
}

// NewSystemCollector creates a new SystemCollector
func NewSystemCollector(client *vergeos.Client) *SystemCollector {
	return &SystemCollector{
		BaseCollector: *NewBaseCollector(client),
		systemVersion: prometheus.NewDesc(
			"vergeos_system_version",
			"Current version of the VergeOS system (always 1, version in label)",
			[]string{"system_name", "version"},
			nil,
		),
		systemInfo: prometheus.NewDesc(
			"vergeos_system_info",
			"Information about the VergeOS system",
			[]string{"system_name", "version", "hash"},
			nil,
		),
	}
}

// Describe implements prometheus.Collector
func (sc *SystemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- sc.systemVersion
	ch <- sc.systemInfo
}

// Collect implements prometheus.Collector
func (sc *SystemCollector) Collect(ch chan<- prometheus.Metric) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	ctx := context.Background()

	// Get system name using BaseCollector (SDK)
	systemName, err := sc.GetSystemName(ctx)
	if err != nil {
		log.Printf("Error getting system name: %v", err)
		return
	}

	// Get system info using SDK
	info, err := sc.client.System.GetInfo(ctx)
	if err != nil {
		log.Printf("Error getting system info: %v", err)
		return
	}

	// Emit system version metric
	ch <- prometheus.MustNewConstMetric(
		sc.systemVersion,
		prometheus.GaugeValue,
		1.0,
		systemName, info.Version,
	)

	// Emit system info metric with available fields
	ch <- prometheus.MustNewConstMetric(
		sc.systemInfo,
		prometheus.GaugeValue,
		1.0,
		systemName, info.Version, info.Hash,
	)
}
