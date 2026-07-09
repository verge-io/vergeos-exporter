//go:build !windows

package main

import "fmt"

// runWindowsService is a no-op on non-Windows platforms. If the user passes
// -service anyway, report a clear error rather than silently ignoring it.
// Returning handled=false lets main() fall through to a normal foreground run.
func runWindowsService(action string) (handled bool, err error) {
	if action != "" {
		return true, fmt.Errorf("-service is only supported on Windows")
	}
	return false, nil
}
