---
id: TASK-029
title: Provide simple issue and pull request templates
status: Done
assignee:
  - '@codex'
created_date: '2026-10-27T19:56:00Z'
updated_date: '2026-10-27T20:00:00Z'
labels: []
dependencies: []
references:
  - https://github.com/stevemao/github-issue-templates
  - .github/pull_request_template.md
  - docs/CONTRIBUTING.md
  - https://gitmore.io/blog/github-pull-request-template
  - https://github.com/Raithlin/gha/pull/28
documentation:
  - docs/CONTRIBUTING.md
modified_files:
  - .github/ISSUE_TEMPLATE/bug_report.md
  - .github/ISSUE_TEMPLATE/feature_request.md
  - docs/CONTRIBUTING.md
  - .github/pull_request_template.md
type: docs
ordinal: 61000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The user supplied issue-template examples and a PR-template guide while preparing the contributor workflow for publication, and explicitly requested a simple but useful PR template. Public reports need focused prompts without a large contributor checklist.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 GitHub offers separate bug-report and feature-request templates tailored to GHA commands, build identity, stdout/stderr evidence, and workflow value.
- [x] #2 The contributor guide links to the relevant issue templates; the PR template asks for the problem/outcome, validation evidence, and optional reviewer notes without a large checklist.
- [x] #3 Template metadata and documentation links validate, and no nonexistent channels, labels, or assignment policies are introduced.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [x] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added focused bug-report and feature-request templates, linked them from the contributor guide, and reduced the default PR template to Summary, Validation, and optional Review notes. YAML metadata and template links validate; all 14 architecture-tool tests pass. Hosted Build and Test passed at 9be13d1 in run 40573919646.
<!-- SECTION:FINAL_SUMMARY:END -->
