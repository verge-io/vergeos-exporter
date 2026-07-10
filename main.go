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
	vergeAPIKey   = flag.String("verge.apikey", "", "API key for VergeOS API authentication (alternative to username/password)")
	scrapeTimeout = flag.Duration("scrape.timeout", 30*time.Second, "Timeout for scraping VergeOS API")
	insecure      = flag.Bool("insecure", false, "Skip TLS certificate verification (use for self-signed certificates)")
	logFile       = flag.String("log.file", "", "Write logs to this file instead of stderr (useful when running as a service).")
	serviceAction = flag.String("service", "", "Windows service control action: install, uninstall, start, stop, run (Windows only).")
)

func main() {
	flag.Parse()

	// Redirect logging to a file when requested (services have no console).
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file %s: %v", *logFile, err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

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
	if *vergeAPIKey == "" {
		*vergeAPIKey = os.Getenv("VERGE_API_KEY")
	}

	// Windows service control/hosting. On non-Windows platforms this is a no-op
	// unless -service was passed, in which case it reports a clear error.
	if handled, err := runWindowsService(*serviceAction); handled {
		if err != nil {
			log.Fatalf("Service error: %v", err)
		}
		return
	}

	// Foreground execution (all platforms).
	if err := runExporter(nil); err != nil {
		log.Fatal(err)
	}
}

// runExporter builds the SDK client, registers collectors, and serves metrics
// until stop is closed, an OS signal arrives, or the HTTP server fails. It is
// platform-neutral: main() calls it directly for foreground runs, and the
// Windows service handler calls it with a stop channel driven by the SCM.
func runExporter(stop <-chan struct{}) error {
	auth, err := authOption(*vergeAPIKey, *vergeUsername, *vergePassword)
	if err != nil {
		return err
	}

	if *insecure {
		log.Printf("WARNING: TLS certificate verification is disabled (--insecure flag)")
	}

	// Create SDK client for API operations
	client, err := vergeos.NewClient(
		vergeos.WithBaseURL(*vergeURL),
		auth,
		vergeos.WithInsecureTLS(*insecure),
		vergeos.WithTimeout(*scrapeTimeout),
	)
	if err != nil {
		return fmt.Errorf("failed to create VergeOS client: %w", err)
	}

	// Validate credentials at startup (Bug #34: fail fast with clear error message)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cloudName, err := client.Settings.GetCloudName(ctx)
	if err != nil {
		if vergeos.IsAuthError(err) {
			return fmt.Errorf("authentication failed: check API key or username/password for %s", *vergeURL)
		}
		return fmt.Errorf("failed to connect to VergeOS API at %s: %w", *vergeURL, err)
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
	vnetCollector := collectors.NewVNetCollector(client, *scrapeTimeout)

	// Use a dedicated registry so repeated runs don't collide on the global default.
	registry := prometheus.NewRegistry()
	registry.MustRegister(nodeCollector)
	registry.MustRegister(storageCollector)
	registry.MustRegister(networkCollector)
	registry.MustRegister(clusterCollector)
	registry.MustRegister(systemCollector)
	registry.MustRegister(tenantCollector)
	registry.MustRegister(vmCollector)
	registry.MustRegister(vnetCollector)

	mux := http.NewServeMux()
	mux.Handle(*metricsPath, promhttp.HandlerFor(
		registry,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: *scrapeTimeout + 5*time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM or when the caller closes stop
	// (the latter is how the Windows service handler asks us to stop).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-quit:
		case <-stop:
		}
		log.Println("Shutting down...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Starting VergeOS exporter on %s", *listenAddress)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func authOption(apiKey, username, password string) (vergeos.ClientOption, error) {
	if apiKey != "" {
		return vergeos.WithAPIKey(apiKey), nil
	}
	if username == "" || password == "" {
		return nil, fmt.Errorf("authentication required: provide verge.apikey (or VERGE_API_KEY) or both verge.username and verge.password (or VERGE_USERNAME/VERGE_PASSWORD)")
	}
	return vergeos.WithCredentials(username, password), nil
}
