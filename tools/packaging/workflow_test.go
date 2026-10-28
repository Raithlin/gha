package packaging

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPackageWorkflowChangeSelection(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/packages.yml")
	require.NoError(t, err)
	var workflow struct {
		On map[string]struct {
			Paths    []string
			Branches []string
		} `yaml:"on"`
		Concurrency struct {
			Group  string
			Cancel bool `yaml:"cancel-in-progress"`
		}
	}
	require.NoError(t, yaml.Unmarshal(data, &workflow))
	require.Contains(t, workflow.On, "workflow_dispatch")
	require.Equal(t, []string{"master"}, workflow.On["push"].Branches)
	for _, event := range []string{"pull_request", "push"} {
		t.Run(event, func(t *testing.T) {
			patterns := workflow.On[event].Paths
			require.NotEmpty(t, patterns, "package builds must filter unrelated changes")
			for _, file := range []string{
				"main.go", "cmd/gha/main.go", "internal/commands/review.go",
				"pkg/model/review.go", "go.mod", "go.sum", ".goreleaser.yaml",
				"skills/gha/SKILL.md", "skills/gha/AGENT-GUIDANCE.md",
				"tools/packaging/smoke-linux.sh", "tools/packaging/smoke_macos.py",
				".github/workflows/packages.yml", ".github/workflows/release.yml",
			} {
				require.True(t, packagePathMatches(patterns, file), "must run for %s", file)
			}
			for _, file := range []string{
				"README.md", "docs/images/social.jpg", "docs/package-distribution.md",
				".backlog/tasks/task-032.md", ".groma/config.json",
			} {
				require.False(t, packagePathMatches(patterns, file), "must skip %s", file)
			}
			// One relevant file must still trigger a mixed documentation/code change.
			require.True(t, packagePathMatches(patterns, "README.md") ||
				packagePathMatches(patterns, "internal/commands/review.go"))
		})
	}
	require.Equal(t, "${{ github.workflow }}-${{ github.ref }}", workflow.Concurrency.Group)
	require.True(t, workflow.Concurrency.Cancel)
}

// Interpret the *, ** and **/ path patterns used by this workflow. Unlike
// filepath.Match, GitHub's ** also matches across directory boundaries.
func packagePathMatches(patterns []string, file string) bool {
	for _, pattern := range patterns {
		expression := regexp.QuoteMeta(pattern)
		expression = strings.ReplaceAll(expression, `\*\*/`, "(?:.*/)?")
		expression = strings.ReplaceAll(expression, `\*\*`, ".*")
		expression = strings.ReplaceAll(expression, `\*`, "[^/]*")
		if regexp.MustCompile("^" + expression + "$").MatchString(file) {
			return true
		}
	}
	return false
}
