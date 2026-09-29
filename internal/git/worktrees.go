package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// WorktreeAddOptions selects an existing branch or creates a named branch in a
// new linked worktree.
type WorktreeAddOptions struct {
	Path      string
	Branch    string
	From      string
	NewBranch bool
}

// WorktreeService inspects and safely changes a repository's worktree set.
type WorktreeService struct {
	workdir string
}

// NewWorktreeService creates a worktree service rooted at workdir. An empty
// workdir uses the current directory.
func NewWorktreeService(workdir string) *WorktreeService {
	return &WorktreeService{workdir: workdir}
}

// List returns a bounded inventory of worktrees registered with the selected
// checkout. Status failures for individual worktrees are reported per entry.
func (s *WorktreeService) List(ctx context.Context, limit int) (*model.WorktreeInventory, error) {
	if limit < 1 {
		return nil, fmt.Errorf("limit must be at least 1")
	}
	entries, sourcePath, err := s.entries(ctx)
	if err != nil {
		return nil, err
	}
	total := len(entries)
	truncated := total > limit
	if truncated {
		entries = entries[:limit]
	}
	return &model.WorktreeInventory{
		SchemaVersion: model.WorktreeInventorySchemaVersion,
		Path:          sourcePath,
		Limit:         limit,
		Total:         total,
		Truncated:     truncated,
		Worktrees:     entries,
	}, nil
}

// BranchPath returns the path of the worktree that currently has branch
// checked out, or an empty string when no worktree has it checked out.
func (s *WorktreeService) BranchPath(ctx context.Context, branch string) (string, error) {
	entries, _, err := s.entries(ctx)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.Branch == branch {
			return entry.Path, nil
		}
	}
	return "", nil
}

// OtherBranchPath returns the path of a worktree other than the selected
// checkout that has branch checked out.
func (s *WorktreeService) OtherBranchPath(ctx context.Context, branch string) (string, error) {
	entries, _, err := s.entries(ctx)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.Branch == branch && !entry.Current {
			return entry.Path, nil
		}
	}
	return "", nil
}

func (s *WorktreeService) branchPaths(ctx context.Context) (map[string]string, error) {
	entries, _, err := s.entries(ctx)
	if err != nil {
		return nil, err
	}
	paths := make(map[string]string)
	for _, entry := range entries {
		if entry.Branch != "" {
			paths[entry.Branch] = entry.Path
		}
	}
	return paths, nil
}

// PlanAdd validates a worktree-add request without changing Git state.
func (s *WorktreeService) PlanAdd(ctx context.Context, options WorktreeAddOptions) (*model.WorktreeMutation, error) {
	sourcePath, err := s.validateAdd(ctx, options)
	if err != nil {
		return nil, err
	}
	path, err := absolutePath(options.Path, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree path: %w", err)
	}
	return &model.WorktreeMutation{SchemaVersion: model.WorktreeMutationSchemaVersion, Operation: "add", Path: path, Branch: options.Branch, From: options.From, DryRun: true, State: "planned"}, nil
}

// Add validates and creates a linked worktree, creating a branch only when
// NewBranch is explicitly selected.
func (s *WorktreeService) Add(ctx context.Context, options WorktreeAddOptions) (*model.WorktreeMutation, error) {
	sourcePath, err := s.validateAdd(ctx, options)
	if err != nil {
		return nil, err
	}
	path, err := absolutePath(options.Path, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree path: %w", err)
	}
	args := []string{"worktree", "add"}
	if options.NewBranch {
		args = append(args, "-b", options.Branch)
	}
	args = append(args, path)
	if options.NewBranch && options.From != "" {
		args = append(args, options.From)
	} else if !options.NewBranch {
		args = append(args, options.Branch)
	}
	if _, err := s.run(ctx, args...); err != nil {
		return nil, fmt.Errorf("add worktree: %w", err)
	}
	return &model.WorktreeMutation{SchemaVersion: model.WorktreeMutationSchemaVersion, Operation: "add", Path: path, Branch: options.Branch, From: options.From, State: "completed"}, nil
}

// PlanRemove validates a worktree-removal request without changing Git state.
func (s *WorktreeService) PlanRemove(ctx context.Context, target string) (*model.WorktreeMutation, error) {
	entry, err := s.removalTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	return &model.WorktreeMutation{SchemaVersion: model.WorktreeMutationSchemaVersion, Operation: "remove", Path: entry.Path, Branch: entry.Branch, DryRun: true, State: "planned"}, nil
}

// Remove validates and removes a clean, unlocked linked worktree. Git's own
// worktree-remove checks remain in force when executing the final write.
func (s *WorktreeService) Remove(ctx context.Context, target string) (*model.WorktreeMutation, error) {
	entry, err := s.removalTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	if _, err := s.run(ctx, "worktree", "remove", entry.Path); err != nil {
		return nil, fmt.Errorf("remove worktree: %w", err)
	}
	return &model.WorktreeMutation{SchemaVersion: model.WorktreeMutationSchemaVersion, Operation: "remove", Path: entry.Path, Branch: entry.Branch, State: "completed"}, nil
}

