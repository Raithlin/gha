// Package output renders command data for users and scripts.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/raithlin/gha/pkg/model"
)

// Format is a supported output format.
type Format string

const (
	// Text is the default human-readable format.
	Text Format = "text"
	// JSON is an indented JSON format for scripts.
	JSON Format = "json"
	// YAML is a YAML format for scripts.
	YAML Format = "yaml"
)

// ParseFormat validates and normalizes a format value.
func ParseFormat(value string) (Format, error) {
	switch value {
	case "", "text", "human":
		return Text, nil
	case string(JSON):
		return JSON, nil
	case string(YAML):
		return YAML, nil
	default:
		return "", fmt.Errorf("unsupported format %q (use text, json, or yaml)", value)
	}
}

// PullRequest renders a pull request.
func PullRequest(writer io.Writer, format Format, pr *model.PullRequest) error {
	if format != Text {
		return structured(writer, format, pr)
	}
	styles := newStyles(writer)

	_, err := fmt.Fprintf(writer, "%s\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n",
		styles.heading(fmt.Sprintf("Pull Request #%d: %s", pr.Number, sanitizeTerminal(pr.Title))),
		styles.label("State"), styles.state(sanitizeTerminal(pr.State)),
		styles.label("Author"), styles.username(sanitizeTerminal(pr.User.Login)),
		styles.label("Created"), styles.muted(sanitizeTerminal(pr.CreatedAt)),
		styles.label("Updated"), styles.muted(sanitizeTerminal(pr.UpdatedAt)))
	if err != nil {
		return err
	}
	if pr.ClosedAt != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Closed"), styles.muted(sanitizeTerminal(pr.ClosedAt))); err != nil {
			return err
		}
	}
	if pr.MergedAt != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Merged"), styles.muted(sanitizeTerminal(pr.MergedAt))); err != nil {
			return err
		}
	}
	if pr.Head.Ref != "" || pr.Base.Ref != "" || pr.Head.SHA != "" || pr.Base.SHA != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s %s  →  %s %s\n",
			styles.label("Branches"), sanitizeTerminal(pr.Head.Ref), styles.commitID(sanitizeTerminal(pr.Head.SHA)), sanitizeTerminal(pr.Base.Ref), styles.commitID(sanitizeTerminal(pr.Base.SHA))); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(writer, "\n%s:\n%s\n\n%s: %d comments, %d commits, +%d/-%d\n",
		styles.label("Description"), styles.description(sanitizeTerminal(terminalBody(pr))), styles.label("Stats"), pr.Comments, pr.Commits, pr.Additions, pr.Deletions)
	return err
}

// terminalBody prefers GitHub's plain-text rendering. Body remains available
// for JSON and YAML output, and is used as a fallback for older API responses.
func terminalBody(pr *model.PullRequest) string {
	if pr.BodyText != "" {
		return pr.BodyText
	}
	return pr.Body
}

