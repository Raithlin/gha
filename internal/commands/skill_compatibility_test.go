package commands

import (
	"testing"

	ghaskill "github.com/raithlin/gha/skills/gha"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedSkillRequiresCapabilitiesAvailableInThisBuild(t *testing.T) {
	missing, err := validateSkillCapabilities(ghaskill.Skill, ghaCapabilities())
	require.NoError(t, err)
	assert.Empty(t, missing)
}

func TestValidateSkillCapabilitiesReportsUnsupportedCommands(t *testing.T) {
	skill := skillWithRequirements("capabilities,dashboard,worktree create")
	missing, err := validateSkillCapabilities(skill, ghaCapabilities())
	require.NoError(t, err)
	assert.Equal(t, []string{"dashboard", "worktree create"}, missing)
}

func TestValidateSkillCapabilitiesRequiresAManifest(t *testing.T) {
	_, err := validateSkillCapabilities([]byte("---\nname: gha\n---\nInstructions"), ghaCapabilities())
	require.ErrorContains(t, err, "does not declare gha-required-capabilities")
}

func TestValidateSkillCapabilitiesRejectsMalformedMetadata(t *testing.T) {
	for _, skill := range [][]byte{
		[]byte("not frontmatter"),
		[]byte("---\nname: gha\nmetadata:\n  gha-required-capabilities: capabilities\n"),
		[]byte("---\nname: gha\nmetadata: [\n---\nInstructions"),
		skillWithRequirements(""),
		skillWithRequirements("capabilities,,update"),
	} {
		_, err := validateSkillCapabilities(skill, ghaCapabilities())
		require.Error(t, err)
	}
}

func TestValidateSkillCapabilitiesTreatsUnknownInventoryAsUnavailable(t *testing.T) {
	missing, err := validateSkillCapabilities(skillWithRequirements("capabilities"), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"capabilities"}, missing)
}

func skillWithRequirements(requirements string) []byte {
	return []byte("---\nname: gha\nmetadata:\n  gha-required-capabilities: \"" + requirements + "\"\n---\nInstructions\n")
}