func (s *WorktreeService) validateAdd(ctx context.Context, options WorktreeAddOptions) (string, error) {
	if err := validateWorktreeAddOptions(options); err != nil {
		return "", err
	}
	entries, sourcePath, err := s.entries(ctx)
	if err != nil {
		return "", err
	}
	current := currentEntry(entries)
	if current == nil || current.Bare {
		return "", fmt.Errorf("worktree creation requires a non-bare Git checkout")
	}
	if err := s.validateAddBranch(ctx, options, entries); err != nil {
		return "", err
	}
	target, err := absolutePath(options.Path, sourcePath)
	if err != nil {
		return "", fmt.Errorf("resolve worktree path: %w", err)
	}
	if err := validateNewWorktreePath(target); err != nil {
		return "", err
	}
	return sourcePath, nil
}

func validateWorktreeAddOptions(options WorktreeAddOptions) error {
	if strings.TrimSpace(options.Path) == "" {
		return fmt.Errorf("worktree path must not be empty")
	}
	if strings.TrimSpace(options.Branch) == "" {
		return fmt.Errorf("select a local branch with --branch")
	}
	if strings.HasPrefix(options.Branch, "-") {
		return fmt.Errorf("invalid branch name %q", options.Branch)
	}
	if options.NewBranch && options.From != "" && strings.HasPrefix(options.From, "-") {
		return fmt.Errorf("invalid start point %q", options.From)
	}
	if !options.NewBranch && options.From != "" {
		return fmt.Errorf("--from requires --new-branch")
	}
	return nil
}

func (s *WorktreeService) validateAddBranch(ctx context.Context, options WorktreeAddOptions, entries []model.Worktree) error {
	if _, err := s.run(ctx, "check-ref-format", "--branch", options.Branch); err != nil {
		return fmt.Errorf("invalid branch name %q", options.Branch)
	}
	if options.From != "" {
		if _, err := s.run(ctx, "rev-parse", "--verify", "--quiet", "--end-of-options", options.From+"^{commit}"); err != nil {
			return fmt.Errorf("start point %q does not resolve to a commit", options.From)
		}
	}
	exists, err := s.localBranchExists(ctx, options.Branch)
	if err != nil {
		return err
	}
	if options.NewBranch {
		if exists {
			return fmt.Errorf("branch %q already exists", options.Branch)
		}
		return nil
	}
	if !exists {
		return fmt.Errorf("local branch %q does not exist", options.Branch)
	}
	for _, entry := range entries {
		if entry.Branch == options.Branch {
			return fmt.Errorf("branch %q is already checked out at %s", options.Branch, entry.Path)
		}
	}
	return nil
}

