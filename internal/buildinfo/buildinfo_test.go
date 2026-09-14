package buildinfo

import (
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
