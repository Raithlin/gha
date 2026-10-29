package output

import (
	"fmt"
	"io"

	"github.com/raithlin/gha/pkg/model"
)

// CleanupExecution renders observed effects from an explicitly selected cleanup.
func CleanupExecution(writer io.Writer, format Format, result *model.CleanupExecution) error {
	if format != Text {
		return structured(writer, format, result)
	}
	if result.Selection != "" {
		if _, err := fmt.Fprintf(writer, "Cleanup selection: %s (dry run: %t)\n", sanitizeTerminal(result.Selection), result.DryRun); err != nil {
			return err
		}
	}
	for _, entry := range result.Results {
		if _, err := fmt.Fprintf(writer, "%s: %s (forced: %t)\n  local: %s; origin: %s; worktree: %s\n  %s\n", sanitizeTerminal(entry.Name), sanitizeTerminal(entry.State), entry.Forced, sanitizeTerminal(entry.Local), sanitizeTerminal(entry.Origin), sanitizeTerminal(entry.Worktree), sanitizeTerminal(entry.Reason)); err != nil {
			return err
		}
		if entry.WorktreePath != "" {
			if _, err := fmt.Fprintf(writer, "  worktree path: %s\n", sanitizeTerminal(entry.WorktreePath)); err != nil {
				return err
			}
		}
		if entry.Recovery != "" {
			if _, err := fmt.Fprintf(writer, "  recovery: %s\n", sanitizeTerminal(entry.Recovery)); err != nil {
				return err
			}
		}
	}
	return nil
}
