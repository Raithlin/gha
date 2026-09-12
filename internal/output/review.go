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
		styles.heading(fmt.Sprintf("Pull Request #%d: %s", pr.Number, pr.Title)),
		styles.label("State"), styles.state(pr.State),
		styles.label("Author"), styles.username(pr.User.Login),
		styles.label("Created"), styles.muted(pr.CreatedAt),
		styles.label("Updated"), styles.muted(pr.UpdatedAt))
	if err != nil {
		return err
	}
	if pr.ClosedAt != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Closed"), styles.muted(pr.ClosedAt)); err != nil {
			return err
		}
	}
	if pr.MergedAt != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", styles.label("Merged"), styles.muted(pr.MergedAt)); err != nil {
			return err
		}
	}
	if pr.Head.Ref != "" || pr.Base.Ref != "" || pr.Head.SHA != "" || pr.Base.SHA != "" {
		if _, err := fmt.Fprintf(writer, "%s: %s %s  →  %s %s\n",
			styles.label("Branches"), pr.Head.Ref, styles.commitID(pr.Head.SHA), pr.Base.Ref, styles.commitID(pr.Base.SHA)); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(writer, "\n%s:\n%s\n\n%s: %d comments, %d commits, +%d/-%d\n",
		styles.label("Description"), styles.description(terminalBody(pr)), styles.label("Stats"), pr.Comments, pr.Commits, pr.Additions, pr.Deletions)
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
			if _, err := fmt.Fprintf(writer, "  [%s] %s: %s\n", styles.severity(signal.Severity), signal.Kind, signal.Detail); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(writer, "\n"+styles.heading("Recommended next actions")+":"); err != nil {
		return err
	}
	for _, action := range summary.RecommendedActions {
		if _, err := fmt.Fprintf(writer, "  %s: %s\n", styles.action(action.Action), action.Reason); err != nil {
			return err
		}
	}
	return nil
}

// PullRequestList renders a list of pull requests.
func PullRequestList(writer io.Writer, format Format, prs []*model.PullRequest, title string) error {
	if format != Text {
		return structured(writer, format, prs)
	}

	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "%s %s:\n\n", styles.heading(title), styles.muted(fmt.Sprintf("(%d total)", len(prs)))); err != nil {
		return err
	}
	for _, pr := range prs {
		if _, err := fmt.Fprintf(writer, "#%-5d %-40s [%s] by %s\n", pr.Number, truncate(pr.Title, 40), styles.state(pr.State), styles.username(pr.User.Login)); err != nil {
			return err
		}
	}
	return nil
}

func structured(writer io.Writer, format Format, value interface{}) error {
	switch format {
	case JSON:
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	case YAML:
		encoder := yaml.NewEncoder(writer)
		defer encoder.Close()
		return encoder.Encode(value)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func truncate(value string, length int) string {
	if len(value) <= length {
		return value
	}
	if length <= 3 {
		return value[:length]
	}
	return value[:length-3] + "..."
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
		logins = append(logins, styles.username(user.Login))
	}
	_, err := fmt.Fprintf(writer, "%s: %s\n", label, strings.Join(logins, ", "))
	return err
}
