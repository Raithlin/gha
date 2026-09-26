package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	outputfmt "github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type updateRoundTripper func(*http.Request) (*http.Response, error)

func (f updateRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) { return 0, fmt.Errorf("read failed") }
func (failingReadCloser) Close() error             { return nil }

type failingCloseBody struct{ *strings.Reader }

func (failingCloseBody) Close() error { return fmt.Errorf("close failed") }

func TestUpdateRefreshesRecordedFilesAndPreservesPersonalInstructions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	home := t.TempDir()
	skill := home + "/skills/gha/SKILL.md"
	instructions := home + "/AGENTS.md"
	target := agentInstallation{ID: "codex", Name: "Codex", SkillPath: skill, InstructionsPath: instructions}
	require.NoError(t, recordAgentInstallation(target))
	require.NoError(t, os.MkdirAll(filepath.Dir(instructions), 0o755))
	require.NoError(t, os.WriteFile(instructions, []byte("# Personal\n\n<!-- gha:begin -->\nold\n<!-- gha:end -->\n"), 0o644))
	client := &http.Client{Transport: updateRoundTripper(func(request *http.Request) (*http.Response, error) {
		body := `[{"tag_name":"v9.0.0","published_at":"2026-09-26T00:00:00Z"}]`
		if strings.Contains(request.URL.Host, "raw.githubusercontent.com") {
			if strings.HasSuffix(request.URL.Path, "SKILL.md") {
				body = "name: gha\nversion: new\n"
			} else {
				body = "<!-- gha:begin -->\nnew guidance\n<!-- gha:end -->\n"
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	var output strings.Builder
	require.NoError(t, runGuidanceUpdate(context.Background(), &output, client, false, outputfmt.Text))
	updatedSkill, err := os.ReadFile(skill)
	require.NoError(t, err)
	assert.Equal(t, "name: gha\nversion: new\n", string(updatedSkill))
	updatedInstructions, err := os.ReadFile(instructions)
	require.NoError(t, err)
	assert.Contains(t, string(updatedInstructions), "# Personal")
	assert.Contains(t, string(updatedInstructions), "new guidance")
	assert.NotContains(t, string(updatedInstructions), "old")
	assert.Contains(t, output.String(), "v9.0.0")
}

func TestUpdateCreatesMissingManagedInstructions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	target := agentInstallation{ID: "codex", Name: "Codex", SkillPath: filepath.Join(root, "skills", "gha", "SKILL.md"), InstructionsPath: filepath.Join(root, "AGENTS.md")}
	require.NoError(t, recordAgentInstallation(target))
	client := &http.Client{Transport: updateRoundTripper(func(request *http.Request) (*http.Response, error) {
		body := `[{"tag_name":"v1","published_at":"2026-09-26T00:00:00Z"}]`
		if strings.Contains(request.URL.Host, "raw.githubusercontent.com") {
			body = "name: gha\n"
			if strings.HasSuffix(request.URL.Path, "AGENT-GUIDANCE.md") {
				body = "<!-- gha:begin -->\nmanaged guidance\n<!-- gha:end -->"
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	require.NoError(t, runGuidanceUpdate(context.Background(), io.Discard, client, false, outputfmt.JSON))
	guidance, err := os.ReadFile(target.InstructionsPath)
	require.NoError(t, err)
	assert.Contains(t, string(guidance), "managed guidance")
}

func TestUpdateRequiresExplicitConfirmation(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"update"})
	err := root.Execute()
	require.ErrorContains(t, err, "requires --confirm")
}

func TestUpdateCommandDryRunWithNoInstallations(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"update", "--dry-run"})
	var output strings.Builder
	root.SetOut(&output)
	require.NoError(t, root.Execute())
	assert.Contains(t, output.String(), "nothing to update")
}

func TestUpdateCommandRendersStructuredResult(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"update", "--dry-run", "--format", "json"})
	var rendered strings.Builder
	root.SetOut(&rendered)
	require.NoError(t, root.Execute())
	var result model.GuidanceUpdate
	require.NoError(t, json.Unmarshal([]byte(rendered.String()), &result))
	assert.Equal(t, model.GuidanceUpdateSchemaVersion, result.SchemaVersion)
	assert.True(t, result.DryRun)
	assert.False(t, result.BinaryUpdated)
}

func TestUpdateCommandRejectsUnsupportedFormat(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"update", "--confirm", "--format", "xml"})
	err := root.Execute()
	require.ErrorContains(t, err, "unsupported format")
}

