// Package version provides build-time version information injected via ldflags.
package version

import (
	"fmt"
	"runtime"
)

// These variables are set at build time via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info returns a formatted version string.
func Info() string {
	return fmt.Sprintf("iron %s (commit: %s, built: %s, go: %s)",
		Version, Commit, Date, runtime.Version())
}

// Short returns just the version number.
func Short() string {
	return Version
}