// ReviewSummary renders a decision-ready review summary.
func ReviewSummary(writer io.Writer, format Format, summary *model.ReviewSummary) error {
	if format != Text {
		return structured(writer, format, summary)
	}
	if err := PullRequest(writer, Text, summary.PullRequest); err != nil {
		return err
	}
	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "\n%s:\n  %s: %s\n  %s: %s\n  %s: %s\n", styles.heading("Review readiness"), styles.label("Mergeable"), mergeableText(summary.Readiness.Mergeable), styles.label("CI"), summary.Readiness.CIStatus, styles.label("Review threads"), summary.Readiness.ReviewThreadsState); err != nil {
		return err
	}
	if err := writeUsers(writer, styles, "  Approved by", summary.Readiness.ApprovedBy); err != nil {
		return err
	}
	if err := writeUsers(writer, styles, "  Changes requested by", summary.Readiness.ChangesRequestedBy); err != nil {
		return err
	}
	if err := writeUsers(writer, styles, "  Pending reviewers", summary.Readiness.PendingReviewers); err != nil {
		return err
	}
	if len(summary.RiskSignals) > 0 {
		if _, err := fmt.Fprintln(writer, "\n"+styles.heading("Risk signals")+":"); err != nil {
			return err
		}
		for _, signal := range summary.RiskSignals {
			if _, err := fmt.Fprintf(writer, "  [%s] %s: %s\n", styles.severity(sanitizeTerminal(signal.Severity)), sanitizeTerminal(signal.Kind), sanitizeTerminal(signal.Detail)); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(writer, "\n"+styles.heading("Recommended next actions")+":"); err != nil {
		return err
	}
	for _, action := range summary.RecommendedActions {
		if _, err := fmt.Fprintf(writer, "  %s: %s\n", styles.action(sanitizeTerminal(action.Action)), sanitizeTerminal(action.Reason)); err != nil {
			return err
		}
	}
	return nil
}

// ReleaseNotes renders release notes generated from merged pull requests.
func ReleaseNotes(writer io.Writer, format Format, notes *model.ReleaseNotes) error {
	if format != Text {
		return structured(writer, format, notes)
	}

	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "%s\n%s: %s\n%s: %s\n\n", styles.heading("Release notes"), styles.label("Repository"), sanitizeTerminal(notes.Repository.String()), styles.label("Merged since"), styles.muted(sanitizeTerminal(notes.Since))); err != nil {
		return err
	}
	if len(notes.PullRequests) == 0 {
		_, err := fmt.Fprintln(writer, "No pull requests were merged in this window.")
		return err
	}
	if notes.Truncated {
		if _, err := fmt.Fprintln(writer, styles.muted("Additional merged pull requests omitted; increase --limit.")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(writer, styles.heading("Changes")+":"); err != nil {
		return err
	}
	for _, pr := range notes.PullRequests {
		if _, err := fmt.Fprintf(writer, "  - #%d %s (%s)\n", pr.Number, sanitizeTerminal(pr.Title), styles.username(sanitizeTerminal(pr.User.Login))); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(writer, "\n"+styles.heading("Contributors")+":"); err != nil {
		return err
	}
	contributors := make([]string, 0, len(notes.Contributors))
	for _, contributor := range notes.Contributors {
		contributors = append(contributors, styles.username(sanitizeTerminal(contributor.Login)))
	}
	_, err := fmt.Fprintln(writer, "  "+strings.Join(contributors, ", "))
	return err
}

// PullRequestPreparation renders a preflight and, after confirmation, the
// resulting provider creation without requiring a script to parse terminal text.
func PullRequestPreparation(writer io.Writer, format Format, preparation *model.PullRequestPreparation) error {
	if format != Text {
		return structured(writer, format, preparation)
	}
	styles := newStyles(writer)
	if err := writePullRequestPreparationHeader(writer, styles, preparation); err != nil {
		return err
	}
	if err := writePullRequestComparison(writer, styles, preparation); err != nil {
		return err
	}
	if err := writeSignal(writer, styles, "Existing pull requests", preparation.ExistingRequests, fmt.Sprintf("%d", len(preparation.ExistingPullRequests))); err != nil {
		return err
	}
	if err := writeSignal(writer, styles, "Can push", preparation.Permissions, booleanText(preparation.CanPush)); err != nil {
		return err
	}
	if err := writeRiskSignals(writer, styles, preparation.RiskSignals); err != nil {
		return err
	}
	return writePullRequestCreation(writer, styles, preparation)
}

func writePullRequestPreparationHeader(writer io.Writer, styles styles, preparation *model.PullRequestPreparation) error {
	if _, err := fmt.Fprintf(writer, "%s\n%s: %s\n%s: %s → %s\n%s: %s\n",
		styles.heading("Pull request preparation"),
		styles.label("Repository"), sanitizeTerminal(preparation.Repository.String()),
		styles.label("Branches"), sanitizeTerminal(preparation.Head), sanitizeTerminal(preparation.Base),
		styles.label("Title"), sanitizeTerminal(preparation.Title)); err != nil {
		return err
	}
	if preparation.DryRun {
		if _, err := fmt.Fprintln(writer, styles.muted("Dry run: no pull request was created.")); err != nil {
			return err
		}
	}
	return nil
}

func writePullRequestComparison(writer io.Writer, styles styles, preparation *model.PullRequestPreparation) error {
	if _, err := fmt.Fprintf(writer, "%s: %s (%d ahead, %d behind)\n", styles.label("Comparison"), sanitizeTerminal(preparation.Comparison.State), preparation.Comparison.AheadBy, preparation.Comparison.BehindBy); err != nil {
		return err
	}
	if preparation.Comparison.Message != "" {
		if _, err := fmt.Fprintf(writer, "  %s\n", styles.muted(sanitizeTerminal(preparation.Comparison.Message))); err != nil {
			return err
		}
	}
	return nil
}

func writeRiskSignals(writer io.Writer, styles styles, signals []model.RiskSignal) error {
	if len(signals) > 0 {
		if _, err := fmt.Fprintln(writer, "\n"+styles.heading("Risk signals")+":"); err != nil {
			return err
		}
		for _, signal := range signals {
			if _, err := fmt.Fprintf(writer, "  [%s] %s: %s\n", styles.severity(sanitizeTerminal(signal.Severity)), sanitizeTerminal(signal.Kind), sanitizeTerminal(signal.Detail)); err != nil {
				return err
			}
		}
	}
	return nil
}

func writePullRequestCreation(writer io.Writer, styles styles, preparation *model.PullRequestPreparation) error {
	if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Creation"), sanitizeTerminal(preparation.Creation)); err != nil {
		return err
	}
	if len(preparation.RecommendedActions) > 0 {
		if _, err := fmt.Fprintln(writer, styles.heading("Recommended next actions")+":"); err != nil {
			return err
		}
		for _, action := range preparation.RecommendedActions {
			if _, err := fmt.Fprintf(writer, "  %s: %s\n", styles.action(sanitizeTerminal(action.Action)), sanitizeTerminal(action.Reason)); err != nil {
				return err
			}
		}
	}
	if preparation.CreatedPullRequest != nil {
		_, err := fmt.Fprintf(writer, "%s: #%d %s\n", styles.label("Created"), preparation.CreatedPullRequest.Number, sanitizeTerminal(preparation.CreatedPullRequest.Title))
		return err
	}
	return nil
}

// RepositoryAnalysis renders an offline local Git repository snapshot.
func RepositoryAnalysis(writer io.Writer, format Format, analysis *model.RepositoryAnalysis) error {
	if format != Text {
		return structured(writer, format, analysis)
	}
	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "%s\n%s: %s\n%s: %s\n", styles.heading("Repository analysis"), styles.label("Path"), sanitizeTerminal(analysis.Path), styles.label("Analyzed"), styles.muted(sanitizeTerminal(analysis.AnalyzedAt))); err != nil {
		return err
	}
	if analysis.Head.State == "available" {
		branch := analysis.Head.Branch
		if branch == "" {
			branch = "detached HEAD"
		}
		if _, err := fmt.Fprintf(writer, "%s: %s %s (%d commits)\n", styles.label("HEAD"), sanitizeTerminal(branch), styles.commitID(sanitizeTerminal(analysis.Head.SHA)), analysis.Head.Commits); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("HEAD"), styles.muted(sanitizeTerminal(analysis.Head.State))); err != nil {
		return err
	}
	worktree := analysis.Worktree
	if _, err := fmt.Fprintf(writer, "%s: %s (%d staged, %d unstaged, %d untracked, %d conflicted)\n", styles.label("Worktree"), styles.state(sanitizeTerminal(worktree.State)), worktree.Staged, worktree.Unstaged, worktree.Untracked, worktree.Conflicted); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "%s: %d loose objects, %d KiB loose, %d KiB packed\n", styles.label("Object storage"), analysis.Storage.LooseObjects, analysis.Storage.LooseKiB, analysis.Storage.PackedKiB); err != nil {
		return err
	}
	if err := writeWorktreeChanges(writer, styles, worktree); err != nil {
		return err
	}
	if err := writeRecentCommits(writer, styles, analysis); err != nil {
		return err
	}
	return writeLargestFiles(writer, styles, analysis)
}

