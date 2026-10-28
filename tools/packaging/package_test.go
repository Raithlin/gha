package packaging

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Check the distribution contract in addition to the native installation CI.
func TestReleasePackagingContract(t *testing.T) {
	data, err := os.ReadFile("../../.goreleaser.yaml")
	require.NoError(t, err)
	var config struct {
		Casks []struct {
			Name       string   `yaml:"name"`
			Binaries   []string `yaml:"binaries"`
			SkipUpload string   `yaml:"skip_upload"`
			Repository struct{ Owner, Name, Token string }
			Hooks      any
		} `yaml:"homebrew_casks"`
		Packages []struct {
			Formats      []string
			Bindir       string
			Dependencies []string
			Scripts      any
			Contents     any
		} `yaml:"nfpms"`
	}
	require.NoError(t, yaml.Unmarshal(data, &config))
	require.Len(t, config.Casks, 1)
	cask := config.Casks[0]
	require.Equal(t, "gha", cask.Name)
	require.Equal(t, []string{"gha"}, cask.Binaries)
	require.Equal(t, "false", cask.SkipUpload, "alpha releases must also update the tap")
	require.Equal(t, "Raithlin", cask.Repository.Owner)
	require.Equal(t, "homebrew-tap", cask.Repository.Name)
	require.Equal(t, "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}", cask.Repository.Token)
	require.Nil(t, cask.Hooks, "installation must never configure agent guidance")
	require.Len(t, config.Packages, 1)
	packages := config.Packages[0]
	require.ElementsMatch(t, []string{"deb", "rpm", "archlinux"}, packages.Formats)
	require.Equal(t, "/usr/bin", packages.Bindir)
	require.Contains(t, packages.Dependencies, "git")
	require.Nil(t, packages.Scripts, "package upgrades must never configure agent guidance")
	require.Nil(t, packages.Contents, "only the executable belongs in the package")
}

func TestManualTapAccessCheckCannotPublishRelease(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/release.yml")
	require.NoError(t, err)
	var workflow struct {
		On   map[string]any `yaml:"on"`
		Jobs map[string]struct {
			If    string
			Needs []string
			Steps []struct {
				Run string
			}
		}
	}
	require.NoError(t, yaml.Unmarshal(data, &workflow))
	require.Contains(t, workflow.On, "workflow_dispatch")
	require.Contains(t, workflow.Jobs["release"].Needs, "tap-access")
	require.Equal(t, "github.event_name == 'push'", workflow.Jobs["release"].If)
	access := workflow.Jobs["tap-access"]
	require.Len(t, access.Steps, 1)
	require.Contains(t, access.Steps[0].Run, "gh api repos/Raithlin/homebrew-tap")
	require.NotContains(t, access.Steps[0].Run, "goreleaser")
}
