---
id: TASK-018
title: >-
  Extend paired agent evaluations to external repositories and independent
  provider tasks
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
ordinal: 18000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap evidence next step. Existing pilots and the implemented-capability sweep suggest compact guidance can reduce tokens, but command-shaped prompts and limited independent provider scenarios constrain generalization.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Add independently defined tasks on additional external repositories, using the same task and repository snapshot with and without GHA and pinned agent model/instructions.
- [ ] #2 Retain complete raw run records and task definitions, recording full-run tokens, completion, elapsed time, and objective correctness.
- [ ] #3 Report tokens per correct result alongside correctness and limitations; blind human review assesses subjective quality.
- [ ] #4 Publish reproducible methods and results in the agent-efficiency documentation and explain implications for roadmap choices without treating measurements as CLI contracts.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->