func writeWorktreeChanges(writer io.Writer, styles styles, worktree model.WorktreeSummary) error {
	if _, err := fmt.Fprintln(writer, "\n"+styles.heading("Changed files")+":"); err != nil {
		return err
	}
	if worktree.ChangesTruncated {
		if _, err := fmt.Fprintln(writer, styles.muted("  additional changed files omitted; increase --limit")); err != nil {
			return err
		}
	}
	if len(worktree.Changes) == 0 {
		_, err := fmt.Fprintln(writer, "  none")
		return err
	}
	for _, change := range worktree.Changes {
		path := sanitizeTerminal(change.Path)
		if change.OriginalPath != "" {
			path = sanitizeTerminal(change.OriginalPath) + " → " + path
		}
		if _, err := fmt.Fprintf(writer, "  %s %s\n", styles.muted(change.IndexStatus+change.WorktreeStatus), path); err != nil {
			return err
		}
	}
	return nil
}

func writeRecentCommits(writer io.Writer, styles styles, analysis *model.RepositoryAnalysis) error {
	if _, err := fmt.Fprintln(writer, "\n"+styles.heading("Recent commits")+":"); err != nil {
		return err
	}
	if analysis.RecentCommitsTruncated {
		if _, err := fmt.Fprintln(writer, styles.muted("  additional commits omitted; increase --limit")); err != nil {
			return err
		}
	}
	if len(analysis.RecentCommits) == 0 {
		_, err := fmt.Fprintln(writer, "  none")
		return err
	}
	for _, commit := range analysis.RecentCommits {
		if _, err := fmt.Fprintf(writer, "  %s %s %s\n", styles.commitID(sanitizeTerminal(commit.SHA)), sanitizeTerminal(commit.Subject), styles.muted(sanitizeTerminal(commit.AuthoredAt))); err != nil {
			return err
		}
	}
	return nil
}

