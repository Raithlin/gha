package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

func TestParseRepository(t *testing.T) {
	target, err := ParseRepository("Raithlin/gha")
	require.NoError(t, err)
	assert.Equal(t, model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, target)

	_, err = ParseRepository("Raithlin/gha/extra")
	assert.Error(t, err)
}

func TestParseRepositoryRemote(t *testing.T) {
	for _, remote := range []string{
		"git@github.com:Raithlin/gha.git",
		"https://github.com/Raithlin/gha.git",
		"ssh://git@github.com/Raithlin/gha.git",
	} {
		target, err := ParseRepositoryRemote(remote)
		require.NoError(t, err, remote)
		assert.Equal(t, model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, target)
	}
}
