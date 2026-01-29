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
type StorageCollector struct {
	BaseCollector
	mutex sync.Mutex

	// VSAN tier capacity metrics
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
}

// NewStorageCollector creates a new StorageCollector.
// Note: Per-drive metrics removed due to SDK gaps (no node mapping in MachineDrivePhys).
// See .claude/GAPS.md for details.
func NewStorageCollector(client *vergeos.Client) *StorageCollector {
	tierLabels := []string{"system_name", "tier", "description"}
	tierStatusLabels := []string{"system_name", "tier", "status"}

	sc := &StorageCollector{
		BaseCollector: *NewBaseCollector(client),

		// VSAN tier capacity metrics
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
		}, tierStatusLabels),
		vsanTierRepairs: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_tier_repairs",
			Help: "Number of repairs in VSAN tier",
		}, tierStatusLabels),
		vsanTierState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_tier_state",
			Help: "VSAN tier state (1=working, 0=not working)",
		}, tierStatusLabels),
		vsanBadDrives: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_bad_drives",
			Help: "Number of bad drives in VSAN tier",
		}, tierStatusLabels),
		vsanEncryptionStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_encryption_status",
			Help: "VSAN tier encryption status (1=encrypted, 0=not encrypted)",
		}, tierStatusLabels),
		vsanRedundant: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_redundant",
			Help: "VSAN tier redundancy status (1=redundant, 0=not redundant)",
		}, tierStatusLabels),
		vsanLastWalkTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_last_walk_time_ms",
			Help: "Last walk time in milliseconds",
		}, tierStatusLabels),
		vsanLastFullwalkTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_last_fullwalk_time_ms",
			Help: "Last full walk time in milliseconds",
		}, tierStatusLabels),
		vsanFullwalkStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_fullwalk_status",
			Help: "VSAN tier fullwalk status (1=in progress, 0=not in progress)",
		}, tierStatusLabels),
		vsanFullwalkProgress: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_fullwalk_progress",
			Help: "VSAN tier fullwalk progress percentage",
		}, tierStatusLabels),
		vsanCurSpaceThrottle: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_vsan_cur_space_throttle_ms",
			Help: "Current space throttle in milliseconds",
		}, tierStatusLabels),
	}

	return sc
}

// Describe implements prometheus.Collector
func (sc *StorageCollector) Describe(ch chan<- *prometheus.Desc) {
	// VSAN tier capacity metrics
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
		sc.vsanCapacity.WithLabelValues(systemName, tierStr, tier.Description).Set(float64(tier.Capacity))
		sc.vsanUsed.WithLabelValues(systemName, tierStr, tier.Description).Set(float64(tier.Used))
		sc.vsanAllocated.WithLabelValues(systemName, tierStr, tier.Description).Set(float64(tier.Allocated))
		// SDK DedupeRatio is uint32 (multiply by 0.01 for ratio), convert to float
		sc.vsanDedupeRatio.WithLabelValues(systemName, tierStr, tier.Description).Set(float64(tier.DedupeRatio) / 100.0)
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
			systemName,
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
	}

	// Collect all metrics
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
}

// boolToFloat64 converts a boolean to 1.0 or 0.0 for Prometheus gauges.
func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
