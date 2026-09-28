package ghaskill

import (
	"strings"
	"testing"
)

func TestDefaultGuidanceIsCompactAndKeepsSkillAvailable(t *testing.T) {
	guidance := string(Guidance)
	if len(Guidance) > 450 {
		t.Fatalf("default guidance is %d bytes; keep the always-present block compact", len(Guidance))
	}
	for _, phrase := range []string{"gha capabilities --format json", "--dry-run", "Use direct Git when simpler"} {
		if !strings.Contains(guidance, phrase) {
			t.Errorf("default guidance lacks %q", phrase)
		}
	}
	if strings.Contains(guidance, "use the installed `gha` skill before") {
		t.Error("default guidance should not require loading the detailed skill on every task")
	}
	if !strings.Contains(string(Skill), "# GHA") {
		t.Error("the detailed skill must remain bundled for explicit use")
	}
}
