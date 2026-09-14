package commands

import (
	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
)

// newVersionCmd constructs the installed-build identification command.
func newVersionCmd(info model.VersionInfo) *cobra.Command {
	var format string
	command := &cobra.Command{
		Use:   "version",
		Short: "Show the installed GHA build identity",
		Long: `Show the version, commit, and build time embedded in this GHA binary.

Use --format json for the stable automation contract.`,
		Args: noArgsWithFormat(&format),
		RunE: func(cmd *cobra.Command, _ []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			return output.VersionInfo(cmd.OutOrStdout(), outputFormat, &info)
		},
	}
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}
