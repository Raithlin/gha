package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
)

// ReportedError marks an error already rendered to the command's diagnostic stream.
type ReportedError struct {
	err error
}

// NewReportedError wraps an error after its diagnostic was rendered by a command.
func NewReportedError(err error) error {
	return &ReportedError{err: err}
}

func (e *ReportedError) Error() string { return e.err.Error() }

func (e *ReportedError) Unwrap() error { return e.err }

// IsReportedError reports whether a command has already rendered err.
func IsReportedError(err error) bool {
	var reported *ReportedError
	return errors.As(err, &reported)
}

// renderCommandError writes stable structured diagnostics when the caller
// selected JSON or YAML, preserving stdout for successful command data.
func renderCommandError(cmd *cobra.Command, format output.Format, code string, err error) error {
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

func noArgsWithFormat(format *string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		outputFormat, err := output.ParseFormat(*format)
		if err != nil {
			return err
		}
		return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("accepts no arguments, received %d", len(args)))
	}
}

func exactArgsWithFormat(count int, format *string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == count {
			return nil
		}
		outputFormat, err := output.ParseFormat(*format)
		if err != nil {
			return err
		}
		return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("accepts %d arg(s), received %d", count, len(args)))
	}
}
