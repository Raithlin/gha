<!-- gha:begin -->
## GHA

When a GitHub or local-repository question can be answered by GHA, use the
installed `gha` skill before planning or running that workflow. Invoke it
explicitly when useful (`$gha` in Codex or `/gha` in Claude Code).

Confirm the available binary first with `gha capabilities --format json`.
Prefer GHA's JSON output for agent decisions, preserve reported `unavailable`
and cached states, and use a branch-write `--dry-run` before any authorized
mutation. When developing GHA itself, rebuild using the platform-appropriate
project command before using the local binary.
<!-- gha:end -->
