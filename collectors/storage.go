package collectors

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/goVergeOS"
)

// StorageCollector collects metrics about VergeOS storage tiers.
//
// All metrics use MustNewConstMetric pattern to avoid stale label issues (Bug #28).
// This ensures only current data is emitted each scrape - no stale labels persist.
type StorageCollector struct {
	BaseCollector
	mutex sync.Mutex

	// VSAN tier capacity metrics (labels: system_name, tier, description)
	vsanCapacity    *prometheus.Desc
	vsanUsed        *prometheus.Desc
	vsanUsedPct     *prometheus.Desc
	vsanAllocated   *prometheus.Desc
	vsanDedupeRatio *prometheus.Desc

	// VSAN tier detailed metrics (labels: system_name, tier, status)
	vsanTierTransaction  *prometheus.Desc
	vsanTierRepairs      *prometheus.Desc
	vsanTierState        *prometheus.Desc
	vsanBadDrives        *prometheus.Desc
	vsanEncryptionStatus *prometheus.Desc
	vsanRedundant        *prometheus.Desc
	vsanLastWalkTime     *prometheus.Desc
	vsanLastFullwalkTime *prometheus.Desc
	vsanFullwalkStatus   *prometheus.Desc
	vsanFullwalkProgress *prometheus.Desc
	vsanCurSpaceThrottle *prometheus.Desc

	// Drive I/O metrics from MachineDriveStats (Issue 3)
	driveReadOps     *prometheus.Desc
	driveWriteOps    *prometheus.Desc
	driveReadBytes   *prometheus.Desc
	driveWriteBytes  *prometheus.Desc
	driveUtil        *prometheus.Desc
	driveServiceTime *prometheus.Desc

	// Drive hardware metrics from MachineDrivePhys (Issue 4)
	driveTemperature    *prometheus.Desc
	driveWearLevel      *prometheus.Desc
	drivePowerOnHours   *prometheus.Desc
	driveReallocSectors *prometheus.Desc
	driveReadErrors     *prometheus.Desc
	driveWriteErrors    *prometheus.Desc
	driveRepairs        *prometheus.Desc
	driveThrottle       *prometheus.Desc

	// Drive state counting (Issue 4)
	vsanDriveStates *prometheus.Desc

	// Online node/drive counts (Issue 6)
	vsanNodesOnline  *prometheus.Desc
	vsanDrivesOnline *prometheus.Desc
}

