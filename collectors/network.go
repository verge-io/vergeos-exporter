package collectors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// NetworkCollector collects metrics about network interfaces
type NetworkCollector struct {
	BaseCollector
	mutex sync.Mutex

	// System info
	systemName string

	// Metrics
	nicTxPacketsDesc *prometheus.Desc
	nicRxPacketsDesc *prometheus.Desc
	nicTxBytesDesc   *prometheus.Desc
	nicRxBytesDesc   *prometheus.Desc
	nicTxErrorsDesc  *prometheus.Desc
	nicRxErrorsDesc  *prometheus.Desc
	nicStatusDesc    *prometheus.Desc
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Stats  struct {
		TxPackets uint64 `json:"tx_packets"`
		RxPackets uint64 `json:"rx_packets"`
		TxBytes   uint64 `json:"tx_bytes"`
		RxBytes   uint64 `json:"rx_bytes"`
		TxErrors  uint64 `json:"tx_errors"`
		RxErrors  uint64 `json:"rx_errors"`
	} `json:"stats"`
}

// NewNetworkCollector creates a new NetworkCollector
func NewNetworkCollector(url string, client *http.Client, username, password string) *NetworkCollector {
	nc := &NetworkCollector{
		BaseCollector: BaseCollector{
			url:        url,
			httpClient: client,
		},
		systemName: "unknown", // Will be updated in Collect
		nicTxPacketsDesc: prometheus.NewDesc(
			"vergeos_nic_tx_packets_total",
			"Total number of packets transmitted",
			[]string{"system_name", "cluster", "node_name", "interface"},
			nil,
		),
		nicRxPacketsDesc: prometheus.NewDesc(
			"vergeos_nic_rx_packets_total",
			"Total number of packets received",
			[]string{"system_name", "cluster", "node_name", "interface"},
			nil,
		),
		nicTxBytesDesc: prometheus.NewDesc(
			"vergeos_nic_tx_bytes_total",
			"Total number of bytes transmitted",
			[]string{"system_name", "cluster", "node_name", "interface"},
			nil,
		),
		nicRxBytesDesc: prometheus.NewDesc(
			"vergeos_nic_rx_bytes_total",
			"Total number of bytes received",
			[]string{"system_name", "cluster", "node_name", "interface"},
			nil,
		),
		nicTxErrorsDesc: prometheus.NewDesc(
			"vergeos_nic_tx_errors_total",
			"Total number of transmit errors",
			[]string{"system_name", "cluster", "node_name", "interface"},
			nil,
		),
		nicRxErrorsDesc: prometheus.NewDesc(
			"vergeos_nic_rx_errors_total",
			"Total number of receive errors",
			[]string{"system_name", "cluster", "node_name", "interface"},
			nil,
		),
		nicStatusDesc: prometheus.NewDesc(
			"vergeos_nic_status",
			"Network interface status (1=up, 0=down)",
			[]string{"system_name", "cluster", "node_name", "interface"},
			nil,
		),
	}

	// Authenticate with the API
	if err := nc.authenticate(username, password); err != nil {
		fmt.Printf("Error authenticating with VergeOS API: %v\n", err)
	}

	return nc
}

// Describe implements prometheus.Collector
func (nc *NetworkCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nc.nicTxPacketsDesc
	ch <- nc.nicRxPacketsDesc
	ch <- nc.nicTxBytesDesc
	ch <- nc.nicRxBytesDesc
	ch <- nc.nicTxErrorsDesc
	ch <- nc.nicRxErrorsDesc
	ch <- nc.nicStatusDesc
}

// Collect implements prometheus.Collector
func (nc *NetworkCollector) Collect(ch chan<- prometheus.Metric) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	// Get system name
	req, err := nc.makeRequest("GET", "/api/v4/settings?fields=most&filter=key%20eq%20%22cloud_name%22")
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	resp, err := nc.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var settings []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		fmt.Printf("Error decoding settings response: %v\n", err)
		return
	}

	for _, setting := range settings {
		if setting.Key == "cloud_name" {
			nc.systemName = setting.Value
			break
		}
	}

	// Get list of physical nodes
	req, err = nc.makeRequest("GET", "/api/v4/nodes?filter=physical%20eq%20true")
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	resp, err = nc.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var nodes []Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return
	}

	for _, node := range nodes {
		req, err = nc.makeRequest("GET", fmt.Sprintf("/api/v4/nodes/%d?fields=dashboard", node.ID))
		if err != nil {
			fmt.Printf("Error creating request for node %s: %v\n", node.Name, err)
			continue
		}

		resp, err = nc.httpClient.Do(req)
		if err != nil {
			fmt.Printf("Error executing request for node %s: %v\n", node.Name, err)
			continue
		}

		var nodeData struct {
			Machine struct {
				NICs []struct {
					Name   string `json:"name"`
					Status string `json:"status"`
					Stats  struct {
						TxPackets uint64 `json:"tx_packets"`
						RxPackets uint64 `json:"rx_packets"`
						TxBytes   uint64 `json:"tx_bytes"`
						RxBytes   uint64 `json:"rx_bytes"`
						TxErrors  uint64 `json:"tx_errors"`
						RxErrors  uint64 `json:"rx_errors"`
					} `json:"stats"`
				} `json:"nics"`
			} `json:"machine"`
			ClusterDisplay string `json:"cluster_display"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&nodeData); err != nil {
			fmt.Printf("Error decoding node data for %s: %v\n", node.Name, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for _, nic := range nodeData.Machine.NICs {
			// Set interface status
			statusValue := 0.0
			if nic.Status == "up" {
				statusValue = 1.0
			}
			ch <- prometheus.MustNewConstMetric(nc.nicStatusDesc, prometheus.CounterValue, statusValue, nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name)
			// Set network metrics
			ch <- prometheus.MustNewConstMetric(nc.nicTxPacketsDesc, prometheus.CounterValue, float64(nic.Stats.TxPackets), nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name)
			ch <- prometheus.MustNewConstMetric(nc.nicRxPacketsDesc, prometheus.CounterValue, float64(nic.Stats.RxPackets), nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name)
			ch <- prometheus.MustNewConstMetric(nc.nicTxBytesDesc, prometheus.CounterValue, float64(nic.Stats.TxBytes), nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name)
			ch <- prometheus.MustNewConstMetric(nc.nicRxBytesDesc, prometheus.CounterValue, float64(nic.Stats.RxBytes), nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name)
			ch <- prometheus.MustNewConstMetric(nc.nicTxErrorsDesc, prometheus.CounterValue, float64(nic.Stats.TxErrors), nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name)
			ch <- prometheus.MustNewConstMetric(nc.nicRxErrorsDesc, prometheus.CounterValue, float64(nic.Stats.RxErrors), nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name)

		}
	}
}
