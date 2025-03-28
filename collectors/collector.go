package collectors

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Collector is the interface that all VergeOS collectors must implement
type Collector interface {
	// Describe sends the super-set of all possible descriptors of metrics
	// collected by this Collector to the provided channel.
	Describe(ch chan<- *prometheus.Desc)

	// Collect is called by Prometheus when collecting metrics.
	Collect(ch chan<- prometheus.Metric)
}
