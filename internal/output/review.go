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
		if _, err := fmt.Fprintf(writer, "#%-5d %-40s [%s] by %s\n", pr.Number, truncate(sanitizeTerminal(pr.Title), 40), styles.state(sanitizeTerminal(pr.State)), styles.username(sanitizeTerminal(pr.User.Login))); err != nil {
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
