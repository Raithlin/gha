package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
)

// repositoryAnalyzer isolates the command contract from Git process details.
type repositoryAnalyzer interface {
	Analyze(context.Context, int) (*model.RepositoryAnalysis, error)
}

// newAnalyzeCmd constructs the read-only local repository analysis command.
func newAnalyzeCmd(analyzer repositoryAnalyzer) *cobra.Command {
	var format string
	var path string
	var limit int

	command := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze a local Git repository",
		Long:  "Combine local Git worktree, history, storage, and tracked-file facts into one offline snapshot.\n\nThe command reads the current Git checkout, or the checkout supplied with --path. It does not fetch or contact a remote. --limit applies independently to changed files, recent commits, and largest tracked files.",
		Args:  noArgsWithFormat(&format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if limit < 1 || limit > 100 {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("limit must be between 1 and 100"))
			}
			analysisService := analyzer
			if path != "" {
				analysisService = git.NewAnalyzer(path)
			}
			if analysisService == nil {
				analysisService = git.NewAnalyzer("")
			}
			analysis, err := analysisService.Analyze(cmd.Context(), limit)
			if err != nil {
				return renderCommandError(cmd, outputFormat, "repository_analysis_failed", err)
			}
			return output.RepositoryAnalysis(cmd.OutOrStdout(), outputFormat, analysis)
		},
	}
	command.SilenceUsage = true
	command.SilenceErrors = true
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&path, "path", "", "Local Git checkout to inspect")
	command.Flags().IntVarP(&limit, "limit", "l", 30, "Maximum entries to return per analysis section (1-100)")
	return command
}
