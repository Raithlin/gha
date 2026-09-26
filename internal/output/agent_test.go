package output

import (
	"bytes"
	"errors"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type agentOutputFailure struct{}

func (agentOutputFailure) Write([]byte) (int, error) { return 0, errors.New("write failed") }

type agentOutputFailAfter struct{ remaining int }

func (writer *agentOutputFailAfter) Write(data []byte) (int, error) {
	if writer.remaining == 0 {
		return 0, errors.New("write failed")
	}
	writer.remaining--
	return len(data), nil
}

func TestAgentInstallationsRendersEmptyAndConfiguredResults(t *testing.T) {
	var rendered bytes.Buffer
	empty := &model.AgentInstallationList{SchemaVersion: "v1", Agents: []model.AgentInstallation{}}
	require.NoError(t, AgentInstallations(&rendered, Text, empty))
	assert.Contains(t, rendered.String(), "No coding-agent harnesses are recorded as configured by GHA")

	list := &model.AgentInstallationList{SchemaVersion: "v1", Agents: []model.AgentInstallation{{ID: "gemini", Name: "Gemini CLI", SkillPath: "/home/test/.agents/skills/gha/SKILL.md", InstructionsPath: "/home/test/.gemini/GEMINI.md", SkillState: "present", GuidanceState: "missing"}}}
	rendered.Reset()
	require.NoError(t, AgentInstallations(&rendered, Text, list))
	assert.Contains(t, rendered.String(), "Gemini CLI (gemini)")
	assert.Contains(t, rendered.String(), "GEMINI.md (missing)")
	assert.Contains(t, rendered.String(), "SKILL.md (present)")

	rendered.Reset()
	require.NoError(t, AgentInstallations(&rendered, JSON, list))
	assert.Contains(t, rendered.String(), `"skill_state": "present"`)
	rendered.Reset()
	require.NoError(t, AgentInstallations(&rendered, YAML, list))
	assert.Contains(t, rendered.String(), "guidance_state: missing")
}

func TestAgentInstallationsReturnsWriterErrors(t *testing.T) {
	empty := &model.AgentInstallationList{SchemaVersion: "v1", Agents: []model.AgentInstallation{}}
	assert.Error(t, AgentInstallations(agentOutputFailure{}, Text, empty))
	list := &model.AgentInstallationList{SchemaVersion: "v1", Agents: []model.AgentInstallation{{Name: "Pi", ID: "pi"}}}
	assert.Error(t, AgentInstallations(agentOutputFailure{}, Text, list))
}

func TestGuidanceUpdateRendersTextAndStructuredForms(t *testing.T) {
	update := &model.GuidanceUpdate{
		SchemaVersion: model.GuidanceUpdateSchemaVersion,
		LatestVersion: "v1.2.3",
		BinaryVersion: "v1.0.0",
		DryRun:        true,
		Targets:       []model.GuidanceUpdateTarget{{AgentID: "codex", AgentName: "Codex", SkillPath: "/home/test/skills/gha/SKILL.md", InstructionsPath: "/home/test/AGENTS.md", State: "planned"}},
	}
	var rendered bytes.Buffer
	require.NoError(t, GuidanceUpdate(&rendered, Text, update))
	assert.Contains(t, rendered.String(), "Codex: planned skill")
	assert.Contains(t, rendered.String(), "AGENTS.md")
	assert.Contains(t, rendered.String(), "executable remains at version v1.0.0")
	rendered.Reset()
	require.NoError(t, GuidanceUpdate(&rendered, JSON, update))
	assert.Contains(t, rendered.String(), `"latest_version": "v1.2.3"`)
	rendered.Reset()
	require.NoError(t, GuidanceUpdate(&rendered, YAML, update))
	assert.Contains(t, rendered.String(), "binary_updated: false")
}

func TestGuidanceUpdateRendersNoTargetsAndWriterErrors(t *testing.T) {
	update := &model.GuidanceUpdate{BinaryVersion: "dev", Targets: []model.GuidanceUpdateTarget{}}
	var rendered bytes.Buffer
	require.NoError(t, GuidanceUpdate(&rendered, Text, update))
	assert.Contains(t, rendered.String(), "nothing to update")
	assert.Error(t, GuidanceUpdate(agentOutputFailure{}, Text, update))
	update.Targets = []model.GuidanceUpdateTarget{{AgentName: "Copilot", SkillPath: "/tmp/SKILL.md", State: "updated"}}
	assert.Error(t, GuidanceUpdate(agentOutputFailure{}, Text, update))
	update.Targets[0].InstructionsPath = "/tmp/AGENTS.md"
	assert.Error(t, GuidanceUpdate(&agentOutputFailAfter{remaining: 2}, Text, update))
}
