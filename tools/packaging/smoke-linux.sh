#!/usr/bin/env bash
# Run only inside a disposable root container, never on a developer's host.
set -euo pipefail
manager=$1
packages=$2
test "$(id -u)" = 0
test "${GHA_DISPOSABLE_PACKAGE_TEST:-}" = 1

case "$manager" in
  deb)
    apt-get update -qq
    apt-get install -y git python3
    suffix=deb
    ;;
  rpm)
    dnf install -y git python3
    suffix=rpm
    ;;
  archlinux)
    pacman -Syu --noconfirm git python
    suffix=pkg.tar.zst
    ;;
  *) echo "Unknown package manager: $manager" >&2; exit 1 ;;
esac

mkdir -p "$HOME/.codex" "$HOME/.agents/skills/gha"
printf 'Personal instructions\n' > "$HOME/.codex/AGENTS.md"
printf 'Customized skill\n' > "$HOME/.agents/skills/gha/SKILL.md"
guidance_before=$(sha256sum "$HOME/.codex/AGENTS.md" "$HOME/.agents/skills/gha/SKILL.md")

for version in 0.0.1-test 0.0.2-test; do
  directory="$packages/$version"
  (cd "$directory" && sha256sum --check --ignore-missing checksums.txt)
  # Both architectures ship; use the native x86_64 package on these runners.
  mapfile -t candidates < <(find "$directory" -maxdepth 1 -name "*_amd64.$suffix" | sort)
  test "${#candidates[@]}" = 1
  package=${candidates[0]}
  case "$manager" in
    deb) dpkg -i "$package" ;;
    rpm) rpm -U "$package" ;;
    archlinux) pacman -U --noconfirm "$package" ;;
  esac
  gha version --format json | python3 -c 'import json,sys; assert json.load(sys.stdin)["version"] == sys.argv[1]' "$version"
  gha capabilities --format json | python3 -c 'import json,sys; assert json.load(sys.stdin)["commands"]'
  gha version
  test "$(sha256sum "$HOME/.codex/AGENTS.md" "$HOME/.agents/skills/gha/SKILL.md")" = "$guidance_before"
  test ! -e "$HOME/.config/gha"
  case "$manager" in
    deb) dpkg-query -S /usr/bin/gha ;;
    rpm) rpm -qf /usr/bin/gha ;;
    archlinux) pacman -Qo /usr/bin/gha ;;
  esac
done