func TestUpdateReportsNoRecordedInstallations(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var output strings.Builder
	require.NoError(t, runGuidanceUpdate(context.Background(), &output, http.DefaultClient, false, outputfmt.Text))
	assert.Contains(t, output.String(), "No GHA-configured coding agents found; nothing to update.")
}

func TestFetchLatestReleaseRejectsBadStatus(t *testing.T) {
	client := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Status: "503 unavailable", Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	_, err := fetchLatestRelease(context.Background(), client)
	require.Error(t, err)
	assert.Equal(t, "GitHub returned 503 unavailable", err.Error())
}

func TestFetchLatestReleaseRejectsMalformedAndEmptyResponses(t *testing.T) {
	for _, response := range []string{"not-json", `[{"tag_name":" ","published_at":"2026-09-26T00:00:00Z"}]`, `[{"tag_name":"v1","draft":true,"published_at":"2026-09-26T00:00:00Z"}]`, `[{"tag_name":"v1"}]`} {
		client := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(response))}, nil
		})}
		_, err := fetchLatestRelease(context.Background(), client)
		require.Error(t, err)
	}
}

func TestFetchLatestReleaseReportsTransportFailure(t *testing.T) {
	client := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("offline") })}
	_, err := fetchLatestRelease(context.Background(), client)
	require.ErrorContains(t, err, "offline")
}

func TestFetchLatestReleaseSelectsPublishedPrereleaseAfterDraft(t *testing.T) {
	client := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		body := `[{"tag_name":"v1-alpha","prerelease":true,"published_at":"2026-09-25T02:00:00Z"},{"tag_name":"v2","draft":true,"published_at":"2026-09-26T02:00:00Z"},{"tag_name":"v0.9-alpha","prerelease":true,"published_at":"2026-09-24T02:00:00Z"}]`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	version, err := fetchLatestRelease(context.Background(), client)
	require.NoError(t, err)
	assert.Equal(t, "v1-alpha", version)
}

func TestFetchLatestReleaseReportsEmptyReleaseList(t *testing.T) {
	client := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("[]"))}, nil
	})}
	_, err := fetchLatestRelease(context.Background(), client)
	require.ErrorContains(t, err, "no published release tag")
}

func TestFetchLatestReleaseChecksResponseClose(t *testing.T) {
	client := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		body := failingCloseBody{strings.NewReader(`[{"tag_name":"v1","published_at":"2026-09-26T00:00:00Z"}]`)}
		return &http.Response{StatusCode: 200, Body: body}, nil
	})}
	_, err := fetchLatestRelease(context.Background(), client)
	require.ErrorContains(t, err, "close release response")
}

func TestFetchLatestReleasePrioritizesCloseFailureOnHTTPError(t *testing.T) {
	client := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 404, Status: "404 Not Found", Body: failingCloseBody{strings.NewReader("")}}, nil
	})}
	_, err := fetchLatestRelease(context.Background(), client)
	require.ErrorContains(t, err, "close release response")
}

func TestFetchReleaseGuidanceRejectsInvalidPayload(t *testing.T) {
	client := &http.Client{Transport: updateRoundTripper(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("bad"))}, nil
	})}
	_, _, err := fetchReleaseGuidance(context.Background(), client, "v1")
	require.ErrorContains(t, err, "invalid")
}

