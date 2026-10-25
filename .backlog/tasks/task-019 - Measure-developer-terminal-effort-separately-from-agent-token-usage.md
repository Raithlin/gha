---
id: TASK-019
title: Measure developer terminal effort separately from agent token usage
status: To Do
assignee: []
created_date: '2026-10-25T13:55:00Z'
labels:
  - roadmap
  - evidence
dependencies: []
references:
  - docs/roadmap.md
  - DESIGN.md
  - docs/agent-efficiency-evaluation.md
  - docs/agent-efficiency-test-results.md
priority: medium
type: task
ordinal: 19000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap evidence next step. Agent token usage does not measure human keystrokes or time to a correct terminal decision, so developer workflow value needs a separate evaluation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Define reproducible familiar-use and first-use terminal scenarios with GHA and appropriate git/gh comparisons.
- [ ] #2 Measure commands and characters entered, time to a correct decision, and mistakes or rework, using explicit correctness criteria.
- [ ] #3 Retain task definitions and raw observations and report correctness alongside effort, separating human measures from agent token results.
- [ ] #4 Document method, results, and limitations in the efficiency documentation so future roadmap decisions can use the evidence.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->
