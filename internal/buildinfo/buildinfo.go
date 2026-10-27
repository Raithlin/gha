// Package buildinfo exposes the identity embedded in a GHA binary.
package buildinfo

import (
	"runtime/debug"

	"github.com/raithlin/gha/pkg/model"
)

var (
	// Version is replaced by release builds through Go linker flags.
	Version = "dev"
	// Commit is replaced by release builds through Go linker flags.
	Commit = "none"
	// Date is replaced by release builds through Go linker flags.
	Date = "unknown"
)

// Current returns linker identity with embedded module and VCS fallbacks.
func Current() model.VersionInfo { return fromBuildInfo(debug.ReadBuildInfo()) }

func fromBuildInfo(info *debug.BuildInfo, ok bool) model.VersionInfo {
	result := model.VersionInfo{SchemaVersion: model.VersionInfoSchemaVersion, Version: Version, Commit: Commit, Date: Date}
	if !ok {
		return result
	}
	if result.Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		result.Version = info.Main.Version
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if result.Commit == "none" && setting.Value != "" {
				result.Commit = setting.Value
			}
		case "vcs.time":
			if result.Date == "unknown" && setting.Value != "" {
				result.Date = setting.Value
			}
		case "vcs.modified":
			if setting.Value == "true" || setting.Value == "false" {
				dirty := setting.Value == "true"
				result.Dirty = &dirty
			}
		}
	}
	return result
}
