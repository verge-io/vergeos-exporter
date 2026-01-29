package collectors

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/goVergeOS"
)

// StorageCollector collects metrics about VergeOS storage
type StorageCollector struct {
	BaseCollector
	mutex sync.Mutex

	// Temporary HTTP client for drive metrics until Phase 3b migration
	// Storage tiers now use SDK (Phase 3a complete)
	httpClient *http.Client
	url        string
	username   string
	password   string

	// System name for labeling metrics
	systemName string

	// Drive metrics
	driveReadOps        *prometheus.CounterVec
	driveWriteOps       *prometheus.CounterVec
	driveReadBytes      *prometheus.CounterVec
	driveWriteBytes     *prometheus.CounterVec
	driveUtil           *prometheus.GaugeVec
	driveReadErrors     *prometheus.CounterVec
	driveWriteErrors    *prometheus.CounterVec
	driveRepairs        *prometheus.CounterVec
	driveThrottle       *prometheus.GaugeVec
	driveWearLevel      *prometheus.GaugeVec
	drivePowerOnHours   *prometheus.CounterVec
	driveReallocSectors *prometheus.CounterVec
	driveTemperature    *prometheus.GaugeVec
	driveServiceTime    *prometheus.GaugeVec

	// VSAN metrics
	vsanCapacity    *prometheus.GaugeVec
	vsanUsed        *prometheus.GaugeVec
	vsanAllocated   *prometheus.GaugeVec
	vsanDedupeRatio *prometheus.GaugeVec

	// VSAN tier detailed metrics
	vsanTierTransaction  *prometheus.CounterVec
	vsanTierRepairs      *prometheus.GaugeVec
	vsanTierState        *prometheus.GaugeVec
	vsanBadDrives        *prometheus.GaugeVec
	vsanEncryptionStatus *prometheus.GaugeVec
	vsanRedundant        *prometheus.GaugeVec
	vsanLastWalkTime     *prometheus.GaugeVec
	vsanLastFullwalkTime *prometheus.GaugeVec
	vsanFullwalkStatus   *prometheus.GaugeVec
	vsanFullwalkProgress *prometheus.GaugeVec
	vsanCurSpaceThrottle *prometheus.GaugeVec
	// Note: vsanNodesOnline and vsanDrivesOnline removed - SDK doesn't provide nodes_online/drives_online fields
	// See .claude/GAPS.md for details. Use drive state metrics for detailed counts.

	// VSAN drive state metrics (per node)
	vsanDriveOnline       *prometheus.GaugeVec
	vsanDriveOffline      *prometheus.GaugeVec
	vsanDriveRepairing    *prometheus.GaugeVec
	vsanDriveInitializing *prometheus.GaugeVec
	vsanDriveVerifying    *prometheus.GaugeVec
	vsanDriveNoredundant  *prometheus.GaugeVec
	vsanDriveOutofspace   *prometheus.GaugeVec
}

