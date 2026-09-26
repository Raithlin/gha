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
)

const updateRepository = "Raithlin/gha"

func newUpdateCmd() *cobra.Command {
	var dryRun bool
	var format string
	command := &cobra.Command{
		Use:   "update",
		Short: "Refresh GHA guidance for configured coding agents",
		Long: `Find the latest published GHA release and refresh guidance and skills only
for harnesses recorded by ` + "`gha agent install`" + `. This command does not update the GHA executable.

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
	result := &model.GuidanceUpdate{SchemaVersion: model.GuidanceUpdateSchemaVersion, BinaryVersion: buildinfo.Version, BinaryUpdated: false, DryRun: dryRun, SourceState: "not_checked", Targets: []model.GuidanceUpdateTarget{}}
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
	for _, target := range ownership.Agents {
		state := "updated"
		if dryRun {
			state = "planned"
		} else {
			if err := writeFileAtomically(target.SkillPath, skill); err != nil {
				return fmt.Errorf("update skill for %s: %w", target.Name, err)
			}
			if target.InstructionsPath != "" {
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
			}
		}
		result.Targets = append(result.Targets, model.GuidanceUpdateTarget{AgentID: target.ID, AgentName: target.Name, SkillPath: target.SkillPath, InstructionsPath: target.InstructionsPath, State: state})
	}
	return output.GuidanceUpdate(writer, format, result)
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
