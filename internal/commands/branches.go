package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
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
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if limit < 1 || limit > 100 {
				return renderBranchError(cmd, outputFormat, "invalid_argument", fmt.Errorf("limit must be between 1 and 100"))
			}
			inventory, err := service.Inventory(cmd.Context(), limit)
			if err != nil {
				return renderBranchError(cmd, outputFormat, "branch_inventory_failed", err)
			}
			return output.BranchInventory(cmd.OutOrStdout(), outputFormat, inventory)
		},
	}
	command.SilenceUsage = true
	command.SilenceErrors = true

	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().IntVarP(&limit, "limit", "l", 30, "Maximum branches to return per source (1-100)")
	return command
}

func renderBranchError(cmd *cobra.Command, format output.Format, code string, err error) error {
	if format == output.JSON || format == output.YAML {
		if renderErr := output.CommandError(cmd.ErrOrStderr(), format, &model.CommandError{
			SchemaVersion: model.ErrorSchemaVersion,
			Code:          code,
			Message:       err.Error(),
		}); renderErr == nil {
			return NewReportedError(err)
		}
	}
	return err
}
