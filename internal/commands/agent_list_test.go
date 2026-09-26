package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentListReportsRecordedHarnessAndCurrentFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	rootDir := t.TempDir()
	target := agentInstallation{ID: "codex", Name: "Codex", SkillPath: filepath.Join(rootDir, "skills", "gha", "SKILL.md"), InstructionsPath: filepath.Join(rootDir, "AGENTS.md")}
	require.NoError(t, os.MkdirAll(filepath.Dir(target.SkillPath), 0o755))
	require.NoError(t, os.WriteFile(target.SkillPath, []byte("skill"), 0o644))
	require.NoError(t, recordAgentInstallation(target))

	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "list", "--format", "json"})
	require.NoError(t, root.Execute())
	assert.JSONEq(t, `{"schema_version":"v1","agents":[{"id":"codex","name":"Codex","skill_path":"`+target.SkillPath+`","instructions_path":"`+target.InstructionsPath+`","skill_state":"present","guidance_state":"missing"}]}`, output.String())

	root = NewRootCmd(nil, nil, nil)
	output.Reset()
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "list"})
	require.NoError(t, root.Execute())
	assert.Contains(t, output.String(), "Codex (codex)")
	assert.Contains(t, output.String(), "Skill:")
	assert.Contains(t, output.String(), "present")
	assert.Contains(t, output.String(), "Guidance:")
	assert.Contains(t, output.String(), "missing")
}

func TestAgentListEmptyInventoryAndYAML(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "list", "--format", "yaml"})
	require.NoError(t, root.Execute())
	assert.True(t, strings.Contains(output.String(), "schema_version: v1"))
	assert.True(t, strings.Contains(output.String(), "agents: []"))
}

func TestAgentListRejectsBadStateAndReportsFilesystemStatuses(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	path := filepath.Join(config, "gha", "agent-installations.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("broken"), 0o644))
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "list", "--format", "json"})
	assert.ErrorContains(t, root.Execute(), "decode agent ownership")

	root = NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "list", "--format", "xml"})
	assert.ErrorContains(t, root.Execute(), "unsupported format")

	directory := t.TempDir()
	assert.Equal(t, "not_configured", fileState(""))
	assert.Equal(t, "missing", fileState(filepath.Join(directory, "missing")))
	assert.Equal(t, "unavailable", fileState(directory))
	file := filepath.Join(directory, "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	assert.Equal(t, "present", fileState(file))
}
