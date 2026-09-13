package commands

import "errors"

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
