package commands

import (
	"fmt"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
	"github.com/spf13/cobra"
)

type releaseAssetOptions struct {
	repository, path, format, outputDir string
	limit                               int
	dryRun                              bool
}

func newReleaseAssetsCmd(resolver *git.RepositoryResolver, provider release.AssetProvider) *cobra.Command {
	options := &releaseAssetOptions{limit: 100}
	command := &cobra.Command{Use: "assets <tag>", Short: "List a bounded hosted release asset inventory",
		Long: "List stable asset identities, names, sizes and provider digests for one hosted release. The result says whether the bounded inventory is complete. This command makes no local or remote changes.",
		Args: exactArgsWithFormat(1, &options.format), RunE: func(cmd *cobra.Command, args []string) error {
			format, target, err := assetCommandTarget(cmd, args[0], options, resolver)
			if err != nil {
				return err
			}
			inventory, err := release.NewAssetService(provider).List(cmd.Context(), target, args[0], options.limit)
			if err != nil {
				return renderCommandError(cmd, format, "release_assets_failed", err)
			}
			return output.ReleaseAssets(cmd.OutOrStdout(), format, inventory)
		}}
	addReleaseAssetFlags(command, options)
	command.SilenceUsage, command.SilenceErrors = true, true
	return command
}

func newReleaseVerifyCmd(resolver *git.RepositoryResolver, provider release.AssetProvider) *cobra.Command {
	options := &releaseAssetOptions{limit: 100}
	command := &cobra.Command{Use: "verify <tag>", Short: "Download and verify hosted release assets",
		Long: "Download release assets into an existing --output-dir and check provider SHA-256 digests and optional SHA256SUMS. Existing files are never overwritten. --dry-run previews destinations without downloading. This command never changes the remote release.",
		Args: exactArgsWithFormat(1, &options.format), RunE: func(cmd *cobra.Command, args []string) error {
			format, target, err := assetCommandTarget(cmd, args[0], options, resolver)
			if err != nil {
				return err
			}
			if options.outputDir == "" {
				return renderCommandError(cmd, format, "invalid_argument", fmt.Errorf("--output-dir is required"))
			}
			result, err := release.NewAssetService(provider).Verify(cmd.Context(), target, args[0], options.outputDir, options.limit, options.dryRun)
			if err != nil {
				return renderCommandError(cmd, format, "release_verify_failed", err)
			}
			if err := output.ReleaseVerification(cmd.OutOrStdout(), format, result); err != nil {
				return err
			}
			if result.State != "verified" && result.State != "planned" {
				return fmt.Errorf("release asset verification failed; inspect the reported asset states")
			}
			return nil
		}}
	addReleaseAssetFlags(command, options)
	command.Flags().StringVar(&options.outputDir, "output-dir", "", "Existing directory for verified assets (required)")
	command.Flags().BoolVar(&options.dryRun, "dry-run", false, "Preview inventory and destinations without downloading")
	command.SilenceUsage, command.SilenceErrors = true, true
	return command
}

func addReleaseAssetFlags(command *cobra.Command, options *releaseAssetOptions) {
	command.Flags().StringVarP(&options.repository, "repo", "r", "", "Provider repository (owner/repo)")
	command.Flags().StringVar(&options.path, "path", "", "Local checkout whose origin selects the repository")
	command.Flags().StringVarP(&options.format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().IntVarP(&options.limit, "limit", "l", 100, "Maximum assets to return (1-100)")
}

func assetCommandTarget(cmd *cobra.Command, tag string, options *releaseAssetOptions, resolver *git.RepositoryResolver) (output.Format, model.RepositoryRef, error) {
	format, err := output.ParseFormat(options.format)
	if err != nil {
		return format, model.RepositoryRef{}, err
	}
	if !validReleaseStatusTag(tag) {
		return format, model.RepositoryRef{}, renderCommandError(cmd, format, "invalid_argument", fmt.Errorf("invalid tag name %q", tag))
	}
	if options.limit < 1 || options.limit > 100 {
		return format, model.RepositoryRef{}, renderCommandError(cmd, format, "invalid_argument", fmt.Errorf("asset limit must be between 1 and 100"))
	}
	if resolver == nil {
		return format, model.RepositoryRef{}, renderCommandError(cmd, format, "repository_resolution_failed", fmt.Errorf("repository resolver is not configured"))
	}
	target, err := resolver.ResolveAtPath(cmd.Context(), options.repository, options.path)
	if err != nil {
		return format, model.RepositoryRef{}, renderCommandError(cmd, format, "repository_resolution_failed", err)
	}
	return format, target, nil
}
