package collectors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// BaseCollector provides common functionality for all collectors
type BaseCollector struct {
	url        string
	httpClient *http.Client
	username   string
	password   string
	mutex      sync.Mutex
}

// AuthResponse represents the API response for authentication
type AuthResponse struct {
	Location string `json:"location"`
	DBPath   string `json:"dbpath"`
	Row      int    `json:"$row"`
	Key      string `json:"$key"`
}

// makeRequest creates an HTTP request with proper authentication
func (bc *BaseCollector) makeRequest(method, path string) (*http.Request, error) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	req, err := http.NewRequest(method, bc.url+path, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Use basic auth for all requests
	req.SetBasicAuth(bc.username, bc.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-JSON-Non-Compact", "1")

	return req, nil
}

// authenticate is no longer needed since we use basic auth
func (bc *BaseCollector) authenticate(username, password string) error {
	bc.username = username
	bc.password = password
	return nil
}

// getSystemName retrieves the system name from the settings API
func (bc *BaseCollector) getSystemName() (string, error) {
	// Get system name
	req, err := bc.makeRequest("GET", "/api/v4/settings?fields=most&filter=key%20eq%20%22cloud_name%22")
	if err != nil {
		return "", fmt.Errorf("error creating system name request: %v", err)
	}

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error getting system name: %v", err)
	}
	defer resp.Body.Close()

	// Read and log the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}
	fmt.Printf("Base collector settings API response: %s\n", string(bodyBytes))

	// Create a new reader with the same bytes for JSON decoding
	var systemNameResp []struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&systemNameResp); err != nil {
		return "", fmt.Errorf("error decoding system name response: %v", err)
	}

	if len(systemNameResp) == 0 {
		return "", fmt.Errorf("no system name found in response")
	}

	return systemNameResp[0].Value, nil
}
