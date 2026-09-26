package commands

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ghaskill "github.com/raithlin/gha/skills/gha"
)

type failOnWriteNumber struct{ count, failAt int }

func (w *failOnWriteNumber) Write(p []byte) (int, error) {
	w.count++
	if w.count == w.failAt {
		return 0, errors.New("writer failed")
	}
	return len(p), nil
}

func TestAgentInstallCopiesCodexSkillAndGuidanceIdempotently(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "claude"))
	codexHome := os.Getenv("CODEX_HOME")
	require.NoError(t, os.MkdirAll(codexHome, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(codexHome, "AGENTS.md"), []byte("# Personal instructions\n"), 0o644))

	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "install", "--agent", "codex"})
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
	root.SetArgs([]string{"agent", "install", "--agent", "codex"})
	require.NoError(t, root.Execute())
	guidance, err = os.ReadFile(guidancePath)
	require.NoError(t, err)
	assert.Equal(t, firstInstall, string(guidance))
	ownership, err := readAgentOwnership()
	require.NoError(t, err)
	require.Len(t, ownership.Agents, 1)
	assert.Equal(t, "codex", ownership.Agents[0].ID)
}

func TestAgentUninstallRetainsSharedGuidanceAndUsesRecordedDestination(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	shared := filepath.Join(t.TempDir(), "shared")
	t.Setenv("CODEX_HOME", shared)
	t.Setenv("CLAUDE_CONFIG_DIR", shared)
	codex, err := codexInstallation()
	require.NoError(t, err)
	claude, err := claudeInstallation()
	require.NoError(t, err)
	require.NoError(t, installAgentGuidance(codex))
	require.NoError(t, recordAgentInstallation(codex))
	require.NoError(t, installAgentGuidance(claude))
	require.NoError(t, recordAgentInstallation(claude))
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "moved"))

	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex"})
	require.NoError(t, root.Execute())
	_, err = os.Stat(codex.SkillPath)
	require.NoError(t, err, "the shared skill must remain available to Claude Code")
	_, err = os.Stat(codex.InstructionsPath)
	assert.True(t, os.IsNotExist(err), "uninstall should use and remove the recorded Codex instruction destination")
	ownership, err := readAgentOwnership()
	require.NoError(t, err)
	require.Len(t, ownership.Agents, 1)
	assert.Equal(t, "claude", ownership.Agents[0].ID)
}

func TestAgentInstallSupportsMultipleAdditionalHarnessesAndSharedSkillOwnership(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, "pi-agent"))
	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "install", "--agent", "pi,opencode,copilot,gemini"})
	require.NoError(t, root.Execute())
	ownership, err := readAgentOwnership()
	require.NoError(t, err)
	require.Len(t, ownership.Agents, 4)
	shared := filepath.Join(home, ".agents", "skills", "gha", "SKILL.md")
	for _, agent := range ownership.Agents {
		assert.Equal(t, shared, agent.SkillPath)
	}
	assert.Equal(t, "not_configured", fileState(ownership.Agents[2].InstructionsPath))
	assert.Contains(t, output.String(), "Installed gha skill for GitHub Copilot.")

	root = NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "pi,opencode,copilot,gemini"})
	require.NoError(t, root.Execute())
	_, err = os.Stat(shared)
	assert.True(t, os.IsNotExist(err), "uninstalling every owner should remove the shared skill")
	ownership, err = readAgentOwnership()
	require.NoError(t, err)
	assert.Empty(t, ownership.Agents)
}

func TestAgentSelectionAcceptsInteractiveMultipleSelection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	targets, err := selectedAgentInstallations("", "configure", strings.NewReader("pi, gemini\n"), &bytes.Buffer{})
	require.NoError(t, err)
	assert.Equal(t, []string{"pi", "gemini"}, []string{targets[0].ID, targets[1].ID})
	targets, err = selectedAgentInstallations("codex,claude", "configure", strings.NewReader(""), &bytes.Buffer{})
	require.NoError(t, err)
	assert.Equal(t, []string{"codex", "claude"}, []string{targets[0].ID, targets[1].ID})
	_, err = selectedAgentInstallations("both", "configure", strings.NewReader(""), &bytes.Buffer{})
	assert.ErrorContains(t, err, "invalid agent")
}

