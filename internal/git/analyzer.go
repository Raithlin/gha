package git

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/raithlin/gha/pkg/model"
)

// Analyzer builds a bounded, read-only summary of one local Git checkout.
type Analyzer struct {
	workdir string
}

// NewAnalyzer creates an analyzer rooted at workdir. An empty workdir uses the
// current directory.
func NewAnalyzer(workdir string) *Analyzer {
	return &Analyzer{workdir: workdir}
}

// Analyze returns local Git facts only. It never fetches, contacts a remote, or
// changes repository state.
func (a *Analyzer) Analyze(ctx context.Context, limit int) (*model.RepositoryAnalysis, error) {
	path, err := a.run(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("resolve repository root: %w", err)
	}
	worktree, err := a.worktree(ctx, limit)
	if err != nil {
		return nil, err
	}
	storage, err := a.storage(ctx)
	if err != nil {
		return nil, err
	}

	analysis := &model.RepositoryAnalysis{
		SchemaVersion: model.RepositoryAnalysisSchemaVersion,
		AnalyzedAt:    time.Now().UTC().Format(time.RFC3339),
		Path:          strings.TrimSpace(path),
		Limit:         limit,
		Worktree:      worktree,
		Storage:       storage,
	}

	head, err := a.head(ctx)
	if err != nil {
		return nil, err
	}
	analysis.Head = head
	if head.State == "unborn" {
		analysis.RecentCommits = []model.LocalCommit{}
		analysis.LargestFiles = []model.LargestFile{}
		analysis.LargestFilesSignal = model.AnalysisSignal{State: "unavailable", Message: "HEAD is unborn; no tracked files are available"}
		return analysis, nil
	}

	commits, commitsTruncated, err := a.recentCommits(ctx, limit)
	if err != nil {
		return nil, err
	}
	largest, filesTruncated, err := a.largestFiles(ctx, limit)
	if err != nil {
		return nil, err
	}
	analysis.RecentCommits = commits
	analysis.RecentCommitsTruncated = commitsTruncated
	analysis.LargestFiles = largest
	analysis.LargestFilesTruncated = filesTruncated
	analysis.LargestFilesSignal = model.AnalysisSignal{State: "available"}
	return analysis, nil
}

func (a *Analyzer) head(ctx context.Context) (model.AnalysisHead, error) {
	sha, err := a.run(ctx, "rev-parse", "--verify", "HEAD")
	if err != nil {
		if isExitError(err) {
			return model.AnalysisHead{State: "unborn"}, nil
		}
		return model.AnalysisHead{}, fmt.Errorf("read HEAD: %w", err)
	}
	count, err := a.run(ctx, "rev-list", "--count", "HEAD")
	if err != nil {
		return model.AnalysisHead{}, fmt.Errorf("count commits: %w", err)
	}
	commits, err := strconv.Atoi(strings.TrimSpace(count))
	if err != nil {
		return model.AnalysisHead{}, fmt.Errorf("parse commit count: %w", err)
	}
	branch, err := a.run(ctx, "branch", "--show-current")
	if err != nil {
		return model.AnalysisHead{}, fmt.Errorf("read current branch: %w", err)
	}
	return model.AnalysisHead{State: "available", Branch: strings.TrimSpace(branch), SHA: strings.TrimSpace(sha), Commits: commits}, nil
}

