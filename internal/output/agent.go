package output

import (
	"fmt"
	"io"

	"github.com/raithlin/gha/pkg/model"
)

// AgentInstallations renders the versioned installed-agent inventory.
func AgentInstallations(writer io.Writer, format Format, list *model.AgentInstallationList) error {
	if format != Text {
		return structured(writer, format, list)
	}
	if _, err := fmt.Fprintln(writer, "GHA-configured agents"); err != nil {
		return err
	}
	if len(list.Agents) == 0 {
		_, err := fmt.Fprintln(writer, "No coding-agent harnesses are recorded as configured by GHA.")
		return err
	}
	for _, agent := range list.Agents {
		if _, err := fmt.Fprintf(writer, "%s (%s)\n  Guidance: %s (%s)\n  Skill: %s (%s)\n", sanitizeTerminal(agent.Name), sanitizeTerminal(agent.ID), sanitizeTerminal(agent.InstructionsPath), sanitizeTerminal(agent.GuidanceState), sanitizeTerminal(agent.SkillPath), sanitizeTerminal(agent.SkillState)); err != nil {
			return err
		}
	}
	return nil
}
