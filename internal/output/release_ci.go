package output

import (
	"fmt"
	"io"

	"github.com/raithlin/gha/internal/release"
)

func releaseCIText(writer io.Writer, ci release.CIState) error {
	for _, signal := range []struct {
		name  string
		value release.CISignal
	}{
		{"Required-check policy", ci.Policy}, {"Workflow inventory", ci.Workflows}, {"Check inventory", ci.Checks},
	} {
		if _, err := fmt.Fprintf(writer, "%s: %s (%s)\n", signal.name, sanitizeTerminal(signal.value.State), sanitizeTerminal(signal.value.Message)); err != nil {
			return err
		}
	}
	for _, check := range ci.RequiredChecks {
		if _, err := fmt.Fprintf(writer, "  Required check %s (app %d)\n", sanitizeTerminal(check.Context), check.AppID); err != nil {
			return err
		}
	}
	for _, run := range ci.Runs {
		if _, err := fmt.Fprintf(writer, "  Workflow %s: run %d attempt %d %s/%s\n", sanitizeTerminal(run.Name), run.ID, run.RunAttempt, sanitizeTerminal(run.Status), sanitizeTerminal(run.Conclusion)); err != nil {
			return err
		}
	}
	return nil
}