func writeLargestFiles(writer io.Writer, styles styles, analysis *model.RepositoryAnalysis) error {
	if _, err := fmt.Fprintln(writer, "\n"+styles.heading("Largest tracked files in HEAD")+":"); err != nil {
		return err
	}
	if analysis.LargestFilesSignal.State != "available" {
		_, err := fmt.Fprintf(writer, "  %s\n", styles.muted(signalText(model.ProviderSignal{State: analysis.LargestFilesSignal.State})))
		return err
	}
	if analysis.LargestFilesTruncated {
		if _, err := fmt.Fprintln(writer, styles.muted("  additional tracked files omitted; increase --limit")); err != nil {
			return err
		}
	}
	if len(analysis.LargestFiles) == 0 {
		_, err := fmt.Fprintln(writer, "  none")
		return err
	}
	for _, file := range analysis.LargestFiles {
		if _, err := fmt.Fprintf(writer, "  %10d B  %s\n", file.Bytes, sanitizeTerminal(file.Path)); err != nil {
			return err
		}
	}
	return nil
}

// BranchInventory renders bounded local and origin branch views.
func BranchInventory(writer io.Writer, format Format, inventory *model.BranchInventory) error {
	if format != Text {
		return structured(writer, format, inventory)
	}

	styles := newStyles(writer)
	if _, err := fmt.Fprintln(writer, styles.heading("Branches")); err != nil {
		return err
	}
	if inventory.Origin != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Origin"), sanitizeTerminal(inventory.Origin)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Origin state"), sanitizeTerminal(inventory.OriginState)); err != nil {
		return err
	}
	if err := writeBranches(writer, styles, "Local branches", inventory.Local, inventory.LocalTruncated, true); err != nil {
		return err
	}
	return writeBranches(writer, styles, "Origin branches", inventory.OriginBranches, inventory.OriginTruncated, false)
}