// NewStorageCollector creates a new StorageCollector.
func NewStorageCollector(client *vergeos.Client) *StorageCollector {
	tierLabels := []string{"system_name", "tier", "description"}
	tierStatusLabels := []string{"system_name", "tier", "status"}
	driveLabels := []string{"system_name", "node_name", "drive_name", "tier", "serial"}

	sc := &StorageCollector{
		BaseCollector: *NewBaseCollector(client),

		// VSAN tier capacity metrics
		vsanCapacity: prometheus.NewDesc(
			"vergeos_vsan_tier_capacity",
			"VSAN tier capacity in bytes",
			tierLabels, nil,
		),
		vsanUsed: prometheus.NewDesc(
			"vergeos_vsan_tier_used",
			"VSAN tier used space in bytes",
			tierLabels, nil,
		),
		vsanUsedPct: prometheus.NewDesc(
			"vergeos_vsan_tier_used_pct",
			"VSAN tier used space percentage",
			tierLabels, nil,
		),
		vsanAllocated: prometheus.NewDesc(
			"vergeos_vsan_tier_allocated",
			"VSAN tier allocated space in bytes",
			tierLabels, nil,
		),
		vsanDedupeRatio: prometheus.NewDesc(
			"vergeos_vsan_tier_dedupe_ratio",
			"VSAN tier deduplication ratio",
			tierLabels, nil,
		),

		// VSAN tier detailed metrics
		vsanTierTransaction: prometheus.NewDesc(
			"vergeos_vsan_tier_transaction",
			"VSAN tier transaction count",
			tierStatusLabels, nil,
		),
		vsanTierRepairs: prometheus.NewDesc(
			"vergeos_vsan_tier_repairs",
			"Number of repairs in VSAN tier",
			tierStatusLabels, nil,
		),
		vsanTierState: prometheus.NewDesc(
			"vergeos_vsan_tier_state",
			"VSAN tier state (1=working, 0=not working)",
			tierStatusLabels, nil,
		),
		vsanBadDrives: prometheus.NewDesc(
			"vergeos_vsan_bad_drives",
			"Number of bad drives in VSAN tier",
			tierStatusLabels, nil,
		),
		vsanEncryptionStatus: prometheus.NewDesc(
			"vergeos_vsan_encryption_status",
			"VSAN tier encryption status (1=encrypted, 0=not encrypted)",
			tierStatusLabels, nil,
		),
		vsanRedundant: prometheus.NewDesc(
			"vergeos_vsan_redundant",
			"VSAN tier redundancy status (1=redundant, 0=not redundant)",
			tierStatusLabels, nil,
		),
		vsanLastWalkTime: prometheus.NewDesc(
			"vergeos_vsan_last_walk_time_ms",
			"Last walk time in milliseconds",
			tierStatusLabels, nil,
		),
		vsanLastFullwalkTime: prometheus.NewDesc(
			"vergeos_vsan_last_fullwalk_time_ms",
			"Last full walk time in milliseconds",
			tierStatusLabels, nil,
		),
		vsanFullwalkStatus: prometheus.NewDesc(
			"vergeos_vsan_fullwalk_status",
			"VSAN tier fullwalk status (1=in progress, 0=not in progress)",
			tierStatusLabels, nil,
		),
		vsanFullwalkProgress: prometheus.NewDesc(
			"vergeos_vsan_fullwalk_progress",
			"VSAN tier fullwalk progress percentage",
			tierStatusLabels, nil,
		),
		vsanCurSpaceThrottle: prometheus.NewDesc(
			"vergeos_vsan_cur_space_throttle_ms",
			"Current space throttle in milliseconds",
			tierStatusLabels, nil,
		),

		// Drive I/O metrics (Issue 3)
		driveReadOps: prometheus.NewDesc(
			"vergeos_drive_read_ops",
			"Total drive read operations",
			driveLabels, nil,
		),
		driveWriteOps: prometheus.NewDesc(
			"vergeos_drive_write_ops",
			"Total drive write operations",
			driveLabels, nil,
		),
		driveReadBytes: prometheus.NewDesc(
			"vergeos_drive_read_bytes",
			"Total drive bytes read",
			driveLabels, nil,
		),
		driveWriteBytes: prometheus.NewDesc(
			"vergeos_drive_write_bytes",
			"Total drive bytes written",
			driveLabels, nil,
		),
		driveUtil: prometheus.NewDesc(
			"vergeos_drive_util",
			"Drive I/O utilization percentage",
			driveLabels, nil,
		),
		driveServiceTime: prometheus.NewDesc(
			"vergeos_drive_service_time",
			"Drive average I/O service time in milliseconds",
			driveLabels, nil,
		),

		// Drive hardware metrics (Issue 4)
		driveTemperature: prometheus.NewDesc(
			"vergeos_drive_temperature",
			"Drive temperature in Celsius",
			driveLabels, nil,
		),
		driveWearLevel: prometheus.NewDesc(
			"vergeos_drive_wear_level",
			"Drive wear level percentage",
			driveLabels, nil,
		),
		drivePowerOnHours: prometheus.NewDesc(
			"vergeos_drive_power_on_hours",
			"Drive power-on hours",
			driveLabels, nil,
		),
		driveReallocSectors: prometheus.NewDesc(
			"vergeos_drive_reallocated_sectors",
			"Drive reallocated sector count",
			driveLabels, nil,
		),
		driveReadErrors: prometheus.NewDesc(
			"vergeos_drive_read_errors",
			"VSAN drive read error count",
			driveLabels, nil,
		),
		driveWriteErrors: prometheus.NewDesc(
			"vergeos_drive_write_errors",
			"VSAN drive write error count",
			driveLabels, nil,
		),
		driveRepairs: prometheus.NewDesc(
			"vergeos_drive_repairs",
			"VSAN drive blocks being repaired",
			driveLabels, nil,
		),
		driveThrottle: prometheus.NewDesc(
			"vergeos_drive_throttle",
			"VSAN drive write throttle in bytes per second",
			driveLabels, nil,
		),

		// Drive state counting (Issue 4)
		vsanDriveStates: prometheus.NewDesc(
			"vergeos_vsan_drive_states",
			"Count of drives in each state per tier",
			[]string{"system_name", "tier", "state"}, nil,
		),

		// Online node/drive counts (Issue 6)
		vsanNodesOnline: prometheus.NewDesc(
			"vergeos_vsan_nodes_online",
			"Count of online nodes for VSAN tier",
			tierStatusLabels, nil,
		),
		vsanDrivesOnline: prometheus.NewDesc(
			"vergeos_vsan_drives_online",
			"Count of online drives for VSAN tier",
			tierStatusLabels, nil,
		),
	}

	return sc
}

