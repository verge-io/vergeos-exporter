package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	vergeos "github.com/verge-io/govergeos"
)

func TestAuthOptionUsesAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/version.json") {
			// The version probe is unauthenticated by design (govergeos ADR-016).
			fmt.Fprint(w, `{"version":"26.0.0"}`)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want Bearer token", got)
		}
		fmt.Fprint(w, `[{"$key":1}]`)
	}))
	defer server.Close()

	auth, err := authOption("test-key", "ignored-user", "ignored-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vergeos.NewClient(vergeos.WithBaseURL(server.URL), auth); err != nil {
		t.Fatal(err)
	}

	if _, err := authOption("", "user", ""); err == nil {
		t.Fatal("expected incomplete username/password to fail")
	}
}

// Issue: VSM #148. Exporter 2.1.2 (govergeos v0.3.0) rejected every major
// other than 26. The SDK floor is now a minimum: 26 and later majors pass.
func TestClientAcceptsLaterMajorVersions(t *testing.T) {
	for _, tc := range []struct {
		version string
		wantErr bool
	}{
		{"26.2.1.2.7", false},
		{"27.1.0.dev-3-gab03828", false},
		{"25.9.9", true},
	} {
		t.Run(tc.version, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/version.json") {
					fmt.Fprintf(w, `{"version":%q}`, tc.version)
					return
				}
				fmt.Fprint(w, `[{"$key":1}]`)
			}))
			defer server.Close()
			_, err := vergeos.NewClient(vergeos.WithBaseURL(server.URL), vergeos.WithAPIKey("k"))
			if tc.wantErr && !vergeos.IsUnsupportedVersionError(err) {
				t.Fatalf("version %s: want UnsupportedVersionError, got %v", tc.version, err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("version %s: unexpected error: %v", tc.version, err)
			}
		})
	}
}

func TestRunExporterRejectsInvalidAPIKey(t *testing.T) {
	var authenticatedRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/version.json") {
			fmt.Fprint(w, `{"version":"26.0.0"}`)
			return
		}
		authenticatedRequests.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer rejected-key" {
			t.Errorf("Authorization = %q, want Bearer rejected-key", got)
		}
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"err":"Login required"}`)
	}))
	defer server.Close()

	oldURL, oldAPIKey, oldListenAddress := *vergeURL, *vergeAPIKey, *listenAddress
	t.Cleanup(func() {
		*vergeURL, *vergeAPIKey, *listenAddress = oldURL, oldAPIKey, oldListenAddress
	})
	*vergeURL, *vergeAPIKey = server.URL, "rejected-key"
	// Fail without binding a port if startup unexpectedly reaches the listener.
	*listenAddress = "invalid-address"

	err := runExporter(nil)
	want := fmt.Sprintf("authentication failed: check API key or username/password for %s", server.URL)
	if err == nil || err.Error() != want {
		t.Fatalf("runExporter() = %v, want %q", err, want)
	}
	if got := authenticatedRequests.Load(); got != 1 {
		t.Errorf("authenticated requests = %d, want 1 (no credential retries)", got)
	}
}
