package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

func (e *Exporter) getToken() error {
	authData := map[string]string{
		"login":    e.username,
		"password": e.password,
	}
	jsonData, err := json.Marshal(authData)
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/sys/tokens", e.url), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create token request: %v", err)
	}

	req.Header.Set("X-JSON-Non-Compact", "1")
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute token request: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %v", err)
	}

	e.token = tokenResp.Key
	return nil
}

func (e *Exporter) ensureToken() error {
	if e.token == "" {
		return e.getToken()
	}
	return nil
}

func (e *Exporter) collectNodeMetrics(ch chan<- prometheus.Metric) {
	if err := e.ensureToken(); err != nil {
		fmt.Printf("Error getting token: %v\n", err)
		return
	}

	// Get list of physical nodes
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/nodes?filter=physical%%20eq%%20true", e.url), nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	req.Header.Set("x-yottabyte-token", e.token)

	resp, err := e.httpClient.Do(req)
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

	// Set total number of physical nodes
	e.nodesTotal.Set(float64(len(nodes)))

	// Process each node
	for _, node := range nodes {
		// Set IPMI status
		status := 0.0
		if node.IPMIStatus == "online" {
			status = 1.0
		}
		e.nodeIPMIStatus.WithLabelValues(node.Name).Set(status)

		// Get detailed node stats
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/nodes/%d?fields=dashboard", e.url, node.ID), nil)
		if err != nil {
			fmt.Printf("Error creating stats request for node %s: %v\n", node.Name, err)
			continue
		}
		req.Header.Set("x-yottabyte-token", e.token)

		resp, err := e.httpClient.Do(req)
		if err != nil {
			fmt.Printf("Error executing stats request for node %s: %v\n", node.Name, err)
			continue
		}

		var nodeStats NodeStats
		if err := json.NewDecoder(resp.Body).Decode(&nodeStats); err != nil {
			fmt.Printf("Error decoding stats response for node %s: %v\n", node.Name, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		stats := nodeStats.Machine.Stats

		// Set CPU core usage metrics
		for i, usage := range stats.CoreUsageList {
			e.nodeCPUCoreUsage.WithLabelValues(
				node.Name,
				strconv.Itoa(i),
			).Set(usage)
		}

		// Set CPU temperature
		e.nodeCoreTemp.WithLabelValues(node.Name).Set(stats.CoreTemp)

		// Set RAM metrics
		e.nodeRAMUsed.WithLabelValues(node.Name).Set(float64(stats.RAMUsed))
		e.nodeRAMPercent.WithLabelValues(node.Name).Set(stats.RAMPct)

		// Set drive metrics
		for _, drive := range nodeStats.Machine.Drives {
			labels := []string{node.Name, drive.Name}

			e.driveReadOps.WithLabelValues(labels...).Add(drive.Stats.ReadOps)
			e.driveWriteOps.WithLabelValues(labels...).Add(drive.Stats.WriteOps)
			e.driveReadBytes.WithLabelValues(labels...).Add(drive.Stats.ReadBytes)
			e.driveWriteBytes.WithLabelValues(labels...).Add(drive.Stats.WriteBytes)
			e.driveUtil.WithLabelValues(labels...).Set(drive.Stats.Util)
		}

		// Set NIC metrics
		for _, nic := range nodeStats.Machine.Nics {
			labels := []string{node.Name, nic.Name}

			e.nicTxPackets.WithLabelValues(labels...).Add(nic.Stats.TxPackets)
			e.nicRxPackets.WithLabelValues(labels...).Add(nic.Stats.RxPackets)
			e.nicTxBytes.WithLabelValues(labels...).Add(nic.Stats.TxBytes)
			e.nicRxBytes.WithLabelValues(labels...).Add(nic.Stats.RxBytes)
		}
	}
}

func (e *Exporter) collectVSANMetrics(ch chan<- prometheus.Metric) {
	if err := e.ensureToken(); err != nil {
		fmt.Printf("Error getting token: %v\n", err)
		return
	}

	// Get VSAN tier stats
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/cluster_tiers?fields=all%%2Cstatus%%5Ball%%5D", e.url), nil)
	if err != nil {
		fmt.Printf("Error creating VSAN request: %v\n", err)
		return
	}
	req.Header.Set("x-yottabyte-token", e.token)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing VSAN request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var tiers []VSANTier
	if err := json.NewDecoder(resp.Body).Decode(&tiers); err != nil {
		fmt.Printf("Error decoding VSAN response: %v\n", err)
		return
	}

	// Process each tier
	for _, tier := range tiers {
		tierID := strconv.Itoa(tier.Tier)

		// Set basic tier metrics
		e.vsanTierCapacity.WithLabelValues(tierID).Set(float64(tier.Status.Capacity))
		e.vsanTierUsed.WithLabelValues(tierID).Set(float64(tier.Status.Used))
		e.vsanTierUsedPct.WithLabelValues(tierID).Set(float64(tier.Status.UsedPct))

		// Set operation metrics
		e.vsanTierTransaction.WithLabelValues(tierID).Add(float64(tier.Status.Transaction))
		e.vsanTierRepairs.WithLabelValues(tierID).Add(float64(tier.Status.Repairs))

		// Set state with the state string as a label
		stateValue := 0.0
		if tier.Status.State == "online" {
			stateValue = 1.0
		}
		e.vsanTierState.WithLabelValues(tierID, tier.Status.State).Set(stateValue)

		e.vsanBadDrives.WithLabelValues(tierID).Set(float64(tier.Status.BadDrives))

		encryptionStatus := 0.0
		if tier.Status.Encrypted {
			encryptionStatus = 1.0
		}
		e.vsanEncryptionStatus.WithLabelValues(tierID).Set(encryptionStatus)

		redundantStatus := 0.0
		if tier.Status.Redundant {
			redundantStatus = 1.0
		}
		e.vsanRedundant.WithLabelValues(tierID).Set(redundantStatus)

		e.vsanLastWalkTimeMs.WithLabelValues(tierID).Set(float64(tier.Status.LastWalkTimeMs))
		e.vsanLastFullwalkTimeMs.WithLabelValues(tierID).Set(float64(tier.Status.LastFullwalkTimeMs))

		fullwalkStatus := 0.0
		if tier.Status.Fullwalk {
			fullwalkStatus = 1.0
		}
		e.vsanFullwalkStatus.WithLabelValues(tierID).Set(fullwalkStatus)

		e.vsanFullwalkProgress.WithLabelValues(tierID).Set(tier.Status.Progress)
		e.vsanCurSpaceThrottleMs.WithLabelValues(tierID).Set(float64(tier.Status.CurSpaceThrottleMs))

		// Count online nodes and drives
		onlineNodes := 0
		if tier.NodesOnline.Nodes != nil {
			for _, node := range tier.NodesOnline.Nodes {
				if node.State == "online" {
					onlineNodes++
				}
			}
		}
		e.vsanNodesOnline.WithLabelValues(tierID).Set(float64(onlineNodes))

		onlineDrives := 0
		if tier.DrivesOnline != nil {
			for _, drive := range tier.DrivesOnline {
				if drive.State == "online" {
					onlineDrives++
				}
			}
		}
		e.vsanDrivesOnline.WithLabelValues(tierID).Set(float64(onlineDrives))

		// Set drive-specific metrics
		if tier.VSANDrives != nil {
			for _, drive := range tier.VSANDrives {
				driveID := strconv.Itoa(drive.Key)
				e.vsanDriveTemp.WithLabelValues(tierID, driveID).Set(float64(drive.Temp))
				// Note: Wear level might not be directly available in the API response
				// Add it when the field is available
			}
		}
	}
}

func (e *Exporter) collectClusterMetrics(ch chan<- prometheus.Metric) {
	if err := e.ensureToken(); err != nil {
		fmt.Printf("Error getting token: %v\n", err)
		return
	}

	// Get cluster stats
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/clusters?fields=most", e.url), nil)
	if err != nil {
		fmt.Printf("Error creating cluster request: %v\n", err)
		return
	}
	req.Header.Set("x-yottabyte-token", e.token)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing cluster request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var clusters []ClusterInfo
	if err := json.NewDecoder(resp.Body).Decode(&clusters); err != nil {
		fmt.Printf("Error decoding cluster response: %v\n", err)
		return
	}

	// Set total clusters metric
	e.clustersTotal.Set(float64(len(clusters)))

	// Process each cluster
	for _, cluster := range clusters {
		// Set enabled status
		enabledStatus := 0.0
		if cluster.Enabled {
			enabledStatus = 1.0
		}
		e.clusterEnabled.WithLabelValues(cluster.Name).Set(enabledStatus)

		// Set RAM and cores per unit
		e.clusterRamPerUnit.WithLabelValues(cluster.Name).Set(float64(cluster.RamPerUnit))
		e.clusterCoresPerUnit.WithLabelValues(cluster.Name).Set(float64(cluster.CoresPerUnit))

		// Set target RAM percentage
		e.clusterTargetRamPct.WithLabelValues(cluster.Name).Set(cluster.TargetRamPct)

		// Set cluster status (1 for online/status=1, 0 for all other states)
		statusValue := 0.0
		if cluster.Status == 1 {
			statusValue = 1.0
		}
		e.clusterStatus.WithLabelValues(cluster.Name).Set(statusValue)

		// Get detailed cluster stats
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/cluster_stats_history_short/%d?fields=all", e.url, cluster.Key), nil)
		if err != nil {
			fmt.Printf("Error creating cluster stats request for cluster %s: %v\n", cluster.Name, err)
			continue
		}
		req.Header.Set("x-yottabyte-token", e.token)

		resp, err := e.httpClient.Do(req)
		if err != nil {
			fmt.Printf("Error executing cluster stats request for cluster %s: %v\n", cluster.Name, err)
			continue
		}

		var stats ClusterStats
		if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
			fmt.Printf("Error decoding cluster stats response for cluster %s: %v\n", cluster.Name, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// Set cluster stats metrics
		e.clusterTotalNodes.WithLabelValues(cluster.Name).Set(float64(stats.TotalNodes))
		e.clusterOnlineNodes.WithLabelValues(cluster.Name).Set(float64(stats.OnlineNodes))
		e.clusterRunningMachines.WithLabelValues(cluster.Name).Set(float64(stats.RunningMachines))
		e.clusterTotalRam.WithLabelValues(cluster.Name).Set(float64(stats.TotalRam))
		e.clusterOnlineRam.WithLabelValues(cluster.Name).Set(float64(stats.OnlineRam))
		e.clusterUsedRam.WithLabelValues(cluster.Name).Set(float64(stats.UsedRam))
		e.clusterTotalCores.WithLabelValues(cluster.Name).Set(float64(stats.TotalCores))
		e.clusterOnlineCores.WithLabelValues(cluster.Name).Set(float64(stats.OnlineCores))
		e.clusterUsedCores.WithLabelValues(cluster.Name).Set(float64(stats.UsedCores))
		e.clusterPhysRamUsed.WithLabelValues(cluster.Name).Set(float64(stats.PhysRamUsed))
	}
}
