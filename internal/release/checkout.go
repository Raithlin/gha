package release

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/raithlin/gha/internal/git"
)

// GitCheckout inspects one checkout and writes only its annotated tag.
type GitCheckout struct {
	Path     string
	Workflow string
}

func (g GitCheckout) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.Path
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Inspect reads the selected commit, cleanliness, origin tip, tags, and trigger.
func (g GitCheckout) Inspect(ctx context.Context, tag string) (CheckoutState, error) {
	var state CheckoutState
	var err error
	if state.Commit, err = g.run(ctx, "rev-parse", "HEAD"); err != nil {
		return state, err
	}
	if state.Branch, err = g.run(ctx, "symbolic-ref", "--quiet", "--short", "HEAD"); err != nil {
		return state, fmt.Errorf("release requires a checked-out branch: %w", err)
	}
	status, err := g.run(ctx, "status", "--porcelain")
	if err != nil {
		return state, err
	}
	state.Clean = status == ""
	origin, err := g.run(ctx, "config", "--get", "remote.origin.url")
	if err != nil {
		return state, err
	}
	state.Repository, err = git.ParseRepositoryRemote(origin)
	if err != nil {
		return state, err
	}
	branchRef := "refs/heads/" + state.Branch
	remote, err := g.run(ctx, "ls-remote", "--exit-code", "origin", branchRef)
	if err != nil {
		return state, fmt.Errorf("read origin branch tip: %w", err)
	}
	fields := strings.Fields(remote)
	if len(fields) != 2 || fields[1] != branchRef {
		return state, fmt.Errorf("origin branch result is invalid")
	}
	state.OriginCommit = fields[0]
	localTag, err := g.run(ctx, "tag", "--list", "--", tag)
	if err != nil {
		return state, err
	}
	state.LocalTag = localTag
	state.Workflow = g.workflow(ctx, tag)
	state.ReleaseNotes = g.releaseNotes(ctx, tag, state.Workflow)
	return state, nil
}

func (g GitCheckout) workflow(ctx context.Context, tag string) WorkflowState {
	files, err := g.run(ctx, "ls-tree", "-r", "--name-only", "HEAD", "--", ".github/workflows")
	if err != nil {
		return WorkflowState{State: "unavailable", Message: err.Error()}
	}
	matches := make([]WorkflowState, 0)
	for _, file := range strings.Split(files, "\n") {
		if file == "" || (!strings.HasSuffix(file, ".yml") && !strings.HasSuffix(file, ".yaml")) {
			continue
		}
		if g.Workflow != "" && file != g.Workflow {
			continue
		}
		content, readErr := g.run(ctx, "show", "HEAD:"+file)
		if readErr != nil {
			return WorkflowState{State: "unavailable", Path: file, Message: readErr.Error()}
		}
		notesDir, notesRequired := workflowReleaseNotesDir(content)
		for _, pattern := range matchingTagPatterns(content, tag) {
			matches = append(matches, WorkflowState{State: "available", Path: file, TagPattern: pattern, ReleaseNotesDir: notesDir, ReleaseNotesRequired: notesRequired})
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	if len(matches) > 1 {
		return WorkflowState{State: "unavailable", Message: "multiple tag-triggered workflows match; choose one with --workflow"}
	}
	return WorkflowState{State: "unavailable", Path: g.Workflow, Message: "no committed workflow has a matching push tag trigger"}
}

func matchingTagPatterns(content, tag string) []string {
	var config struct {
		On struct {
			Push struct {
				Tags []string `yaml:"tags"`
			} `yaml:"push"`
		} `yaml:"on"`
	}
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil
	}
	matches := make([]string, 0)
	for _, pattern := range config.On.Push.Tags {
		matched, matchErr := path.Match(pattern, tag)
		if matchErr == nil && matched {
			matches = append(matches, pattern)
		}
	}
	return matches
}

func workflowReleaseNotesDir(content string) (string, bool) {
	var config struct {
		Env map[string]string `yaml:"env"`
	}
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return "", false
	}
	directory, configured := config.Env["GHA_RELEASE_NOTES_DIR"]
	return directory, configured
}

func (g GitCheckout) releaseNotes(ctx context.Context, tag string, workflow WorkflowState) NotesState {
	if !workflow.ReleaseNotesRequired {
		return NotesState{State: "not_required"}
	}
	directory := path.Clean(workflow.ReleaseNotesDir)
	if path.IsAbs(directory) || directory == "." || directory == ".." || strings.HasPrefix(directory, "../") || strings.Contains(directory, "\\") {
		return NotesState{State: "unavailable", Message: "workflow release-notes directory must be a repository-relative path"}
	}
	notes := NotesState{State: "unavailable", Path: path.Join(directory, tag+".md")}
	content, err := g.run(ctx, "show", "HEAD:"+notes.Path)
	if err != nil {
		notes.Message = "file is not committed at HEAD"
		return notes
	}
	if content == "" {
		notes.Message = "file is empty"
		return notes
	}
	if strings.Contains(content, "REPLACE_ME") {
		notes.Message = "file still contains template placeholders"
		return notes
	}
	notes.State = "available"
	return notes
}

// CreateTag makes an annotated tag at the checked commit.
func (g GitCheckout) CreateTag(ctx context.Context, tag, commit string) error {
	_, err := g.run(ctx, "tag", "-a", tag, "-m", "Release "+tag, commit)
	return err
}

// PushTag pushes exactly the reviewed tag ref.
func (g GitCheckout) PushTag(ctx context.Context, tag string) error {
	_, err := g.run(ctx, "push", "origin", "refs/tags/"+tag+":refs/tags/"+tag)
	return err
}
