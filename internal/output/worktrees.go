package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// WorktreeInventory renders a bounded inventory of registered Git worktrees.
func WorktreeInventory(writer io.Writer, format Format, inventory *model.WorktreeInventory) error {
	if inventory == nil {
		return fmt.Errorf("worktree inventory is nil")
	}
	if format != Text {
		return structured(writer, format, inventory)
	}
	styles := newStyles(writer)
	if err := writeWorktreeInventoryHeader(writer, styles, inventory); err != nil {
		return err
	}
	if len(inventory.Worktrees) == 0 {
		_, err := fmt.Fprintln(writer, "  none")
		return err
	}
	for _, worktree := range inventory.Worktrees {
		if err := writeWorktreeEntry(writer, styles, worktree); err != nil {
			return err
		}
	}
	return nil
}

func writeWorktreeInventoryHeader(writer io.Writer, styles styles, inventory *model.WorktreeInventory) error {
	if _, err := fmt.Fprintln(writer, styles.heading("Git worktrees")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "%s: %s\n%s: %d\n", styles.label("Checkout"), sanitizeTerminal(inventory.Path), styles.label("Total"), inventory.Total); err != nil {
		return err
	}
	if inventory.Truncated {
		_, err := fmt.Fprintln(writer, styles.muted("Additional worktrees omitted; increase --limit."))
		return err
	}
	return nil
}

func writeWorktreeEntry(writer io.Writer, styles styles, worktree model.Worktree) error {
	if err := writeWorktreeIdentity(writer, styles, worktree); err != nil {
		return err
	}
	if flags := worktreeFlags(worktree); len(flags) > 0 {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Flags"), sanitizeTerminal(strings.Join(flags, ", "))); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Status"), styles.state(sanitizeTerminal(worktree.StatusState))); err != nil {
		return err
	}
	return writeWorktreeStatusDetails(writer, styles, worktree)
}

func writeWorktreeIdentity(writer io.Writer, styles styles, worktree model.Worktree) error {
	label := worktree.Branch
	if worktree.Detached {
		label = "Detached"
	} else if worktree.Bare {
		label = "Bare repository"
	}
	if label == "" {
		label = "(no branch)"
	}
	_, err := fmt.Fprintf(writer, "\n%s: %s\n%s: %s\n%s: %s\n", styles.label("Path"), sanitizeTerminal(worktree.Path), styles.label("Branch"), sanitizeTerminal(label), styles.label("HEAD"), styles.commitID(sanitizeTerminal(worktree.Head)))
	return err
}

func worktreeFlags(worktree model.Worktree) []string {
	flags := make([]string, 0, 4)
	if worktree.Main {
		flags = append(flags, "main")
	}
	if worktree.Current {
		flags = append(flags, "current")
	}
	if worktree.Locked {
		flags = append(flags, "locked")
	}
	if worktree.Prunable {
		flags = append(flags, "prunable")
	}
	return flags
}

func writeWorktreeStatusDetails(writer io.Writer, styles styles, worktree model.Worktree) error {
	for _, detail := range []struct{ label, value string }{
		{"Lock reason", worktree.LockReason},
		{"Prunable reason", worktree.PrunableReason},
		{"Status detail", worktree.StatusMessage},
	} {
		if detail.value == "" {
			continue
		}
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label(detail.label), sanitizeTerminal(detail.value)); err != nil {
			return err
		}
	}
	return nil
}

// WorktreeMutation renders a worktree add or remove plan and observed result.
func WorktreeMutation(writer io.Writer, format Format, mutation *model.WorktreeMutation) error {
	if mutation == nil {
		return fmt.Errorf("worktree mutation is nil")
	}
	if format != Text {
		return structured(writer, format, mutation)
	}
	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "%s\n%s: %s\n", styles.heading("Worktree "+sanitizeTerminal(mutation.Operation)), styles.label("Path"), sanitizeTerminal(mutation.Path)); err != nil {
		return err
	}
	if mutation.Branch != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Branch"), sanitizeTerminal(mutation.Branch)); err != nil {
			return err
		}
	}
	if mutation.From != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("From"), sanitizeTerminal(mutation.From)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("State"), sanitizeTerminal(mutation.State)); err != nil {
		return err
	}
	if mutation.DryRun {
		_, err := fmt.Fprintln(writer, styles.muted("Dry run: no changes were made."))
		return err
	}
	return nil
}
