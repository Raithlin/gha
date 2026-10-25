---
type: Groma Flow
title: Publish a release tag
groma:
  id: publish-a-release-tag
---

The workflow runs only for an authorized invocation. --dry-run stops after preparation and rendering. Without it, publication repeats preflight, creates and pushes the annotated tag, and performs one Actions observation; it does not wait for release assets.

## Steps

| From | To | Action |
| --- | --- | --- |
| [Developer](../actors/developer.md) | [GHA CLI](../systems/gha/containers/gha-cli/container.md) | Run release publish <version>, optionally with --dry-run |
| [cli-runtime](../systems/gha/containers/gha-cli/components/cli-runtime.md) | [release-commands](../systems/gha/containers/gha-cli/components/release-commands.md) | Dispatch release publication |
| [release-commands](../systems/gha/containers/gha-cli/components/release-commands.md) | [repository-context](../systems/gha/containers/gha-cli/components/repository-context.md) | Resolve the selected repository and checkout |
| [release-commands](../systems/gha/containers/gha-cli/components/release-commands.md) | [release-service](../systems/gha/containers/gha-cli/components/release-service.md) | Prepare the versioned release plan and blockers |
| [release-service](../systems/gha/containers/gha-cli/components/release-service.md) | [release-checkout](../systems/gha/containers/gha-cli/components/release-checkout.md) | Read clean HEAD, live origin tip, committed workflow, tags, and required notes |
| [release-service](../systems/gha/containers/gha-cli/components/release-service.md) | [github-provider](../systems/gha/containers/gha-cli/components/github-provider.md) | Read repository permissions, CI, and remote tag collision |
| [release-service](../systems/gha/containers/gha-cli/components/release-service.md) | [release-checkout](../systems/gha/containers/gha-cli/components/release-checkout.md) | Only when ready and not a dry run: recheck and perform annotated-tag effects |
| [release-checkout](../systems/gha/containers/gha-cli/components/release-checkout.md) | [tag-writer](../systems/gha/containers/gha-cli/components/tag-writer.md) | Create the annotated tag and push its exact ref |
| [tag-writer](../systems/gha/containers/gha-cli/components/tag-writer.md) | [git-and-repository-storage](../externals/git-and-repository-storage.md) | Execute the selected Git tag and origin push |
| [release-service](../systems/gha/containers/gha-cli/components/release-service.md) | [github-provider](../systems/gha/containers/gha-cli/components/github-provider.md) | Observe the tag-triggered Actions workflow once |
| [release-commands](../systems/gha/containers/gha-cli/components/release-commands.md) | [result-rendering](../systems/gha/containers/gha-cli/components/result-rendering.md) | Render blockers, local/origin effects, and workflow observation separately |
