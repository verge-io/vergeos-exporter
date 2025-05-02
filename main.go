package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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

	httpClient := &http.Client{
		Timeout: *scrapeTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: *insecure},
		},
	}

	nodeCollector := collectors.NewNodeCollector(*vergeURL, httpClient, *vergeUsername, *vergePassword)
	storageCollector := collectors.NewStorageCollector(*vergeURL, httpClient, *vergeUsername, *vergePassword)
	networkCollector := collectors.NewNetworkCollector(*vergeURL, httpClient, *vergeUsername, *vergePassword)
	clusterCollector := collectors.NewClusterCollector(*vergeURL, httpClient, *vergeUsername, *vergePassword)
	systemCollector := collectors.NewSystemCollector(*vergeURL, httpClient, *vergeUsername, *vergePassword)

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
