// Package commands implements the gha command tree.
package commands

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	SkillDigest      string `json:"skill_digest,omitempty"`
	InstructionsPath string `json:"instructions_path"`
	skillOwned       bool   `json:"-"`
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
	skillModified    bool
	skillUnowned     bool
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
	var binaryOnly bool
	command := &cobra.Command{
		Use:   "install",
		Short: "Install the gha skill for supported coding agents",
		Long: `Copy the bundled gha skill and managed GHA guidance for selected supported coding agents.

Without --agent, detected supported harnesses are configured automatically.
Use --binary-only to skip harness setup, or --agent to select harnesses
explicitly. Use --dry-run to inspect the destination paths.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runAgentInstall(cmd, agent, dryRun, binaryOnly) },
	}
	command.Flags().StringVar(&agent, "agent", "", "Comma-separated agents (codex, claude, pi, opencode, copilot, gemini, cursor, hermes, openclaw, all); detects configured harnesses when omitted")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the files that would be written")
	command.Flags().BoolVar(&binaryOnly, "binary-only", false, "Skip coding-agent harness setup")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

func runAgentInstall(cmd *cobra.Command, agent string, dryRun, binaryOnly bool) error {
	if binaryOnly && agent != "" {
		return fmt.Errorf("--binary-only cannot be combined with --agent")
	}
	if binaryOnly {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "GHA binary setup selected; harness configuration skipped.")
		return err
	}
	targets, err := detectedOrSelectedAgentInstallations(agent)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No supported coding-agent harnesses were detected. GHA is ready to use; install setup later with `gha agent install --agent <name>` or select `--binary-only`.")
		return err
	}
	return installAgentTargets(cmd, targets, dryRun)
}

func installAgentTargets(cmd *cobra.Command, targets []agentInstallation, dryRun bool) error {
	ownership, err := readAgentOwnership()
	if err != nil {
		return err
	}
	plannedSkills := map[string]bool{}
	for _, target := range targets {
		if plannedSkills[target.SkillPath] {
			continue
		}
		if err := validateSkillInstallTarget(target, ownership); err != nil {
			return fmt.Errorf("install skill for %s: %w", target.Name, err)
		}
		plannedSkills[target.SkillPath] = true
	}
	installedSkills := map[string]bool{}
	for _, target := range targets {
		target.SkillDigest = skillDigest(ghaskill.Skill)
		if dryRun {
			if err := printAgentInstallPlan(cmd.OutOrStdout(), target); err != nil {
				return err
			}
			continue
		}
		if err := installAgentGuidanceOnce(target, installedSkills); err != nil {
			return err
		}
		if err := recordAgentInstallation(target); err != nil {
			return fmt.Errorf("record %s installation ownership: %w", target.Name, err)
		}
		if err := printAgentInstalled(cmd.OutOrStdout(), target); err != nil {
			return err
		}
	}
	return nil
}

func printAgentInstallPlan(output io.Writer, target agentInstallation) error {
	if _, err := fmt.Fprintf(output, "Would install gha skill for %s at %s", target.Name, target.SkillPath); err != nil {
		return err
	}
	if target.InstructionsPath != "" {
		if _, err := fmt.Fprintf(output, " and update %s", target.InstructionsPath); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(output)
	return err
}

func printAgentInstalled(output io.Writer, target agentInstallation) error {
	message := fmt.Sprintf("Installed gha skill for %s.\n", target.Name)
	if target.InstructionsPath != "" {
		message = fmt.Sprintf("Installed gha skill and guidance for %s.\n", target.Name)
	}
	_, err := fmt.Fprint(output, message)
	return err
}

func installAgentGuidanceOnce(target agentInstallation, installedSkills map[string]bool) error {
	if !installedSkills[target.SkillPath] {
		if err := writeFileAtomically(target.SkillPath, ghaskill.Skill); err != nil {
			return fmt.Errorf("install skill for %s: %w", target.Name, err)
		}
		installedSkills[target.SkillPath] = true
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

func detectedOrSelectedAgentInstallations(agent string) ([]agentInstallation, error) {
	if agent != "" {
		return selectedAgentInstallations(agent, "install the gha skill for", strings.NewReader(""), io.Discard)
	}
	var targets []agentInstallation
	for _, id := range supportedAgentIDs() {
		target, err := agentInstallationFor(id)
		if err != nil {
			return nil, err
		}
		root := filepath.Dir(target.InstructionsPath)
		if target.InstructionsPath == "" {
			root = filepath.Dir(filepath.Dir(filepath.Dir(target.SkillPath)))
		}
		if target.ID == "copilot" {
			root, err = agentConfigRoot("COPILOT_HOME", ".copilot")
			if err != nil {
				return nil, err
			}
		}
		if target.ID == "cursor" {
			root = filepath.Dir(filepath.Dir(filepath.Dir(target.SkillPath)))
		}
		info, err := os.Stat(root)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("detect %s configuration: %w", target.Name, err)
		}
		if err == nil && info.IsDir() {
			targets = append(targets, target)
		}
	}
	return targets, nil
}

func newAgentUninstallCmd() *cobra.Command {
	var agent string
	var dryRun bool
	command := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove GHA guidance and skill for selected agents",
		Long: `Remove the managed GHA guidance section and bundled gha skill for selected supported agents.

Every instruction outside the marked GHA section, other files in the skill
directory, and skills modified after installation are preserved. Without --agent, choose an agent interactively. Use
--dry-run to inspect the destination paths. Removal runs unless --dry-run is specified.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAgentUninstall(cmd, agent, dryRun)
		},
	}
	command.Flags().StringVar(&agent, "agent", "", "Comma-separated agents (codex, claude, pi, opencode, copilot, gemini, cursor, hermes, openclaw, all); prompts when omitted")
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
		recordedSkill := false
		for _, recorded := range ownership.Agents {
			if recorded.ID == target.ID {
				target = recorded
				recordedSkill = true
				break
			}
		}
		target.skillOwned = recordedSkill
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
		return printAgentUninstallPlan(output, target, remaining)
	}

	result, err := uninstallAgentChanges(target, remaining)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(output, agentUninstallResultMessage(target.Name, result))
	return err
}