// Describe implements prometheus.Collector
func (sc *StorageCollector) Describe(ch chan<- *prometheus.Desc) {
	// VSAN tier capacity metrics
	ch <- sc.vsanCapacity
	ch <- sc.vsanUsed
	ch <- sc.vsanUsedPct
	ch <- sc.vsanAllocated
	ch <- sc.vsanDedupeRatio

	// VSAN tier detailed metrics
	ch <- sc.vsanTierTransaction
	ch <- sc.vsanTierRepairs
	ch <- sc.vsanTierState
	ch <- sc.vsanBadDrives
	ch <- sc.vsanEncryptionStatus
	ch <- sc.vsanRedundant
	ch <- sc.vsanLastWalkTime
	ch <- sc.vsanLastFullwalkTime
	ch <- sc.vsanFullwalkStatus
	ch <- sc.vsanFullwalkProgress
	ch <- sc.vsanCurSpaceThrottle

	// Drive I/O metrics
	ch <- sc.driveReadOps
	ch <- sc.driveWriteOps
	ch <- sc.driveReadBytes
	ch <- sc.driveWriteBytes
	ch <- sc.driveUtil
	ch <- sc.driveServiceTime

	// Drive hardware metrics
	ch <- sc.driveTemperature
	ch <- sc.driveWearLevel
	ch <- sc.drivePowerOnHours
	ch <- sc.driveReallocSectors
	ch <- sc.driveReadErrors
	ch <- sc.driveWriteErrors
	ch <- sc.driveRepairs
	ch <- sc.driveThrottle

	// Drive state counting
	ch <- sc.vsanDriveStates

	// Online node/drive counts
	ch <- sc.vsanNodesOnline
	ch <- sc.vsanDrivesOnline
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

	// Collect VSAN tier metrics
	sc.collectTierMetrics(ctx, ch, systemName)

	// Collect drive metrics
	sc.collectDriveMetrics(ctx, ch, systemName)
}

// collectTierMetrics handles VSAN tier capacity and status metrics
func (sc *StorageCollector) collectTierMetrics(ctx context.Context, ch chan<- prometheus.Metric, systemName string) {
	// Collect VSAN tier metrics using SDK
	storageTiers, err := sc.Client().StorageTiers.List(ctx)
	if err != nil {
		log.Printf("Error fetching storage tiers: %v", err)
		return
	}

	// Build validTiers set for Bug #27 (phantom tiers fix)
	validTiers := make(map[int]bool)
	for _, tier := range storageTiers {
		validTiers[tier.Tier] = true

		tierStr := fmt.Sprintf("%d", tier.Tier)

		ch <- prometheus.MustNewConstMetric(
			sc.vsanCapacity, prometheus.GaugeValue,
			float64(tier.Capacity),
			systemName, tierStr, tier.Description,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanUsed, prometheus.GaugeValue,
			float64(tier.Used),
			systemName, tierStr, tier.Description,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanUsedPct, prometheus.GaugeValue,
			float64(tier.UsedPct),
			systemName, tierStr, tier.Description,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanAllocated, prometheus.GaugeValue,
			float64(tier.Allocated),
			systemName, tierStr, tier.Description,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanDedupeRatio, prometheus.GaugeValue,
			float64(tier.DedupeRatio)/100.0,
			systemName, tierStr, tier.Description,
		)
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

		tierStr := fmt.Sprintf("%d", tier.Tier)
		status := tier.Status.Status

		ch <- prometheus.MustNewConstMetric(
			sc.vsanTierTransaction, prometheus.CounterValue,
			float64(tier.Status.Transaction),
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanTierRepairs, prometheus.GaugeValue,
			float64(tier.Status.Repairs),
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanTierState, prometheus.GaugeValue,
			boolToFloat64(tier.Status.Working),
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanBadDrives, prometheus.GaugeValue,
			tier.Status.BadDrives,
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanEncryptionStatus, prometheus.GaugeValue,
			boolToFloat64(tier.Status.Encrypted),
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanRedundant, prometheus.GaugeValue,
			boolToFloat64(tier.Status.Redundant),
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanLastWalkTime, prometheus.GaugeValue,
			float64(tier.Status.LastWalkTimeMs),
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanLastFullwalkTime, prometheus.GaugeValue,
			float64(tier.Status.LastFullwalkTimeMs),
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanFullwalkStatus, prometheus.GaugeValue,
			boolToFloat64(tier.Status.Fullwalk),
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanFullwalkProgress, prometheus.GaugeValue,
			tier.Status.Progress,
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanCurSpaceThrottle, prometheus.GaugeValue,
			tier.Status.CurSpaceThrottleMs,
			systemName, tierStr, status,
		)

		// Online node/drive counts (Issue 6)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanNodesOnline, prometheus.GaugeValue,
			float64(tier.CountOnlineNodes()),
			systemName, tierStr, status,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.vsanDrivesOnline, prometheus.GaugeValue,
			float64(tier.CountOnlineDrives()),
			systemName, tierStr, status,
		)
	}
}

