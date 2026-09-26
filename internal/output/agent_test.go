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
