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
	systemVersionDesc       *prometheus.Desc
	systemVersionLatestDesc *prometheus.Desc
	systemBranchDesc        *prometheus.Desc
	systemInfoDesc          *prometheus.Desc
}

// NewSystemCollector creates a new SystemCollector
func NewSystemCollector(url string, client *http.Client, username, password string) *SystemCollector {
	sc := &SystemCollector{
		BaseCollector: BaseCollector{
			url:        url,
			httpClient: client,
		},
		systemName: "unknown", // Will be updated in Collect
		systemVersionDesc: prometheus.NewDesc(
			"vergeos_system_version",
			"Current version of the VergeOS system (always 1, version in label)",
			[]string{"system_name", "version"},
			nil,
		),
		systemVersionLatestDesc: prometheus.NewDesc(
			"vergeos_system_version_latest",
			"Latest available version of the VergeOS system (always 1, version in label)",
			[]string{"system_name", "version"},
			nil,
		),
		systemBranchDesc: prometheus.NewDesc(
			"vergeos_system_branch",
			"Branch of the VergeOS system (always 1, branch in label)",
			[]string{"system_name", "branch"},
			nil,
		),
		systemInfoDesc: prometheus.NewDesc(
			"vergeos_system_info",
			"Information about the VergeOS system",
			[]string{"system_name", "current_version", "latest_version", "branch"},
			nil,
		),
	}

	// Authenticate with the API
	if err := sc.authenticate(username, password); err != nil {
		fmt.Printf("Error authenticating with VergeOS API: %v\n", err)
	}

	return sc
}

// Describe implements prometheus.Collector
func (sc *SystemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- sc.systemVersionDesc
	ch <- sc.systemVersionLatestDesc
	ch <- sc.systemBranchDesc
	ch <- sc.systemInfoDesc
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
			ch <- prometheus.MustNewConstMetric(sc.systemVersionDesc, prometheus.GaugeValue, 1, sc.systemName, pkg.Version)

			// Set branch as a string label
			ch <- prometheus.MustNewConstMetric(sc.systemBranchDesc, prometheus.GaugeValue, 1, sc.systemName, pkg.Branch)

			// Get latest version from the first source package
			latestVersion := ""
			if len(pkg.SourcePackages) > 0 {
				latestVersion = pkg.SourcePackages[0].Version
				ch <- prometheus.MustNewConstMetric(sc.systemVersionLatestDesc, prometheus.GaugeValue, 1, sc.systemName, latestVersion)

			}

			// Set system info metric with all information as labels
			ch <- prometheus.MustNewConstMetric(sc.systemInfoDesc, prometheus.GaugeValue, 1, sc.systemName, pkg.Version, latestVersion, pkg.Branch)

			break
		}
	}

	// Collect all metrics
}