//nolint:gocyclo // Git's porcelain status format has several distinct entry cases.
func (a *Analyzer) worktree(ctx context.Context, limit int) (model.WorktreeSummary, error) {
	output, err := a.run(ctx, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return model.WorktreeSummary{}, fmt.Errorf("read worktree status: %w", err)
	}
	summary := model.WorktreeSummary{State: "clean", Changes: []model.WorktreeChange{}}
	entries := strings.Split(output, "\x00")
	for index := 0; index < len(entries); index++ {
		entry := entries[index]
		if entry == "" {
			continue
		}
		if len(entry) < 4 || entry[2] != ' ' {
			return model.WorktreeSummary{}, fmt.Errorf("parse worktree status entry %q", entry)
		}
		indexStatus, worktreeStatus := string(entry[0]), string(entry[1])
		change := model.WorktreeChange{Path: entry[3:], IndexStatus: indexStatus, WorktreeStatus: worktreeStatus}
		if indexStatus == "?" && worktreeStatus == "?" {
			summary.Untracked++
		} else {
			if indexStatus != " " {
				summary.Staged++
			}
			if worktreeStatus != " " {
				summary.Unstaged++
			}
			if conflicted(indexStatus + worktreeStatus) {
				summary.Conflicted++
			}
		}
		if indexStatus == "R" || indexStatus == "C" || worktreeStatus == "R" || worktreeStatus == "C" {
			index++
			if index >= len(entries) || entries[index] == "" {
				return model.WorktreeSummary{}, fmt.Errorf("parse renamed worktree entry %q", entry)
			}
			change.OriginalPath = entries[index]
		}
		if len(summary.Changes) < limit {
			summary.Changes = append(summary.Changes, change)
		} else {
			summary.ChangesTruncated = true
		}
	}
	if len(summary.Changes) > 0 {
		summary.State = "dirty"
	}
	return summary, nil
}

func conflicted(status string) bool {
	switch status {
	case "DD", "AU", "UD", "UA", "DU", "AA", "UU":
		return true
	default:
		return false
	}
}

func (a *Analyzer) storage(ctx context.Context) (model.RepositoryStorage, error) {
	output, err := a.run(ctx, "count-objects", "-v")
	if err != nil {
		return model.RepositoryStorage{}, fmt.Errorf("read object storage: %w", err)
	}
	values := map[string]int{}
	for _, line := range lines(output) {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return model.RepositoryStorage{}, fmt.Errorf("parse object storage entry %q", line)
		}
		value, err := strconv.Atoi(fields[1])
		if err != nil {
			return model.RepositoryStorage{}, fmt.Errorf("parse object storage value %q: %w", fields[1], err)
		}
		values[fields[0]] = value
	}
	return model.RepositoryStorage{LooseObjects: values["count"], LooseKiB: values["size"], PackedKiB: values["size-pack"]}, nil
}

func (a *Analyzer) recentCommits(ctx context.Context, limit int) ([]model.LocalCommit, bool, error) {
	output, err := a.run(ctx, "log", "-n", strconv.Itoa(limit+1), "-z", "--format=%H%x00%s%x00%aI")
	if err != nil {
		return nil, false, fmt.Errorf("read recent commits: %w", err)
	}
	fields := strings.Split(output, "\x00")
	commits := make([]model.LocalCommit, 0, limit)
	for index := 0; index < len(fields); {
		if fields[index] == "" {
			index++
			continue
		}
		if index+2 >= len(fields) {
			return nil, false, fmt.Errorf("parse recent commit")
		}
		if len(commits) == limit {
			return commits, true, nil
		}
		commits = append(commits, model.LocalCommit{SHA: fields[index], Subject: fields[index+1], AuthoredAt: fields[index+2]})
		index += 3
	}
	return commits, false, nil
}

func (a *Analyzer) largestFiles(ctx context.Context, limit int) ([]model.LargestFile, bool, error) {
	output, err := a.run(ctx, "ls-tree", "-r", "-l", "-z", "HEAD")
	if err != nil {
		return nil, false, fmt.Errorf("read tracked files: %w", err)
	}
	files := make([]model.LargestFile, 0)
	for _, entry := range strings.Split(output, "\x00") {
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "\t", 2)
		if len(parts) != 2 {
			return nil, false, fmt.Errorf("parse tracked file %q", entry)
		}
		metadata := strings.Fields(parts[0])
		if len(metadata) != 4 || metadata[1] != "blob" || metadata[3] == "-" {
			continue
		}
		size, err := strconv.ParseInt(metadata[3], 10, 64)
		if err != nil {
			return nil, false, fmt.Errorf("parse tracked file size %q: %w", metadata[3], err)
		}
		files = append(files, model.LargestFile{Path: parts[1], Bytes: size})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Bytes == files[j].Bytes {
			return files[i].Path < files[j].Path
		}
		return files[i].Bytes > files[j].Bytes
	})
	if len(files) > limit {
		return files[:limit], true, nil
	}
	return files, false, nil
}

func (a *Analyzer) run(ctx context.Context, args ...string) (string, error) {
	return (&BranchLister{workdir: a.workdir}).run(ctx, args...)
}
