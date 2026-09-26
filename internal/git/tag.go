package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// TagInspection contains the shared local and origin facts for one tag.
type TagInspection struct {
	Tag          string
	Commit       string
	LocalExists  bool
	OriginExists bool
}

// TagWriter performs tag operations against one checkout.
type TagWriter struct{ workdir string }

// NewTagWriter creates tag operations rooted at workdir. An empty path uses
// the current working directory.
func NewTagWriter(workdir string) *TagWriter { return &TagWriter{workdir: workdir} }

// Inspect validates the tag, resolves the commit, and checks both tag refs.
func (w *TagWriter) Inspect(ctx context.Context, tag, revision string) (TagInspection, error) {
	inspection := TagInspection{Tag: tag}
	if _, err := w.run(ctx, "check-ref-format", "refs/tags/"+tag); err != nil {
		return inspection, fmt.Errorf("invalid tag name %q: %w", tag, err)
	}
	commit, err := w.run(ctx, "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if err != nil {
		return inspection, fmt.Errorf("resolve commit %q: %w", revision, err)
	}
	inspection.Commit = commit
	inspection.LocalExists, err = w.LocalTagExists(ctx, tag)
	if err != nil {
		return inspection, err
	}
	inspection.OriginExists, err = w.originTagExists(ctx, tag)
	return inspection, err
}

// LocalTagExists reports whether the exact local tag exists.
func (w *TagWriter) LocalTagExists(ctx context.Context, tag string) (bool, error) {
	if _, err := w.run(ctx, "check-ref-format", "refs/tags/"+tag); err != nil {
		return false, fmt.Errorf("invalid tag name %q: %w", tag, err)
	}
	value, err := w.run(ctx, "tag", "--list", "--", tag)
	return value != "", err
}

// Create creates a lightweight tag when message is empty and an annotated tag
// when message is supplied.
func (w *TagWriter) Create(ctx context.Context, tag, commit, message string) error {
	if _, err := w.run(ctx, "check-ref-format", "refs/tags/"+tag); err != nil {
		return fmt.Errorf("invalid tag name %q: %w", tag, err)
	}
	if message == "" {
		_, err := w.run(ctx, "tag", "--", tag, commit)
		return err
	}
	_, err := w.run(ctx, "tag", "-a", "-m", message, "--", tag, commit)
	return err
}

// Push publishes only the selected tag ref to origin.
func (w *TagWriter) Push(ctx context.Context, tag string) error {
	if _, err := w.run(ctx, "check-ref-format", "refs/tags/"+tag); err != nil {
		return fmt.Errorf("invalid tag name %q: %w", tag, err)
	}
	_, err := w.run(ctx, "push", "origin", "refs/tags/"+tag+":refs/tags/"+tag)
	return err
}

func (w *TagWriter) originTagExists(ctx context.Context, tag string) (bool, error) {
	ref := "refs/tags/" + tag
	value, err := w.run(ctx, "ls-remote", "--exit-code", "--refs", "origin", ref)
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 2 {
			return false, nil
		}
		return false, fmt.Errorf("inspect origin tag %q: %w", tag, err)
	}
	for _, line := range strings.Split(value, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == ref {
			return true, nil
		}
	}
	return false, nil
}

func (w *TagWriter) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = w.workdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}
