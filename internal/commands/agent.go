// Package commands implements the gha command tree.
package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	ghaskill "github.com/raithlin/gha/skills/gha"
)

type agentInstallation struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	SkillPath        string `json:"skill_path"`
	InstructionsPath string `json:"instructions_path"`
}

type agentOwnership struct {
	Version int                 `json:"version"`
	Agents  []agentInstallation `json:"agents"`
}

type agentUninstallResult struct {
	guidanceRemoved  bool
	skillRemoved     bool
	guidanceRetained bool
	skillRetained    bool
}

func newAgentCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "agent",
		Short: "Manage GHA guidance for coding agents",
	}
	command.AddCommand(newAgentInstallCmd(), newAgentUninstallCmd(), newAgentListCmd())
	return command
}

func newAgentInstallCmd() *cobra.Command {
	var agent string
	var dryRun bool
	command := &cobra.Command{
		Use:   "install",
		Short: "Install the gha skill for supported coding agents",
		Long: `Copy the bundled gha skill and managed GHA guidance for selected supported coding agents.

Without --agent, choose an agent interactively. Use --dry-run to inspect the
destination paths. The installation runs unless --dry-run is specified.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			targets, err := selectedAgentInstallations(agent, "install the gha skill for", cmd.InOrStdin(), cmd.OutOrStdout())
			if err != nil {
				return err
			}

			for _, target := range targets {
				if dryRun {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Would install gha skill for %s at %s", target.Name, target.SkillPath); err != nil {
						return err
					}
					if target.InstructionsPath != "" {
						if _, err := fmt.Fprintf(cmd.OutOrStdout(), " and update %s", target.InstructionsPath); err != nil {
							return err
						}
					}
					if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
						return err
					}
					continue
				}
				if err := installAgentGuidance(target); err != nil {
					return err
				}
				if err := recordAgentInstallation(target); err != nil {
					return fmt.Errorf("record %s installation ownership: %w", target.Name, err)
				}
				message := fmt.Sprintf("Installed gha skill for %s.\n", target.Name)
				if target.InstructionsPath != "" {
					message = fmt.Sprintf("Installed gha skill and guidance for %s.\n", target.Name)
				}
				if _, err := fmt.Fprint(cmd.OutOrStdout(), message); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().StringVar(&agent, "agent", "", "Comma-separated agents (codex, claude, pi, opencode, copilot, gemini, all); prompts when omitted")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the files that would be written")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

func newAgentUninstallCmd() *cobra.Command {
	var agent string
	var dryRun bool
	command := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove GHA guidance and skill for selected agents",
		Long: `Remove the managed GHA guidance section and bundled gha skill for selected supported agents.

Every instruction outside the marked GHA section and other files in the skill
directory are preserved. Without --agent, choose an agent interactively. Use
--dry-run to inspect the destination paths. Removal runs unless --dry-run is specified.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAgentUninstall(cmd, agent, dryRun)
		},
	}
	command.Flags().StringVar(&agent, "agent", "", "Comma-separated agents (codex, claude, pi, opencode, copilot, gemini, all); prompts when omitted")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the GHA files that would be removed")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

func runAgentUninstall(cmd *cobra.Command, agent string, dryRun bool) error {
	targets, err := selectedAgentInstallations(agent, "remove GHA guidance and skill for", cmd.InOrStdin(), cmd.OutOrStdout())
	if err != nil {
		return err
	}

	ownership, err := readAgentOwnership()
	if err != nil {
		return err
	}
	selectedIDs := make([]string, 0, len(targets))
	for _, target := range targets {
		selectedIDs = append(selectedIDs, target.ID)
	}
	for _, target := range targets {
		for _, recorded := range ownership.Agents {
			if recorded.ID == target.ID {
				target = recorded
				break
			}
		}
		remaining := ownership.withoutAgents(selectedIDs)
		if !dryRun {
			remaining = ownership.without(target.ID)
		}
		if err := uninstallAgentTarget(cmd.OutOrStdout(), target, dryRun, remaining); err != nil {
			return err
		}
		if !dryRun {
			ownership = remaining
		}
	}
	if !dryRun {
		return writeAgentOwnership(ownership)
	}
	return nil
}

