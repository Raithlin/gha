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

// GuidanceUpdate renders the structured outcome of a guidance refresh.
func GuidanceUpdate(writer io.Writer, format Format, update *model.GuidanceUpdate) error {
	if format != Text {
		return structured(writer, format, update)
	}
	if _, err := fmt.Fprintf(writer, "GHA guidance update (release %s)\n", sanitizeTerminal(update.LatestVersion)); err != nil {
		return err
	}
	if update.SourceState == "unavailable" {
		if _, err := fmt.Fprintf(writer, "Source unavailable: %s\n", sanitizeTerminal(update.SourceMessage)); err != nil {
			return err
		}
	}
	for _, target := range update.Targets {
		if _, err := fmt.Fprintf(writer, "%s: %s skill %s", sanitizeTerminal(target.AgentName), sanitizeTerminal(target.State), sanitizeTerminal(target.SkillPath)); err != nil {
			return err
		}
		if target.InstructionsPath != "" {
			if _, err := fmt.Fprintf(writer, "; guidance %s", sanitizeTerminal(target.InstructionsPath)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}
	if len(update.Targets) == 0 {
		if _, err := fmt.Fprintln(writer, "No GHA-configured coding agents found; nothing to update."); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(writer, "GHA executable remains at version %s.\n", sanitizeTerminal(update.BinaryVersion))
	return err
}
