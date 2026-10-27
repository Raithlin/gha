package buildinfo

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raithlin/gha/pkg/model"
)

func TestCurrentReturnsDevelopmentIdentityWithoutReleaseLinkerFlags(t *testing.T) {
	assert.Equal(t, model.VersionInfo{
		SchemaVersion: model.VersionInfoSchemaVersion,
		Version:       "dev",
		Commit:        "none",
		Date:          "unknown",
	}, Current())
}

func TestEmbeddedBuildIdentity(t *testing.T) {
	originalVersion, originalCommit, originalDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = originalVersion, originalCommit, originalDate })
	for _, test := range []struct {
		name                              string
		info                              *debug.BuildInfo
		ok                                bool
		version, commit, date             string
		wantVersion, wantCommit, wantDate string
		dirty                             *bool
	}{
		{name: "missing", version: "dev", commit: "none", date: "unknown", wantVersion: "dev", wantCommit: "none", wantDate: "unknown"},
		{name: "devel", info: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, ok: true, version: "dev", commit: "none", date: "unknown", wantVersion: "dev", wantCommit: "none", wantDate: "unknown"},
		{name: "module", info: &debug.BuildInfo{Main: debug.Module{Version: "v0.3.0-alpha.0.20261027183555-fd9cda029a29"}}, ok: true, version: "dev", commit: "none", date: "unknown", wantVersion: "v0.3.0-alpha.0.20261027183555-fd9cda029a29", wantCommit: "none", wantDate: "unknown"},
		{name: "vcs", info: &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abcdef0123456789"}, {Key: "vcs.time", Value: "2026-10-27T18:35:55Z"}, {Key: "vcs.modified", Value: "true"}}}, ok: true, version: "dev", commit: "none", date: "unknown", wantVersion: "dev", wantCommit: "abcdef0123456789", wantDate: "2026-10-27T18:35:55Z", dirty: boolPointer(true)},
		{name: "release precedence", info: &debug.BuildInfo{Main: debug.Module{Version: "v0.2.0"}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "embedded"}, {Key: "vcs.time", Value: "embedded time"}, {Key: "vcs.modified", Value: "false"}}}, ok: true, version: "v1.0.0", commit: "release", date: "release time", wantVersion: "v1.0.0", wantCommit: "release", wantDate: "release time", dirty: boolPointer(false)},
		{name: "partial override", info: &debug.BuildInfo{Main: debug.Module{Version: "v0.2.0"}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "embedded"}, {Key: "vcs.time", Value: "embedded time"}, {Key: "vcs.modified", Value: "unknown"}}}, ok: true, version: "v1.0.0", commit: "none", date: "unknown", wantVersion: "v1.0.0", wantCommit: "embedded", wantDate: "embedded time"},
	} {
		t.Run(test.name, func(t *testing.T) {
			Version, Commit, Date = test.version, test.commit, test.date
			got := fromBuildInfo(test.info, test.ok)
			assert.Equal(t, test.wantVersion, got.Version)
			assert.Equal(t, test.wantCommit, got.Commit)
			assert.Equal(t, test.wantDate, got.Date)
			assert.Equal(t, test.dirty, got.Dirty)
		})
	}
}

func boolPointer(value bool) *bool { return &value }
