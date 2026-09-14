// Package buildinfo exposes the identity embedded in a GHA binary.
package buildinfo

import "github.com/raithlin/gha/pkg/model"

var (
	// Version is replaced by release builds through Go linker flags.
	Version = "dev"
	// Commit is replaced by release builds through Go linker flags.
	Commit = "none"
	// Date is replaced by release builds through Go linker flags.
	Date = "unknown"
)

// Current returns the versioned identity of this installed build.
func Current() model.VersionInfo {
	return model.VersionInfo{
		SchemaVersion: model.VersionInfoSchemaVersion,
		Version:       Version,
		Commit:        Commit,
		Date:          Date,
	}
}
