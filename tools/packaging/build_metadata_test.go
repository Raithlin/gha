package packaging

import (
	"debug/buildinfo"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReleaseOutputPreservesVCSMetadata(t *testing.T) {
	repository := t.TempDir()
	ignore, err := os.ReadFile("../../.gitignore")
	require.NoError(t, err)
	write := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(repository, name), []byte(content), 0o600))
	}
	run := func(command string, arguments ...string) string {
		cmd := exec.Command(command, arguments...)
		cmd.Dir = repository
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s %v: %s", command, arguments, output)
		return string(output)
	}
	write(".gitignore", string(ignore))
	write("go.mod", "module example.com/release-metadata\n\ngo 1.23\n")
	const source = "package main\n\nfunc main() {}\n"
	write("main.go", source)
	run("git", "init", "--quiet")
	run("git", "add", ".")
	run("git", "-c", "user.name=Packaging Test", "-c", "user.email=packaging@example.com", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "Initial source")
	revision := run("git", "rev-parse", "HEAD")
	require.NoError(t, os.Mkdir(filepath.Join(repository, "dist"), 0o700))
	write("dist/metadata.json", "{}\n")

	assertModified := func(t *testing.T, expected string) {
		t.Helper()
		run("go", "build", "-o", "dist/probe", ".")
		info, err := buildinfo.ReadFile(filepath.Join(repository, "dist", "probe"))
		require.NoError(t, err)
		settings := make(map[string]string)
		for _, setting := range info.Settings {
			settings[setting.Key] = setting.Value
		}
		require.Equal(t, revision, settings["vcs.revision"]+"\n")
		require.Equal(t, expected, settings["vcs.modified"])
	}
	t.Run("generated release output", func(t *testing.T) {
		assertModified(t, "false")
	})
	t.Run("tracked source edit", func(t *testing.T) {
		write("main.go", source+"// Source changed.\n")
		assertModified(t, "true")
		write("main.go", source)
	})
	t.Run("untracked nested dist source", func(t *testing.T) {
		require.NoError(t, os.MkdirAll(filepath.Join(repository, "internal", "dist"), 0o700))
		write("internal/dist/source.go", "package dist\n")
		assertModified(t, "true")
	})
}
