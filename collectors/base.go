package collectors

import (
	"context"
	"fmt"
	"sync"
	"time"

	vergeos "github.com/verge-io/govergeos"
)

// BaseCollector provides common functionality for all collectors
type BaseCollector struct {
	// SDK client for API operations
	client *vergeos.Client

	// Timeout for scrape operations
	scrapeTimeout time.Duration

	// Cached system name
	systemName string

	mutex sync.Mutex
}

// NewBaseCollector creates a new BaseCollector with SDK client and scrape timeout
func NewBaseCollector(client *vergeos.Client, scrapeTimeout time.Duration) *BaseCollector {
	return &BaseCollector{
		client:        client,
		scrapeTimeout: scrapeTimeout,
	}
}

// ScrapeContext returns a context with the configured scrape timeout.
func (bc *BaseCollector) ScrapeContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), bc.scrapeTimeout)
}

// Client returns the SDK client for direct access by collectors
func (bc *BaseCollector) Client() *vergeos.Client {
	return bc.client
}

// GetSystemName retrieves the system name using the SDK with caching
// This method provides typed error handling for auth/permission issues (Bug #34)
func (bc *BaseCollector) GetSystemName(ctx context.Context) (string, error) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// Return cached value if available
	if bc.systemName != "" {
		return bc.systemName, nil
	}

	// Fetch using SDK
	name, err := bc.client.Settings.GetCloudName(ctx)
	if err != nil {
		// Provide typed error handling for better debugging
		if vergeos.IsAuthError(err) {
			return "", fmt.Errorf("authentication failed (check credentials): %w", err)
		}
		return "", fmt.Errorf("failed to get system name: %w", err)
	}

	bc.systemName = name
	return name, nil
}

// BuildClusterMap creates a mapping from cluster ID to cluster name.
func (bc *BaseCollector) BuildClusterMap(ctx context.Context) (map[int]string, error) {
	clusters, err := bc.client.Clusters.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	clusterMap := make(map[int]string)
	for _, cluster := range clusters {
		clusterMap[int(cluster.Key)] = cluster.Name
	}

	return clusterMap, nil
}