func uninstallAgentTarget(output io.Writer, target agentInstallation, dryRun bool, remaining agentOwnership) error {
	if dryRun {
		message := "Would remove managed GHA guidance for " + target.Name
		if target.InstructionsPath == "" {
			message = "Would remove gha skill for " + target.Name
		}
		if target.InstructionsPath != "" {
			message += " from " + target.InstructionsPath
		}
		if target.SkillPath != "" {
			message += " and gha skill at " + target.SkillPath
		}
		if remaining.owns(target.InstructionsPath) || remaining.owns(target.SkillPath) {
			message += " (shared destination retained)"
		}
		_, err := fmt.Fprintln(output, message)
		return err
	}

	result := agentUninstallResult{}
	var err error
	if remaining.owns(target.InstructionsPath) {
		result.guidanceRetained = true
	} else {
		result.guidanceRemoved, err = removeManagedGuidance(target)
	}
	if err == nil {
		if remaining.owns(target.SkillPath) {
			result.skillRetained = true
		} else {
			result.skillRemoved, err = removeAgentSkill(target.SkillPath)
			if err != nil {
				err = fmt.Errorf("remove skill for %s: %w", target.Name, err)
			}
		}
	}
	if err != nil {
		return err
	}
	message := agentUninstallResultMessage(target.Name, result)
	_, err = fmt.Fprint(output, message)
	return err
}

func agentUninstallResultMessage(name string, result agentUninstallResult) string {
	switch {
	case result.guidanceRemoved && result.skillRetained:
		return fmt.Sprintf("Removed managed GHA guidance for %s; shared gha skill remains in use.\n", name)
	case result.guidanceRetained && result.skillRemoved:
		return fmt.Sprintf("Removed gha skill for %s; shared GHA guidance remains in use.\n", name)
	case result.guidanceRetained && result.skillRetained:
		return fmt.Sprintf("Retained shared managed GHA guidance and skill for %s.\n", name)
	case result.skillRetained:
		return fmt.Sprintf("Retained shared gha skill for %s; no managed GHA guidance was found.\n", name)
	case result.guidanceRetained:
		return fmt.Sprintf("Retained shared managed GHA guidance for %s; no gha skill was found.\n", name)
	case result.guidanceRemoved && result.skillRemoved:
		return fmt.Sprintf("Removed managed GHA guidance and skill for %s.\n", name)
	case result.guidanceRemoved:
		return fmt.Sprintf("Removed managed GHA guidance for %s; no gha skill was found.\n", name)
	case result.skillRemoved:
		return fmt.Sprintf("Removed gha skill for %s; no managed GHA guidance was found.\n", name)
	default:
		return fmt.Sprintf("No managed GHA guidance or skill found for %s.\n", name)
	}
}

func selectedAgentInstallations(agent, action string, input io.Reader, output io.Writer) ([]agentInstallation, error) {
	if agent == "" {
		if _, err := fmt.Fprintf(output, "Which agent(s) should %s?\n  codex) Codex\n  claude) Claude Code\n  pi) Pi\n  opencode) OpenCode\n  copilot) GitHub Copilot\n  gemini) Gemini CLI\n  all) All supported harnesses\nEnter one or more names, separated by commas: ", action); err != nil {
			return nil, err
		}
		line, err := bufio.NewReader(input).ReadString('\n')
		if err != nil && (err != io.EOF || strings.TrimSpace(line) == "") {
			return nil, fmt.Errorf("read agent selection: %w", err)
		}
		agent = strings.TrimSpace(line)
	}

	ids, err := selectedAgentIDs(agent)
	if err != nil {
		return nil, err
	}
	var targets []agentInstallation
	for _, id := range ids {
		target, err := agentInstallationFor(id)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("select at least one agent")
	}
	return targets, nil
}