// NewStorageCollector creates a new StorageCollector
// Note: url, username, password are temporary until Phase 3b migrates drive metrics to SDK
func NewStorageCollector(client *vergeos.Client, url, username, password string) *StorageCollector {
	// Create temporary HTTP client for drive metrics (will be removed in Phase 3b)
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	driveLabels := []string{"system_name", "node_name", "drive_name", "tier", "serial"}
	tierLabels := []string{"system_name", "tier", "description"}
	// New labels for drive state metrics, including node_name
	driveStateLabels := []string{"system_name", "node_name", "tier"}

	sc := &StorageCollector{
		BaseCollector: *NewBaseCollector(client),
		httpClient:    httpClient,
		url:           url,
		username:      username,
		password:      password,
		systemName:    "unknown", // Will be updated in Collect
		driveReadOps: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vergeos_drive_read_ops",
			Help: "Total number of read operations",
		}, driveLabels),
		driveWriteOps: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vergeos_drive_write_ops",
			Help: "Total number of write operations",
		}, driveLabels),
		driveReadBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vergeos_drive_read_bytes",
			Help: "Total number of bytes read",
		}, driveLabels),
		driveWriteBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vergeos_drive_write_bytes",
			Help: "Total number of bytes written",
		}, driveLabels),
		driveUtil: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_drive_util",
			Help: "Drive utilization percentage",
		}, driveLabels),
		driveReadErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vergeos_drive_read_errors",
			Help: "Total number of read errors",
		}, driveLabels),
		driveWriteErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vergeos_drive_write_errors",
			Help: "Total number of write errors",
		}, driveLabels),
		driveRepairs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vergeos_drive_repairs",
			Help: "Total number of drive repairs",
		}, driveLabels),
		driveThrottle: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_drive_throttle",
			Help: "Drive throttle percentage",
		}, driveLabels),
		driveWearLevel: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_drive_wear_level",
			Help: "Drive wear level",
		}, driveLabels),
		drivePowerOnHours: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vergeos_drive_power_on_hours",
			Help: "Total number of hours the drive has been powered on",
		}, driveLabels),
		driveReallocSectors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vergeos_drive_reallocated_sectors",
			Help: "Total number of reallocated sectors",
		}, driveLabels),
		driveTemperature: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_drive_temperature",
			Help: "Drive temperature in Celsius",
		}, driveLabels),
		driveServiceTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_drive_service_time",
			Help: "Drive service time in milliseconds",
		}, driveLabels),
		vsanCapacity: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_tier_capacity",
			Help: "VSAN tier capacity in bytes",
		}, tierLabels),
		vsanUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_tier_used",
			Help: "VSAN tier used space in bytes",
		}, tierLabels),
		vsanAllocated: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_tier_allocated",
			Help: "VSAN tier allocated space in bytes",
		}, tierLabels),
		vsanDedupeRatio: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_tier_dedupe_ratio",
			Help: "VSAN tier deduplication ratio",
		}, tierLabels),

		// VSAN tier detailed metrics
		vsanTierTransaction: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vergeos_vsan_tier_transaction",
			Help: "VSAN tier transaction count",
		}, []string{"system_name", "tier", "status"}),
		vsanTierRepairs: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_tier_repairs",
			Help: "Number of repairs in VSAN tier",
		}, []string{"system_name", "tier", "status"}),
		vsanTierState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_tier_state",
			Help: "VSAN tier state (1=working, 0=not working)",
		}, []string{"system_name", "tier", "status"}),
		vsanBadDrives: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_bad_drives",
			Help: "Number of bad drives in VSAN tier",
		}, []string{"system_name", "tier", "status"}),
		vsanEncryptionStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_encryption_status",
			Help: "VSAN tier encryption status (1=encrypted, 0=not encrypted)",
		}, []string{"system_name", "tier", "status"}),
		vsanRedundant: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_redundant",
			Help: "VSAN tier redundancy status (1=redundant, 0=not redundant)",
		}, []string{"system_name", "tier", "status"}),
		vsanLastWalkTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_last_walk_time_ms",
			Help: "Last walk time in milliseconds",
		}, []string{"system_name", "tier", "status"}),
		vsanLastFullwalkTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_last_fullwalk_time_ms",
			Help: "Last full walk time in milliseconds",
		}, []string{"system_name", "tier", "status"}),
		vsanFullwalkStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_fullwalk_status",
			Help: "VSAN tier fullwalk status (1=in progress, 0=not in progress)",
		}, []string{"system_name", "tier", "status"}),
		vsanFullwalkProgress: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_fullwalk_progress",
			Help: "VSAN tier fullwalk progress percentage",
		}, []string{"system_name", "tier", "status"}),
		vsanCurSpaceThrottle: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_cur_space_throttle_ms",
			Help: "Current space throttle in milliseconds",
		}, []string{"system_name", "tier", "status"}),
		// Note: vsanNodesOnline and vsanDrivesOnline removed - SDK gap (see .claude/GAPS.md)

		// Initialize new drive state metrics
		vsanDriveOnline: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_drive_online_count",
			Help: "Number of drives in the 'online' state per node and tier",
		}, driveStateLabels),
		vsanDriveOffline: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_drive_offline_count",
			Help: "Number of drives in the 'offline' state per node and tier",
		}, driveStateLabels),
		vsanDriveRepairing: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_drive_repairing_count",
			Help: "Number of drives in the 'repairing' state per node and tier",
		}, driveStateLabels),
		vsanDriveInitializing: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_drive_initializing_count",
			Help: "Number of drives in the 'initializing' state per node and tier",
		}, driveStateLabels),
		vsanDriveVerifying: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_drive_verifying_count",
			Help: "Number of drives in the 'verifying' state per node and tier",
		}, driveStateLabels),
		vsanDriveNoredundant: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_drive_noredundant_count",
			Help: "Number of drives in the 'noredundant' state per node and tier",
		}, driveStateLabels),
		vsanDriveOutofspace: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_drive_outofspace_count",
			Help: "Number of drives in the 'outofspace' state per node and tier",
		}, driveStateLabels),
	}

	return sc
}

