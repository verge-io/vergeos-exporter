package collectors

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/govergeos"
)

var _ prometheus.Collector = (*VNetCollector)(nil)

// VNetCollector collects metrics about VergeOS virtual networks (VNets):
// inventory/config state for every network, plus the latest gateway-monitoring
// stats for networks with gateway monitoring enabled.
type VNetCollector struct {
	BaseCollector
	mutex sync.Mutex

	// Inventory/config metrics
	vnetEnabled        *prometheus.Desc
	vnetPowerState     *prometheus.Desc
	vnetMonitorGateway *prometheus.Desc

	// Router NIC traffic counters
	vnetTxBytes   *prometheus.Desc
	vnetRxBytes   *prometheus.Desc
	vnetTxPackets *prometheus.Desc
	vnetRxPackets *prometheus.Desc

	// Gateway monitoring metrics (latest sample per monitored network)
	monitorSent             *prometheus.Desc
	monitorQuality          *prometheus.Desc
	monitorDroppedPct       *prometheus.Desc
	monitorLatencyUsecAvg   *prometheus.Desc
	monitorLatencyUsecPeak  *prometheus.Desc
	monitorDuplicates       *prometheus.Desc
	monitorTruncated        *prometheus.Desc
	monitorDropped          *prometheus.Desc
	monitorBadChecksums     *prometheus.Desc
	monitorBadData          *prometheus.Desc
	monitorTimestampSeconds *prometheus.Desc
}

