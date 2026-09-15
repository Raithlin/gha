package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ghaskill "github.com/raithlin/gha/skills/gha"
)

func TestAgentInstallCopiesCodexSkillAndGuidanceIdempotently(t *testing.T) {
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "claude"))
	codexHome := os.Getenv("CODEX_HOME")
	require.NoError(t, os.MkdirAll(codexHome, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(codexHome, "AGENTS.md"), []byte("# Personal instructions\n"), 0o644))

	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "install", "--agent", "codex", "--confirm"})
	require.NoError(t, root.Execute())

	skill, err := os.ReadFile(filepath.Join(codexHome, "skills", "gha", "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, ghaskill.Skill, skill)
	guidancePath := filepath.Join(codexHome, "AGENTS.md")
	guidance, err := os.ReadFile(guidancePath)
	require.NoError(t, err)
	assert.Contains(t, string(guidance), "# Personal instructions")
	assert.Equal(t, 1, strings.Count(string(guidance), "<!-- gha:begin -->"))

	firstInstall := string(guidance)
	root = NewRootCmd(nil, nil, nil)
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "install", "--agent", "codex", "--confirm"})
	require.NoError(t, root.Execute())
	guidance, err = os.ReadFile(guidancePath)
	require.NoError(t, err)
	assert.Equal(t, firstInstall, string(guidance))
}

func TestAgentInstallPromptsForClaudeCode(t *testing.T) {
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "claude"))
	claudeHome := os.Getenv("CLAUDE_CONFIG_DIR")

	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetIn(strings.NewReader("2\n"))
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "install", "--confirm"})
	require.NoError(t, root.Execute())

	skill, err := os.ReadFile(filepath.Join(claudeHome, "skills", "gha", "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, ghaskill.Skill, skill)
	assert.Contains(t, output.String(), "Claude Code")
}

func TestAgentInstallDryRunDoesNotWrite(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)

	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "install", "--agent", "codex", "--dry-run"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(codexHome)
	assert.True(t, os.IsNotExist(err))
	assert.Contains(t, output.String(), "Would install gha skill for Codex")
}

func TestAgentInstallRequiresConfirmationForWrites(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "install", "--agent", "codex"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--confirm")
}

func TestAgentUninstallRemovesManagedGuidanceAndSkill(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)
	guidancePath := filepath.Join(codexHome, "AGENTS.md")
	skillPath := filepath.Join(codexHome, "skills", "gha", "SKILL.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(skillPath), 0o755))
	require.NoError(t, os.WriteFile(skillPath, []byte("installed skill"), 0o644))
	require.NoError(t, os.WriteFile(guidancePath, []byte("# Personal instructions\n\n<!-- gha:begin -->\nmanaged guidance\n<!-- gha:end -->\n\n# More personal instructions\n"), 0o644))

	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex", "--confirm"})
	require.NoError(t, root.Execute())

	guidance, err := os.ReadFile(guidancePath)
	require.NoError(t, err)
	assert.NotContains(t, string(guidance), "gha:begin")
	assert.Contains(t, string(guidance), "# Personal instructions")
	assert.Contains(t, string(guidance), "# More personal instructions")
	_, err = os.Stat(skillPath)
	assert.True(t, os.IsNotExist(err))
	assert.Contains(t, output.String(), "Removed managed GHA guidance and skill for Codex")
}

func TestAgentUninstallPreservesOtherFilesInTheSkillDirectory(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)
	skillPath := filepath.Join(codexHome, "skills", "gha", "SKILL.md")
	additionalPath := filepath.Join(codexHome, "skills", "gha", "notes.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(skillPath), 0o755))
	require.NoError(t, os.WriteFile(skillPath, []byte("installed skill"), 0o644))
	require.NoError(t, os.WriteFile(additionalPath, []byte("personal note"), 0o644))

	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex", "--confirm"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(skillPath)
	assert.True(t, os.IsNotExist(err))
	note, err := os.ReadFile(additionalPath)
	require.NoError(t, err)
	assert.Equal(t, "personal note", string(note))
}

func TestAgentUninstallRemovesGuidanceFileWhenItContainsOnlyManagedGuidance(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)
	guidancePath := filepath.Join(codexHome, "AGENTS.md")
	require.NoError(t, os.MkdirAll(codexHome, 0o755))
	require.NoError(t, os.WriteFile(guidancePath, ghaskill.Guidance, 0o644))

	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex", "--confirm"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(guidancePath)
	assert.True(t, os.IsNotExist(err))
}

func TestAgentUninstallDryRunAndMissingGuidanceDoNotWrite(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)

	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex", "--dry-run"})
	require.NoError(t, root.Execute())
	_, err := os.Stat(codexHome)
	assert.True(t, os.IsNotExist(err))
	assert.Contains(t, output.String(), "Would remove managed GHA guidance for Codex")
	assert.Contains(t, output.String(), "gha skill")

	root = NewRootCmd(nil, nil, nil)
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex", "--confirm"})
	require.NoError(t, root.Execute())
	_, err = os.Stat(codexHome)
	assert.True(t, os.IsNotExist(err))
}

func TestAgentUninstallPromptsForTarget(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "claude"))

	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetIn(strings.NewReader("2\n"))
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "uninstall", "--dry-run"})
	require.NoError(t, root.Execute())

	assert.Contains(t, output.String(), "remove GHA guidance and skill")
	assert.Contains(t, output.String(), "Claude Code")
}

func TestAgentUninstallRequiresConfirmationAndRejectsMalformedManagedGuidance(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)
	require.NoError(t, os.MkdirAll(codexHome, 0o755))
	guidancePath := filepath.Join(codexHome, "AGENTS.md")
	require.NoError(t, os.WriteFile(guidancePath, []byte("# Personal\n<!-- gha:begin -->\n"), 0o644))

	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--confirm")

	root = NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex", "--confirm"})
	err = root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without <!-- gha:end -->")
}

func TestAgentUninstallResultMessage(t *testing.T) {
	tests := []struct {
		name     string
		result   agentUninstallResult
		expected string
	}{
		{
			name:     "guidance and skill removed",
			result:   agentUninstallResult{guidanceRemoved: true, skillRemoved: true},
			expected: "Removed managed GHA guidance and skill for Codex.\n",
		},
		{
			name:     "guidance removed",
			result:   agentUninstallResult{guidanceRemoved: true},
			expected: "Removed managed GHA guidance for Codex; no gha skill was found.\n",
		},
		{
			name:     "skill removed",
			result:   agentUninstallResult{skillRemoved: true},
			expected: "Removed gha skill for Codex; no managed GHA guidance was found.\n",
		},
		{
			name:     "nothing removed",
			expected: "No managed GHA guidance or skill found for Codex.\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, agentUninstallResultMessage("Codex", test.result))
		})
	}
}
