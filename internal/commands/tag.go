package commands

import (
	"fmt"
	"strings"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	var format, path, commit string
	var dryRun bool
	root := &cobra.Command{Use: "tag", Short: "Publish exact Git tags", Long: "Create and publish a local tag and its exact origin ref with one reviewed workflow."}
	command := &cobra.Command{
		Use: "publish <name>", Short: "Create a tag at a selected commit and push that exact tag",
		Long: "Inspect the selected commit and local and origin tag refs. Use --dry-run to review both effects without writing. If the push fails after local creation, the result reports partial completion.",
		Args: exactArgsWithFormat(1, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTagPublish(cmd, args[0], format, path, commit, dryRun)
		},
	}
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&path, "path", "", "Local checkout")
	command.Flags().StringVar(&commit, "commit", "HEAD", "Commit to tag")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the plan without writing")
	command.SilenceUsage = true
	command.SilenceErrors = true
	root.AddCommand(command)
	return root
}

func runTagPublish(cmd *cobra.Command, tag, format, path, commit string, dryRun bool) error {
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	writer := git.NewTagWriter(path)
	inspection, err := writer.Inspect(cmd.Context(), tag, commit)
	if err != nil {
		code := "tag_publication_failed"
		if strings.Contains(err.Error(), "invalid tag name") {
			code = "invalid_argument"
		}
		return renderCommandError(cmd, f, code, err)
	}
	result := &model.TagPublication{SchemaVersion: model.TagPublicationSchemaVersion, Tag: tag, Commit: inspection.Commit, DryRun: dryRun, Ready: !inspection.LocalExists && !inspection.OriginExists, Blockers: []string{}, Local: "planned", Origin: "planned"}
	if inspection.LocalExists {
		result.Blockers = append(result.Blockers, "local tag already exists")
	}
	if inspection.OriginExists {
		result.Blockers = append(result.Blockers, "origin tag already exists")
	}
	if !result.Ready || dryRun {
		return output.TagPublication(cmd.OutOrStdout(), f, result)
	}
	if err = writer.Create(cmd.Context(), tag, result.Commit, ""); err != nil {
		return renderCommandError(cmd, f, "tag_publication_failed", err)
	}
	result.Local = "completed"
	if err = writer.Push(cmd.Context(), tag); err != nil {
		result.Origin = "failed"
		_ = output.TagPublication(cmd.OutOrStdout(), f, result)
		return renderCommandError(cmd, f, "tag_publication_failed", fmt.Errorf("local tag created but origin push failed: %w", err))
	}
	result.Origin = "completed"
	return output.TagPublication(cmd.OutOrStdout(), f, result)
}
