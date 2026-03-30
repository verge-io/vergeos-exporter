package collectors

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/govergeos"
)

var _ prometheus.Collector = (*NetworkCollector)(nil)

// NetworkCollector collects metrics about physical node network interfaces
// using the MachineNICService for per-NIC traffic counters and link status.
type NetworkCollector struct {
	BaseCollector
	mutex sync.Mutex

	// NIC traffic metrics
	nicTxPackets *prometheus.Desc
	nicRxPackets *prometheus.Desc
	nicTxBytes   *prometheus.Desc
	nicRxBytes   *prometheus.Desc
	nicStatus    *prometheus.Desc
}

// NewNetworkCollector creates a new NetworkCollector.
func NewNetworkCollector(client *vergeos.Client, scrapeTimeout time.Duration) *NetworkCollector {
	nicLabels := []string{"system_name", "cluster", "node_name", "interface"}

	return &NetworkCollector{
		BaseCollector: *NewBaseCollector(client, scrapeTimeout),
		nicTxPackets: prometheus.NewDesc(
			"vergeos_nic_tx_packets_total",
			"Total transmitted packets",
			nicLabels, nil,
		),
		nicRxPackets: prometheus.NewDesc(
			"vergeos_nic_rx_packets_total",
			"Total received packets",
			nicLabels, nil,
		),
		nicTxBytes: prometheus.NewDesc(
			"vergeos_nic_tx_bytes_total",
			"Total transmitted bytes",
			nicLabels, nil,
		),
		nicRxBytes: prometheus.NewDesc(
			"vergeos_nic_rx_bytes_total",
			"Total received bytes",
			nicLabels, nil,
		),
		nicStatus: prometheus.NewDesc(
			"vergeos_nic_status",
			"NIC link status (1=up, 0=other)",
			nicLabels, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (nc *NetworkCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nc.nicTxPackets
	ch <- nc.nicRxPackets
	ch <- nc.nicTxBytes
	ch <- nc.nicRxBytes
	ch <- nc.nicStatus
}

// Collect implements prometheus.Collector.
func (nc *NetworkCollector) Collect(ch chan<- prometheus.Metric) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	ctx, cancel := nc.ScrapeContext()
	defer cancel()

	// Get system name for labeling
	systemName, err := nc.GetSystemName(ctx)
	if err != nil {
		log.Printf("NetworkCollector: Error getting system name: %v", err)
		return
	}

	// Build cluster ID -> name mapping
	clusterMap, err := nc.BuildClusterMap(ctx)
	if err != nil {
		log.Printf("NetworkCollector: Error building cluster map: %v", err)
		return
	}

	// Get physical nodes
	nodes, err := nc.Client().Nodes.ListPhysical(ctx)
	if err != nil {
		log.Printf("NetworkCollector: Error fetching physical nodes: %v", err)
		return
	}

	// Batch-fetch all NICs (avoids N+1 per-node API calls)
	allNICs, err := nc.Client().MachineNICs.List(ctx)
	if err != nil {
		log.Printf("NetworkCollector: Error fetching NICs: %v", err)
		return
	}
	nicMap := make(map[int][]vergeos.MachineNIC)
	for _, nic := range allNICs {
		nicMap[nic.Machine] = append(nicMap[nic.Machine], nic)
	}

	for _, node := range nodes {
		clusterName := clusterMap[node.Cluster]
		if clusterName == "" {
			clusterName = fmt.Sprintf("cluster_%d", node.Cluster)
		}

		for _, nic := range nicMap[node.Machine] {
			labels := []string{systemName, clusterName, node.Name, nic.Name}

			// Traffic counters
			if nic.Stats != nil {
				ch <- prometheus.MustNewConstMetric(
					nc.nicTxPackets, prometheus.CounterValue,
					float64(nic.Stats.TxPckts), labels...,
				)
				ch <- prometheus.MustNewConstMetric(
					nc.nicRxPackets, prometheus.CounterValue,
					float64(nic.Stats.RxPckts), labels...,
				)
				ch <- prometheus.MustNewConstMetric(
					nc.nicTxBytes, prometheus.CounterValue,
					float64(nic.Stats.TxBytes), labels...,
				)
				ch <- prometheus.MustNewConstMetric(
					nc.nicRxBytes, prometheus.CounterValue,
					float64(nic.Stats.RxBytes), labels...,
				)
			}

			// Link status
			if nic.Status != nil {
				statusValue := 0.0
				if nic.Status.Status == "up" {
					statusValue = 1.0
				}
				ch <- prometheus.MustNewConstMetric(
					nc.nicStatus, prometheus.GaugeValue,
					statusValue, labels...,
				)
			}
		}
	}
}
