package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
)

const assetDownloadTimeout = 30 * time.Minute

// ListAssets reads one hosted release and at most limit assets. A sentinel
// request distinguishes an exact limit from a truncated inventory.
func (c *GitHubClient) ListAssets(ctx context.Context, repository model.RepositoryRef, tag string, limit int) (*release.AssetInventory, error) {
	base := fmt.Sprintf("repos/%s/%s", repository.Owner, repository.Name)
	var raw struct {
		ID         int64  `json:"id"`
		Tag        string `json:"tag_name"`
		URL        string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	found, err := c.releaseStatusGet(ctx, base+"/releases/tags/"+url.PathEscape(tag), &raw)
	if err != nil {
		return nil, fmt.Errorf("read release %s: %w", tag, err)
	}
	if !found {
		return nil, fmt.Errorf("release %s is absent", tag)
	}
	if raw.ID <= 0 || raw.Tag != tag {
		return nil, fmt.Errorf("release identity is unavailable or does not match %s", tag)
	}
	result := &release.AssetInventory{ReleaseID: raw.ID, Tag: tag, URL: raw.URL, Draft: raw.Draft, Prerelease: raw.Prerelease,
		Limit: limit, Complete: true, Assets: []release.Asset{}}
	endpoint := fmt.Sprintf("%s/releases/%d/assets", base, raw.ID)
	pageSize := limit + 1
	if pageSize > 100 {
		pageSize = 100
	}
	assets, err := c.assetPage(ctx, endpoint, 1, pageSize)
	if err != nil {
		return nil, err
	}
	result.Assets = assets
	if len(result.Assets) > limit {
		result.Assets = result.Assets[:limit]
		result.Truncated, result.Complete = true, false
		return result, nil
	}
	if len(result.Assets) == limit {
		next, err := c.assetPage(ctx, endpoint, 2, pageSize)
		if err != nil {
			return nil, fmt.Errorf("cannot determine asset inventory completeness: %w", err)
		}
		result.Truncated = len(next) > 0
		result.Complete = !result.Truncated
	}
	return result, nil
}

func (c *GitHubClient) assetPage(ctx context.Context, endpoint string, page, perPage int) ([]release.Asset, error) {
	query := url.Values{"page": {fmt.Sprint(page)}, "per_page": {fmt.Sprint(perPage)}}
	var assets []release.Asset
	found, err := c.releaseStatusGet(ctx, endpoint+"?"+query.Encode(), &assets)
	if err != nil {
		return nil, fmt.Errorf("list release assets: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("release asset inventory is unavailable")
	}
	return assets, nil
}

// DownloadAsset streams authenticated provider bytes to the supplied writer.
func (c *GitHubClient) DownloadAsset(ctx context.Context, repository model.RepositoryRef, id int64, out io.Writer) (returnErr error) {
	endpoint := fmt.Sprintf("repos/%s/%s/releases/assets/%d", repository.Owner, repository.Name, id)
	request, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/octet-stream")
	// The shared client's API timeout covers the response body as well. Give
	// streaming downloads their own deadline without changing API requests.
	downloadClient := *c.HTTPClient
	if downloadClient.Timeout != 0 && downloadClient.Timeout < assetDownloadTimeout {
		downloadClient.Timeout = assetDownloadTimeout
	}
	response, err := downloadClient.Do(request)
	if err != nil {
		return fmt.Errorf("download release asset %d: %w", id, err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil && returnErr == nil {
			returnErr = err
		}
	}()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download release asset %d: provider returned %s", id, response.Status)
	}
	if response.Header.Get("Content-Type") == "application/json" {
		return fmt.Errorf("download release asset %d: provider returned metadata instead of bytes", id)
	}
	_, err = io.Copy(out, response.Body)
	return err
}

var _ release.AssetProvider = (*GitHubClient)(nil)