// NewVNetCollector creates a new VNetCollector.
func NewVNetCollector(client *vergeos.Client, scrapeTimeout time.Duration) *VNetCollector {
	labels := []string{"system_name", "vnet_name", "vnet_id", "cluster", "type", "layer2_type"}

	return &VNetCollector{
		BaseCollector: *NewBaseCollector(client, scrapeTimeout),
		vnetEnabled: prometheus.NewDesc(
			"vergeos_vnet_enabled",
			"Whether the virtual network is enabled (1=enabled, 0=disabled)",
			labels, nil,
		),
		vnetPowerState: prometheus.NewDesc(
			"vergeos_vnet_powerstate",
			"Whether the virtual network's router is running (1=running, 0=stopped)",
			labels, nil,
		),
		vnetTxBytes: prometheus.NewDesc(
			"vergeos_vnet_tx_bytes_total",
			"Total bytes transmitted by the virtual network's router NIC",
			labels, nil,
		),
		vnetRxBytes: prometheus.NewDesc(
			"vergeos_vnet_rx_bytes_total",
			"Total bytes received by the virtual network's router NIC",
			labels, nil,
		),
		vnetTxPackets: prometheus.NewDesc(
			"vergeos_vnet_tx_packets_total",
			"Total packets transmitted by the virtual network's router NIC",
			labels, nil,
		),
		vnetRxPackets: prometheus.NewDesc(
			"vergeos_vnet_rx_packets_total",
			"Total packets received by the virtual network's router NIC",
			labels, nil,
		),
		vnetMonitorGateway: prometheus.NewDesc(
			"vergeos_vnet_monitor_gateway",
			"Whether gateway monitoring is enabled for the virtual network (1=enabled, 0=disabled)",
			labels, nil,
		),
		monitorSent: prometheus.NewDesc(
			"vergeos_vnet_monitor_sent",
			"Monitoring packets sent in the latest gateway-monitoring sample",
			labels, nil,
		),
		monitorQuality: prometheus.NewDesc(
			"vergeos_vnet_monitor_quality",
			"Gateway network quality percentage (100 - dropped_pct) from the latest sample",
			labels, nil,
		),
		monitorDroppedPct: prometheus.NewDesc(
			"vergeos_vnet_monitor_dropped_pct",
			"Percentage of dropped monitoring packets in the latest sample",
			labels, nil,
		),
		monitorLatencyUsecAvg: prometheus.NewDesc(
			"vergeos_vnet_monitor_latency_usec_avg",
			"Average gateway latency in microseconds from the latest sample",
			labels, nil,
		),
		monitorLatencyUsecPeak: prometheus.NewDesc(
			"vergeos_vnet_monitor_latency_usec_peak",
			"Peak gateway latency in microseconds from the latest sample",
			labels, nil,
		),
		monitorDuplicates: prometheus.NewDesc(
			"vergeos_vnet_monitor_duplicates",
			"Duplicate monitoring packets in the latest sample",
			labels, nil,
		),
		monitorTruncated: prometheus.NewDesc(
			"vergeos_vnet_monitor_truncated",
			"Truncated monitoring packets in the latest sample",
			labels, nil,
		),
		monitorDropped: prometheus.NewDesc(
			"vergeos_vnet_monitor_dropped",
			"Dropped monitoring packets in the latest sample",
			labels, nil,
		),
		monitorBadChecksums: prometheus.NewDesc(
			"vergeos_vnet_monitor_bad_checksums",
			"Monitoring packets with bad checksums in the latest sample",
			labels, nil,
		),
		monitorBadData: prometheus.NewDesc(
			"vergeos_vnet_monitor_bad_data",
			"Monitoring packets with bad data in the latest sample",
			labels, nil,
		),
		monitorTimestampSeconds: prometheus.NewDesc(
			"vergeos_vnet_monitor_timestamp_seconds",
			"Unix timestamp of the latest gateway-monitoring sample",
			labels, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (vc *VNetCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- vc.vnetEnabled
	ch <- vc.vnetPowerState
	ch <- vc.vnetMonitorGateway
	ch <- vc.vnetTxBytes
	ch <- vc.vnetRxBytes
	ch <- vc.vnetTxPackets
	ch <- vc.vnetRxPackets
	ch <- vc.monitorSent
	ch <- vc.monitorQuality
	ch <- vc.monitorDroppedPct
	ch <- vc.monitorLatencyUsecAvg
	ch <- vc.monitorLatencyUsecPeak
	ch <- vc.monitorDuplicates
	ch <- vc.monitorTruncated
	ch <- vc.monitorDropped
	ch <- vc.monitorBadChecksums
	ch <- vc.monitorBadData
	ch <- vc.monitorTimestampSeconds
}

// Collect implements prometheus.Collector.
func (vc *VNetCollector) Collect(ch chan<- prometheus.Metric) {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()

	ctx, cancel := vc.ScrapeContext()
	defer cancel()

	systemName, err := vc.GetSystemName(ctx)
	if err != nil {
		log.Printf("VNetCollector: Error getting system name: %v", err)
		return
	}

	clusterMap, err := vc.BuildClusterMap(ctx)
	if err != nil {
		log.Printf("VNetCollector: Error building cluster map: %v", err)
		return
	}

	networks, err := vc.Client().Networks.List(ctx)
	if err != nil {
		log.Printf("VNetCollector: Error fetching networks: %v", err)
		return
	}

	// Map NIC key -> NIC so each vnet's router NIC stats can be joined
	// without per-vnet API calls.
	allNICs, err := vc.Client().MachineNICs.List(ctx)
	if err != nil {
		log.Printf("VNetCollector: Error fetching NICs: %v", err)
		return
	}
	nicByKey := make(map[int]vergeos.MachineNIC, len(allNICs))
	for _, nic := range allNICs {
		nicByKey[int(nic.Key)] = nic
	}

	for _, network := range networks {
		clusterName := clusterMap[int(network.Cluster)]
		if clusterName == "" {
			clusterName = fmt.Sprintf("cluster_%d", network.Cluster)
		}
		labels := []string{
			systemName, network.Name, strconv.Itoa(int(network.ID)),
			clusterName, network.Type, network.Layer2Type,
		}

		ch <- prometheus.MustNewConstMetric(
			vc.vnetEnabled, prometheus.GaugeValue,
			boolToFloat64(network.Enabled), labels...,
		)
		// Running (machine#status#running) is the true state; the vnets
		// `powerstate` DB field is not maintained by the platform.
		ch <- prometheus.MustNewConstMetric(
			vc.vnetPowerState, prometheus.GaugeValue,
			boolToFloat64(network.Running), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.vnetMonitorGateway, prometheus.GaugeValue,
			boolToFloat64(network.MonitorGateway), labels...,
		)

		// Router NIC traffic counters (primary NIC only; DMZ NIC not exposed)
		if nic, ok := nicByKey[int(network.NIC)]; ok && network.NIC != 0 && nic.Stats != nil {
			ch <- prometheus.MustNewConstMetric(
				vc.vnetTxBytes, prometheus.CounterValue, float64(nic.Stats.TxBytes), labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				vc.vnetRxBytes, prometheus.CounterValue, float64(nic.Stats.RxBytes), labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				vc.vnetTxPackets, prometheus.CounterValue, float64(nic.Stats.TxPckts), labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				vc.vnetRxPackets, prometheus.CounterValue, float64(nic.Stats.RxPckts), labels...,
			)
		}

		if !network.MonitorGateway {
			continue
		}

		stats, err := vc.Client().Networks.GetLatestStatistics(ctx, int(network.ID))
		if err != nil {
			log.Printf("VNetCollector: Error fetching monitor stats for network %s: %v", network.Name, err)
			continue
		}
		if stats == nil {
			// Monitoring enabled but no stats row yet (e.g. network just started)
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			vc.monitorSent, prometheus.GaugeValue, float64(stats.Sent), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.monitorQuality, prometheus.GaugeValue, float64(stats.Quality), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.monitorDroppedPct, prometheus.GaugeValue, float64(stats.DroppedPct), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.monitorLatencyUsecAvg, prometheus.GaugeValue, float64(stats.LatencyUSAvg), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.monitorLatencyUsecPeak, prometheus.GaugeValue, float64(stats.LatencyUSPeak), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.monitorDuplicates, prometheus.GaugeValue, float64(stats.Duplicates), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.monitorTruncated, prometheus.GaugeValue, float64(stats.Truncated), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.monitorDropped, prometheus.GaugeValue, float64(stats.Dropped), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.monitorBadChecksums, prometheus.GaugeValue, float64(stats.BadChecksums), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.monitorBadData, prometheus.GaugeValue, float64(stats.BadData), labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			vc.monitorTimestampSeconds, prometheus.GaugeValue, float64(stats.Timestamp), labels...,
		)
	}
}
