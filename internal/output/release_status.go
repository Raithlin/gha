package output

import (
	"fmt"
	"io"

	"github.com/raithlin/gha/internal/release"
)

// ReleaseStatus renders one release observation for people or agents.
func ReleaseStatus(writer io.Writer, format Format, result *release.Status) error {
	if format != Text {
		return structured(writer, format, result)
	}
	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "%s\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n",
		styles.heading("Release status"), styles.label("Repository"), sanitizeTerminal(result.Repository.String()),
		styles.label("Tag"), sanitizeTerminal(result.Tag), styles.label("Tag state"), sanitizeTerminal(result.TagState),
		styles.label("Commit"), sanitizeTerminal(result.Commit), styles.label("State"), sanitizeTerminal(result.State)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "%s: %s (%s)\n%s: %d, attempt %d, %s/%s\n%s: %s\n%s: %s\n%s: %s (%d observations)\n",
		styles.label("Workflow"), sanitizeTerminal(result.Workflow.State), sanitizeTerminal(result.Workflow.Path),
		styles.label("Run"), result.Workflow.RunID, result.Workflow.Attempt, sanitizeTerminal(result.Workflow.Status), sanitizeTerminal(result.Workflow.Conclusion),
		styles.label("Run URL"), sanitizeTerminal(result.Workflow.URL), styles.label("Hosted release"), sanitizeTerminal(result.HostedRelease.State),
		styles.label("Wait"), sanitizeTerminal(result.Wait.State), result.Wait.Observations); err != nil {
		return err
	}
	if result.HostedRelease.URL != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Release URL"), sanitizeTerminal(result.HostedRelease.URL)); err != nil {
			return err
		}
	}
	if result.Workflow.Message != "" {
		if _, err := fmt.Fprintf(writer, "  %s\n", sanitizeTerminal(result.Workflow.Message)); err != nil {
			return err
		}
	}
	if result.HostedRelease.Message != "" {
		if _, err := fmt.Fprintf(writer, "  %s\n", sanitizeTerminal(result.HostedRelease.Message)); err != nil {
			return err
		}
	}
	return nil
}