// makeRequest creates an HTTP request with proper authentication
// TODO: Remove after Phase 3b migration of drive metrics to SDK
func (sc *StorageCollector) makeRequest(method, path string) (*http.Request, error) {
	req, err := http.NewRequest(method, sc.url+path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.SetBasicAuth(sc.username, sc.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-JSON-Non-Compact", "1")

	return req, nil
}

// Describe implements prometheus.Collector
func (sc *StorageCollector) Describe(ch chan<- *prometheus.Desc) {
	sc.driveReadOps.Describe(ch)
	sc.driveWriteOps.Describe(ch)
	sc.driveReadBytes.Describe(ch)
	sc.driveWriteBytes.Describe(ch)
	sc.driveUtil.Describe(ch)
	sc.driveReadErrors.Describe(ch)
	sc.driveWriteErrors.Describe(ch)
	sc.driveRepairs.Describe(ch)
	sc.driveThrottle.Describe(ch)
	sc.driveWearLevel.Describe(ch)
	sc.drivePowerOnHours.Describe(ch)
	sc.driveReallocSectors.Describe(ch)
	sc.driveTemperature.Describe(ch)
	sc.driveServiceTime.Describe(ch)
	sc.vsanCapacity.Describe(ch)
	sc.vsanUsed.Describe(ch)
	sc.vsanAllocated.Describe(ch)
	sc.vsanDedupeRatio.Describe(ch)

	// VSAN tier detailed metrics
	sc.vsanTierTransaction.Describe(ch)
	sc.vsanTierRepairs.Describe(ch)
	sc.vsanTierState.Describe(ch)
	sc.vsanBadDrives.Describe(ch)
	sc.vsanEncryptionStatus.Describe(ch)
	sc.vsanRedundant.Describe(ch)
	sc.vsanLastWalkTime.Describe(ch)
	sc.vsanLastFullwalkTime.Describe(ch)
	sc.vsanFullwalkStatus.Describe(ch)
	sc.vsanFullwalkProgress.Describe(ch)
	sc.vsanCurSpaceThrottle.Describe(ch)
	// Note: vsanNodesOnline and vsanDrivesOnline removed - SDK gap

	// Describe new drive state metrics
	sc.vsanDriveOnline.Describe(ch)
	sc.vsanDriveOffline.Describe(ch)
	sc.vsanDriveRepairing.Describe(ch)
	sc.vsanDriveInitializing.Describe(ch)
	sc.vsanDriveVerifying.Describe(ch)
	sc.vsanDriveNoredundant.Describe(ch)
	sc.vsanDriveOutofspace.Describe(ch)
}

// Collect implements prometheus.Collector
func (sc *StorageCollector) Collect(ch chan<- prometheus.Metric) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	ctx := context.Background()

	// Get system name using SDK (via BaseCollector)
	systemName, err := sc.GetSystemName(ctx)
	if err != nil {
		log.Printf("Error getting system name: %v", err)
		return
	}
	sc.systemName = systemName

	// Collect VSAN tier metrics using SDK
	storageTiers, err := sc.Client().StorageTiers.List(ctx)
	if err != nil {
		log.Printf("Error fetching storage tiers: %v", err)
		return
	}

	// Build validTiers set for Bug #27 (phantom tiers fix)
	// Only tiers returned by storage_tiers API are valid
	validTiers := make(map[int]bool)
	for _, tier := range storageTiers {
		validTiers[tier.Tier] = true

		tierStr := fmt.Sprintf("%d", tier.Tier)
		sc.vsanCapacity.WithLabelValues(sc.systemName, tierStr, tier.Description).Set(float64(tier.Capacity))
		sc.vsanUsed.WithLabelValues(sc.systemName, tierStr, tier.Description).Set(float64(tier.Used))
		sc.vsanAllocated.WithLabelValues(sc.systemName, tierStr, tier.Description).Set(float64(tier.Allocated))
		// SDK DedupeRatio is uint32 (multiply by 0.01 for ratio), convert to float
		sc.vsanDedupeRatio.WithLabelValues(sc.systemName, tierStr, tier.Description).Set(float64(tier.DedupeRatio) / 100.0)
	}

	// Get VSAN tier details using SDK
	clusterTiers, err := sc.Client().ClusterTiers.List(ctx)
	if err != nil {
		log.Printf("Error fetching cluster tiers: %v", err)
		return
	}

	// Process tier details
	for _, tier := range clusterTiers {
		// Bug #27: Skip phantom tiers not in validTiers
		if !validTiers[tier.Tier] {
			log.Printf("Skipping phantom tier %d (not configured in storage_tiers)", tier.Tier)
			continue
		}

		// Skip tiers without status
		if tier.Status == nil {
			continue
		}

		labels := []string{
			sc.systemName,
			fmt.Sprintf("%d", tier.Tier),
			tier.Status.Status, // SDK uses Status field (not StatusDisplay)
		}

		// Update counters
		sc.vsanTierTransaction.WithLabelValues(labels...).Add(float64(tier.Status.Transaction))

		// Update gauges
		sc.vsanTierRepairs.WithLabelValues(labels...).Set(float64(tier.Status.Repairs))
		sc.vsanTierState.WithLabelValues(labels...).Set(boolToFloat64(tier.Status.Working))
		sc.vsanBadDrives.WithLabelValues(labels...).Set(tier.Status.BadDrives)
		sc.vsanEncryptionStatus.WithLabelValues(labels...).Set(boolToFloat64(tier.Status.Encrypted))
		sc.vsanRedundant.WithLabelValues(labels...).Set(boolToFloat64(tier.Status.Redundant))
		sc.vsanLastWalkTime.WithLabelValues(labels...).Set(float64(tier.Status.LastWalkTimeMs))
		sc.vsanLastFullwalkTime.WithLabelValues(labels...).Set(float64(tier.Status.LastFullwalkTimeMs))
		sc.vsanFullwalkStatus.WithLabelValues(labels...).Set(boolToFloat64(tier.Status.Fullwalk))
		sc.vsanFullwalkProgress.WithLabelValues(labels...).Set(tier.Status.Progress)
		sc.vsanCurSpaceThrottle.WithLabelValues(labels...).Set(tier.Status.CurSpaceThrottleMs)

		// Note: vsanNodesOnline and vsanDrivesOnline removed - SDK gap (see .claude/GAPS.md)
	}

	// Collect drive state metrics using the new API endpoint
	sc.collectDriveStateMetrics(ch)

	// Collect drive metrics (TODO: Migrate to SDK in Phase 3b)
	req, err := sc.makeRequest("GET", "/api/v4/nodes?filter=physical%20eq%20true")
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		log.Printf("Error executing request: %v", err)
		return
	}

	var nodeResp PhysicalNodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&nodeResp); err != nil {
		log.Printf("Error decoding nodes response: %v", err)
		return
	}
	resp.Body.Close()

	for _, node := range nodeResp {
		nodeReq, err := sc.makeRequest("GET", fmt.Sprintf("/api/v4/nodes/%d?fields=dashboard", node.ID))
		if err != nil {
			log.Printf("Error creating request for node %s: %v", node.Name, err)
			continue
		}

		nodeResp, err := sc.httpClient.Do(nodeReq)
		if err != nil {
			log.Printf("Error executing request for node %s: %v", node.Name, err)
			continue
		}

		var nodeStats NodeResponse
		if err := json.NewDecoder(nodeResp.Body).Decode(&nodeStats); err != nil {
			log.Printf("Error decoding stats for node %s: %v", node.Name, err)
			nodeResp.Body.Close()
			continue
		}
		nodeResp.Body.Close()

		// Process drive metrics
		for _, drive := range nodeStats.Machine.Drives {
			// Drive metrics
			labels := []string{
				sc.systemName,
				node.Name,
				drive.Name,
				fmt.Sprintf("%d", drive.PhysicalStatus.VsanTier),
				drive.PhysicalStatus.Serial,
			}

			sc.driveReadOps.WithLabelValues(labels...).Add(float64(drive.Stats.ReadOps))
			sc.driveWriteOps.WithLabelValues(labels...).Add(float64(drive.Stats.WriteOps))
			sc.driveReadBytes.WithLabelValues(labels...).Add(float64(drive.Stats.ReadBytes))
			sc.driveWriteBytes.WithLabelValues(labels...).Add(float64(drive.Stats.WriteBytes))
			sc.driveUtil.WithLabelValues(labels...).Set(drive.Stats.Util)
			sc.driveReadErrors.WithLabelValues(labels...).Add(float64(drive.Stats.ReadErrors))
			sc.driveWriteErrors.WithLabelValues(labels...).Add(float64(drive.Stats.WriteErrors))
			sc.driveRepairs.WithLabelValues(labels...).Add(float64(drive.Stats.Repairs))
			sc.driveThrottle.WithLabelValues(labels...).Set(drive.Stats.Throttle)
			// Use the wear level from physical_status
			sc.driveWearLevel.WithLabelValues(labels...).Set(float64(drive.PhysicalStatus.WearLevel))
			sc.drivePowerOnHours.WithLabelValues(labels...).Add(float64(drive.PhysicalStatus.Hours))
			// Use the reallocated sectors from physical_status instead of stats
			sc.driveReallocSectors.WithLabelValues(labels...).Add(float64(drive.PhysicalStatus.ReallocSectors))
			sc.driveTemperature.WithLabelValues(labels...).Set(drive.PhysicalStatus.Temp)
			sc.driveServiceTime.WithLabelValues(labels...).Set(drive.Stats.ServiceTime)
		}
	}

	// Collect all metrics
	sc.driveReadOps.Collect(ch)
	sc.driveWriteOps.Collect(ch)
	sc.driveReadBytes.Collect(ch)
	sc.driveWriteBytes.Collect(ch)
	sc.driveUtil.Collect(ch)
	sc.driveReadErrors.Collect(ch)
	sc.driveWriteErrors.Collect(ch)
	sc.driveRepairs.Collect(ch)
	sc.driveThrottle.Collect(ch)
	sc.driveWearLevel.Collect(ch)
	sc.drivePowerOnHours.Collect(ch)
	sc.driveReallocSectors.Collect(ch)
	sc.driveTemperature.Collect(ch)
	sc.driveServiceTime.Collect(ch)

	sc.vsanCapacity.Collect(ch)
	sc.vsanUsed.Collect(ch)
	sc.vsanAllocated.Collect(ch)
	sc.vsanDedupeRatio.Collect(ch)

	sc.vsanTierTransaction.Collect(ch)
	sc.vsanTierRepairs.Collect(ch)
	sc.vsanTierState.Collect(ch)
	sc.vsanBadDrives.Collect(ch)
	sc.vsanEncryptionStatus.Collect(ch)
	sc.vsanRedundant.Collect(ch)
	sc.vsanLastWalkTime.Collect(ch)
	sc.vsanLastFullwalkTime.Collect(ch)
	sc.vsanFullwalkStatus.Collect(ch)
	sc.vsanFullwalkProgress.Collect(ch)
	sc.vsanCurSpaceThrottle.Collect(ch)
	// Note: vsanNodesOnline and vsanDrivesOnline removed - SDK gap

	// Collect new drive state metrics
	sc.vsanDriveOnline.Collect(ch)
	sc.vsanDriveOffline.Collect(ch)
	sc.vsanDriveRepairing.Collect(ch)
	sc.vsanDriveInitializing.Collect(ch)
	sc.vsanDriveVerifying.Collect(ch)
	sc.vsanDriveNoredundant.Collect(ch)
	sc.vsanDriveOutofspace.Collect(ch)
}

