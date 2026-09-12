package output

import (
	"io"
	"os"
	"regexp"
	"strings"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiBlue   = "\x1b[34m"
	ansiCyan   = "\x1b[36m"
	ansiPurple = "\x1b[35m"
)

// styles applies ANSI styling only when the writer is an interactive terminal.
type styles struct {
	enabled bool
}

var terminalReferencePattern = regexp.MustCompile(`(?i)@[a-z0-9](?:[a-z0-9-]{0,37})|\b[0-9a-f]{7,40}\b`)

func newStyles(writer io.Writer) styles {
	return styles{enabled: supportsColor(writer)}
}

func supportsColor(writer io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0" || os.Getenv("TERM") == "" || os.Getenv("TERM") == "dumb" {
		return false
	}

	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (s styles) wrap(code, value string) string {
	if !s.enabled || value == "" {
		return value
	}
	return code + value + ansiReset
}

func (s styles) heading(value string) string  { return s.wrap(ansiBold+ansiCyan, value) }
func (s styles) label(value string) string    { return s.wrap(ansiBold, value) }
func (s styles) muted(value string) string    { return s.wrap(ansiDim, value) }
func (s styles) action(value string) string   { return s.wrap(ansiCyan, value) }
func (s styles) username(value string) string { return s.wrap(ansiPurple, value) }
func (s styles) commitID(value string) string { return s.wrap(ansiBlue, shortCommitID(value)) }

func (s styles) state(value string) string {
	switch value {
	case "open":
		return s.wrap(ansiGreen, value)
	case "closed":
		return s.wrap(ansiRed, value)
	case "merged":
		return s.wrap(ansiPurple, value)
	default:
		return s.wrap(ansiYellow, value)
	}
}

func shortCommitID(value string) string {
	const shortLength = 12
	if len(value) <= shortLength {
		return value
	}
	return value[:shortLength]
}

func (s styles) severity(value string) string {
	switch value {
	case "high":
		return s.wrap(ansiRed, value)
	case "medium":
		return s.wrap(ansiYellow, value)
	case "low":
		return s.wrap(ansiGreen, value)
	default:
		return s.muted(value)
	}
}

// description highlights GitHub mentions and abbreviated or full SHA IDs in
// human-readable PR text. Structured output always retains the source body.
func (s styles) description(value string) string {
	if !s.enabled || value == "" {
		return value
	}

	matches := terminalReferencePattern.FindAllStringIndex(value, -1)
	if len(matches) == 0 {
		return value
	}

	var rendered strings.Builder
	rendered.Grow(len(value) + len(matches)*len(ansiReset))
	previous := 0
	for _, match := range matches {
		rendered.WriteString(value[previous:match[0]])
		reference := value[match[0]:match[1]]
		if reference[0] == '@' {
			rendered.WriteString(s.username(reference))
		} else {
			rendered.WriteString(s.commitID(reference))
		}
		previous = match[1]
	}
	rendered.WriteString(value[previous:])
	return rendered.String()
}
