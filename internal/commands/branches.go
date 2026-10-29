package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
)

// newBranchesCmd constructs the branch inventory command.
func newBranchesCmd(service *branch.Service) *cobra.Command {
	var format string
	var path string
	var limit int
	var refreshOrigin bool
	var dryRun bool

	command := &cobra.Command{
		Use:   "branches",
		Short: "Inspect local and origin branches",
		Long: `Inspect local branches and the remote-tracking branches for origin without changing Git state by default.

The command reads the current Git repository, or the checkout supplied with
		--path. --limit applies independently to the local and origin branch lists.

--refresh-origin explicitly fetches and prunes origin. Review its no-write plan
with --dry-run to inspect the plan; otherwise the refresh runs.`,
		Args: noArgsWithFormat(&format),
		RunE: func(cmd *cobra.Command, _ []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if limit < 1 || limit > 100 {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("limit must be between 1 and 100"))
			}
			if !refreshOrigin && dryRun {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("--dry-run requires --refresh-origin"))
			}
			inventoryService := service
			if path != "" {
				if inventoryService == nil {
					inventoryService = branch.NewService(git.NewBranchLister(path))
				} else {
					inventoryService = inventoryService.WithLister(git.NewBranchLister(path))
				}
			}
			var inventory *model.BranchInventory
			if refreshOrigin {
				inventory, err = inventoryService.RefreshOrigin(cmd.Context(), limit, dryRun)
			} else {
				inventory, err = inventoryService.Inventory(cmd.Context(), limit)
			}
			if err != nil {
				return renderCommandError(cmd, outputFormat, "branch_inventory_failed", err)
			}
			return output.BranchInventory(cmd.OutOrStdout(), outputFormat, inventory)
		},
	}
	command.SilenceUsage = true
	command.SilenceErrors = true

	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&path, "path", "", "Local Git checkout to inspect")
	command.Flags().IntVarP(&limit, "limit", "l", 30, "Maximum branches to return per source (1-100)")
	command.Flags().BoolVar(&refreshOrigin, "refresh-origin", false, "Fetch and prune origin before listing branches")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the origin refresh plan without fetching")
	command.AddCommand(newBranchesCleanupCmd(service))
	return command
}

func newBranchesCleanupCmd(services ...*branch.Service) *cobra.Command {
	var format string
	var path string
	var base string
	var limit int
	var mergedPRs bool
	command := &cobra.Command{
		Use:   "cleanup",
		Short: "Review bounded local branch cleanup candidates",
		Long: `Review local branches without changing Git or origin.

A branch is a cleanup candidate only when its tip is reachable from --base.
Without --base, GHA uses the cached origin/HEAD branch and requires that branch
to exist locally. This is not an age-based "stale" classification; provider
safety is not inferred and this command does not delete any branch.

With --merged-prs, query fresh PR evidence from the checkout's fetch origin.
Candidates require one merged PR whose repository, base, head branch and exact
head commit match the local tip. This recognises squash and rebase merges.
Reused names, unavailable evidence and checked-out branches are retained.
Each branch lookup reads at most two PR identities and one PR detail; --limit
bounds the number of local branches inspected. No fetch or write occurs.`,
		Args: noArgsWithFormat(&format),
		RunE: func(cmd *cobra.Command, _ []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			var service *branch.Service
			if len(services) > 0 {
				service = services[0]
			}
			service = service.WithLister(git.NewBranchLister(path))
			var repository model.RepositoryRef
			var repositoryError error
			if mergedPRs {
				repository, repositoryError = git.PRFetchRepository(cmd.Context(), path, "")
			}
			cleanup, err := service.Cleanup(cmd.Context(), base, limit, repository, repositoryError, mergedPRs)
			if err != nil {
				return renderCommandError(cmd, outputFormat, "branch_cleanup_failed", err)
			}
			return output.BranchCleanup(cmd.OutOrStdout(), outputFormat, cleanup)
		},
	}
	command.SilenceUsage = true
	command.SilenceErrors = true
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&path, "path", "", "Local Git checkout to inspect")
	command.Flags().StringVar(&base, "base", "", "Local branch that candidate tips must be reachable from")
	command.Flags().IntVarP(&limit, "limit", "l", 30, "Maximum local branches to review (1-100)")
	command.Flags().BoolVar(&mergedPRs, "merged-prs", false, "Require fresh merged-PR evidence matching each exact local branch tip")
	var service *branch.Service
	if len(services) > 0 {
		service = services[0]
	}
	command.AddCommand(newCleanupApplyCmd(service))
	return command
}
