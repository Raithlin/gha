// Package output renders command data for users and scripts.
package output

import (
	"encoding/json"
	"fmt"
	"io"

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

	_, err := fmt.Fprintf(writer, "Pull Request #%d: %s\nState: %s\nAuthor: %s\nCreated: %s\nUpdated: %s\n",
		pr.Number, pr.Title, pr.State, pr.User.Login, pr.CreatedAt, pr.UpdatedAt)
	if err != nil {
		return err
	}
	if pr.ClosedAt != "" {
		if _, err := fmt.Fprintf(writer, "Closed: %s\n", pr.ClosedAt); err != nil {
			return err
		}
	}
	if pr.MergedAt != "" {
		if _, err := fmt.Fprintf(writer, "Merged: %s\n", pr.MergedAt); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(writer, "\nDescription:\n%s\n\nStats: %d comments, %d commits, +%d/-%d\n",
		pr.Body, pr.Comments, pr.Commits, pr.Additions, pr.Deletions)
	return err
}

// PullRequestList renders a list of pull requests.
func PullRequestList(writer io.Writer, format Format, prs []*model.PullRequest, title string) error {
	if format != Text {
		return structured(writer, format, prs)
	}

	if _, err := fmt.Fprintf(writer, "%s (%d total):\n\n", title, len(prs)); err != nil {
		return err
	}
	for _, pr := range prs {
		if _, err := fmt.Fprintf(writer, "#%-5d %-40s [%s] by %s\n", pr.Number, truncate(pr.Title, 40), pr.State, pr.User.Login); err != nil {
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