func (s *WorktreeService) localBranchExists(ctx context.Context, branch string) (bool, error) {
	_, err := s.run(ctx, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	if err == nil {
		return true, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("check local branch %q: %w", branch, err)
}

func validateNewWorktreePath(target string) error {
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("worktree path %s already exists", target)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect worktree path %s: %w", target, err)
	}
	parentInfo, err := os.Stat(filepath.Dir(target))
	if err != nil {
		return fmt.Errorf("inspect worktree parent directory: %w", err)
	}
	if !parentInfo.IsDir() {
		return fmt.Errorf("worktree parent directory is not a directory")
	}
	return nil
}

func (s *WorktreeService) removalTarget(ctx context.Context, target string) (model.Worktree, error) {
	if strings.TrimSpace(target) == "" {
		return model.Worktree{}, fmt.Errorf("worktree path must not be empty")
	}
	entries, sourcePath, err := s.entries(ctx)
	if err != nil {
		return model.Worktree{}, err
	}
	current := currentEntry(entries)
	if current == nil || current.Bare {
		return model.Worktree{}, fmt.Errorf("worktree removal requires a non-bare Git checkout")
	}
	path, err := absolutePath(target, sourcePath)
	if err != nil {
		return model.Worktree{}, fmt.Errorf("resolve worktree path: %w", err)
	}
	for _, entry := range entries {
		entryPath, pathErr := absolutePath(entry.Path, sourcePath)
		if pathErr != nil || entryPath != path {
			continue
		}
		if entry.Main {
			return model.Worktree{}, fmt.Errorf("refusing to remove the main worktree %s", entry.Path)
		}
		if entry.Current {
			return model.Worktree{}, fmt.Errorf("refusing to remove the selected checkout %s; select another worktree with --path", entry.Path)
		}
		if entry.Prunable {
			return model.Worktree{}, fmt.Errorf("worktree %s is prunable because its checkout is unavailable; use git worktree prune after reviewing it", entry.Path)
		}
		if entry.Locked {
			return model.Worktree{}, fmt.Errorf("worktree %s is locked; unlock it with git worktree unlock after reviewing the lock", entry.Path)
		}
		if entry.StatusState != "clean" {
			return model.Worktree{}, fmt.Errorf("worktree %s cannot be safely removed; status is %s", entry.Path, entry.StatusState)
		}
		return entry, nil
	}
	return model.Worktree{}, fmt.Errorf("worktree path %s is not registered with this repository", path)
}

func (s *WorktreeService) entries(ctx context.Context) ([]model.Worktree, string, error) {
	output, err := s.run(ctx, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, "", fmt.Errorf("list Git worktrees: %w", err)
	}
	entries, err := parseWorktreeList(output)
	if err != nil {
		return nil, "", err
	}
	sourcePath, err := s.run(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		bare, bareErr := s.run(ctx, "rev-parse", "--is-bare-repository")
		if bareErr != nil || strings.TrimSpace(bare) != "true" {
			return nil, "", fmt.Errorf("resolve selected Git checkout: %w", err)
		}
		sourcePath = s.workdir
	}
	sourcePath, err = absolutePath(strings.TrimSpace(sourcePath), "")
	if err != nil {
		return nil, "", fmt.Errorf("resolve selected Git checkout: %w", err)
	}
	for index := range entries {
		entries[index].Main = index == 0
		entryPath, pathErr := absolutePath(entries[index].Path, sourcePath)
		if pathErr == nil {
			entries[index].Path = entryPath
		}
		entries[index].Current = entries[index].Path == sourcePath
		if entries[index].Bare {
			entries[index].StatusState = "not_applicable"
			continue
		}
		if entries[index].Prunable || pathErr != nil {
			entries[index].StatusState = "unavailable"
			entries[index].StatusMessage = "worktree checkout is unavailable"
			continue
		}
		status, statusErr := runGitAt(ctx, entries[index].Path, "status", "--porcelain=v1", "--untracked-files=all")
		if statusErr != nil {
			entries[index].StatusState = "unavailable"
			entries[index].StatusMessage = statusErr.Error()
		} else if strings.TrimSpace(status) == "" {
			entries[index].StatusState = "clean"
		} else {
			entries[index].StatusState = "dirty"
		}
	}
	return entries, sourcePath, nil
}

func currentEntry(entries []model.Worktree) *model.Worktree {
	for index := range entries {
		if entries[index].Current {
			return &entries[index]
		}
	}
	return nil
}

func parseWorktreeList(output string) ([]model.Worktree, error) {
	if output == "" {
		return nil, nil
	}
	var entries []model.Worktree
	var current *model.Worktree
	for _, field := range strings.Split(output, "\x00") {
		if field == "" {
			if err := finishWorktreeRecord(&current, &entries); err != nil {
				return nil, err
			}
			continue
		}
		if err := parseWorktreeField(&current, field, &entries); err != nil {
			return nil, err
		}
	}
	if err := finishWorktreeRecord(&current, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func parseWorktreeField(current **model.Worktree, field string, entries *[]model.Worktree) error {
	key, value, hasValue := strings.Cut(field, " ")
	if key == "worktree" {
		if err := finishWorktreeRecord(current, entries); err != nil {
			return err
		}
		*current = &model.Worktree{Path: value}
		return nil
	}
	if *current == nil {
		return fmt.Errorf("parse Git worktree list: field %q appears before a worktree path", field)
	}
	if !hasValue {
		value = ""
	}
	switch key {
	case "HEAD":
		(*current).Head = value
	case "branch":
		(*current).Branch = strings.TrimPrefix(value, "refs/heads/")
	case "detached":
		(*current).Detached = true
	case "bare":
		(*current).Bare = true
	case "locked":
		(*current).Locked = true
		(*current).LockReason = value
	case "prunable":
		(*current).Prunable = true
		(*current).PrunableReason = value
	}
	return nil
}

func finishWorktreeRecord(current **model.Worktree, entries *[]model.Worktree) error {
	if *current == nil {
		return nil
	}
	if (*current).Path == "" {
		return fmt.Errorf("parse Git worktree list: record is missing its path")
	}
	if (*current).Head == "" && !(*current).Bare {
		return fmt.Errorf("parse Git worktree list: worktree %q is missing its HEAD", (*current).Path)
	}
	*entries = append(*entries, **current)
	*current = nil
	return nil
}

func absolutePath(path, base string) (string, error) {
	if !filepath.IsAbs(path) {
		if base != "" {
			path = filepath.Join(base, path)
		}
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	if resolved, resolveErr := filepath.EvalSymlinks(absPath); resolveErr == nil {
		return resolved, nil
	}
	return absPath, nil
}

func (s *WorktreeService) run(ctx context.Context, args ...string) (string, error) {
	return runGitAt(ctx, s.workdir, args...)
}

func runGitAt(ctx context.Context, workdir string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = workdir
	output, err := command.CombinedOutput()
	if err == nil {
		return string(output), nil
	}
	diagnostic := strings.TrimSpace(string(output))
	if diagnostic != "" {
		return "", fmt.Errorf("%w: %s", err, diagnostic)
	}
	return "", err
}
