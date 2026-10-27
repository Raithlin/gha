---
id: TASK-002
title: Groma setup
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T13:20:00Z'
updated_date: '2026-10-27T19:13:00Z'
labels: []
dependencies: []
references:
  - gha
  - gha-cli
  - branch-service
  - branch-writer
  - capabilities
  - code-host-provider
  - config
  - release-service
  - review-service
  - cli-runtime
  - build-identity
  - branch-commands
  - pull-request-commands
  - release-commands
  - tag-commands
  - worktree-commands
  - agent-integration
  - guidance-update
  - repository-context
  - branch-inspection
  - github-provider
  - domain-models
  - result-rendering
  - repository-analysis
  - analysis-command
  - release-checkout
  - tag-writer
  - worktree-operations
modified_files:
  - .groma/config.json
  - .groma/systems/gha/containers/gha-gha/components/root.md
  - .groma/systems/gha/containers/gha-gha/components/main.md
  - .groma/systems/gha/containers/gha-gha/components/errors.md
  - .groma/systems/gha/containers/gha-gha/components/dashboard.md
  - .groma/systems/gha/containers/gha-gha/components/buildinfo.md
  - .groma/systems/gha/containers/gha-gha/components/version.md
  - .groma/systems/gha/containers/gha-gha/components/branch.md
  - .groma/systems/gha/containers/gha-gha/components/commands-branches.md
  - .groma/systems/gha/containers/gha-gha/components/commands-review.md
  - .groma/systems/gha/containers/gha-gha/components/prs.md
  - .groma/systems/gha/containers/gha-gha/components/pr.md
  - .groma/systems/gha/containers/gha-gha/components/commands-release.md
  - .groma/systems/gha/containers/gha-gha/components/releases.md
  - .groma/systems/gha/containers/gha-gha/components/release-publish.md
  - .groma/systems/gha/containers/gha-gha/components/commands-agent.md
  - .groma/systems/gha/containers/gha-gha/components/agent-list.md
  - .groma/systems/gha/containers/gha-gha/components/embed.md
  - .groma/systems/gha/containers/gha-gha/components/update.md
  - .groma/systems/gha/containers/gha-gha/components/skill-compatibility.md
  - .groma/systems/gha/containers/gha-gha/components/repository.md
  - .groma/systems/gha/containers/gha-gha/components/current-branch.md
  - .groma/systems/gha/containers/gha-gha/components/git-branches.md
  - .groma/systems/gha/containers/gha-gha/components/cleanup.md
  - .groma/systems/gha/containers/gha-gha/components/client.md
  - .groma/systems/gha/containers/gha-gha/components/github-release.md
  - .groma/systems/gha/containers/gha-gha/components/model.md
  - .groma/systems/gha/containers/gha-gha/components/model-worktrees.md
  - .groma/systems/gha/containers/gha-gha/components/output-review.md
  - .groma/systems/gha/containers/gha-gha/components/color.md
  - .groma/systems/gha/containers/gha-gha/components/output-agent.md
  - .groma/systems/gha/containers/gha-gha/components/output-worktrees.md
  - .groma/systems/gha/containers/gha-cli/container.md
  - .groma/systems/gha/containers/gha-cli/components/analyze.md
  - .groma/systems/gha/containers/gha-cli/components/analyzer.md
  - .groma/systems/gha/containers/gha-cli/components/branch.md
  - .groma/systems/gha/containers/gha-cli/components/branch-service.md
  - .groma/systems/gha/containers/gha-cli/components/branch-writer.md
  - .groma/systems/gha/containers/gha-cli/components/buildinfo.md
  - .groma/systems/gha/containers/gha-cli/components/capabilities.md
  - .groma/systems/gha/containers/gha-cli/components/checkout.md
  - .groma/systems/gha/containers/gha-cli/components/client.md
  - .groma/systems/gha/containers/gha-cli/components/code-host-provider.md
  - .groma/systems/gha/containers/gha-cli/components/commands-agent.md
  - .groma/systems/gha/containers/gha-cli/components/commands-release.md
  - .groma/systems/gha/containers/gha-cli/components/commands-review.md
  - .groma/systems/gha/containers/gha-cli/components/commands-tag.md
  - .groma/systems/gha/containers/gha-cli/components/commands-worktrees.md
  - .groma/systems/gha/containers/gha-cli/components/config.md
  - .groma/systems/gha/containers/gha-cli/components/git-branches.md
  - .groma/systems/gha/containers/gha-cli/components/git-tag.md
  - .groma/systems/gha/containers/gha-cli/components/git-worktrees.md
  - .groma/systems/gha/containers/gha-cli/components/model.md
  - .groma/systems/gha/containers/gha-cli/components/output-review.md
  - .groma/systems/gha/containers/gha-cli/components/release-service.md
  - .groma/systems/gha/containers/gha-cli/components/repository.md
  - .groma/systems/gha/containers/gha-cli/components/review-service.md
  - .groma/systems/gha/containers/gha-cli/components/root.md
  - .groma/systems/gha/containers/gha-cli/components/update.md
  - .groma/systems/gha/containers/gha-gha/container.md
  - .groma/systems/gha/containers/gha-gha/components/analyze.md
  - .groma/systems/gha/containers/gha-gha/components/analyzer.md
  - .groma/systems/gha/containers/gha-gha/components/branch-service.md
  - .groma/systems/gha/containers/gha-gha/components/branch-writer.md
  - .groma/systems/gha/containers/gha-gha/components/capabilities.md
  - .groma/systems/gha/containers/gha-gha/components/checkout.md
  - .groma/systems/gha/containers/gha-gha/components/code-host-provider.md
  - .groma/systems/gha/containers/gha-gha/components/commands-tag.md
  - .groma/systems/gha/containers/gha-gha/components/commands-worktrees.md
  - .groma/systems/gha/containers/gha-gha/components/config.md
  - .groma/systems/gha/containers/gha-gha/components/git-tag.md
  - .groma/systems/gha/containers/gha-gha/components/git-worktrees.md
  - .groma/systems/gha/containers/gha-gha/components/release-service.md
  - .groma/systems/gha/containers/gha-gha/components/review-service.md
  - .groma/systems/gha/containers/gha-cli/components/cli-runtime.md
  - .groma/systems/gha/containers/gha-cli/components/build-identity.md
  - .groma/systems/gha/containers/gha-cli/components/branch-commands.md
  - .groma/systems/gha/containers/gha-cli/components/pull-request-commands.md
  - .groma/systems/gha/containers/gha-cli/components/release-commands.md
  - .groma/systems/gha/containers/gha-cli/components/tag-commands.md
  - .groma/systems/gha/containers/gha-cli/components/worktree-commands.md
  - .groma/systems/gha/containers/gha-cli/components/agent-integration.md
  - .groma/systems/gha/containers/gha-cli/components/guidance-update.md
  - .groma/systems/gha/containers/gha-cli/components/repository-context.md
  - .groma/systems/gha/containers/gha-cli/components/branch-inspection.md
  - .groma/systems/gha/containers/gha-cli/components/github-provider.md
  - .groma/systems/gha/containers/gha-cli/components/domain-models.md
  - .groma/systems/gha/containers/gha-cli/components/result-rendering.md
  - .groma/systems/gha/containers/gha-cli/components/repository-analysis.md
  - .groma/systems/gha/containers/gha-cli/components/analysis-command.md
  - .groma/systems/gha/containers/gha-cli/components/release-checkout.md
  - .groma/systems/gha/containers/gha-cli/components/tag-writer.md
  - .groma/systems/gha/containers/gha-cli/components/worktree-operations.md
  - .groma/index.md
  - .groma/actors/developer.md
  - .groma/externals/coding-agent-harnesses.md
  - .groma/externals/github.md
  - .groma/externals/git-and-repository-storage.md
  - .groma/externals/github-cli-authentication.md
  - .groma/systems/gha/system.md
  - .groma/project.md
  - .groma/flows/inspect-a-pull-request.md
  - .groma/flows/plan-a-branch-deletion.md
  - .groma/flows/publish-a-release-tag.md
  - .groma/flows/refresh-coding-agent-guidance.md
priority: high
type: chore
ordinal: 23000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Curate this repository's architecture with Groma. Run groma agent-instructions and follow its curation workflow. Describe the architecture as it exists today; do not change application code.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [x] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Completed and accepted by the user. Curated .groma with one system, one CLI container, 26 responsibility components, external integrations, exact source ownership, and four scenario flows. Two rescans preserve the architecture; the local build and tests pass. No application code changed. Retained verification limitations: CI was not verified, exact existing coverage is 94.96%, and the Groma source drawer returns Failed to fetch. User explicitly authorized Done with these limitations.
<!-- SECTION:FINAL_SUMMARY:END -->
