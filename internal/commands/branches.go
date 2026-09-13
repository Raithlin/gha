package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/output"
)

// newBranchesCmd constructs the read-only branch inventory command.
func newBranchesCmd(service *branch.Service) *cobra.Command {
	var format string
	var limit int

	command := &cobra.Command{
		Use:   "branches",
		Short: "Inspect local and origin branches",
		Long: `Inspect local branches and the remote-tracking branches for origin without changing Git state.

The command reads the current Git repository. --limit applies independently to
the local and origin branch lists.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 1 || limit > 100 {
				return fmt.Errorf("limit must be between 1 and 100")
			}
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			inventory, err := service.Inventory(cmd.Context(), limit)
			if err != nil {
				return err
			}
			return output.BranchInventory(cmd.OutOrStdout(), outputFormat, inventory)
		},
	}

	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().IntVarP(&limit, "limit", "l", 30, "Maximum branches to return per source (1-100)")
	return command
}
