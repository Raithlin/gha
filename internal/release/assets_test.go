package release

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

type assetProviderStub struct {
	inventory *AssetInventory
	bytes     map[int64]string
}

type failingAssetProvider struct {
	assetProviderStub
	destination string
	fail        bool
}

func (p failingAssetProvider) DownloadAsset(ctx context.Context, repo model.RepositoryRef, id int64, out io.Writer) error {
	if p.destination != "" {
		if err := os.WriteFile(p.destination, []byte("keep"), 0600); err != nil {
			return err
		}
	}
	if p.fail {
		return errors.New("download interrupted")
	}
	return p.assetProviderStub.DownloadAsset(ctx, repo, id, out)
}

func (p assetProviderStub) ListAssets(_ context.Context, _ model.RepositoryRef, _ string, _ int) (*AssetInventory, error) {
	return p.inventory, nil
}
func (p assetProviderStub) DownloadAsset(_ context.Context, _ model.RepositoryRef, id int64, out io.Writer) error {
	_, err := io.Copy(out, strings.NewReader(p.bytes[id]))
	return err
}

func assetDigest(value string) string { return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(value))) }

func TestVerifyAssetsChecksProviderAndManifestBytes(t *testing.T) {
	manifest := fmt.Sprintf("%x  app.tar.gz\n", sha256.Sum256([]byte("binary")))
	p := assetProviderStub{inventory: &AssetInventory{Tag: "v1", Complete: true, Assets: []Asset{
		{ID: 1, Name: "app.tar.gz", Size: 6, Digest: assetDigest("binary")},
		{ID: 2, Name: "checksums.txt", Size: int64(len(manifest)), Digest: assetDigest(manifest)},
	}}, bytes: map[int64]string{1: "binary", 2: manifest}}
	dir := t.TempDir()
	result, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", dir, 100, false)
	require.NoError(t, err)
	require.Equal(t, "verified", result.State)
	require.Equal(t, "verified", result.Assets[0].ProviderDigestState)
	require.Equal(t, "verified", result.Assets[0].ManifestState)
	require.Equal(t, "verified", result.Assets[1].ProviderDigestState)
	data, err := os.ReadFile(filepath.Join(dir, "app.tar.gz"))
	require.NoError(t, err)
	require.Equal(t, "binary", string(data))
}

func TestVerifyAssetsRejectsMismatchAndLeavesNoFinalFile(t *testing.T) {
	p := assetProviderStub{inventory: &AssetInventory{Complete: true, Assets: []Asset{{ID: 1, Name: "app.tar.gz", Size: 6, Digest: assetDigest("other!")}}}, bytes: map[int64]string{1: "binary"}}
	dir := t.TempDir()
	result, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", dir, 100, false)
	require.NoError(t, err)
	require.Equal(t, "failed", result.State)
	require.Equal(t, "mismatch", result.Assets[0].ProviderDigestState)
	_, err = os.Stat(filepath.Join(dir, "app.tar.gz"))
	require.True(t, os.IsNotExist(err))
}

func TestVerifyAssetsDryRunAndExistingDestination(t *testing.T) {
	p := assetProviderStub{inventory: &AssetInventory{Complete: true, Assets: []Asset{{ID: 1, Name: "app.tar.gz", Size: 6, Digest: assetDigest("binary")}}}, bytes: map[int64]string{1: "binary"}}
	dir := t.TempDir()
	plan, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", dir, 100, true)
	require.NoError(t, err)
	require.Equal(t, "planned", plan.State)
	_, err = os.Stat(filepath.Join(dir, "app.tar.gz"))
	require.True(t, os.IsNotExist(err))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.tar.gz"), []byte("keep"), 0600))
	_, err = NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", dir, 100, false)
	require.ErrorContains(t, err, "already exists")
	data, err := os.ReadFile(filepath.Join(dir, "app.tar.gz"))
	require.NoError(t, err)
	require.Equal(t, "keep", string(data))
}

func TestVerifyAssetsRefusesIncompleteAndUnsafeNames(t *testing.T) {
	for _, inventory := range []*AssetInventory{
		{Complete: true, Assets: []Asset{{ID: 1, Name: "../escape", Digest: assetDigest("x")}}},
		{Complete: true, Assets: []Asset{{ID: 1, Name: "same", Digest: assetDigest("x")}, {ID: 2, Name: "same", Digest: assetDigest("x")}}},
	} {
		_, err := NewAssetService(assetProviderStub{inventory: inventory}).Verify(context.Background(), model.RepositoryRef{}, "v1", t.TempDir(), 100, false)
		require.Error(t, err)
	}
}