func TestAgentInstallationsUseDocumentedGlobalPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, "pi"))
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(home, "opencode-custom"))
	for _, tc := range []struct{ id, instruction string }{
		{"pi", filepath.Join(home, "pi", "AGENTS.md")},
		{"opencode", filepath.Join(home, "opencode-custom", "AGENTS.md")},
		{"copilot", ""},
		{"gemini", filepath.Join(home, ".gemini", "GEMINI.md")},
	} {
		target, err := agentInstallationFor(tc.id)
		require.NoError(t, err)
		assert.Equal(t, tc.instruction, target.InstructionsPath)
		assert.Equal(t, filepath.Join(home, ".agents", "skills", "gha", "SKILL.md"), target.SkillPath)
	}
}

func TestAgentOwnershipReportsCorruptVersionsAndFilesystemFailures(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	path := filepath.Join(config, "gha", "agent-installations.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("{"), 0o644))
	_, err := readAgentOwnership()
	assert.ErrorContains(t, err, "decode agent ownership")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":2,"agents":[]}`), 0o644))
	_, err = readAgentOwnership()
	assert.ErrorContains(t, err, "unsupported agent ownership version")
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.Mkdir(path, 0o755))
	_, err = readAgentOwnership()
	assert.ErrorContains(t, err, "read agent ownership")
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.Remove(filepath.Dir(path)))
	require.NoError(t, os.WriteFile(filepath.Dir(path), []byte("blocker"), 0o644))
	assert.ErrorContains(t, writeAgentOwnership(agentOwnership{Version: 1, Agents: []agentInstallation{{ID: "pi"}}}), "write agent ownership")

	require.NoError(t, os.Remove(filepath.Dir(path)))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.Mkdir(path, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(path, "keep"), []byte("x"), 0o644))
	assert.ErrorContains(t, writeAgentOwnership(agentOwnership{Version: 1}), "remove agent ownership")
}

func TestAgentGlobalPathResolutionFailuresAndEmptySelection(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("PI_CODING_AGENT_DIR", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	_, err := sharedAgentSkillPath()
	assert.ErrorContains(t, err, "shared agent skills")
	_, err = piInstallation()
	assert.ErrorContains(t, err, "PI_CODING_AGENT_DIR")
	_, err = openCodeInstallation()
	assert.ErrorContains(t, err, "OpenCode config directory")
	_, err = copilotInstallation()
	assert.ErrorContains(t, err, "shared agent skills")
	_, err = geminiInstallation()
	assert.ErrorContains(t, err, "home directory")
	_, err = selectedAgentIDs("")
	assert.ErrorContains(t, err, "invalid agent")
	ids, err := selectedAgentIDs("all")
	require.NoError(t, err)
	assert.Equal(t, []string{"codex", "claude", "pi", "opencode", "copilot", "gemini"}, ids)
	assert.ErrorContains(t, agentOwnershipReadError(), "user config directory")
}

func TestAgentInstallReportsEveryOutputAndOwnershipFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	for _, failAt := range []int{2, 3} {
		root := NewRootCmd(nil, nil, nil)
		root.SetOut(&failOnWriteNumber{failAt: failAt})
		root.SetArgs([]string{"agent", "install", "--agent", "codex", "--dry-run"})
		assert.ErrorContains(t, root.Execute(), "writer failed")
	}

	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(blocked, []byte("x"), 0o644))
	t.Setenv("CODEX_HOME", filepath.Join(blocked, "codex"))
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "install", "--agent", "codex"})
	assert.ErrorContains(t, root.Execute(), "install skill for Codex")

	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	require.NoError(t, os.WriteFile(filepath.Join(config, "gha"), []byte("x"), 0o644))
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	root = NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "install", "--agent", "codex"})
	assert.ErrorContains(t, root.Execute(), "record Codex installation ownership")
}