// collectDriveStateMetrics collects metrics about drive states using the new API endpoint.
// It counts the number of drives in each state per node and tier.
func (sc *StorageCollector) collectDriveStateMetrics(ch chan<- prometheus.Metric) {
	// Use the new API endpoint to get drive information including node name
	req, err := sc.makeRequest("GET", "/api/v4/machine_drives?fields=%24key%2C%20name%2C%20machine%23parent%23%24key%20as%20node%2C%20machine%23type%20as%20type%2C%20machine%23name%20as%20node_display%2C%20status%23status%20as%20statuslist%2C%20physical_status%2C%20physical_status%23vsan_tier%20as%20vsan_tier%2C%20physical_status%23vsan_repairing%20as%20vsan_repairing&filter=type%20eq%20%27node%27")
	if err != nil {
		fmt.Printf("Error creating request for machine drives: %v\n", err)
		return
	}

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing request for machine drives: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var drives MachineDriveResponse
	if err := json.NewDecoder(resp.Body).Decode(&drives); err != nil {
		fmt.Printf("Error decoding machine drives response: %v\n", err)
		return
	}

	// Reset metrics before setting new values to handle removed/changed drives/nodes/tiers
	sc.vsanDriveOnline.Reset()
	sc.vsanDriveOffline.Reset()
	sc.vsanDriveRepairing.Reset()
	sc.vsanDriveInitializing.Reset()
	sc.vsanDriveVerifying.Reset()
	sc.vsanDriveNoredundant.Reset()
	sc.vsanDriveOutofspace.Reset()

	// Define a struct to use as a map key
	type driveKey struct {
		nodeName string
		tier     string
	}

	// Maps to store counts for each state per node/tier
	onlineCounts := make(map[driveKey]float64)
	offlineCounts := make(map[driveKey]float64)
	repairingCounts := make(map[driveKey]float64)
	initializingCounts := make(map[driveKey]float64)
	verifyingCounts := make(map[driveKey]float64)
	noredundantCounts := make(map[driveKey]float64)
	outofspaceCounts := make(map[driveKey]float64)
	allKeys := make(map[driveKey]struct{}) // Keep track of all node/tier combinations

	// Process each drive and increment the appropriate counter
	for _, drive := range drives {
		// Skip drives not assigned to a VSAN tier
		if drive.VsanTier < 0 {
			continue
		}

		nodeName := drive.NodeDisplay // Use the fetched node name
		tierStr := fmt.Sprintf("%d", drive.VsanTier)
		key := driveKey{nodeName: nodeName, tier: tierStr}
		allKeys[key] = struct{}{} // Record this node/tier combination

		// Determine the primary state
		state := drive.StatusList
		if drive.VsanRepairing > 0 { // Override if repairing
			state = "repairing"
		}

		// Increment the counter for the determined state
		switch state {
		case "online":
			onlineCounts[key]++
		case "offline":
			offlineCounts[key]++
		case "repairing":
			repairingCounts[key]++
		case "initializing":
			initializingCounts[key]++
		case "verifying":
			verifyingCounts[key]++
		case "noredundant":
			noredundantCounts[key]++
		case "outofspace":
			outofspaceCounts[key]++
		}
	}

	// Ensure all states are set for every encountered node/tier combination
	for key := range allKeys {
		labels := prometheus.Labels{
			"system_name": sc.systemName,
			"node_name":   key.nodeName,
			"tier":        key.tier,
		}

		// Set gauge for each state, defaulting to 0 if the key wasn't in the specific count map
		sc.vsanDriveOnline.With(labels).Set(onlineCounts[key])
		sc.vsanDriveOffline.With(labels).Set(offlineCounts[key])
		sc.vsanDriveRepairing.With(labels).Set(repairingCounts[key])
		sc.vsanDriveInitializing.With(labels).Set(initializingCounts[key])
		sc.vsanDriveVerifying.With(labels).Set(verifyingCounts[key])
		sc.vsanDriveNoredundant.With(labels).Set(noredundantCounts[key])
		sc.vsanDriveOutofspace.With(labels).Set(outofspaceCounts[key])
	}

	// Note: The 'ch' parameter is not used here as metrics are collected
	// in the main Collect method which calls this function.
}

// boolToFloat64 converts a boolean to 1.0 or 0.0 for Prometheus gauges.
func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