// BranchInspection renders one branch's local state and provider safety facts.
//
//nolint:gocyclo // This renderer presents independent safety signals in a stable text layout.
func BranchInspection(writer io.Writer, format Format, inspection *model.BranchInspection) error {
	if format != Text {
		return structured(writer, format, inspection)
	}
	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.heading("Branch"), sanitizeTerminal(inspection.Name)); err != nil {
		return err
	}
	if inspection.Repository != nil {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Repository"), sanitizeTerminal(inspection.Repository.String())); err != nil {
			return err
		}
	}
	if inspection.Origin != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s (%s)\n", styles.label("Origin"), sanitizeTerminal(inspection.Origin), sanitizeTerminal(inspection.OriginState)); err != nil {
			return err
		}
	}
	if err := writeInspectedBranch(writer, styles, "Local", inspection.Local); err != nil {
		return err
	}
	if err := writeInspectedBranch(writer, styles, "Cached origin", inspection.OriginBranch); err != nil {
		return err
	}

	safety := inspection.Safety
	if _, err := fmt.Fprintln(writer, "\n"+styles.heading("Safety signals")+":"); err != nil {
		return err
	}
	if safety.Provider != "" {
		if _, err := fmt.Fprintf(writer, "  %s: %s\n", styles.label("Provider"), providerName(safety.Provider)); err != nil {
			return err
		}
	}
	if safety.CheckedAt != "" {
		if _, err := fmt.Fprintf(writer, "  %s: %s\n", styles.label("Safety checked"), styles.muted(sanitizeTerminal(safety.CheckedAt))); err != nil {
			return err
		}
	}
	if safety.Provider != "" {
		if _, err := fmt.Fprintf(writer, "  %s: %s\n", styles.label("Provider status"), styles.muted(providerStatus(safety))); err != nil {
			return err
		}
	}
	if err := writeSignal(writer, styles, "Open pull requests", safety.Requests, fmt.Sprintf("%d", len(safety.OpenPullRequests))); err != nil {
		return err
	}
	if normalizedSignalState(safety.Requests) == "available" {
		for _, pr := range safety.OpenPullRequests {
			if _, err := fmt.Fprintf(writer, "    #%d %s\n", pr.Number, sanitizeTerminal(pr.Title)); err != nil {
				return err
			}
		}
	}
	if err := writeSignal(writer, styles, "Protected", safety.Protection, booleanText(safety.Protected)); err != nil {
		return err
	}
	if err := writeSignal(writer, styles, "Can push", safety.Permissions, booleanText(safety.CanPush)); err != nil {
		return err
	}
	if err := writeSignal(writer, styles, "Default branch", safety.DefaultBranch, sanitizeTerminal(safety.DefaultBranchName)); err != nil {
		return err
	}
	if normalizedSignalState(safety.DefaultBranch) == "available" {
		if _, err := fmt.Fprintf(writer, "  %s: %s\n", styles.label("Is default branch"), yesNoText(safety.IsDefault)); err != nil {
			return err
		}
	}
	return writeSignal(writer, styles, "Mergeable", safety.Merge, booleanText(safety.Mergeable))
}

// BranchMutation renders a branch write result without requiring scripts to
// infer whether local or origin state changed.
func BranchMutation(writer io.Writer, format Format, mutation *model.BranchMutation) error {
	if format != Text {
		return structured(writer, format, mutation)
	}
	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "%s: %s %s\n", styles.heading("Branch mutation"), sanitizeTerminal(mutation.Operation), sanitizeTerminal(mutation.Name)); err != nil {
		return err
	}
	if mutation.NewName != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("New name"), sanitizeTerminal(mutation.NewName)); err != nil {
			return err
		}
	}
	if mutation.From != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("From"), sanitizeTerminal(mutation.From)); err != nil {
			return err
		}
	}
	if mutation.CheckedOut != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Checked out"), sanitizeTerminal(mutation.CheckedOut)); err != nil {
			return err
		}
	}
	if mutation.DryRun {
		if _, err := fmt.Fprintln(writer, styles.muted("Dry run: no changes were made.")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "%s: %s\n%s: %s\n", styles.label("Local"), sanitizeTerminal(mutation.Local), styles.label("Origin"), sanitizeTerminal(mutation.Origin)); err != nil {
		return err
	}
	return nil
}

