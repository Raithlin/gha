package github

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestReleaseAssetInventoryIsBoundedAndComplete(t *testing.T) {
	calls := 0
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/tool/releases/tags/v1":
			_, _ = io.WriteString(w, `{"id":9,"tag_name":"v1","draft":true,"prerelease":true,"html_url":"https://example.test/v1"}`)
		case "/repos/acme/tool/releases/9/assets":
			calls++
			require.Equal(t, "3", r.URL.Query().Get("per_page"))
			_, _ = io.WriteString(w, fmt.Sprintf(`[{"id":%d,"name":"asset-%d","size":6,"digest":"sha256:abc"}]`, calls, calls))
		default:
			t.Fatalf("unexpected %s", r.URL.String())
		}
	}))
	defer closeServer()
	inventory, err := client.ListAssets(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1", 2)
	require.NoError(t, err)
	require.True(t, inventory.Complete)
	require.False(t, inventory.Truncated)
	require.True(t, inventory.Draft)
	require.True(t, inventory.Prerelease)
	require.Equal(t, int64(9), inventory.ReleaseID)
	require.Len(t, inventory.Assets, 1)
	require.Equal(t, 1, calls)
}

func TestReleaseAssetInventoryReportsTruncation(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/acme/tool/releases/tags/v1" {
			_, _ = io.WriteString(w, `{"id":9,"tag_name":"v1"}`)
			return
		}
		_, _ = io.WriteString(w, `[{"id":1,"name":"one"},{"id":2,"name":"two"}]`)
	}))
	defer closeServer()
	inventory, err := client.ListAssets(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1", 1)
	require.NoError(t, err)
	require.True(t, inventory.Truncated)
	require.False(t, inventory.Complete)
	require.Len(t, inventory.Assets, 1)
}

func TestDownloadReleaseAssetUsesAuthenticatedAPIBytes(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/acme/tool/releases/assets/11", r.URL.Path)
		require.Equal(t, "application/octet-stream", r.Header.Get("Accept"))
		require.Equal(t, "token test-token", r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, "binary")
	}))
	defer closeServer()
	var out bytes.Buffer
	require.NoError(t, client.DownloadAsset(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, 11, &out))
	require.Equal(t, "binary", out.String())
}

func TestReleaseAssetInventoryReportsProviderErrors(t *testing.T) {
	for _, tc := range []struct {
		name, releaseBody, assetBody string
		releaseStatus, assetStatus   int
	}{
		{name: "missing release", releaseStatus: http.StatusNotFound},
		{name: "release API failure", releaseStatus: http.StatusInternalServerError},
		{name: "missing identity", releaseBody: `{"tag_name":"v1"}`},
		{name: "wrong tag", releaseBody: `{"id":9,"tag_name":"v2"}`},
		{name: "asset API failure", releaseBody: `{"id":9,"tag_name":"v1"}`, assetStatus: http.StatusInternalServerError},
		{name: "missing asset endpoint", releaseBody: `{"id":9,"tag_name":"v1"}`, assetStatus: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repos/acme/tool/releases/tags/v1" {
					if tc.releaseStatus != 0 {
						w.WriteHeader(tc.releaseStatus)
					}
					_, _ = io.WriteString(w, tc.releaseBody)
					return
				}
				if tc.assetStatus != 0 {
					w.WriteHeader(tc.assetStatus)
				}
				_, _ = io.WriteString(w, tc.assetBody)
			}))
			defer closeServer()
			_, err := client.ListAssets(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1", 2)
			require.Error(t, err)
		})
	}
}

func TestReleaseAssetInventoryChecksNextPageAtLimit(t *testing.T) {
	for _, next := range []string{`[]`, `[{"id":3,"name":"third"}]`} {
		client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/repos/acme/tool/releases/tags/v1" {
				_, _ = io.WriteString(w, `{"id":9,"tag_name":"v1"}`)
				return
			}
			require.Equal(t, "3", r.URL.Query().Get("per_page"))
			if r.URL.Query().Get("page") == "2" {
				_, _ = io.WriteString(w, next)
				return
			}
			_, _ = io.WriteString(w, `[{"id":1,"name":"first"},{"id":2,"name":"second"}]`)
		}))
		inventory, err := client.ListAssets(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1", 2)
		closeServer()
		require.NoError(t, err)
		require.Equal(t, next != `[]`, inventory.Truncated)
		require.Equal(t, !inventory.Truncated, inventory.Complete)
	}
}

func TestReleaseAssetInventoryCannotClaimCompletenessWhenSentinelFails(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/acme/tool/releases/tags/v1" {
			_, _ = io.WriteString(w, `{"id":9,"tag_name":"v1"}`)
			return
		}
		if r.URL.Query().Get("page") == "2" {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = io.WriteString(w, `[{"id":1,"name":"first"}]`)
	}))
	defer closeServer()
	_, err := client.ListAssets(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1", 1)
	require.ErrorContains(t, err, "completeness")
}

func TestDownloadReleaseAssetRejectsProviderFailureAndMetadata(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      int
		contentType string
	}{
		{"not found", http.StatusNotFound, ""},
		{"metadata", http.StatusOK, "application/json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.contentType != "" {
					w.Header().Set("Content-Type", tc.contentType)
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, `{}`)
			}))
			defer closeServer()
			var out bytes.Buffer
			require.Error(t, client.DownloadAsset(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, 11, &out))
		})
	}
}

func TestReleaseAssetInventoryLimit100UsesSamePageSizeForSentinel(t *testing.T) {
	assets := make([]string, 100)
	for i := range assets {
		assets[i] = fmt.Sprintf(`{"id":%d,"name":"asset-%d"}`, i+1, i+1)
	}
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/acme/tool/releases/tags/v1" {
			_, _ = io.WriteString(w, `{"id":9,"tag_name":"v1"}`)
			return
		}
		require.Equal(t, "100", r.URL.Query().Get("per_page"))
		if r.URL.Query().Get("page") == "2" {
			_, _ = io.WriteString(w, `[]`)
			return
		}
		_, _ = io.WriteString(w, "["+strings.Join(assets, ",")+"]")
	}))
	defer closeServer()
	inventory, err := client.ListAssets(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1", 100)
	require.NoError(t, err)
	require.True(t, inventory.Complete)
	require.Len(t, inventory.Assets, 100)
}

func TestDownloadReleaseAssetReportsTransportAndWriterErrors(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "binary") }))
	defer closeServer()
	require.Error(t, client.DownloadAsset(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, 11, failedAssetWriter{}))
	client.HTTPClient = &http.Client{Transport: failingTransport{}}
	require.Error(t, client.DownloadAsset(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, 11, &bytes.Buffer{}))
}

type failedAssetWriter struct{}

func (failedAssetWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("disk full") }
