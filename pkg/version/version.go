// Package version provides version information for tfpipboy.
package version

// Version is the current version of tfpipboy
// This should match the version in cmd/tfpipboy/main.go
const Version = "0.7.0-beta.2"

// Name is the application name
const Name = "tfpipboy"

// FullVersion returns the full version string
func FullVersion() string {
	return Name + " v" + Version
}
