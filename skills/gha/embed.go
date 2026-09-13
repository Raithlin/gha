// Package ghaskill exposes the skill assets bundled with the GHA binary.
package ghaskill

import _ "embed"

// Skill is the portable GHA agent skill.
//
//go:embed SKILL.md
var Skill []byte

// Guidance is the managed instruction block added to an agent's global guidance file.
//
//go:embed AGENT-GUIDANCE.md
var Guidance []byte