func TestFetchReleaseGuidanceReportsSecondFileFailure(t *testing.T) {
	client := &http.Client{Transport: updateRoundTripper(func(request *http.Request) (*http.Response, error) {
		if strings.HasSuffix(request.URL.Path, "SKILL.md") {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("name: gha\n"))}, nil
		}
		return &http.Response{StatusCode: 502, Status: "502 unavailable", Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	_, _, err := fetchReleaseGuidance(context.Background(), client, "v1")
	require.ErrorContains(t, err, "502 unavailable")
}

func TestFetchGuidanceFileReportsHTTPAndTransportErrors(t *testing.T) {
	statusClient := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 404, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	_, err := fetchGuidanceFile(context.Background(), statusClient, "https://example.test/SKILL.md")
	require.ErrorContains(t, err, "404 Not Found")
	statusCloseClient := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 404, Status: "404 Not Found", Body: failingCloseBody{strings.NewReader("")}}, nil
	})}
	_, err = fetchGuidanceFile(context.Background(), statusCloseClient, "https://example.test/SKILL.md")
	require.ErrorContains(t, err, "close guidance response")
	transportClient := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("offline")
	})}
	_, err = fetchGuidanceFile(context.Background(), transportClient, "https://example.test/SKILL.md")
	require.ErrorContains(t, err, "offline")
	invalidClient := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("not a skill"))}, nil
	})}
	_, err = fetchGuidanceFile(context.Background(), invalidClient, "https://example.test/SKILL.md")
	require.ErrorContains(t, err, "invalid")
	emptyClient := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	_, err = fetchGuidanceFile(context.Background(), emptyClient, "https://example.test/AGENT-GUIDANCE.md")
	require.ErrorContains(t, err, "empty or invalid")
	badGuidanceClient := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("text without managed markers"))}, nil
	})}
	_, err = fetchGuidanceFile(context.Background(), badGuidanceClient, "https://example.test/AGENT-GUIDANCE.md")
	require.ErrorContains(t, err, "empty or invalid")
	readFailureClient := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: failingReadCloser{}}, nil
	})}
	_, err = fetchGuidanceFile(context.Background(), readFailureClient, "https://example.test/SKILL.md")
	require.ErrorContains(t, err, "read failed")
	closeFailureClient := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: failingCloseBody{strings.NewReader("name: gha\n")}}, nil
	})}
	_, err = fetchGuidanceFile(context.Background(), closeFailureClient, "https://example.test/SKILL.md")
	require.ErrorContains(t, err, "close failed")
}