func TestAgentUninstallReportsManifestAndSharedDestinationErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", filepath.Join(home, "codex"))
	t.Setenv("HOME", "")
	err := runAgentUninstall(NewRootCmd(nil, nil, nil), "codex", true)
	assert.ErrorContains(t, err, "user config directory")

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	target := agentInstallation{Name: "Pi", InstructionsPath: filepath.Join(home, "shared.md"), SkillPath: filepath.Join(home, "skills", "SKILL.md")}
	remaining := agentOwnership{Agents: []agentInstallation{{InstructionsPath: target.InstructionsPath, SkillPath: target.SkillPath}}}
	writer := &bytes.Buffer{}
	require.NoError(t, uninstallAgentTarget(writer, target, false, remaining))
	assert.Contains(t, writer.String(), "Retained shared managed GHA guidance and skill")
	assert.Error(t, uninstallAgentTarget(&failOnWriteNumber{failAt: 1}, target, true, agentOwnership{}))
}

func agentOwnershipReadError() error {
	_, err := readAgentOwnership()
	return err
}

func TestAgentInstallAutomaticallyConfiguresDetectedClaudeCode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "claude"))
	claudeHome := os.Getenv("CLAUDE_CONFIG_DIR")
	require.NoError(t, os.MkdirAll(claudeHome, 0o755))

	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "install"})
	require.NoError(t, root.Execute())

	skill, err := os.ReadFile(filepath.Join(claudeHome, "skills", "gha", "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, ghaskill.Skill, skill)
	assert.Contains(t, output.String(), "Claude Code")
}

func TestAgentInstallReportsNoDetectedHarnessAndSupportsBinaryOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "install"})
	require.NoError(t, root.Execute())
	assert.Contains(t, output.String(), "No supported coding-agent harnesses were detected")
	ownership, err := readAgentOwnership()
	require.NoError(t, err)
	assert.Empty(t, ownership.Agents)

	output.Reset()
	root = NewRootCmd(nil, nil, nil)
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "install", "--binary-only"})
	require.NoError(t, root.Execute())
	assert.Contains(t, output.String(), "harness configuration skipped")

	root = NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "install", "--binary-only", "--agent", "codex"})
	err = root.Execute()
	assert.ErrorContains(t, err, "cannot be combined")
}

func TestAgentInstallDetectsConfiguredHarnessWithoutExecutable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini"), 0o755))
	targets, err := detectedOrSelectedAgentInstallations("")
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "gemini", targets[0].ID)
}

func TestInstallAgentGuidanceOnceReportsSkillAndGuidanceFailures(t *testing.T) {
	root := t.TempDir()
	blockedParent := filepath.Join(root, "blocked")
	require.NoError(t, os.WriteFile(blockedParent, []byte("file"), 0o644))
	target := agentInstallation{Name: "Pi", SkillPath: filepath.Join(blockedParent, "SKILL.md")}
	err := installAgentGuidanceOnce(target, map[string]bool{})
	assert.ErrorContains(t, err, "install skill for Pi")

	target = agentInstallation{Name: "Claude Code", SkillPath: filepath.Join(root, "skill", "SKILL.md"), InstructionsPath: filepath.Join(root, "guidance")}
	require.NoError(t, os.Mkdir(target.InstructionsPath, 0o755))
	err = installAgentGuidanceOnce(target, map[string]bool{})
	assert.ErrorContains(t, err, "read Claude Code guidance")

	malformed := filepath.Join(root, "malformed.md")
	require.NoError(t, os.WriteFile(malformed, []byte("<!-- gha:begin -->"), 0o644))
	target = agentInstallation{Name: "Codex", SkillPath: filepath.Join(root, "other", "SKILL.md"), InstructionsPath: malformed}
	err = installAgentGuidanceOnce(target, map[string]bool{})
	assert.ErrorContains(t, err, "update Codex guidance")
}

func TestAgentDetectionReportsUnreadableConfigurationPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	loop := filepath.Join(home, "loop")
	require.NoError(t, os.Symlink(loop, loop))
	t.Setenv("CODEX_HOME", loop)
	_, err := detectedOrSelectedAgentInstallations("")
	assert.ErrorContains(t, err, "detect Codex configuration")
}

func TestAgentInstallDryRunDoesNotWrite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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

func TestAgentInstallRunsByDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "install", "--agent", "codex"})
	require.NoError(t, root.Execute())
}

