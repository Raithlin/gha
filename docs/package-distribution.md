# Package distribution

GHA's package managers own the executable. Agent setup remains an explicit
`gha agent install` operation. `gha update` refreshes guidance only: it checks
the downloaded skill's required capabilities against the installed binary and
preserves customized skills. Package operations have no agent-setup hooks.

## Homebrew on macOS

```sh
brew install --cask raithlin/tap/gha
gha version --format json

# Later:
brew update
brew upgrade --cask gha
```

The shorthand `raithlin/tap` identifies the public
[Raithlin/homebrew-tap](https://github.com/Raithlin/homebrew-tap) repository.
Homebrew clones it automatically. Its `Casks/gha.rb` selects the Intel or
Apple Silicon release archive and checks a pinned SHA-256 before installing
the `gha` executable. Go is unnecessary for a binary installation; Git is
needed for local repository workflows. Remove it with `brew uninstall --cask gha`.

The tap starts with v0.3.0-alpha, whose published archive bytes were verified
against `checksums.txt`. Prereleases update this same cask while GHA is in alpha;
this is not a separate stable channel. Linux users should use native packages
below: Homebrew casks are the macOS distribution path.

These binaries are not Apple Developer ID signed or notarized. SHA-256 checks
verify the expected release bytes; they do not establish notarization or an
independent signature. macOS security policy may prevent execution. Do not
disable Gatekeeper as an installation step. The native Mac CI reports what it
actually installs and runs, rather than treating a successful download as proof.

### How releases maintain the tap

[GoReleaser configuration](../.goreleaser.yaml) builds the archives and native
packages, writes `checksums.txt`, and generates the cask from the same archives.
After publishing a release, GoReleaser commits the cask to the tap. `skip_upload`
is explicitly false so alpha releases also update it. GoReleaser is pinned to
2.19.0 in release and packaging validation workflows. No install or update hook
writes agent guidance.

The release repository's built-in `GITHUB_TOKEN` can publish GHA releases but
cannot write to another repository. The one-time maintainer setup is:

1. Create a fine-grained GitHub token owned by the account maintaining the tap.
   Select **only `Raithlin/homebrew-tap`**, grant **Contents: Read and write**
   (Metadata read is implicit), and choose an expiration/rotation schedule.
2. In **Raithlin/gha → Settings → Secrets and variables → Actions**, add a
   repository secret named **`HOMEBREW_TAP_GITHUB_TOKEN`**. Enter the token in
   GitHub's secret form, never in a task, PR, shell argument, or chat.
3. Before tagging, verify that the secret exists and that its repository access
   is still valid. The release job checks authenticated tap push permission
   before building or publishing anything. Expired or absent credentials stop
   the job with an actionable error.

For a read-only credential check, manually run the **Release** workflow in
GitHub Actions, or use `gh workflow run release.yml --repo Raithlin/gha`.
Manual dispatch runs only the tap-access check; the publication job requires
a tag push and is skipped. Select a branch explicitly with `--ref` when testing
an unmerged workflow change. The check never prints the token.

Do not reuse a broad account token. GitHub App installation tokens are a future
alternative if centralized credential rotation becomes useful. GoReleaser's
[Homebrew documentation](https://goreleaser.com/customization/publish/homebrew_casks/)
describes the cross-repository token requirement.

If release assets publish but a subsequent tap commit fails, the GitHub Release
and tap are separate observed outcomes. Fix tap access first. Recover the cask
from the exact release build, verify its URLs and hashes against the published
`checksums.txt`, and commit it to the tap; do not rebuild and substitute different
archive hashes. Verify the tap's Mac install/upgrade workflow before claiming
the package-manager rollout succeeded. A tap commit made through a personal
access token triggers that workflow.

## Linux choice

Use native `.deb`, `.rpm`, and Arch `.pkg.tar.zst` packages built by
[GoReleaser/nFPM](https://goreleaser.com/customization/package/nfpm/). They install
`/usr/bin/gha`, declare Git as a dependency, and contain no agent files or package
scripts. Both amd64 and arm64 release artifacts are built. The first release
containing this configuration will introduce these packages; older releases
offer archives only.

| Option | Maintenance and workflow fit | Decision |
| --- | --- | --- |
| Native release packages | Reuse existing release builds and checksums; native tools own files and upgrades. Users download each new version explicitly. | Selected now. |
| Hosted apt/rpm repositories | Enable routine repository upgrades, but add repository signing, key rotation, hosting, and metadata maintenance. | Defer until demand warrants an operated repository. |
| AUR `gha-bin` | Fits Arch discovery and helper upgrades, but adds a separate package definition and submission/maintenance workflow. | Defer; provide native pacman-installable release assets now. |
| Homebrew on Linux | Requires an additional package manager; the cask delivery path is for macOS. | Prefer the distribution's existing tools. |
| Nix, Snap, Flatpak | Additional ecosystem definitions and update channels; sandboxed desktop formats add little to this host CLI. | Defer. |

There is no configured apt/dnf/pacman repository, so `apt upgrade`, `dnf upgrade`,
or `pacman -Syu` alone will not discover new GHA releases. Download the matching
new package and run the same local-package install command to upgrade. All
executable changes remain recorded by that package manager.

Download the package and `checksums.txt` from the **same release** on the
[release page](https://github.com/Raithlin/gha/releases). Select `amd64` for
x86_64 or `arm64` for aarch64. For example, replacing the version and asset below
with the actual release you selected:

```sh
sha256sum --check --ignore-missing checksums.txt

# Debian/Ubuntu (installs dependencies too):
sudo apt install ./gha_VERSION_linux_amd64.deb

# Fedora:
sudo dnf install ./gha_VERSION_linux_amd64.rpm

# Arch/EndeavourOS (Git must already be installed):
sudo pacman -U ./gha_VERSION_linux_amd64.pkg.tar.zst

gha version --format json
```

Require an `OK` result for your selected package and a zero exit code before
installing it. These release packages are unsigned; the manifest from the same
GitHub release detects mismatched or damaged bytes, not compromise of that
release channel. No signing-key trust is implied.

Remove the executable with `apt remove gha`, `dnf remove gha`, or `pacman -R gha`.
Agent files remain under their recorded GHA ownership; use `gha agent uninstall`
before removing the binary if you also want to remove its managed guidance.

## Validation

The [package workflow](../.github/workflows/packages.yml) builds two distinct
versions without publishing. Debian, Fedora and Arch containers install the
first package, upgrade to the second, execute version/capabilities commands,
and verify native file ownership and unchanged personal guidance. Mac runners
on Intel and Apple Silicon install and upgrade the generated casks using a
loopback HTTP fixture server and the original archive hashes. They first verify
that Homebrew rejects a deliberately wrong digest. No quarantine bypass is used.

The tap has its own Mac workflow that installs the previous published release
fixture and upgrades to the current live cask. This checks real release downloads
as well as generated snapshot fixtures. The Go suite separately covers skill
capability rejection, preservation of modified skills, and `binary_updated: false`.

Local checks:

```sh
go test ./tools/packaging
python3 -B -m unittest discover -s tools/packaging -p 'test_*.py'
goreleaser check
go test ./internal/commands -run 'TestUpdate|TestAgentInstall'
make check
```

The native smoke scripts refuse to run outside explicitly marked disposable
test environments. Run package installation on the native CI runners instead
of installing test versions onto a development host.