func TestUpdateDryRunDoesNotWriteRecordedFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	skill := filepath.Join(root, "skill", "SKILL.md")
	require.NoError(t, recordAgentInstallation(agentInstallation{ID: "copilot", Name: "Copilot", SkillPath: skill}))
	client := &http.Client{Transport: updateRoundTripper(func(request *http.Request) (*http.Response, error) {
		body := `[{"tag_name":"v1","published_at":"2026-09-26T00:00:00Z"}]`
		if strings.Contains(request.URL.Host, "raw.githubusercontent.com") {
			body = "name: gha\n"
			if strings.HasSuffix(request.URL.Path, "AGENT-GUIDANCE.md") {
				body = "<!-- gha:begin -->\ncontent\n<!-- gha:end -->"
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	var output strings.Builder
	require.NoError(t, runGuidanceUpdate(context.Background(), &output, client, true, outputfmt.Text))
	_, err := os.Stat(skill)
	assert.True(t, os.IsNotExist(err))
	assert.Contains(t, output.String(), "Copilot: planned skill")
}

func TestUpdateReportsReleaseAndDestinationFailures(t *testing.T) {
	t.Run("release lookup", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		require.NoError(t, recordAgentInstallation(agentInstallation{ID: "copilot", Name: "Copilot", SkillPath: filepath.Join(t.TempDir(), "skill")}))
		client := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Status: "500 failure", Body: io.NopCloser(strings.NewReader(""))}, nil
		})}
		var rendered strings.Builder
		err := runGuidanceUpdate(context.Background(), &rendered, client, true, outputfmt.JSON)
		require.ErrorContains(t, err, "discover latest published")
		var result model.GuidanceUpdate
		require.NoError(t, json.Unmarshal([]byte(rendered.String()), &result))
		assert.Equal(t, "unavailable", result.SourceState)
		require.Len(t, result.Targets, 1)
		assert.Equal(t, "unavailable", result.Targets[0].State)
	})
	t.Run("instruction read failure", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		root := t.TempDir()
		instructionDirectory := filepath.Join(root, "AGENTS.md")
		require.NoError(t, os.Mkdir(instructionDirectory, 0o755))
		require.NoError(t, recordAgentInstallation(agentInstallation{ID: "codex", Name: "Codex", SkillPath: filepath.Join(root, "skill"), InstructionsPath: instructionDirectory}))
		client := &http.Client{Transport: updateRoundTripper(func(request *http.Request) (*http.Response, error) {
			body := `[{"tag_name":"v1","published_at":"2026-09-26T00:00:00Z"}]`
			if strings.Contains(request.URL.Host, "raw.githubusercontent.com") {
				body = "name: gha\n"
				if strings.HasSuffix(request.URL.Path, "AGENT-GUIDANCE.md") {
					body = "<!-- gha:begin -->\nnew\n<!-- gha:end -->"
				}
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
		err := runGuidanceUpdate(context.Background(), io.Discard, client, false, outputfmt.Text)
		require.ErrorContains(t, err, "read Codex guidance")
	})
	t.Run("malformed managed block", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		root := t.TempDir()
		instructions := filepath.Join(root, "AGENTS.md")
		require.NoError(t, os.WriteFile(instructions, []byte("personal\n<!-- gha:begin -->"), 0o644))
		require.NoError(t, recordAgentInstallation(agentInstallation{ID: "codex", Name: "Codex", SkillPath: filepath.Join(root, "skill"), InstructionsPath: instructions}))
		client := &http.Client{Transport: updateRoundTripper(func(request *http.Request) (*http.Response, error) {
			body := `[{"tag_name":"v1","published_at":"2026-09-26T00:00:00Z"}]`
			if strings.Contains(request.URL.Host, "raw.githubusercontent.com") {
				body = "name: gha\n"
				if strings.HasSuffix(request.URL.Path, "AGENT-GUIDANCE.md") {
					body = "<!-- gha:begin -->\nnew\n<!-- gha:end -->"
				}
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
		err := runGuidanceUpdate(context.Background(), io.Discard, client, false, outputfmt.Text)
		require.ErrorContains(t, err, "without <!-- gha:end -->")
	})
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write failed") }

func TestUpdatePropagatesOutputFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, recordAgentInstallation(agentInstallation{ID: "copilot", Name: "Copilot", SkillPath: filepath.Join(t.TempDir(), "skill")}))
	client := &http.Client{Transport: updateRoundTripper(func(request *http.Request) (*http.Response, error) {
		body := `[{"tag_name":"v1","published_at":"2026-09-26T00:00:00Z"}]`
		if strings.Contains(request.URL.Host, "raw.githubusercontent.com") {
			body = "name: gha\n"
			if strings.HasSuffix(request.URL.Path, "AGENT-GUIDANCE.md") {
				body = "<!-- gha:begin -->\nnew\n<!-- gha:end -->"
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	err := runGuidanceUpdate(context.Background(), failingWriter{}, client, true, outputfmt.Text)
	require.ErrorContains(t, err, "write failed")
}

func TestUpdateReportsOutputFailureWhenSourceIsUnavailable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, recordAgentInstallation(agentInstallation{ID: "copilot", Name: "Copilot", SkillPath: filepath.Join(t.TempDir(), "skill")}))
	client := &http.Client{Transport: updateRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Status: "503 unavailable", Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	err := runGuidanceUpdate(context.Background(), failingWriter{}, client, true, outputfmt.JSON)
	require.ErrorContains(t, err, "write failed")
}

func TestUpdateReportsUnavailableGuidanceFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, recordAgentInstallation(agentInstallation{ID: "copilot", Name: "Copilot", SkillPath: filepath.Join(t.TempDir(), "skill")}))
	client := &http.Client{Transport: updateRoundTripper(func(request *http.Request) (*http.Response, error) {
		if strings.Contains(request.URL.Host, "raw.githubusercontent.com") {
			return &http.Response{StatusCode: 404, Status: "404 missing", Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"tag_name":"v1","published_at":"2026-09-26T00:00:00Z"}]`))}, nil
	})}
	var rendered strings.Builder
	err := runGuidanceUpdate(context.Background(), &rendered, client, true, outputfmt.JSON)
	require.ErrorContains(t, err, "fetch GHA guidance for release")
	var result model.GuidanceUpdate
	require.NoError(t, json.Unmarshal([]byte(rendered.String()), &result))
	assert.Equal(t, "v1", result.LatestVersion)
	assert.Equal(t, "unavailable", result.SourceState)
	assert.Equal(t, "unavailable", result.Targets[0].State)
}

func TestUpdateReportsSkillWriteFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	blockedParent := filepath.Join(root, "not-a-directory")
	require.NoError(t, os.WriteFile(blockedParent, []byte("file"), 0o644))
	require.NoError(t, recordAgentInstallation(agentInstallation{ID: "copilot", Name: "Copilot", SkillPath: filepath.Join(blockedParent, "SKILL.md")}))
	client := &http.Client{Transport: updateRoundTripper(func(request *http.Request) (*http.Response, error) {
		body := `[{"tag_name":"v1","published_at":"2026-09-26T00:00:00Z"}]`
		if strings.Contains(request.URL.Host, "raw.githubusercontent.com") {
			body = "name: gha\n"
			if strings.HasSuffix(request.URL.Path, "AGENT-GUIDANCE.md") {
				body = "<!-- gha:begin -->\nnew\n<!-- gha:end -->"
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	err := runGuidanceUpdate(context.Background(), io.Discard, client, false, outputfmt.Text)
	require.ErrorContains(t, err, "update skill for Copilot")
}
