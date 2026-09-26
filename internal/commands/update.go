package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/buildinfo"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
	ghaskill "github.com/raithlin/gha/skills/gha"
)

const updateRepository = "Raithlin/gha"

func newUpdateCmd() *cobra.Command {
	var dryRun bool
	var format string
	command := &cobra.Command{
		Use:   "update",
		Short: "Refresh compatible GHA guidance for configured coding agents",
		Long: `Find the latest published GHA release and refresh guidance and skills only
for harnesses recorded by ` + "`gha agent install`" + `. The skill declares the GHA commands it requires;
updates are rejected if this executable does not provide every required capability. This command does not update the GHA executable.

Use --dry-run to inspect configured destinations without writing files.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			return runGuidanceUpdate(cmd.Context(), cmd.OutOrStdout(), http.DefaultClient, dryRun, outputFormat)
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show configured destinations without writing")
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

func runGuidanceUpdate(ctx context.Context, writer io.Writer, client *http.Client, dryRun bool, format output.Format) error {
	ownership, err := readAgentOwnership()
	if err != nil {
		return err
	}
	result := &model.GuidanceUpdate{SchemaVersion: model.GuidanceUpdateSchemaVersion, BinaryVersion: buildinfo.Version, BinaryUpdated: false, DryRun: dryRun, SourceState: "not_checked", CompatibilityState: "not_checked", Targets: []model.GuidanceUpdateTarget{}}
	if len(ownership.Agents) == 0 {
		return output.GuidanceUpdate(writer, format, result)
	}
	release, err := fetchLatestRelease(ctx, client)
	if err != nil {
		result.SourceMessage = err.Error()
		return reportUnavailableGuidanceSource(writer, format, result, ownership, fmt.Errorf("discover latest published GHA release: %w", err))
	}
	skill, guidance, err := fetchReleaseGuidance(ctx, client, release)
	if err != nil {
		result.LatestVersion = release
		result.SourceMessage = err.Error()
		return reportUnavailableGuidanceSource(writer, format, result, ownership, fmt.Errorf("fetch GHA guidance for release %s: %w", release, err))
	}
	result.LatestVersion = release
	result.SourceState = "available"
	missing, compatibilityErr := validateSkillCapabilities(skill, ghaCapabilities())
	if compatibilityErr != nil || len(missing) > 0 {
		result.CompatibilityState = "incompatible"
		result.MissingCapabilities = missing
		var message string
		if compatibilityErr != nil {
			message = compatibilityErr.Error()
		} else {
			message = "skill requires GHA capabilities not available in this binary: " + strings.Join(missing, ", ")
		}
		result.CompatibilityMessage = message
		for _, target := range ownership.Agents {
			result.Targets = append(result.Targets, model.GuidanceUpdateTarget{AgentID: target.ID, AgentName: target.Name, SkillPath: target.SkillPath, InstructionsPath: target.InstructionsPath, State: "incompatible"})
		}
		if err := output.GuidanceUpdate(writer, format, result); err != nil {
			return err
		}
		return fmt.Errorf("refuse GHA skill update: %s", message)
	}
	result.CompatibilityState = "compatible"
	targets, err := refreshGuidanceTargets(ownership, skill, guidance, dryRun)
	if err != nil {
		return err
	}
	result.Targets = targets
	return output.GuidanceUpdate(writer, format, result)
}

func refreshGuidanceTargets(ownership agentOwnership, skill, guidance []byte, dryRun bool) ([]model.GuidanceUpdateTarget, error) {
	for _, target := range ownership.Agents {
		if err := validateOwnedSkillForUpdate(target, ownership); err != nil {
			return nil, fmt.Errorf("update skill for %s: %w", target.Name, err)
		}
	}
	updatedSkills := map[string]bool{}
	var result []model.GuidanceUpdateTarget
	for _, target := range ownership.Agents {
		state := "updated"
		if dryRun {
			state = "planned"
		} else {
			if err := updateAgentTargetFiles(target, skill, guidance, updatedSkills); err != nil {
				return nil, err
			}
		}
		if !dryRun {
			for i := range ownership.Agents {
				if ownership.Agents[i].SkillPath == target.SkillPath {
					ownership.Agents[i].SkillDigest = skillDigest(skill)
				}
			}
		}
		result = append(result, model.GuidanceUpdateTarget{AgentID: target.ID, AgentName: target.Name, SkillPath: target.SkillPath, InstructionsPath: target.InstructionsPath, State: state})
	}
	if !dryRun {
		if err := writeAgentOwnership(ownership); err != nil {
			return nil, fmt.Errorf("record updated skill ownership: %w", err)
		}
	}
	return result, nil
}

func updateAgentTargetFiles(target agentInstallation, skill, guidance []byte, updatedSkills map[string]bool) error {
	if !updatedSkills[target.SkillPath] {
		if err := writeFileAtomically(target.SkillPath, skill); err != nil {
			return fmt.Errorf("update skill for %s: %w", target.Name, err)
		}
		updatedSkills[target.SkillPath] = true
	}
	if target.InstructionsPath == "" {
		return nil
	}
	existing, err := os.ReadFile(target.InstructionsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s guidance: %w", target.Name, err)
	}
	updated, err := withManagedGuidanceContent(existing, guidance)
	if err != nil {
		return fmt.Errorf("update %s guidance: %w", target.Name, err)
	}
	if err := writeFileAtomically(target.InstructionsPath, updated); err != nil {
		return fmt.Errorf("write %s guidance: %w", target.Name, err)
	}
	return nil
}

func validateOwnedSkillForUpdate(target agentInstallation, ownership agentOwnership) error {
	content, err := os.ReadFile(target.SkillPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	expected, owned := ownership.skillDigest(target.SkillPath)
	if !owned {
		return fmt.Errorf("skill at %s is not recorded as GHA-managed", target.SkillPath)
	}
	if expected == "" {
		if !bytes.Equal(content, ghaskill.Skill) {
			return fmt.Errorf("skill at %s has changed and has no ownership digest; preserve it or restore the GHA version", target.SkillPath)
		}
		return nil
	}
	if skillDigest(content) != expected {
		return fmt.Errorf("skill at %s has changed; preserve it or restore the GHA version", target.SkillPath)
	}
	return nil
}

func reportUnavailableGuidanceSource(writer io.Writer, format output.Format, result *model.GuidanceUpdate, ownership agentOwnership, sourceErr error) error {
	result.SourceState = "unavailable"
	for _, target := range ownership.Agents {
		result.Targets = append(result.Targets, model.GuidanceUpdateTarget{AgentID: target.ID, AgentName: target.Name, SkillPath: target.SkillPath, InstructionsPath: target.InstructionsPath, State: "unavailable"})
	}
	if err := output.GuidanceUpdate(writer, format, result); err != nil {
		return err
	}
	return sourceErr
}

func fetchLatestRelease(ctx context.Context, client *http.Client) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+updateRepository+"/releases?per_page=100", nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "GHA-GitHub-Assistant")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		status := response.Status
		if err := response.Body.Close(); err != nil {
			return "", fmt.Errorf("close release response: %w", err)
		}
		return "", fmt.Errorf("GitHub returned %s", status)
	}
	var releases []struct {
		TagName     string `json:"tag_name"`
		Draft       bool   `json:"draft"`
		PublishedAt string `json:"published_at"`
	}
	decodeErr := json.NewDecoder(response.Body).Decode(&releases)
	closeErr := response.Body.Close()
	if decodeErr != nil {
		return "", decodeErr
	}
	if closeErr != nil {
		return "", fmt.Errorf("close release response: %w", closeErr)
	}
	latestTag, latestPublishedAt := "", ""
	for _, release := range releases {
		if release.Draft || release.PublishedAt == "" || strings.TrimSpace(release.TagName) == "" {
			continue
		}
		if release.PublishedAt > latestPublishedAt {
			latestTag, latestPublishedAt = release.TagName, release.PublishedAt
		}
	}
	if latestTag != "" {
		return latestTag, nil
	}
	return "", fmt.Errorf("release list contains no published release tag")
}

func fetchReleaseGuidance(ctx context.Context, client *http.Client, release string) ([]byte, []byte, error) {
	base := "https://raw.githubusercontent.com/" + updateRepository + "/" + url.PathEscape(release) + "/skills/gha/"
	skill, err := fetchGuidanceFile(ctx, client, base+path.Base("SKILL.md"))
	if err != nil {
		return nil, nil, err
	}
	guidance, err := fetchGuidanceFile(ctx, client, base+path.Base("AGENT-GUIDANCE.md"))
	if err != nil {
		return nil, nil, err
	}
	return skill, guidance, nil
}

func fetchGuidanceFile(ctx context.Context, client *http.Client, address string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "GHA-GitHub-Assistant")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		status := response.Status
		if err := response.Body.Close(); err != nil {
			return nil, fmt.Errorf("close guidance response: %w", err)
		}
		return nil, fmt.Errorf("GitHub returned %s", status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	closeErr := response.Body.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close guidance response: %w", closeErr)
	}
	if len(data) == 0 || (strings.Contains(address, "SKILL.md") && !strings.Contains(string(data), "name:")) || (strings.Contains(address, "AGENT-GUIDANCE.md") && !bytes.Contains(data, []byte("gha:begin"))) {
		return nil, fmt.Errorf("downloaded guidance is empty or invalid")
	}
	return data, nil
}