// BranchPublication renders a guarded branch publication plan or result.
func BranchPublication(writer io.Writer, format Format, publication *model.BranchPublication) error {
	if format != Text {
		return structured(writer, format, publication)
	}
	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "%s: %s → %s\n", styles.heading("Branch publication"), sanitizeTerminal(publication.Name), sanitizeTerminal(publication.Target)); err != nil {
		return err
	}
	if publication.Repository != nil {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Repository"), sanitizeTerminal(publication.Repository.String())); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "%s: %s (%s)\n", styles.label("Origin"), sanitizeTerminal(publication.Origin), sanitizeTerminal(publication.OriginState)); err != nil {
		return err
	}
	if err := writeInspectedBranch(writer, styles, "Local branch", publication.Local); err != nil {
		return err
	}
	if err := writePublicationTracking(writer, styles, publication.Local); err != nil {
		return err
	}
	if err := writeInspectedBranch(writer, styles, "Cached origin branch", publication.OriginBranch); err != nil {
		return err
	}
	if err := writeSignal(writer, styles, "Can push", publication.Permissions, booleanText(publication.CanPush)); err != nil {
		return err
	}
	if publication.DryRun {
		if _, err := fmt.Fprintln(writer, styles.muted("Dry run: no changes were made.")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Publication"), sanitizeTerminal(publication.Publication))
	return err
}

func writePublicationTracking(writer io.Writer, styles styles, branch *model.Branch) error {
	if branch == nil {
		return nil
	}
	upstream := branch.Upstream
	if upstream == "" {
		upstream = "none"
	}
	if _, err := fmt.Fprintf(writer, "  %s: %s\n", styles.label("Upstream"), sanitizeTerminal(upstream)); err != nil {
		return err
	}
	divergence := branch.DivergenceState
	if divergence == "" {
		divergence = "unavailable"
	}
	if divergence == "available" && branch.Ahead != nil && branch.Behind != nil {
		divergence = fmt.Sprintf("%d ahead, %d behind", *branch.Ahead, *branch.Behind)
	}
	if divergence == "not_tracked" {
		divergence = "not tracked"
	}
	_, err := fmt.Fprintf(writer, "  %s: %s\n", styles.label("Divergence"), sanitizeTerminal(divergence))
	return err
}

func writeInspectedBranch(writer io.Writer, styles styles, label string, branch *model.Branch) error {
	if branch == nil {
		_, err := fmt.Fprintf(writer, "%s: none\n", styles.label(label))
		return err
	}
	tracking := ""
	if branch.Upstream != "" {
		tracking = " → " + sanitizeTerminal(branch.Upstream)
		if branch.Ahead != nil && branch.Behind != nil {
			tracking += fmt.Sprintf(" (%d ahead, %d behind)", *branch.Ahead, *branch.Behind)
		}
	}
	_, err := fmt.Fprintf(writer, "%s: %s%s\n", styles.label(label), styles.commitID(sanitizeTerminal(branch.SHA)), styles.muted(tracking))
	return err
}

func writeSignal(writer io.Writer, styles styles, label string, signal model.ProviderSignal, value string) error {
	if normalizedSignalState(signal) == "available" {
		_, err := fmt.Fprintf(writer, "  %s: %s\n", styles.label(label), value)
		return err
	}
	_, err := fmt.Fprintf(writer, "  %s: %s\n", styles.label(label), styles.muted(signalText(signal)))
	return err
}

func normalizedSignalState(signal model.ProviderSignal) string {
	if signal.State == "" {
		return "unavailable"
	}
	return signal.State
}

func signalText(signal model.ProviderSignal) string {
	switch normalizedSignalState(signal) {
	case "not_applicable":
		return "not applicable"
	case "unavailable":
		return "unavailable"
	default:
		return normalizedSignalState(signal)
	}
}

func providerName(provider string) string {
	if strings.EqualFold(provider, "github") {
		return "GitHub"
	}
	return sanitizeTerminal(provider)
}

func providerStatus(safety model.BranchSafety) string {
	signals := []model.ProviderSignal{safety.Requests, safety.Protection, safety.Permissions, safety.DefaultBranch, safety.Merge}
	unavailable := 0
	for _, signal := range signals {
		if normalizedSignalState(signal) == "unavailable" {
			unavailable++
		}
	}
	switch {
	case unavailable == 0:
		return "available"
	case unavailable == len(signals):
		return "unavailable; use --format json for details"
	default:
		return fmt.Sprintf("partial; %d signals unavailable (use --format json for details)", unavailable)
	}
}

func booleanText(value *bool) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%t", *value)
}

func yesNoText(value *bool) string {
	if value == nil {
		return "unknown"
	}
	if *value {
		return "yes"
	}
	return "no"
}

func writeBranches(writer io.Writer, styles styles, title string, branches []*model.Branch, truncated, showTracking bool) error {
	if _, err := fmt.Fprintf(writer, "\n%s %s:\n", styles.heading(title), styles.muted(fmt.Sprintf("(%d total)", len(branches)))); err != nil {
		return err
	}
	if truncated {
		if _, err := fmt.Fprintln(writer, styles.muted("  additional branches omitted; increase --limit")); err != nil {
			return err
		}
	}
	if len(branches) == 0 {
		_, err := fmt.Fprintln(writer, "  none")
		return err
	}
	for _, branch := range branches {
		marker := " "
		if branch.Current {
			marker = "*"
		}
		tracking := ""
		if showTracking && branch.Upstream != "" {
			tracking = " → " + sanitizeTerminal(branch.Upstream)
			if branch.Ahead != nil && branch.Behind != nil {
				tracking += fmt.Sprintf(" (%d ahead, %d behind)", *branch.Ahead, *branch.Behind)
			} else if branch.DivergenceState == "unavailable" {
				tracking += " (divergence unavailable)"
			}
		}
		if _, err := fmt.Fprintf(writer, "  %s %-32s %s%s\n", marker, sanitizeTerminal(branch.Name), styles.commitID(sanitizeTerminal(branch.SHA)), styles.muted(tracking)); err != nil {
			return err
		}
	}
	return nil
}