func TestAgentUninstallRemovesManagedGuidanceAndSkill(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex"})
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
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)
	skillPath := filepath.Join(codexHome, "skills", "gha", "SKILL.md")
	additionalPath := filepath.Join(codexHome, "skills", "gha", "notes.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(skillPath), 0o755))
	require.NoError(t, os.WriteFile(skillPath, []byte("installed skill"), 0o644))
	require.NoError(t, os.WriteFile(additionalPath, []byte("personal note"), 0o644))

	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(skillPath)
	assert.True(t, os.IsNotExist(err))
	note, err := os.ReadFile(additionalPath)
	require.NoError(t, err)
	assert.Equal(t, "personal note", string(note))
}

func TestAgentUninstallRemovesGuidanceFileWhenItContainsOnlyManagedGuidance(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)
	guidancePath := filepath.Join(codexHome, "AGENTS.md")
	require.NoError(t, os.MkdirAll(codexHome, 0o755))
	require.NoError(t, os.WriteFile(guidancePath, ghaskill.Guidance, 0o644))

	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(guidancePath)
	assert.True(t, os.IsNotExist(err))
}

func TestAgentUninstallDryRunAndMissingGuidanceDoNotWrite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex"})
	require.NoError(t, root.Execute())
	_, err = os.Stat(codexHome)
	assert.True(t, os.IsNotExist(err))
}

func TestAgentUninstallPromptsForTarget(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "claude"))

	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetIn(strings.NewReader("claude\n"))
	root.SetOut(&output)
	root.SetArgs([]string{"agent", "uninstall", "--dry-run"})
	require.NoError(t, root.Execute())

	assert.Contains(t, output.String(), "remove GHA guidance and skill")
	assert.Contains(t, output.String(), "Claude Code")
}

func TestAgentUninstallRunsByDefaultAndRejectsMalformedManagedGuidance(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)
	require.NoError(t, os.MkdirAll(codexHome, 0o755))
	guidancePath := filepath.Join(codexHome, "AGENTS.md")
	require.NoError(t, os.WriteFile(guidancePath, []byte("# Personal\n<!-- gha:begin -->\n"), 0o644))

	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without <!-- gha:end -->")

	root = NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"agent", "uninstall", "--agent", "codex"})
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

func TestAgentSelectionAndManagedGuidanceHelpers(t *testing.T) {
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "claude"))
	for _, agent := range []string{"codex", "claude", "codex,claude", "pi"} {
		targets, err := selectedAgentInstallations(agent, "configure", strings.NewReader(""), &bytes.Buffer{})
		require.NoError(t, err, agent)
		assert.NotEmpty(t, targets)
	}
	_, err := selectedAgentInstallations("unknown", "configure", strings.NewReader(""), &bytes.Buffer{})
	assert.ErrorContains(t, err, "invalid agent")

	updated, err := withManagedGuidance([]byte("# Personal\n\n" + string(ghaskill.Guidance) + "\n# After\n"))
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(updated), "<!-- gha:begin -->"))
	without, removed, err := withoutManagedGuidance(updated)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Contains(t, string(without), "# Personal")
	unchanged, removed, err := withoutManagedGuidance([]byte("# Personal\n"))
	require.NoError(t, err)
	assert.False(t, removed)
	assert.Equal(t, "# Personal\n", string(unchanged))
	_, err = withManagedGuidance([]byte("<!-- gha:begin -->"))
	assert.ErrorContains(t, err, "without <!-- gha:end -->")
}

func TestWriteFileAtomicallyPreservesExistingModeAndRemovesSkillDirectories(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "file")
	require.NoError(t, writeFileAtomically(path, []byte("first")))
	require.NoError(t, os.Chmod(path, 0o600))
	require.NoError(t, writeFileAtomically(path, []byte("second")))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "second", string(content))
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	skill := filepath.Join(root, "skills", "gha", "SKILL.md")
	require.NoError(t, writeFileAtomically(skill, []byte("skill")))
	removed, err := removeAgentSkill(skill)
	require.NoError(t, err)
	assert.True(t, removed)
	removed, err = removeAgentSkill(skill)
	require.NoError(t, err)
	assert.False(t, removed)
}