// driveStateKey is a key for grouping drives by tier and status
type driveStateKey struct {
	Tier   string
	Status string
}

// collectDriveMetrics handles per-drive I/O and hardware metrics
func (sc *StorageCollector) collectDriveMetrics(ctx context.Context, ch chan<- prometheus.Metric, systemName string) {
	// Fetch all physical drives (now includes NodeDisplay and StatusList)
	drives, err := sc.Client().MachineDrivePhys.List(ctx)
	if err != nil {
		log.Printf("Error fetching machine drive phys: %v", err)
		return
	}

	// Fetch physical drive stats and build lookup map
	driveStats, err := sc.Client().MachineDriveStats.ListPhysical(ctx)
	if err != nil {
		log.Printf("Error fetching machine drive stats: %v", err)
		// Continue without I/O stats - hardware metrics can still be emitted
		driveStats = nil
	}

	statsMap := make(map[int]vergeos.MachineDriveStats)
	for _, s := range driveStats {
		statsMap[s.ParentDrive] = s
	}

	// Track drive states for counting
	stateCounts := make(map[driveStateKey]int)

	for _, drive := range drives {
		// Skip drives not in a VSAN tier
		if drive.VSANTier < 0 {
			continue
		}

		tierStr := fmt.Sprintf("%d", drive.VSANTier)
		nodeName := drive.NodeDisplay
		driveName := drive.Path
		serial := drive.Serial

		driveLabels := []string{systemName, nodeName, driveName, tierStr, serial}

		// Drive hardware metrics
		ch <- prometheus.MustNewConstMetric(
			sc.driveTemperature, prometheus.GaugeValue,
			float64(drive.Temp), driveLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.driveWearLevel, prometheus.CounterValue,
			float64(drive.WearLevel), driveLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.drivePowerOnHours, prometheus.CounterValue,
			float64(drive.Hours), driveLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.driveReallocSectors, prometheus.CounterValue,
			float64(drive.ReallocSectors), driveLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.driveReadErrors, prometheus.CounterValue,
			float64(drive.VSANReadErrors), driveLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.driveWriteErrors, prometheus.CounterValue,
			float64(drive.VSANWriteErrors), driveLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.driveRepairs, prometheus.CounterValue,
			float64(drive.VSANRepairing), driveLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			sc.driveThrottle, prometheus.GaugeValue,
			float64(drive.VSANThrottle), driveLabels...,
		)

		// Drive I/O stats (if available)
		if stats, ok := statsMap[drive.ParentDrive]; ok {
			ch <- prometheus.MustNewConstMetric(
				sc.driveReadOps, prometheus.CounterValue,
				float64(stats.Reads), driveLabels...,
			)
			ch <- prometheus.MustNewConstMetric(
				sc.driveWriteOps, prometheus.CounterValue,
				float64(stats.Writes), driveLabels...,
			)
			ch <- prometheus.MustNewConstMetric(
				sc.driveReadBytes, prometheus.CounterValue,
				float64(stats.ReadBytes), driveLabels...,
			)
			ch <- prometheus.MustNewConstMetric(
				sc.driveWriteBytes, prometheus.CounterValue,
				float64(stats.WriteBytes), driveLabels...,
			)
			ch <- prometheus.MustNewConstMetric(
				sc.driveUtil, prometheus.GaugeValue,
				stats.Util, driveLabels...,
			)
			ch <- prometheus.MustNewConstMetric(
				sc.driveServiceTime, prometheus.GaugeValue,
				stats.ServiceTime, driveLabels...,
			)
		}

		// Count drive states
		if drive.StatusList != "" {
			key := driveStateKey{Tier: tierStr, Status: drive.StatusList}
			stateCounts[key]++
		}
	}

	// Emit drive state counts
	for key, count := range stateCounts {
		ch <- prometheus.MustNewConstMetric(
			sc.vsanDriveStates, prometheus.GaugeValue,
			float64(count),
			systemName, key.Tier, key.Status,
		)
	}
}

// boolToFloat64 converts a boolean to 1.0 or 0.0 for Prometheus gauges.
func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
