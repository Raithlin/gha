---
type: C4 Container
title: GHA CLI
status: stable
groma:
  id: gha-cli
  parent: gha
  technology: Go 1.26.5; Cobra; Git subprocesses; net/http; JSON and YAML
description: Runs repository workflows and emits stable structured results or readable terminal output.
---

A single Go executable constructed explicitly in cmd/gha/main.go loads configuration, creates the GitHub adapter and workflow services, and executes a fresh Cobra command tree. Commands select --repo and --path targets, validate inputs, coordinate services or local Git helpers, and render domain results. JSON and YAML data go to stdout; command diagnostics use stderr. Git subprocesses provide local and origin operations; HTTPS provides GitHub signals.

State lives in selected Git checkouts and refs, installed harness skill/instruction files, and a versioned agent-installations.json ownership file under the operating-system user configuration directory. There is no general query cache or database: origin remote-tracking refs are cached Git data, labelled accordingly. Read commands do not implicitly fetch. Mutations generally execute after preflight unless --dry-run is set; explicit targets and flags vary by command. Release-tag publication may read origin during preflight and observes Actions asynchronously after pushing.