func uninstallAgentChanges(target agentInstallation, remaining agentOwnership) (agentUninstallResult, error) {
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
		} else if !target.skillOwned {
			result.skillRetained = true
			result.skillUnowned = true
		} else {
			result.skillRemoved, result.skillModified, err = removeAgentSkillIfUnchanged(target.SkillPath, target.SkillDigest)
			if err != nil {
				err = fmt.Errorf("remove skill for %s: %w", target.Name, err)
			}
		}
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func printAgentUninstallPlan(output io.Writer, target agentInstallation, remaining agentOwnership) error {
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
	} else if !target.skillOwned {
		message += " (skill is not recorded as GHA-managed and will be preserved)"
	} else if target.SkillPath != "" {
		if data, err := os.ReadFile(target.SkillPath); err == nil && skillChanged(data, target.SkillDigest) {
			message += " (modified skill will be preserved)"
		}
	}
	_, err := fmt.Fprintln(output, message)
	return err
}

func agentUninstallResultMessage(name string, result agentUninstallResult) string {
	if result.skillUnowned {
		return unownedSkillRemovalMessage(name, result.guidanceRemoved)
	}
	if result.skillModified {
		return modifiedSkillRemovalMessage(name, result.guidanceRemoved)
	}
	return ordinaryAgentUninstallMessage(name, result)
}

func unownedSkillRemovalMessage(name string, guidanceRemoved bool) string {
	if guidanceRemoved {
		return fmt.Sprintf("Removed managed GHA guidance for %s; preserved the unowned gha skill.\n", name)
	}
	return fmt.Sprintf("Preserved the unowned gha skill for %s; no managed guidance was found.\n", name)
}

func modifiedSkillRemovalMessage(name string, guidanceRemoved bool) string {
	if guidanceRemoved {
		return fmt.Sprintf("Removed managed GHA guidance for %s; preserved the modified gha skill.\n", name)
	}
	return fmt.Sprintf("Preserved the modified gha skill for %s; no managed guidance was found.\n", name)
}

