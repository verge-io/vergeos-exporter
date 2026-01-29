package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	vergeos "github.com/verge-io/goVergeOS"

	"vergeos-exporter/collectors"
)

var (
	listenAddress = flag.String("web.listen-address", ":9888", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	vergeURL      = flag.String("verge.url", "http://localhost", "Base URL of the VergeOS API")
	vergeUsername = flag.String("verge.username", "", "Username for VergeOS API authentication")
	vergePassword = flag.String("verge.password", "", "Password for VergeOS API authentication")
	scrapeTimeout = flag.Duration("scrape.timeout", 30*time.Second, "Timeout for scraping VergeOS API")
	insecure      = flag.Bool("insecure", true, "Skip TLS certificate verification (default: true since VergeOS typically uses self-signed certificates)")
)

func main() {
	flag.Parse()

	// Validate required flags
	if *vergeUsername == "" || *vergePassword == "" {
		log.Fatal("verge.username and verge.password are required")
	}

	// Create SDK client for API operations
	client, err := vergeos.NewClient(
		vergeos.WithBaseURL(*vergeURL),
		vergeos.WithCredentials(*vergeUsername, *vergePassword),
		vergeos.WithInsecureTLS(*insecure),
		vergeos.WithTimeout(*scrapeTimeout),
	)
	if err != nil {
		log.Fatalf("Failed to create VergeOS client: %v", err)
	}

	// Validate credentials at startup (Bug #34: fail fast with clear error message)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cloudName, err := client.Settings.GetCloudName(ctx)
	if err != nil {
		if vergeos.IsAuthError(err) {
			log.Fatalf("Authentication failed: check username/password for %s", *vergeURL)
		}
		log.Fatalf("Failed to connect to VergeOS API at %s: %v", *vergeURL, err)
	}
	log.Printf("Successfully connected to VergeOS system: %s", cloudName)

	// Initialize collectors with SDK client
	// StorageCollector is fully migrated to SDK (Phase 3a/3b/3c complete)
	// NodeCollector is fully migrated to SDK (Phase 4 complete)
	// Other collectors still need URL/credentials until their migration phase
	storageCollector := collectors.NewStorageCollector(client)
	nodeCollector := collectors.NewNodeCollector(client)
	networkCollector := collectors.NewNetworkCollector(client, *vergeURL, *vergeUsername, *vergePassword)
	clusterCollector := collectors.NewClusterCollector(client, *vergeURL, *vergeUsername, *vergePassword)
	systemCollector := collectors.NewSystemCollector(client, *vergeURL, *vergeUsername, *vergePassword)

	prometheus.MustRegister(nodeCollector)
	prometheus.MustRegister(storageCollector)
	prometheus.MustRegister(networkCollector)
	prometheus.MustRegister(clusterCollector)
	prometheus.MustRegister(systemCollector)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>VergeOS Exporter</title></head>
			<body>
			<h1>VergeOS Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Printf("Starting VergeOS exporter on %s", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