func TestVerifyAssetsReportsIncompleteInventory(t *testing.T) {
	p := assetProviderStub{inventory: &AssetInventory{Complete: false, Truncated: true, Assets: []Asset{{ID: 1, Name: "app", Size: 6, Digest: assetDigest("binary")}}}}
	result, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", t.TempDir(), 1, false)
	require.NoError(t, err)
	require.Equal(t, "incomplete", result.State)
	require.False(t, result.InventoryComplete)
	require.Empty(t, result.Assets)
}

func TestVerifyAssetsCannotTrustManifestWithoutProviderDigest(t *testing.T) {
	manifest := fmt.Sprintf("%x  app\n", sha256.Sum256([]byte("binary")))
	p := assetProviderStub{inventory: &AssetInventory{Complete: true, Assets: []Asset{
		{ID: 1, Name: "app", Size: 6, Digest: assetDigest("binary")},
		{ID: 2, Name: "SHA256SUMS", Size: int64(len(manifest))},
	}}, bytes: map[int64]string{1: "binary", 2: manifest}}
	result, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", t.TempDir(), 100, false)
	require.NoError(t, err)
	require.Equal(t, "failed", result.State)
	require.Equal(t, "unverified", result.ManifestState)
	require.Equal(t, "missing", result.Assets[1].ProviderDigestState)
}

func TestVerifyAssetsRejectsUnknownManifestEntry(t *testing.T) {
	manifest := fmt.Sprintf("%x  other.tar.gz\n", sha256.Sum256([]byte("binary")))
	p := assetProviderStub{inventory: &AssetInventory{Complete: true, Assets: []Asset{
		{ID: 1, Name: "app.tar.gz", Size: 6, Digest: assetDigest("binary")},
		{ID: 2, Name: "SHA256SUMS", Size: int64(len(manifest)), Digest: assetDigest(manifest)},
	}}, bytes: map[int64]string{1: "binary", 2: manifest}}
	result, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", t.TempDir(), 100, false)
	require.NoError(t, err)
	require.Equal(t, "failed", result.State)
	require.Equal(t, "unsafe", result.ManifestState)
}

func TestVerifyAssetsDigestEvidenceStates(t *testing.T) {
	cases := []struct{ name, digest, state string }{
		{"missing", "", "missing"},
		{"unsupported algorithm", "sha512:abc", "unsupported"},
		{"malformed digest", "sha256:bad", "unsupported"},
		{"invalid hex", "sha256:" + strings.Repeat("z", 64), "unsupported"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := assetProviderStub{inventory: &AssetInventory{Complete: true, Assets: []Asset{{ID: 1, Name: "app", Size: 6, Digest: tc.digest}}}, bytes: map[int64]string{1: "binary"}}
			result, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", t.TempDir(), 100, false)
			require.NoError(t, err)
			require.Equal(t, "failed", result.State)
			require.Equal(t, tc.state, result.Assets[0].ProviderDigestState)
		})
	}
}

func TestVerifyAssetsManifestMissingAndMismatch(t *testing.T) {
	for _, tc := range []struct{ name, manifest, state string }{
		{"missing", "", "missing"},
		{"mismatch", fmt.Sprintf("%x  app\n", sha256.Sum256([]byte("other!"))), "mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := assetProviderStub{inventory: &AssetInventory{Complete: true, Assets: []Asset{
				{ID: 1, Name: "app", Size: 6, Digest: assetDigest("binary")},
				{ID: 2, Name: "SHA256SUMS", Size: int64(len(tc.manifest)), Digest: assetDigest(tc.manifest)},
			}}, bytes: map[int64]string{1: "binary", 2: tc.manifest}}
			result, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", t.TempDir(), 100, false)
			require.NoError(t, err)
			require.Equal(t, "failed", result.State)
			require.Equal(t, tc.state, result.Assets[0].ManifestState)
		})
	}
}

