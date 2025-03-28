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
	nicTxPackets *prometheus.CounterVec
	nicRxPackets *prometheus.CounterVec
	nicTxBytes   *prometheus.CounterVec
	nicRxBytes   *prometheus.CounterVec
	nicTxErrors  *prometheus.CounterVec
	nicRxErrors  *prometheus.CounterVec
	nicStatus    *prometheus.GaugeVec
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
		nicTxPackets: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_nic_tx_packets_total",
				Help: "Total number of packets transmitted",
			},
			[]string{"system_name", "cluster", "node_name", "interface"},
		),
		nicRxPackets: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_nic_rx_packets_total",
				Help: "Total number of packets received",
			},
			[]string{"system_name", "cluster", "node_name", "interface"},
		),
		nicTxBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_nic_tx_bytes_total",
				Help: "Total number of bytes transmitted",
			},
			[]string{"system_name", "cluster", "node_name", "interface"},
		),
		nicRxBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_nic_rx_bytes_total",
				Help: "Total number of bytes received",
			},
			[]string{"system_name", "cluster", "node_name", "interface"},
		),
		nicTxErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_nic_tx_errors_total",
				Help: "Total number of transmit errors",
			},
			[]string{"system_name", "cluster", "node_name", "interface"},
		),
		nicRxErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vergeos_nic_rx_errors_total",
				Help: "Total number of receive errors",
			},
			[]string{"system_name", "cluster", "node_name", "interface"},
		),
		nicStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vergeos_nic_status",
				Help: "Network interface status (1=up, 0=down)",
			},
			[]string{"system_name", "cluster", "node_name", "interface"},
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
	nc.nicTxPackets.Describe(ch)
	nc.nicRxPackets.Describe(ch)
	nc.nicTxBytes.Describe(ch)
	nc.nicRxBytes.Describe(ch)
	nc.nicTxErrors.Describe(ch)
	nc.nicRxErrors.Describe(ch)
	nc.nicStatus.Describe(ch)
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
			nc.nicStatus.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name).Set(statusValue)

			// Set network metrics
			nc.nicTxPackets.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name).Add(float64(nic.Stats.TxPackets))
			nc.nicRxPackets.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name).Add(float64(nic.Stats.RxPackets))
			nc.nicTxBytes.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name).Add(float64(nic.Stats.TxBytes))
			nc.nicRxBytes.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name).Add(float64(nic.Stats.RxBytes))
			nc.nicTxErrors.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name).Add(float64(nic.Stats.TxErrors))
			nc.nicRxErrors.WithLabelValues(nc.systemName, nodeData.ClusterDisplay, node.Name, nic.Name).Add(float64(nic.Stats.RxErrors))
		}
	}

	nc.nicTxPackets.Collect(ch)
	nc.nicRxPackets.Collect(ch)
	nc.nicTxBytes.Collect(ch)
	nc.nicRxBytes.Collect(ch)
	nc.nicTxErrors.Collect(ch)
	nc.nicRxErrors.Collect(ch)
	nc.nicStatus.Collect(ch)
}
