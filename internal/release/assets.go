package release

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// Asset is one provider release asset. ID, not name, is its stable identity.
type Asset struct {
	ID          int64  `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Size        int64  `json:"size" yaml:"size"`
	Digest      string `json:"digest" yaml:"digest"`
	ContentType string `json:"content_type,omitempty" yaml:"content_type,omitempty"`
	DownloadURL string `json:"browser_download_url,omitempty" yaml:"browser_download_url,omitempty"`
}

// AssetInventory is a bounded snapshot of one hosted release and its assets.
type AssetInventory struct {
	SchemaVersion string              `json:"schema_version" yaml:"schema_version"`
	Repository    model.RepositoryRef `json:"repository" yaml:"repository"`
	ReleaseID     int64               `json:"release_id" yaml:"release_id"`
	Tag           string              `json:"tag" yaml:"tag"`
	URL           string              `json:"url" yaml:"url"`
	Draft         bool                `json:"draft" yaml:"draft"`
	Prerelease    bool                `json:"prerelease" yaml:"prerelease"`
	Limit         int                 `json:"limit" yaml:"limit"`
	Truncated     bool                `json:"truncated" yaml:"truncated"`
	Complete      bool                `json:"complete" yaml:"complete"`
	Assets        []Asset             `json:"assets" yaml:"assets"`
}

// AssetProvider supplies bounded release metadata and authenticated bytes.
type AssetProvider interface {
	ListAssets(context.Context, model.RepositoryRef, string, int) (*AssetInventory, error)
	DownloadAsset(context.Context, model.RepositoryRef, int64, io.Writer) error
}

// AssetService coordinates release asset inspection and verification.
type AssetService struct{ provider AssetProvider }

// NewAssetService constructs an asset workflow from a provider.
func NewAssetService(provider AssetProvider) *AssetService { return &AssetService{provider: provider} }

// List returns one bounded provider asset inventory.
func (s *AssetService) List(ctx context.Context, repository model.RepositoryRef, tag string, limit int) (*AssetInventory, error) {
	if s == nil || s.provider == nil {
		return nil, fmt.Errorf("release asset provider is not configured")
	}
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("asset limit must be between 1 and 100")
	}
	result, err := s.provider.ListAssets(ctx, repository, tag, limit)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("provider returned no release asset inventory")
	}
	result.SchemaVersion, result.Repository, result.Tag, result.Limit = SchemaVersion, repository, tag, limit
	if result.Assets == nil {
		result.Assets = []Asset{}
	}
	return result, nil
}

// AssetVerification keeps independent provider and optional manifest evidence.
type AssetVerification struct {
	ID                  int64  `json:"id" yaml:"id"`
	Name                string `json:"name" yaml:"name"`
	Destination         string `json:"destination" yaml:"destination"`
	Size                int64  `json:"size" yaml:"size"`
	ActualSize          int64  `json:"actual_size,omitempty" yaml:"actual_size,omitempty"`
	ActualSHA256        string `json:"actual_sha256,omitempty" yaml:"actual_sha256,omitempty"`
	ProviderDigest      string `json:"provider_digest" yaml:"provider_digest"`
	ProviderDigestState string `json:"provider_digest_state" yaml:"provider_digest_state"`
	ManifestState       string `json:"manifest_state" yaml:"manifest_state"`
	Message             string `json:"message,omitempty" yaml:"message,omitempty"`
}

// Verification reports the final state and independent asset evidence.
type Verification struct {
	SchemaVersion     string              `json:"schema_version" yaml:"schema_version"`
	Repository        model.RepositoryRef `json:"repository" yaml:"repository"`
	Tag               string              `json:"tag" yaml:"tag"`
	ReleaseID         int64               `json:"release_id" yaml:"release_id"`
	OutputDir         string              `json:"output_dir" yaml:"output_dir"`
	DryRun            bool                `json:"dry_run" yaml:"dry_run"`
	State             string              `json:"state" yaml:"state"`
	InventoryComplete bool                `json:"inventory_complete" yaml:"inventory_complete"`
	ManifestState     string              `json:"manifest_state" yaml:"manifest_state"`
	Assets            []AssetVerification `json:"assets" yaml:"assets"`
}

type stagedAsset struct {
	path string
	sum  string
	size int64
}

// Verify stages every download in the selected directory. Final names are
// created only when all available verification evidence matches.
func (s *AssetService) Verify(ctx context.Context, repository model.RepositoryRef, tag, outputDir string, limit int, dryRun bool) (*Verification, error) {
	inventory, err := s.List(ctx, repository, tag, limit)
	if err != nil {
		return nil, err
	}
	dir, err := verifyOutputDir(outputDir)
	if err != nil {
		return nil, err
	}
	if !inventory.Complete || inventory.Truncated {
		return &Verification{SchemaVersion: SchemaVersion, Repository: repository, Tag: tag, ReleaseID: inventory.ReleaseID,
			OutputDir: dir, DryRun: dryRun, State: "incomplete", InventoryComplete: false, ManifestState: "unavailable", Assets: []AssetVerification{}}, nil
	}
	if len(inventory.Assets) == 0 {
		return nil, fmt.Errorf("release has no assets to verify")
	}
	result, manifestIndex, err := planVerification(inventory, repository, tag, dir, dryRun)
	if err != nil {
		return nil, err
	}
	if dryRun {
		return result, nil
	}
	staged := make([]stagedAsset, len(inventory.Assets))
	defer func() {
		for _, item := range staged {
			if item.path != "" {
				_ = os.Remove(item.path)
			}
		}
	}()
	s.stageDownloads(ctx, repository, inventory, result, staged)
	if manifestIndex >= 0 {
		checkManifest(inventory, manifestIndex, result, staged)
	}
	if result.State == "verified" {
		placeVerifiedAssets(result, staged)
	}
	return result, nil
}

func planVerification(inventory *AssetInventory, repository model.RepositoryRef, tag, dir string, dryRun bool) (*Verification, int, error) {
	result := &Verification{SchemaVersion: SchemaVersion, Repository: repository, Tag: tag, ReleaseID: inventory.ReleaseID,
		OutputDir: dir, DryRun: dryRun, State: "planned", InventoryComplete: true, ManifestState: "not_present", Assets: make([]AssetVerification, 0, len(inventory.Assets))}
	seen := map[string]bool{}
	manifestIndex := -1
	for i, asset := range inventory.Assets {
		if asset.ID <= 0 || asset.Size < 0 || !safeAssetName(asset.Name) {
			return nil, -1, fmt.Errorf("unsafe release asset identity, size or name %q", asset.Name)
		}
		if seen[asset.Name] {
			return nil, -1, fmt.Errorf("duplicate release asset name %q", asset.Name)
		}
		seen[asset.Name] = true
		if asset.Name == "SHA256SUMS" || asset.Name == "checksums.txt" {
			if manifestIndex >= 0 {
				return nil, -1, fmt.Errorf("multiple checksum manifests are ambiguous")
			}
			manifestIndex, result.ManifestState = i, "present"
		}
		destination := filepath.Join(dir, asset.Name)
		if _, err := os.Lstat(destination); err == nil {
			return nil, -1, fmt.Errorf("destination already exists: %s", destination)
		} else if !os.IsNotExist(err) {
			return nil, -1, err
		}
		result.Assets = append(result.Assets, AssetVerification{ID: asset.ID, Name: asset.Name, Destination: destination, Size: asset.Size,
			ProviderDigest: asset.Digest, ProviderDigestState: "planned", ManifestState: "not_applicable"})
	}
	return result, manifestIndex, nil
}

func (s *AssetService) stageDownloads(ctx context.Context, repository model.RepositoryRef, inventory *AssetInventory, result *Verification, staged []stagedAsset) {
	result.State = "verified"
	for i, asset := range inventory.Assets {
		file, err := os.CreateTemp(result.OutputDir, ".gha-release-*")
		if err != nil {
			result.Assets[i].ProviderDigestState, result.Assets[i].Message, result.State = "unavailable", err.Error(), "failed"
			continue
		}
		staged[i].path = file.Name()
		hash := sha256.New()
		size, downloadErr := downloadAndHash(ctx, s.provider, repository, asset.ID, file, hash)
		staged[i].size, staged[i].sum = size, hex.EncodeToString(hash.Sum(nil))
		item := &result.Assets[i]
		item.ActualSize, item.ActualSHA256 = size, staged[i].sum
		if downloadErr != nil {
			item.ProviderDigestState, item.Message, result.State = "unavailable", downloadErr.Error(), "failed"
			continue
		}
		item.ProviderDigestState = compareDigest(asset.Digest, staged[i].sum)
		if size != asset.Size {
			item.ProviderDigestState, item.Message = "mismatch", "downloaded size differs from provider metadata"
		}
		if item.ProviderDigestState != "verified" {
			result.State = "failed"
		}
	}
}

func checkManifest(inventory *AssetInventory, index int, result *Verification, staged []stagedAsset) {
	if result.Assets[index].ProviderDigestState == "unavailable" {
		result.ManifestState, result.State = "unavailable", "failed"
		for i := range result.Assets {
			if i != index {
				result.Assets[i].ManifestState = "unavailable"
			}
		}
		return
	}
	manifest, err := readManifest(staged[index].path)
	if err == nil {
		err = validateManifestNames(manifest, inventory.Assets, inventory.Assets[index].Name)
	}
	if err != nil {
		result.ManifestState, result.State = "unsafe", "failed"
	} else {
		result.ManifestState = "verified"
	}
	for i := range result.Assets {
		if i == index {
			continue
		}
		item := &result.Assets[i]
		if err != nil {
			item.ManifestState = "unsafe"
			continue
		}
		expected, exists := manifest[item.Name]
		if !exists {
			item.ManifestState = "missing"
		} else if expected == staged[i].sum {
			item.ManifestState = "verified"
		} else {
			item.ManifestState = "mismatch"
		}
		if item.ManifestState != "verified" {
			result.State = "failed"
			if result.ManifestState == "verified" || item.ManifestState == "mismatch" {
				result.ManifestState = item.ManifestState
			}
		}
	}
	if result.Assets[index].ProviderDigestState != "verified" {
		result.ManifestState, result.State = "unverified", "failed"
	}
}

func validateManifestNames(manifest map[string]string, assets []Asset, manifestName string) error {
	validNames := make(map[string]bool, len(assets))
	for _, asset := range assets {
		validNames[asset.Name] = true
	}
	for name := range manifest {
		if !validNames[name] || name == manifestName {
			return fmt.Errorf("manifest names an unknown or unsafe asset %q", name)
		}
	}
	return nil
}

func placeVerifiedAssets(result *Verification, staged []stagedAsset) {
	created := make([]string, 0, len(staged))
	for i := range staged {
		if err := os.Link(staged[i].path, result.Assets[i].Destination); err != nil {
			for _, path := range created {
				_ = os.Remove(path)
			}
			result.State = "failed"
			result.Assets[i].Message = fmt.Sprintf("place verified asset: %v", err)
			return
		}
		created = append(created, result.Assets[i].Destination)
	}
}

func verifyOutputDir(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("--output-dir is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("output directory must be an existing directory, not a symlink: %s", abs)
	}
	return abs, nil
}

func safeAssetName(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name && !strings.ContainsAny(name, "/\\\x00\r\n")
}

func downloadAndHash(ctx context.Context, provider AssetProvider, repository model.RepositoryRef, id int64, file *os.File, hash io.Writer) (int64, error) {
	writer := io.MultiWriter(file, hash)
	counter := &countWriter{writer: writer}
	err := provider.DownloadAsset(ctx, repository, id, counter)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return counter.count, err
}

type countWriter struct {
	writer io.Writer
	count  int64
}

func (w *countWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.count += int64(n)
	return n, err
}

func compareDigest(expected, actual string) string {
	if expected == "" {
		return "missing"
	}
	algorithm, value, ok := strings.Cut(expected, ":")
	if !ok || algorithm != "sha256" || len(value) != 64 {
		return "unsupported"
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return "unsupported"
	}
	if strings.EqualFold(value, actual) {
		return "verified"
	}
	return "mismatch"
}

func readManifest(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	entries, parseErr := parseManifest(file)
	closeErr := file.Close()
	if parseErr != nil {
		return nil, parseErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return entries, nil
}

func parseManifest(file *os.File) (map[string]string, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > 4<<20 {
		return nil, fmt.Errorf("checksum manifest exceeds 4 MiB")
	}
	entries := map[string]string{}
	scanner := bufio.NewScanner(io.LimitReader(file, 4<<20))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 67 || line[64:66] != "  " && line[64:66] != " *" {
			return nil, fmt.Errorf("unsafe checksum manifest entry")
		}
		digest, name := line[:64], line[66:]
		if !safeAssetName(name) || entries[name] != "" {
			return nil, fmt.Errorf("unsafe checksum manifest entry")
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return nil, fmt.Errorf("invalid manifest digest: %w", err)
		}
		entries[name] = strings.ToLower(digest)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
