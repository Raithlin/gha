// Package commands implements the gha command tree.
package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	ghaskill "github.com/raithlin/gha/skills/gha"
)

type agentInstallation struct {
	name             string
	skillPath        string
	instructionsPath string
}

func newAgentCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "agent",
		Short: "Install GHA guidance for coding agents",
	}
	command.AddCommand(newAgentInstallCmd())
	return command
}

func newAgentInstallCmd() *cobra.Command {
	var agent string
	var confirm bool
	var dryRun bool
	command := &cobra.Command{
		Use:   "install",
		Short: "Install the gha skill for Codex or Claude Code",
		Long: `Copy the bundled gha skill and managed GHA guidance for Codex, Claude Code, or both.

Without --agent, choose an agent interactively. Use --dry-run to inspect the
destination paths. Writing requires --confirm.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			targets, err := selectedAgentInstallations(agent, cmd.InOrStdin(), cmd.OutOrStdout())
			if err != nil {
				return err
			}
			if !dryRun && !confirm {
				return fmt.Errorf("agent installation changes files; rerun with --confirm or inspect with --dry-run")
			}

			for _, target := range targets {
				if dryRun {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Would install gha skill for %s at %s and update %s\n", target.name, target.skillPath, target.instructionsPath); err != nil {
						return err
					}
					continue
				}
				if err := installAgentGuidance(target); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Installed gha skill and guidance for %s.\n", target.name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().StringVar(&agent, "agent", "", "Agent to configure (codex, claude, both); prompts when omitted")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm writing the selected agent configuration")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the files that would be written")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

func selectedAgentInstallations(agent string, input io.Reader, output io.Writer) ([]agentInstallation, error) {
	if agent == "" {
		if _, err := fmt.Fprint(output, "Install the gha skill for which agent?\n  1) Codex\n  2) Claude Code\n  3) Both\nSelection [1-3]: "); err != nil {
			return nil, err
		}
		if _, err := fmt.Fscanln(input, &agent); err != nil {
			return nil, fmt.Errorf("read agent selection: %w", err)
		}
	}

	switch agent {
	case "1", "codex":
		codex, err := codexInstallation()
		return installationOrError(codex, err)
	case "2", "claude":
		claude, err := claudeInstallation()
		return installationOrError(claude, err)
	case "3", "both":
		codex, err := codexInstallation()
		if err != nil {
			return nil, err
		}
		claude, err := claudeInstallation()
		if err != nil {
			return nil, err
		}
		return []agentInstallation{codex, claude}, nil
	default:
		return nil, fmt.Errorf("invalid agent %q; use codex, claude, or both", agent)
	}
}

func installationOrError(installation agentInstallation, err error) ([]agentInstallation, error) {
	if err != nil {
		return nil, err
	}
	return []agentInstallation{installation}, nil
}

func codexInstallation() (agentInstallation, error) {
	root, err := agentConfigRoot("CODEX_HOME", ".codex")
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{
		name:             "Codex",
		skillPath:        filepath.Join(root, "skills", "gha", "SKILL.md"),
		instructionsPath: filepath.Join(root, "AGENTS.md"),
	}, nil
}

func claudeInstallation() (agentInstallation, error) {
	root, err := agentConfigRoot("CLAUDE_CONFIG_DIR", ".claude")
	if err != nil {
		return agentInstallation{}, err
	}
	return agentInstallation{
		name:             "Claude Code",
		skillPath:        filepath.Join(root, "skills", "gha", "SKILL.md"),
		instructionsPath: filepath.Join(root, "CLAUDE.md"),
	}, nil
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
	if err := writeFileAtomically(target.skillPath, ghaskill.Skill); err != nil {
		return fmt.Errorf("install skill for %s: %w", target.name, err)
	}

	existing, err := os.ReadFile(target.instructionsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s guidance: %w", target.name, err)
	}
	guidance, err := withManagedGuidance(existing)
	if err != nil {
		return fmt.Errorf("update %s guidance: %w", target.name, err)
	}
	if err := writeFileAtomically(target.instructionsPath, guidance); err != nil {
		return fmt.Errorf("write %s guidance: %w", target.name, err)
	}
	return nil
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
