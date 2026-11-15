package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestReleaseAssetInventoryOutput(t *testing.T) {
	result := &release.AssetInventory{Repository: model.RepositoryRef{Owner: "acme", Name: "tool"}, Tag: "v1", ReleaseID: 9,
		Draft: true, Prerelease: true, Limit: 1, Truncated: true, Assets: []release.Asset{{ID: 11, Name: "app", Size: 6, Digest: "sha256:abc"}}}
	var text bytes.Buffer
	require.NoError(t, ReleaseAssets(&text, Text, result))
	require.Contains(t, text.String(), "truncated")
	require.Contains(t, text.String(), "app")
	var structured bytes.Buffer
	require.NoError(t, ReleaseAssets(&structured, JSON, result))
	var parsed release.AssetInventory
	require.NoError(t, json.Unmarshal(structured.Bytes(), &parsed))
	require.True(t, parsed.Truncated)
	require.Error(t, ReleaseAssets(failingWriter{}, Text, result))
	require.Error(t, ReleaseAssets(failingWriter{}, JSON, result))
	require.Error(t, ReleaseAssets(&failingWriterAt{failAt: 2}, Text, result))
}

func TestReleaseVerificationOutput(t *testing.T) {
	result := &release.Verification{Repository: model.RepositoryRef{Owner: "acme", Name: "tool"}, Tag: "v1", State: "failed", OutputDir: "/tmp/assets", ManifestState: "mismatch",
		Assets: []release.AssetVerification{{Name: "app", Destination: "/tmp/assets/app", ProviderDigestState: "verified", ManifestState: "mismatch", Message: "bad checksum"}}}
	var text bytes.Buffer
	require.NoError(t, ReleaseVerification(&text, Text, result))
	require.Contains(t, text.String(), "bad checksum")
	require.Contains(t, text.String(), "mismatch")
	var structured bytes.Buffer
	require.NoError(t, ReleaseVerification(&structured, JSON, result))
	var parsed release.Verification
	require.NoError(t, json.Unmarshal(structured.Bytes(), &parsed))
	require.Equal(t, "failed", parsed.State)
	require.Error(t, ReleaseVerification(failingWriter{}, Text, result))
	require.Error(t, ReleaseVerification(failingWriter{}, JSON, result))
	require.Error(t, ReleaseVerification(&failingWriterAt{failAt: 2}, Text, result))
	require.Error(t, ReleaseVerification(&failingWriterAt{failAt: 3}, Text, result))
}
