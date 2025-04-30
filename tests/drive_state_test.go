package tests

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"
)

// Configuration options
var (
	testExporterURL   = flag.String("exporter.url", "http://localhost:9888", "URL of the VergeOS exporter")
	testMetricsPath   = flag.String("exporter.metrics-path", "/metrics", "Path to metrics endpoint")
	testVergeURL      = flag.String("verge.url", "", "VergeOS API URL")
	testVergeUser     = flag.String("verge.username", "", "VergeOS API username")
	testVergePass     = flag.String("verge.password", "", "VergeOS API password")
	testStartExporter = flag.Bool("start-exporter", false, "Start the exporter as part of the test")
)

// DriveStateMetric represents a parsed drive state metric
type DriveStateMetric struct {
	SystemName string
	Tier       string
	State      string
	Value      float64
}

func main() {
	flag.Parse()

	// Validate required parameters
	if *testVergeURL == "" || *testVergeUser == "" || *testVergePass == "" {
		fmt.Println("Error: VergeOS URL, username, and password are required")
		flag.Usage()
		os.Exit(1)
	}

	// Start the exporter if requested
	if *testStartExporter {
		go startVergeOSExporter(*testVergeURL, *testVergeUser, *testVergePass)
		// Wait for the exporter to start
		time.Sleep(2 * time.Second)
	}

	// Fetch metrics from the exporter
	metricsURL := *testExporterURL + *testMetricsPath
	fmt.Printf("Fetching metrics from %s\n", metricsURL)

	resp, err := http.Get(metricsURL)
	if err != nil {
		fmt.Printf("Error fetching metrics: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: received status code %d\n", resp.StatusCode)
		os.Exit(1)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		os.Exit(1)
	}

	// Parse the drive state metrics
	metrics := parseDriveStateMetrics(string(body))

	// Print the metrics in a table format
	printMetricsTable(metrics)

	// Verify that all expected states are present
	verifyAllStatesPresent(metrics)
}

// startVergeOSExporter starts the VergeOS exporter
func startVergeOSExporter(vergeURL, username, password string) {
	cmd := exec.Command("../vergeos-exporter",
		"-verge.url="+vergeURL,
		"-verge.username="+username,
		"-verge.password="+password)

	fmt.Printf("Starting exporter: %v\n", cmd.Args)

	// Start the command
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Error starting exporter: %v\n", err)
		os.Exit(1)
	}
}

// parseDriveStateMetrics parses the drive state metrics from the response
func parseDriveStateMetrics(metricsText string) []DriveStateMetric {
	var metrics []DriveStateMetric

	// Regular expression to match drive state metrics
	re := regexp.MustCompile(`vergeos_vsan_drive_states{state="([^"]+)",system_name="([^"]+)",tier="([^"]+)"} ([0-9.]+)`)

	// Find all matches
	matches := re.FindAllStringSubmatch(metricsText, -1)

	for _, match := range matches {
		if len(match) == 5 {
			state := match[1]
			systemName := match[2]
			tier := match[3]
			value := match[4]

			// Parse the value as a float
			var floatValue float64
			fmt.Sscanf(value, "%f", &floatValue)

			metrics = append(metrics, DriveStateMetric{
				State:      state,
				SystemName: systemName,
				Tier:       tier,
				Value:      floatValue,
			})
		}
	}

	return metrics
}

// printMetricsTable prints the metrics in a table format
func printMetricsTable(metrics []DriveStateMetric) {
	// Print header
	fmt.Println("\nDrive State Metrics:")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("%-15s %-10s %-15s %-10s\n", "System", "Tier", "State", "Count")
	fmt.Println("------------------------------------------------------------")

	// Print metrics
	for _, metric := range metrics {
		fmt.Printf("%-15s %-10s %-15s %-10.0f\n",
			metric.SystemName,
			metric.Tier,
			metric.State,
			metric.Value)
	}
	fmt.Println("------------------------------------------------------------")
}

// verifyAllStatesPresent checks if all expected states are present in the metrics
func verifyAllStatesPresent(metrics []DriveStateMetric) {
	expectedStates := []string{
		"online",
		"offline",
		"repairing",
		"initializing",
		"verifying",
		"noredundant",
		"outofspace",
	}

	// Get unique tiers
	tierMap := make(map[string]bool)
	for _, metric := range metrics {
		tierMap[metric.Tier] = true
	}

	// Convert map to slice
	var tiers []string
	for tier := range tierMap {
		tiers = append(tiers, tier)
	}

	// Check if all states are present for each tier
	missingStates := false
	fmt.Println("\nVerifying all states are present:")

	for _, tier := range tiers {
		tierMetrics := filterMetricsByTier(metrics, tier)
		statesMap := make(map[string]bool)

		for _, metric := range tierMetrics {
			statesMap[metric.State] = true
		}

		for _, state := range expectedStates {
			if !statesMap[state] {
				fmt.Printf("WARNING: State '%s' is missing for tier %s\n", state, tier)
				missingStates = true
			}
		}
	}

	if !missingStates {
		fmt.Println("All expected states are present for all tiers.")
	}
}

// filterMetricsByTier returns metrics for a specific tier
func filterMetricsByTier(metrics []DriveStateMetric, tier string) []DriveStateMetric {
	var filtered []DriveStateMetric

	for _, metric := range metrics {
		if metric.Tier == tier {
			filtered = append(filtered, metric)
		}
	}

	return filtered
}
