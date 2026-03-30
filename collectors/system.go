package collectors

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	vergeos "github.com/verge-io/govergeos"
)

var _ prometheus.Collector = (*SystemCollector)(nil)

// SystemCollector collects metrics about VergeOS system versions
type SystemCollector struct {
	BaseCollector
	mutex sync.Mutex

	// Metric descriptors (using MustNewConstMetric pattern to avoid stale metrics)
	systemVersion       *prometheus.Desc
	systemInfo          *prometheus.Desc
	systemBranch        *prometheus.Desc
	systemVersionLatest *prometheus.Desc
}

// NewSystemCollector creates a new SystemCollector
func NewSystemCollector(client *vergeos.Client, scrapeTimeout time.Duration) *SystemCollector {
	return &SystemCollector{
		BaseCollector: *NewBaseCollector(client, scrapeTimeout),
		systemVersion: prometheus.NewDesc(
			"vergeos_system_version",
			"Current version of the VergeOS system (always 1, version in label)",
			[]string{"system_name", "version"},
			nil,
		),
		systemInfo: prometheus.NewDesc(
			"vergeos_system_info",
			"Information about the VergeOS system",
			[]string{"system_name", "current_version", "latest_version", "branch", "hash"},
			nil,
		),
		systemBranch: prometheus.NewDesc(
			"vergeos_system_branch",
			"Update branch of the VergeOS system (always 1, branch in label)",
			[]string{"system_name", "branch"},
			nil,
		),
		systemVersionLatest: prometheus.NewDesc(
			"vergeos_system_version_latest",
			"Latest available version of the VergeOS system (always 1, version in label)",
			[]string{"system_name", "version"},
			nil,
		),
	}
}

// Describe implements prometheus.Collector
func (sc *SystemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- sc.systemVersion
	ch <- sc.systemInfo
	ch <- sc.systemBranch
	ch <- sc.systemVersionLatest
}

// Collect implements prometheus.Collector
func (sc *SystemCollector) Collect(ch chan<- prometheus.Metric) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	ctx, cancel := sc.ScrapeContext()
	defer cancel()

	// Get system name using BaseCollector (SDK)
	systemName, err := sc.GetSystemName(ctx)
	if err != nil {
		log.Printf("Error getting system name: %v", err)
		return
	}

	// Get system info using SDK
	info, err := sc.client.System.GetInfo(ctx)
	if err != nil {
		log.Printf("Error getting system info: %v", err)
		return
	}

	// Emit system version metric
	ch <- prometheus.MustNewConstMetric(
		sc.systemVersion,
		prometheus.GaugeValue,
		1.0,
		systemName, info.Version,
	)

	// Get update settings for branch and latest version
	branchName := ""
	latestVersion := ""

	settings, err := sc.client.UpdateSettings.Get(ctx)
	if err != nil {
		log.Printf("Error getting update settings: %v", err)
	} else {
		branchName = settings.BranchName

		// Emit branch metric
		ch <- prometheus.MustNewConstMetric(
			sc.systemBranch,
			prometheus.GaugeValue,
			1.0,
			systemName, branchName,
		)

		// Get latest available version from source packages
		pkgs, err := sc.client.UpdateSourcePackages.ListByBranchAndSource(ctx, settings.Branch, settings.Source)
		if err != nil {
			log.Printf("Error getting update source packages: %v", err)
		} else {
			for _, pkg := range pkgs {
				if pkg.Name == "ybos" {
					latestVersion = pkg.Version
					ch <- prometheus.MustNewConstMetric(
						sc.systemVersionLatest,
						prometheus.GaugeValue,
						1.0,
						systemName, latestVersion,
					)
					break
				}
			}
		}
	}

	// Emit system info metric with all available fields
	ch <- prometheus.MustNewConstMetric(
		sc.systemInfo,
		prometheus.GaugeValue,
		1.0,
		systemName, info.Version, latestVersion, branchName, info.Hash,
	)
}