func selectedAgentIDs(value string) ([]string, error) {
	selected := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		id, err := normalizeAgentID(item)
		if err != nil {
			return nil, err
		}
		switch id {
		case "all":
			selected["all"] = true
		default:
			selected[id] = true
		}
	}
	if selected["all"] {
		return []string{"codex", "claude", "pi", "opencode", "copilot", "gemini"}, nil
	}
	ids := make([]string, 0, len(selected))
	for _, id := range []string{"codex", "claude", "pi", "opencode", "copilot", "gemini"} {
		if selected[id] {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func normalizeAgentID(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "codex", "claude", "pi", "opencode", "copilot", "gemini", "all":
		return value, nil
	default:
		return "", fmt.Errorf("invalid agent %q; use codex, claude, pi, opencode, copilot, gemini, or all", value)
	}
}

func codexInstallation() (agentInstallation, error) {
	root, err := agentConfigRoot("CODEX_HOME", ".codex")
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{
		ID:               "codex",
		Name:             "Codex",
		SkillPath:        filepath.Join(root, "skills", "gha", "SKILL.md"),
		InstructionsPath: filepath.Join(root, "AGENTS.md"),
	}, nil
}

func claudeInstallation() (agentInstallation, error) {
	root, err := agentConfigRoot("CLAUDE_CONFIG_DIR", ".claude")
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{
		ID:               "claude",
		Name:             "Claude Code",
		SkillPath:        filepath.Join(root, "skills", "gha", "SKILL.md"),
		InstructionsPath: filepath.Join(root, "CLAUDE.md"),
	}, nil
}

func agentInstallationFor(id string) (agentInstallation, error) {
	switch id {
	case "codex":
		return codexInstallation()
	case "claude":
		return claudeInstallation()
	case "pi":
		return piInstallation()
	case "opencode":
		return openCodeInstallation()
	case "copilot":
		return copilotInstallation()
	case "gemini":
		return geminiInstallation()
	default:
		return agentInstallation{}, fmt.Errorf("unsupported agent %q", id)
	}
}

func sharedAgentSkillPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory for shared agent skills: %w", err)
	}
	return filepath.Join(home, ".agents", "skills", "gha", "SKILL.md"), nil
}

func piInstallation() (agentInstallation, error) {
	root, err := agentConfigRoot("PI_CODING_AGENT_DIR", filepath.Join(".pi", "agent"))
	if err != nil {
		return agentInstallation{}, err
	}
	skill, err := sharedAgentSkillPath()
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{ID: "pi", Name: "Pi", SkillPath: skill, InstructionsPath: filepath.Join(root, "AGENTS.md")}, nil
}

func openCodeInstallation() (agentInstallation, error) {
	root := os.Getenv("OPENCODE_CONFIG_DIR")
	if root == "" {
		var err error
		root, err = os.UserConfigDir()
		if err != nil {
			return agentInstallation{}, fmt.Errorf("find OpenCode config directory: %w", err)
		}
		root = filepath.Join(root, "opencode")
	}
	skill, err := sharedAgentSkillPath()
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{ID: "opencode", Name: "OpenCode", SkillPath: skill, InstructionsPath: filepath.Join(root, "AGENTS.md")}, nil
}

func copilotInstallation() (agentInstallation, error) {
	skill, err := sharedAgentSkillPath()
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{ID: "copilot", Name: "GitHub Copilot", SkillPath: skill}, nil
}

func geminiInstallation() (agentInstallation, error) {
	root, err := agentConfigRoot("", ".gemini")
	if err != nil {
		return agentInstallation{}, err
	}
	skill, err := sharedAgentSkillPath()
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{ID: "gemini", Name: "Gemini CLI", SkillPath: skill, InstructionsPath: filepath.Join(root, "GEMINI.md")}, nil
}

func agentConfigRoot(environment, defaultDirectory string) (string, error) {
	if configured := os.Getenv(environment); configured != "" {
		return configured, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory for %s: %w", environment, err)
	}
	return filepath.Join(home, defaultDirectory), nil
}

func installAgentGuidance(target agentInstallation) error {
	if err := writeFileAtomically(target.SkillPath, ghaskill.Skill); err != nil {
		return fmt.Errorf("install skill for %s: %w", target.Name, err)
	}
	if target.InstructionsPath == "" {
		return nil
	}

	existing, err := os.ReadFile(target.InstructionsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s guidance: %w", target.Name, err)
	}
	guidance, err := withManagedGuidance(existing)
	if err != nil {
		return fmt.Errorf("update %s guidance: %w", target.Name, err)
	}
	if err := writeFileAtomically(target.InstructionsPath, guidance); err != nil {
		return fmt.Errorf("write %s guidance: %w", target.Name, err)
	}
	return nil
}

func removeManagedGuidance(target agentInstallation) (bool, error) {
	if target.InstructionsPath == "" {
		return false, nil
	}
	existing, err := os.ReadFile(target.InstructionsPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s guidance: %w", target.Name, err)
	}
	guidance, removed, err := withoutManagedGuidance(existing)
	if err != nil {
		return false, fmt.Errorf("update %s guidance: %w", target.Name, err)
	}
	if !removed {
		return false, nil
	}
	if bytes.Equal(existing, ghaskill.Guidance) {
		err = os.Remove(target.InstructionsPath)
	} else {
		err = writeFileAtomically(target.InstructionsPath, guidance)
	}
	if err != nil {
		return false, fmt.Errorf("write %s guidance: %w", target.Name, err)
	}
	return true, nil
}

func agentOwnershipPath() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(config, "gha", "agent-installations.json"), nil
}

