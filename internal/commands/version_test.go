package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

func TestVersionCommandReportsVersionedBuildIdentity(t *testing.T) {
	command := newVersionCmd(model.VersionInfo{
		SchemaVersion: model.VersionInfoSchemaVersion,
		Version:       "v1.2.3",
		Commit:        "abcdef0123456789",
		Date:          "2026-09-14T10:00:00Z",
	})
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--format", "json"})

	require.NoError(t, command.Execute())

	var info model.VersionInfo
	require.NoError(t, json.Unmarshal(output.Bytes(), &info))
	assert.Equal(t, model.VersionInfoSchemaVersion, info.SchemaVersion)
	assert.Equal(t, "v1.2.3", info.Version)
	assert.Equal(t, "abcdef0123456789", info.Commit)
	assert.Equal(t, "2026-09-14T10:00:00Z", info.Date)
}

func TestVersionCommandRendersBuildIdentityForTerminals(t *testing.T) {
	command := newVersionCmd(model.VersionInfo{
		SchemaVersion: model.VersionInfoSchemaVersion,
		Version:       "v1.2.3",
		Commit:        "abcdef0123456789",
		Date:          "2026-09-14T10:00:00Z",
	})
	var output bytes.Buffer
	command.SetOut(&output)

	require.NoError(t, command.Execute())
	assert.Equal(t, "GHA version: v1.2.3\nCommit: abcdef012345\nBuilt: 2026-09-14T10:00:00Z\n", output.String())
}