func TestVerifyAssetsSizeMismatchAndUnavailableDownload(t *testing.T) {
	p := assetProviderStub{inventory: &AssetInventory{Complete: true, Assets: []Asset{{ID: 1, Name: "app", Size: 7, Digest: assetDigest("binary")}}}, bytes: map[int64]string{1: "binary"}}
	result, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", t.TempDir(), 100, false)
	require.NoError(t, err)
	require.Equal(t, "mismatch", result.Assets[0].ProviderDigestState)
	p.bytes = nil
	result, err = NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", t.TempDir(), 100, false)
	require.NoError(t, err)
	require.Equal(t, "mismatch", result.Assets[0].ProviderDigestState)
}

func TestVerifyAssetsRejectsAbsentAssetsAndInvalidOutputDirectory(t *testing.T) {
	service := NewAssetService(assetProviderStub{inventory: &AssetInventory{Complete: true}})
	_, err := service.Verify(context.Background(), model.RepositoryRef{}, "v1", t.TempDir(), 100, false)
	require.ErrorContains(t, err, "no assets")
	service = NewAssetService(assetProviderStub{inventory: &AssetInventory{Complete: true, Assets: []Asset{{ID: 1, Name: "app", Size: 1}}}})
	_, err = service.Verify(context.Background(), model.RepositoryRef{}, "v1", "", 100, true)
	require.ErrorContains(t, err, "output-dir")
	_, err = service.Verify(context.Background(), model.RepositoryRef{}, "v1", filepath.Join(t.TempDir(), "missing"), 100, true)
	require.Error(t, err)
	dir := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(dir, link))
	_, err = service.Verify(context.Background(), model.RepositoryRef{}, "v1", link, 100, true)
	require.ErrorContains(t, err, "symlink")
}

func TestManifestParserRejectsUnsafeEntries(t *testing.T) {
	for _, content := range []string{"not a checksum\n", strings.Repeat("z", 64) + "  app\n", strings.Repeat("a", 64) + "  ../app\n", strings.Repeat("a", 64) + "  app\n" + strings.Repeat("b", 64) + "  app\n"} {
		path := filepath.Join(t.TempDir(), "manifest")
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))
		_, err := readManifest(path)
		require.Error(t, err)
	}
	path := filepath.Join(t.TempDir(), "manifest")
	require.NoError(t, os.WriteFile(path, []byte(strings.Repeat("a", 4<<20+1)), 0600))
	_, err := readManifest(path)
	require.ErrorContains(t, err, "4 MiB")
}

func TestVerifyAssetsKeepsFileCreatedDuringDownload(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "app")
	p := failingAssetProvider{assetProviderStub: assetProviderStub{inventory: &AssetInventory{Complete: true, Assets: []Asset{{ID: 1, Name: "app", Size: 6, Digest: assetDigest("binary")}}}, bytes: map[int64]string{1: "binary"}}, destination: destination}
	result, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", dir, 100, false)
	require.NoError(t, err)
	require.Equal(t, "failed", result.State)
	require.Contains(t, result.Assets[0].Message, "place verified asset")
	data, err := os.ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, "keep", string(data))
}

func TestVerifyAssetsReportsUnavailableDownload(t *testing.T) {
	p := failingAssetProvider{assetProviderStub: assetProviderStub{inventory: &AssetInventory{Complete: true, Assets: []Asset{{ID: 1, Name: "app", Size: 6, Digest: assetDigest("binary")}}}}, fail: true}
	result, err := NewAssetService(p).Verify(context.Background(), model.RepositoryRef{}, "v1", t.TempDir(), 100, false)
	require.NoError(t, err)
	require.Equal(t, "failed", result.State)
	require.Equal(t, "unavailable", result.Assets[0].ProviderDigestState)
	require.Contains(t, result.Assets[0].Message, "download interrupted")
}

func TestAssetServiceRejectsMissingProviderAndInvalidLimit(t *testing.T) {
	_, err := NewAssetService(nil).List(context.Background(), model.RepositoryRef{}, "v1", 10)
	require.ErrorContains(t, err, "not configured")
	_, err = NewAssetService(assetProviderStub{}).List(context.Background(), model.RepositoryRef{}, "v1", 0)
	require.ErrorContains(t, err, "limit")
	_, err = NewAssetService(assetProviderStub{}).List(context.Background(), model.RepositoryRef{}, "v1", 10)
	require.ErrorContains(t, err, "no release asset inventory")
}
