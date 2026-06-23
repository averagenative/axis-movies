// Package version holds build-time metadata, injected via -ldflags at release.
package version

var (
	// Version is the semantic version of the build (e.g. v0.1.0).
	Version = "dev"
	// Commit is the git SHA the binary was built from.
	Commit = "unknown"
	// Date is the RFC3339 build timestamp.
	Date = "unknown"
)
