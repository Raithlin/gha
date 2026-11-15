package commands

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type commandAssetProvider struct{ digest string }

type incompleteCommandProvider struct{ commandAssetProvider }

func (incompleteCommandProvider) ListAssets(context.Context, model.RepositoryRef, string, int) (*release.AssetInventory, error) {
	return &release.AssetInventory{Complete: false, Truncated: true, Assets: []release.Asset{{ID: 11, Name: "app.tar.gz", Size: 6}}}, nil
}

func (p commandAssetProvider) ListAssets(_ context.Context, _ model.RepositoryRef, tag string, _ int) (*release.AssetInventory, error) {
	digest := p.digest
	if digest == "" {
		digest = "sha256:deadbeef"
	}
	return &release.AssetInventory{Tag: tag, Complete: true, Assets: []release.Asset{{ID: 11, Name: "app.tar.gz", Size: 6, Digest: digest}}}, nil
}
func (commandAssetProvider) DownloadAsset(_ context.Context, _ model.RepositoryRef, _ int64, out io.Writer) error {
	_, err := io.WriteString(out, "binary")
	return err
}

func TestReleaseAssetsJSONAndText(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		command := newReleaseAssetsCmd(git.NewRepositoryResolver("acme/tool"), commandAssetProvider{})
		command.SetArgs([]string{"v1", "--format", format})
		var stdout bytes.Buffer
		command.SetOut(&stdout)
		require.NoError(t, command.Execute())
		if format == "json" {
			var got release.AssetInventory
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
			require.True(t, got.Complete)
			require.Equal(t, int64(11), got.Assets[0].ID)
		} else {
			require.Contains(t, stdout.String(), "app.tar.gz")
			require.Contains(t, stdout.String(), "complete")
		}
	}
}

func TestReleaseVerifyDryRunDoesNotDownload(t *testing.T) {
	command := newReleaseVerifyCmd(git.NewRepositoryResolver("acme/tool"), commandAssetProvider{})
	command.SetArgs([]string{"v1", "--output-dir", t.TempDir(), "--dry-run", "--format", "json"})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	require.NoError(t, command.Execute())
	var got release.Verification
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	require.Equal(t, "planned", got.State)
	require.True(t, got.DryRun)
	require.Len(t, got.Assets, 1)
}

func TestReleaseAssetCommandsRejectInvalidArguments(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command *cobra.Command
		args    []string
	}{
		{"invalid tag", newReleaseAssetsCmd(git.NewRepositoryResolver("acme/tool"), commandAssetProvider{}), []string{"bad tag"}},
		{"invalid limit", newReleaseAssetsCmd(git.NewRepositoryResolver("acme/tool"), commandAssetProvider{}), []string{"v1", "--limit", "0"}},
		{"invalid format", newReleaseAssetsCmd(git.NewRepositoryResolver("acme/tool"), commandAssetProvider{}), []string{"v1", "--format", "xml"}},
		{"missing resolver", newReleaseAssetsCmd(nil, commandAssetProvider{}), []string{"v1"}},
		{"missing directory", newReleaseVerifyCmd(git.NewRepositoryResolver("acme/tool"), commandAssetProvider{}), []string{"v1"}},
		{"missing provider", newReleaseAssetsCmd(git.NewRepositoryResolver("acme/tool"), nil), []string{"v1"}},
	} {
		t.Run(tc.name, func(t *testing.T) { tc.command.SetArgs(tc.args); require.Error(t, tc.command.Execute()) })
	}
}

func TestReleaseVerifyIncompleteInventoryIsStructuredFailure(t *testing.T) {
	command := newReleaseVerifyCmd(git.NewRepositoryResolver("acme/tool"), incompleteCommandProvider{})
	command.SetArgs([]string{"v1", "--output-dir", t.TempDir(), "--format", "json"})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	require.Error(t, command.Execute())
	var got release.Verification
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	require.Equal(t, "incomplete", got.State)
	require.False(t, got.InventoryComplete)
}

func TestReleaseVerifyFailureKeepsStructuredResult(t *testing.T) {
	command := newReleaseVerifyCmd(git.NewRepositoryResolver("acme/tool"), commandAssetProvider{})
	command.SetArgs([]string{"v1", "--output-dir", t.TempDir(), "--format", "json"})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	require.ErrorContains(t, command.Execute(), "verification failed")
	var got release.Verification
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	require.Equal(t, "failed", got.State)
	require.Equal(t, "unsupported", got.Assets[0].ProviderDigestState)
}

func TestReleaseVerifySuccessfulJSONAndText(t *testing.T) {
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("binary")))
	for _, format := range []string{"json", "text"} {
		t.Run(format, func(t *testing.T) {
			command := newReleaseVerifyCmd(git.NewRepositoryResolver("acme/tool"), commandAssetProvider{digest: digest})
			command.SetArgs([]string{"v1", "--output-dir", t.TempDir(), "--format", format})
			var stdout bytes.Buffer
			command.SetOut(&stdout)
			require.NoError(t, command.Execute())
			if format == "json" {
				var got release.Verification
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
				require.Equal(t, "verified", got.State)
			} else {
				require.Contains(t, stdout.String(), "verified")
				require.Contains(t, stdout.String(), "app.tar.gz")
			}
		})
	}
}