func ordinaryAgentUninstallMessage(name string, result agentUninstallResult) string {
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
		if _, err := fmt.Fprintf(output, "Which agent(s) should %s?\n  codex) Codex\n  claude) Claude Code\n  pi) Pi\n  opencode) OpenCode\n  copilot) GitHub Copilot\n  gemini) Gemini CLI\n  cursor) Cursor\n  hermes) Hermes Agent\n  openclaw) OpenClaw\n  all) All supported harnesses\nEnter one or more names, separated by commas: ", action); err != nil {
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
		return supportedAgentIDs(), nil
	}
	ids := make([]string, 0, len(selected))
	for _, id := range supportedAgentIDs() {
		if selected[id] {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func normalizeAgentID(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "codex", "claude", "pi", "opencode", "copilot", "gemini", "cursor", "hermes", "openclaw", "all":
		return value, nil
	default:
		return "", fmt.Errorf("invalid agent %q; use codex, claude, pi, opencode, copilot, gemini, cursor, hermes, openclaw, or all", value)
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
	case "cursor":
		return cursorInstallation()
	case "hermes":
		return hermesInstallation()
	case "openclaw":
		return openClawInstallation()
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

func cursorInstallation() (agentInstallation, error) {
	root, err := agentConfigRoot("", ".cursor")
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{ID: "cursor", Name: "Cursor", SkillPath: filepath.Join(root, "skills", "gha", "SKILL.md")}, nil
}

func hermesInstallation() (agentInstallation, error) {
	root, err := agentConfigRoot("HERMES_HOME", ".hermes")
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{ID: "hermes", Name: "Hermes Agent", SkillPath: filepath.Join(root, "skills", "gha", "SKILL.md")}, nil
}

func openClawInstallation() (agentInstallation, error) {
	root, err := agentConfigRoot("OPENCLAW_STATE_DIR", ".openclaw")
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{ID: "openclaw", Name: "OpenClaw", SkillPath: filepath.Join(root, "skills", "gha", "SKILL.md")}, nil
}

func supportedAgentIDs() []string {
	return []string{"codex", "claude", "pi", "opencode", "copilot", "gemini", "cursor", "hermes", "openclaw"}
}

func skillDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func validateSkillInstallTarget(target agentInstallation, ownership agentOwnership) error {
	current, err := os.ReadFile(target.SkillPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read existing %s skill: %w", target.Name, err)
	}
	expected, managed := ownership.skillDigest(target.SkillPath)
	if !managed {
		return fmt.Errorf("%s skill already exists at %s and is not managed by GHA; move it or remove it before installing", target.Name, target.SkillPath)
	}
	if expected == "" {
		if bytes.Equal(current, ghaskill.Skill) {
			return nil
		}
		return fmt.Errorf("managed %s skill at %s has changed; preserve it or restore the GHA version before reinstalling", target.Name, target.SkillPath)
	}
	if skillDigest(current) != expected {
		return fmt.Errorf("managed %s skill at %s has changed; preserve it or restore the GHA version before reinstalling", target.Name, target.SkillPath)
	}
	return nil
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
	if target.SkillDigest == "" {
		if data, readErr := os.ReadFile(target.SkillPath); readErr == nil {
			target.SkillDigest = skillDigest(data)
		} else if !os.IsNotExist(readErr) {
			return fmt.Errorf("read installed skill for ownership: %w", readErr)
		}
	}
	ownership.Version = 1
	for i := range ownership.Agents {
		if ownership.Agents[i].SkillPath == target.SkillPath && target.SkillDigest != "" {
			ownership.Agents[i].SkillDigest = target.SkillDigest
		}
	}
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

func (ownership agentOwnership) skillDigest(path string) (string, bool) {
	for _, agent := range ownership.Agents {
		if agent.SkillPath == path {
			return agent.SkillDigest, true
		}
	}
	return "", false
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

func removeAgentSkillIfUnchanged(skillPath, expectedDigest string) (bool, bool, error) {
	content, err := os.ReadFile(skillPath)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if skillChanged(content, expectedDigest) {
		return false, true, nil
	}
	if err := os.Remove(skillPath); err != nil {
		return false, false, err
	}
	removeEmptyDirectory(filepath.Dir(skillPath))
	removeEmptyDirectory(filepath.Dir(filepath.Dir(skillPath)))
	return true, false, nil
}

func skillChanged(content []byte, expectedDigest string) bool {
	if expectedDigest == "" {
		return !bytes.Equal(content, ghaskill.Skill)
	}
	return skillDigest(content) != expectedDigest
}

func removeEmptyDirectory(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return
	}
}

func withManagedGuidance(existing []byte) ([]byte, error) {
	return withManagedGuidanceContent(existing, ghaskill.Guidance)
}

func withManagedGuidanceContent(existing, managed []byte) ([]byte, error) {
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
		return append([]byte(nil), managed...), nil
	}
	return []byte(content + "\n\n" + string(managed)), nil
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
