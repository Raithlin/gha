package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
)

// newBranchesCmd constructs the read-only branch inventory command.
func newBranchesCmd(service *branch.Service) *cobra.Command {
	var format string
	var path string
	var limit int

	command := &cobra.Command{
		Use:   "branches",
		Short: "Inspect local and origin branches",
		Long: `Inspect local branches and the remote-tracking branches for origin without changing Git state.

The command reads the current Git repository, or the checkout supplied with
--path. --limit applies independently to the local and origin branch lists.`,
		Args: noArgsWithFormat(&format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if limit < 1 || limit > 100 {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("limit must be between 1 and 100"))
			}
			inventoryService := service
			if path != "" {
				inventoryService = branch.NewService(git.NewBranchLister(path))
			}
			inventory, err := inventoryService.Inventory(cmd.Context(), limit)
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
	return command
}
