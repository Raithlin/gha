---
type: Groma Project
title: GHA current architecture
groma:
  profile: architecture
description: Current architecture of the agent-first GitHub Assistant CLI.
---

This map describes the source and executable present in this checkout, not the roadmap. One Go CLI coordinates GitHub-backed decisions, local Git workflows, release publication, and coding-agent guidance. Drill into GHA CLI for grouped responsibilities and exact source files.

Coverage: the enabled Go scanner owns production Go source in the Linux/amd64 build context. Tests and development tooling are excluded from product components. Embedded Markdown assets, documentation, shell/install scripts, and GitHub Actions YAML are not separately scanned components; the implementation references are explained in their owning runtime responsibilities. The scanner reports unresolved dependency/type evidence and dynamic dispatch limitations, so important interface-driven interactions are authored from source inspection. GitHub is the sole implemented provider; dashboard, other hosts, and server/database architectures are not implemented. Relationships describe current calls, while flows show selected scenarios and conditional write steps.