// CommandError renders a command diagnostic for structured formats.
func CommandError(writer io.Writer, format Format, commandError *model.CommandError) error {
	return structured(writer, format, commandError)
}

// PullRequestList renders a list of pull requests.
func PullRequestList(writer io.Writer, format Format, list *model.PullRequestList, title string) error {
	if format != Text {
		return structured(writer, format, list)
	}

	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "%s %s:\n\n", styles.heading(title), styles.muted(fmt.Sprintf("(%d returned)", len(list.PullRequests)))); err != nil {
		return err
	}
	if list.Truncated {
		if _, err := fmt.Fprintln(writer, styles.muted("additional pull requests omitted; increase --limit")); err != nil {
			return err
		}
	}
	for _, pr := range list.PullRequests {
		if _, err := fmt.Fprintf(writer, "#%-5d %-40s [%s] by %s\n", pr.Number, truncate(sanitizeTerminal(pr.Title), 40), styles.state(sanitizeTerminal(pr.State)), styles.username(sanitizeTerminal(pr.User.Login))); err != nil {
			return err
		}
	}
	return nil
}

// Capabilities renders the machine-readable command inventory.
func Capabilities(writer io.Writer, format Format, capabilities *model.Capabilities) error {
	if format != Text {
		return structured(writer, format, capabilities)
	}

	styles := newStyles(writer)
	if _, err := fmt.Fprintln(writer, styles.heading("GHA capabilities")); err != nil {
		return err
	}
	for _, capability := range capabilities.Commands {
		note := ""
		if capability.Notes != "" {
			note = ": " + sanitizeTerminal(capability.Notes)
		}
		if _, err := fmt.Fprintf(writer, "  %s [%s]%s\n", styles.action(capability.Command), sanitizeTerminal(capability.Status), note); err != nil {
			return err
		}
	}
	return nil
}

func structured(writer io.Writer, format Format, value interface{}) (returnErr error) {
	switch format {
	case JSON:
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	case YAML:
		encoder := yaml.NewEncoder(writer)
		defer func() {
			if err := encoder.Close(); err != nil && returnErr == nil {
				returnErr = err
			}
		}()
		return encoder.Encode(value)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func truncate(value string, length int) string {
	runes := []rune(value)
	if len(runes) <= length {
		return value
	}
	if length <= 3 {
		return string(runes[:length])
	}
	return string(runes[:length-3]) + "..."
}

func mergeableText(value *bool) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%t", *value)
}

func writeUsers(writer io.Writer, styles styles, label string, users []model.User) error {
	if len(users) == 0 {
		return nil
	}
	logins := make([]string, 0, len(users))
	for _, user := range users {
		logins = append(logins, styles.username(sanitizeTerminal(user.Login)))
	}
	_, err := fmt.Fprintf(writer, "%s: %s\n", label, strings.Join(logins, ", "))
	return err
}

// sanitizeTerminal removes control characters from untrusted GitHub values
// before they are rendered for a terminal. Newlines and tabs remain useful in
// descriptions; JSON and YAML output bypass this function and retain raw data.
//
//nolint:gocyclo // Escape-sequence parsing necessarily has one branch per control form.
func sanitizeTerminal(value string) string {
	var withoutEscapes strings.Builder
	withoutEscapes.Grow(len(value))
	for i := 0; i < len(value); {
		if value[i] != '\x1b' {
			withoutEscapes.WriteByte(value[i])
			i++
			continue
		}

		i++
		if i >= len(value) {
			break
		}
		switch value[i] {
		case '[': // CSI: consume through its final byte.
			i++
			for i < len(value) && (value[i] < 0x40 || value[i] > 0x7e) {
				i++
			}
			if i < len(value) {
				i++
			}
		case ']', 'P', '^', '_': // OSC and other string controls.
			i++
			for i < len(value) {
				if value[i] == '\a' {
					i++
					break
				}
				if value[i] == '\x1b' && i+1 < len(value) && value[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
		default: // A two-byte escape sequence.
			i++
		}
	}

	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return r
		case r < 0x20 || (r >= 0x7f && r <= 0x9f):
			return -1
		default:
			return r
		}
	}, withoutEscapes.String())
}
