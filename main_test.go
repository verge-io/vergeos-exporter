package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	vergeos "github.com/verge-io/govergeos"
)

func TestAuthOptionUsesAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want Bearer token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"version":"26.0.0"}`)
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
