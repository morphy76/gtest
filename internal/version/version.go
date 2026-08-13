// Package version holds build-time version information injected via ldflags.
package version

var (
	// Version is the SemVer version string, injected at build time.
	Version = "dev"

	// Commit is the short git commit hash, injected at build time.
	Commit = "none"

	// BuildTime is the UTC build timestamp, injected at build time.
	BuildTime = "unknown"
)
