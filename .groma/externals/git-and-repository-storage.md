---
type: C4 System
title: Git and repository storage
status: stable
groma:
  id: git-and-repository-storage
  technology: Git CLI; local working trees and refs; configured origin transport
---

Installed Git executable and the repositories it operates on. GHA reads local checkout and cached remote-tracking state through subprocesses, and selected workflows create refs, change worktrees, or push exact refs to origin. Remote reads and writes use Git authentication independently of GHA API credentials. This boundary includes the selected repository data; it is not a GHA-managed database.
