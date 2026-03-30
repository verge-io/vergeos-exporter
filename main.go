package main

import (
	"context"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	vergeos "github.com/verge-io/govergeos"

	"vergeos-exporter/collectors"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	listenAddress = flag.String("web.listen-address", ":9888", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	vergeURL      = flag.String("verge.url", "http://localhost", "Base URL of the VergeOS API")
	vergeUsername = flag.String("verge.username", "", "Username for VergeOS API authentication")
	vergePassword = flag.String("verge.password", "", "Password for VergeOS API authentication")
	scrapeTimeout = flag.Duration("scrape.timeout", 30*time.Second, "Timeout for scraping VergeOS API")
	insecure      = flag.Bool("insecure", false, "Skip TLS certificate verification (use for self-signed certificates)")
)

func main() {
	flag.Parse()

	log.Printf("vergeos-exporter version=%s commit=%s date=%s", version, commit, date)

	// Environment variable fallback for credentials (avoids exposing secrets in /proc/cmdline)
	if *vergeURL == "http://localhost" {
		if v := os.Getenv("VERGE_URL"); v != "" {
			*vergeURL = v
		}
	}
	if *vergeUsername == "" {
		*vergeUsername = os.Getenv("VERGE_USERNAME")
	}
	if *vergePassword == "" {
		*vergePassword = os.Getenv("VERGE_PASSWORD")
	}

	if *vergeUsername == "" || *vergePassword == "" {
		log.Fatal("verge.username and verge.password are required (flags or VERGE_USERNAME/VERGE_PASSWORD env vars)")
	}

	if *insecure {
		log.Printf("WARNING: TLS certificate verification is disabled (--insecure flag)")
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

	// Initialize collectors with SDK client and scrape timeout
	storageCollector := collectors.NewStorageCollector(client, *scrapeTimeout)
	nodeCollector := collectors.NewNodeCollector(client, *scrapeTimeout)
	clusterCollector := collectors.NewClusterCollector(client, *scrapeTimeout)
	networkCollector := collectors.NewNetworkCollector(client, *scrapeTimeout)
	systemCollector := collectors.NewSystemCollector(client, *scrapeTimeout)
	tenantCollector := collectors.NewTenantCollector(client, *scrapeTimeout)
	vmCollector := collectors.NewVMCollector(client, *scrapeTimeout)

	prometheus.MustRegister(nodeCollector)
	prometheus.MustRegister(storageCollector)
	prometheus.MustRegister(networkCollector)
	prometheus.MustRegister(clusterCollector)
	prometheus.MustRegister(systemCollector)
	prometheus.MustRegister(tenantCollector)
	prometheus.MustRegister(vmCollector)

	http.Handle(*metricsPath, promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html>
			<head><title>VergeOS Exporter</title></head>
			<body>
			<h1>VergeOS Exporter</h1>
			<p><a href="%s">Metrics</a></p>
			</body>
			</html>`, html.EscapeString(*metricsPath))
	})

	srv := &http.Server{
		Addr:         *listenAddress,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: *scrapeTimeout + 5*time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Shutting down...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Starting VergeOS exporter on %s", *listenAddress)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
