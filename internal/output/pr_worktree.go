package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// PRWorktree renders the same checkout, review and recovery facts in all formats.
func PRWorktree(writer io.Writer, format Format, result *model.PRWorktree) error {
	if result == nil {
		return fmt.Errorf("PR worktree result is nil")
	}
	if format != Text {
		return structured(writer, format, result)
	}
	if _, err := fmt.Fprintf(writer, "PR %s #%d: %s\n", sanitizeTerminal(result.Repository.String()), result.PullRequest.Number, sanitizeTerminal(result.PullRequest.Title)); err != nil {
		return err
	}
	values := [][2]string{{"Operation", result.Operation}, {"PR state", result.PullRequest.State}, {"Branch", result.Branch}, {"Worktree", result.Path}, {"Decision", result.Decision}, {"State", result.State}, {"Fetch", result.Fetch}, {"Branch effect", result.BranchState}, {"PR association", result.AssociationState}, {"Worktree effect", result.WorktreeState}, {"Publication comparison", result.Comparison.State}, {"Review signals", result.ReviewState.State}}
	if result.Comparison.State != "unavailable" {
		values = append(values, [2]string{"Commits absent from PR", fmt.Sprint(result.Comparison.AheadBy)}, [2]string{"Commits behind PR", fmt.Sprint(result.Comparison.BehindBy)})
	}
	if result.Worktree != nil {
		values = append(values, [2]string{"Checkout status", result.Worktree.StatusState})
	}
	if result.Review != nil {
		values = append(values, [2]string{"CI", result.Review.Readiness.CIStatus}, [2]string{"Merge readiness", result.Review.Readiness.MergeableState}, [2]string{"Review threads", result.Review.Readiness.ReviewThreadsState})
	}
	if result.Push.State == "available" {
		values = append(values, [2]string{"Push repository", result.Push.Repository.String()}, [2]string{"Push URL", result.Push.URL}, [2]string{"Push refspec", result.Push.Refspec})
	}
	if result.ReviewState.Message != "" {
		values = append(values, [2]string{"Review detail", result.ReviewState.Message})
	}
	values = append(values, [2]string{"Reason", strings.Join(result.Reasons, "; ")})
	for _, value := range values {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", value[0], sanitizeTerminal(value[1])); err != nil {
			return err
		}
	}
	if err := writePRWorktreeReviewers(writer, result.Review); err != nil {
		return err
	}
	for _, action := range result.RecommendedActions {
		if _, err := fmt.Fprintf(writer, "Next: %s — %s\n", sanitizeTerminal(action.Action), sanitizeTerminal(action.Reason)); err != nil {
			return err
		}
	}
	if result.DryRun {
		_, err := fmt.Fprintln(writer, "Dry run: no fetch or changes were made.")
		return err
	}
	return nil
}

func writePRWorktreeReviewers(writer io.Writer, review *model.ReviewSummary) error {
	if review == nil {
		return nil
	}
	values := []struct {
		label string
		users []model.User
	}{{"Approved by", review.Readiness.ApprovedBy}, {"Changes requested by", review.Readiness.ChangesRequestedBy}, {"Pending reviewers", review.Readiness.PendingReviewers}}
	for _, value := range values {
		if err := writeUsers(writer, newStyles(writer), value.label, value.users); err != nil {
			return err
		}
	}
	return nil
}