func readAgentOwnership() (agentOwnership, error) {
	path, err := agentOwnershipPath()
	if err != nil {
		return agentOwnership{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return agentOwnership{Version: 1}, nil
	}
	if err != nil {
		return agentOwnership{}, fmt.Errorf("read agent ownership: %w", err)
	}
	var ownership agentOwnership
	if err := json.Unmarshal(data, &ownership); err != nil {
		return ownership, fmt.Errorf("decode agent ownership at %s: %w", path, err)
	}
	if ownership.Version != 1 {
		return ownership, fmt.Errorf("unsupported agent ownership version %d", ownership.Version)
	}
	return ownership, nil
}

func writeAgentOwnership(ownership agentOwnership) error {
	path, err := agentOwnershipPath()
	if err != nil {
		return err
	}
	if len(ownership.Agents) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove agent ownership: %w", err)
		}
		return nil
	}
	data, err := json.MarshalIndent(ownership, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFileAtomically(path, append(data, '\n')); err != nil {
		return fmt.Errorf("write agent ownership: %w", err)
	}
	return nil
}

func recordAgentInstallation(target agentInstallation) error {
	ownership, err := readAgentOwnership()
	if err != nil {
		return err
	}
	ownership.Version = 1
	for i, current := range ownership.Agents {
		if current.ID == target.ID {
			ownership.Agents[i] = target
			return writeAgentOwnership(ownership)
		}
	}
	ownership.Agents = append(ownership.Agents, target)
	return writeAgentOwnership(ownership)
}

func (ownership agentOwnership) without(id string) agentOwnership {
	remaining := make([]agentInstallation, 0, len(ownership.Agents))
	for _, agent := range ownership.Agents {
		if agent.ID != id {
			remaining = append(remaining, agent)
		}
	}
	ownership.Agents = remaining
	return ownership
}

func (ownership agentOwnership) withoutAgents(ids []string) agentOwnership {
	for _, id := range ids {
		ownership = ownership.without(id)
	}
	return ownership
}

func (ownership agentOwnership) owns(path string) bool {
	if path == "" {
		return false
	}
	for _, agent := range ownership.Agents {
		if agent.SkillPath == path || agent.InstructionsPath == path {
			return true
		}
	}
	return false
}

func removeAgentSkill(skillPath string) (bool, error) {
	if err := os.Remove(skillPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	removeEmptyDirectory(filepath.Dir(skillPath))
	removeEmptyDirectory(filepath.Dir(filepath.Dir(skillPath)))
	return true, nil
}

func removeEmptyDirectory(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return
	}
}

func withManagedGuidance(existing []byte) ([]byte, error) {
	const start = "<!-- gha:begin -->"
	const end = "<!-- gha:end -->"
	content := string(existing)
	if marker := strings.Index(content, start); marker >= 0 {
		endOffset := strings.Index(content[marker:], end)
		if endOffset < 0 {
			return nil, fmt.Errorf("found %s without %s", start, end)
		}
		endOffset += marker + len(end)
		content = content[:marker] + content[endOffset:]
	}

	content = strings.TrimRight(content, "\n")
	if content == "" {
		return append([]byte(nil), ghaskill.Guidance...), nil
	}
	return []byte(content + "\n\n" + string(ghaskill.Guidance)), nil
}

func withoutManagedGuidance(existing []byte) ([]byte, bool, error) {
	const start = "<!-- gha:begin -->"
	const end = "<!-- gha:end -->"
	content := string(existing)
	marker := strings.Index(content, start)
	if marker < 0 {
		return existing, false, nil
	}
	endOffset := strings.Index(content[marker:], end)
	if endOffset < 0 {
		return nil, false, fmt.Errorf("found %s without %s", start, end)
	}
	endOffset += marker + len(end)
	return []byte(content[:marker] + content[endOffset:]), true, nil
}

func writeFileAtomically(path string, content []byte) (returnErr error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(path), ".gha-install-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		if err := os.Remove(temporaryPath); err != nil && !os.IsNotExist(err) && returnErr == nil {
			returnErr = fmt.Errorf("remove temporary file: %w", err)
		}
	}()
	if err := temporary.Chmod(mode); err != nil {
		if closeErr := temporary.Close(); closeErr != nil {
			return fmt.Errorf("%w (close temporary file: %v)", err, closeErr)
		}
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		if closeErr := temporary.Close(); closeErr != nil {
			return fmt.Errorf("%w (close temporary file: %v)", err, closeErr)
		}
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
