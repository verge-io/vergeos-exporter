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
// Note: Per-drive metrics and drive state metrics have been removed due to SDK gaps.
// See .claude/GAPS.md for details on MachineDrivePhys limitations.
//
// All metrics use MustNewConstMetric pattern to avoid stale label issues (Bug #28).
// This ensures only current data is emitted each scrape - no stale labels persist.
type StorageCollector struct {
	BaseCollector
	mutex sync.Mutex

	// VSAN tier capacity metrics (labels: system_name, tier, description)
	vsanCapacity    *prometheus.Desc
	vsanUsed        *prometheus.Desc
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
}

// NewStorageCollector creates a new StorageCollector.
// Note: Per-drive metrics removed due to SDK gaps (no node mapping in MachineDrivePhys).
// See .claude/GAPS.md for details.
//
// All metrics use prometheus.Desc with MustNewConstMetric pattern to fix Bug #28
// (stale metrics when tier status changes).
func NewStorageCollector(client *vergeos.Client) *StorageCollector {
	tierLabels := []string{"system_name", "tier", "description"}
	tierStatusLabels := []string{"system_name", "tier", "status"}

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
	}

	return sc
}

// Describe implements prometheus.Collector
func (sc *StorageCollector) Describe(ch chan<- *prometheus.Desc) {
	// VSAN tier capacity metrics
	ch <- sc.vsanCapacity
	ch <- sc.vsanUsed
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
}

// Collect implements prometheus.Collector
// Uses MustNewConstMetric pattern to emit metrics directly, avoiding stale labels (Bug #28).
// Each scrape only emits metrics for the current state - no old label combinations persist.
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

		// Emit capacity metrics using MustNewConstMetric
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
			sc.vsanAllocated, prometheus.GaugeValue,
			float64(tier.Allocated),
			systemName, tierStr, tier.Description,
		)
		// SDK DedupeRatio is uint32 (multiply by 0.01 for ratio), convert to float
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
		status := tier.Status.Status // SDK uses Status field (not StatusDisplay)

		// Emit tier status metrics using MustNewConstMetric
		// Transaction is a cumulative counter from the API - use CounterValue
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
	}
}

// boolToFloat64 converts a boolean to 1.0 or 0.0 for Prometheus gauges.
func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
