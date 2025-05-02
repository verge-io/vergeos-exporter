package collectors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// UpdatePackage represents a package in the update_dashboard response
type UpdatePackage struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Version        string `json:"version"`
	Branch         string `json:"branch"`
	SourcePackages []struct {
		Key        int           `json:"$key"`
		Downloaded bool          `json:"downloaded"`
		Version    string        `json:"version"`
		Files      []interface{} `json:"files"`
	} `json:"source_packages"`
}

// UpdateDashboardResponse represents the API response for update_dashboard
type UpdateDashboardResponse struct {
	Packages []UpdatePackage `json:"packages"`
	// Other fields like logs, branches, settings, etc. are omitted as we don't need them
}

// SystemCollector collects metrics about VergeOS system versions
type SystemCollector struct {
	BaseCollector
	mutex sync.Mutex

	// System info
	systemName string

	// Metrics
	systemVersion       *prometheus.GaugeVec
	systemVersionLatest *prometheus.GaugeVec
	systemBranch        *prometheus.GaugeVec
	systemInfo          *prometheus.GaugeVec
}

// NewSystemCollector creates a new SystemCollector
func NewSystemCollector(url string, client *http.Client, username, password string) *SystemCollector {
	sc := &SystemCollector{
		BaseCollector: BaseCollector{
			url:        url,
			httpClient: client,
		},
		systemName: "unknown", // Will be updated in Collect
		systemVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_system_version",
			Help: "Current version of the VergeOS system (always 1, version in label)",
		}, []string{"system_name", "version"}),
		systemVersionLatest: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_system_version_latest",
			Help: "Latest available version of the VergeOS system (always 1, version in label)",
		}, []string{"system_name", "version"}),
		systemBranch: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_system_branch",
			Help: "Branch of the VergeOS system (always 1, branch in label)",
		}, []string{"system_name", "branch"}),
		systemInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vergeos_system_info",
			Help: "Information about the VergeOS system",
		}, []string{"system_name", "current_version", "latest_version", "branch"}),
	}

	// Authenticate with the API
	if err := sc.authenticate(username, password); err != nil {
		fmt.Printf("Error authenticating with VergeOS API: %v\n", err)
	}

	return sc
}

// Describe implements prometheus.Collector
func (sc *SystemCollector) Describe(ch chan<- *prometheus.Desc) {
	sc.systemVersion.Describe(ch)
	sc.systemVersionLatest.Describe(ch)
	sc.systemBranch.Describe(ch)
	sc.systemInfo.Describe(ch)
}

// Collect implements prometheus.Collector
func (sc *SystemCollector) Collect(ch chan<- prometheus.Metric) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// Get system name
	systemName, err := sc.getSystemName()
	if err != nil {
		fmt.Printf("Error getting system name: %v\n", err)
		return
	}
	sc.systemName = systemName

	// Get update dashboard data
	req, err := sc.makeRequest("GET", "/api/v4/update_dashboard?limit=50")
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var dashboard UpdateDashboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&dashboard); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return
	}

	// Find the ybos package
	for _, pkg := range dashboard.Packages {
		if pkg.Name == "ybos" {
			// Set current version as a string label
			sc.systemVersion.WithLabelValues(sc.systemName, pkg.Version).Set(1)

			// Set branch as a string label
			sc.systemBranch.WithLabelValues(sc.systemName, pkg.Branch).Set(1)

			// Get latest version from the first source package
			latestVersion := ""
			if len(pkg.SourcePackages) > 0 {
				latestVersion = pkg.SourcePackages[0].Version
				sc.systemVersionLatest.WithLabelValues(sc.systemName, latestVersion).Set(1)
			}

			// Set system info metric with all information as labels
			sc.systemInfo.WithLabelValues(sc.systemName, pkg.Version, latestVersion, pkg.Branch).Set(1)

			break
		}
	}

	// Collect all metrics
	sc.systemVersion.Collect(ch)
	sc.systemVersionLatest.Collect(ch)
	sc.systemBranch.Collect(ch)
	sc.systemInfo.Collect(ch)
}
