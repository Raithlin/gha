---
type: C4 System
title: GitHub CLI authentication
status: stable
groma:
  id: github-cli-authentication
  technology: gh auth token subprocess
---

Optional installed GitHub CLI used only to obtain an existing authenticated token when GHA_GITHUB_TOKEN is empty. Ordinary provider calls use GHA HTTP adapters rather than routing through gh.
