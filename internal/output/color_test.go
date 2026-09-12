package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStylesDoNotColorNonTerminalOutput(t *testing.T) {
	assert.False(t, supportsColor(&bytes.Buffer{}))
	assert.Equal(t, "Pull Request", newStyles(&bytes.Buffer{}).heading("Pull Request"))
}

func TestStylesApplyANSIWhenEnabled(t *testing.T) {
	styles := styles{enabled: true}

	assert.Equal(t, "\x1b[1m\x1b[36mPull Request\x1b[0m", styles.heading("Pull Request"))
	assert.Equal(t, "\x1b[32mopen\x1b[0m", styles.state("open"))
	assert.Equal(t, "\x1b[31mhigh\x1b[0m", styles.severity("high"))
	assert.Equal(t, "\x1b[35moctocat\x1b[0m", styles.username("octocat"))
	assert.Equal(t, "\x1b[34m0123456789ab\x1b[0m", styles.commitID("0123456789abcdef"))
}

func TestStylesColorizeReferencesInDescriptions(t *testing.T) {
	styles := styles{enabled: true}

	rendered := styles.description("Commit 168dc1f by @dependabot fixes CVE-2026-42505.")

	assert.Equal(t, "Commit \x1b[34m168dc1f\x1b[0m by \x1b[35m@dependabot\x1b[0m fixes CVE-2026-42505.", rendered)
}
