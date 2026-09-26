package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
)

func newAgentListCmd() *cobra.Command {
	var format string
	command := &cobra.Command{
		Use:   "list",
		Short: "List coding-agent harnesses configured by GHA",
		Long:  "List harnesses recorded by gha agent install, their managed destinations, and whether the guidance and skill files are present.",
		Args:  noArgsWithFormat(&format),
		RunE: func(cmd *cobra.Command, _ []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			ownership, err := readAgentOwnership()
			if err != nil {
				return err
			}
			result := &model.AgentInstallationList{SchemaVersion: model.AgentInstallationListSchemaVersion, Agents: make([]model.AgentInstallation, 0, len(ownership.Agents))}
			for _, agent := range ownership.Agents {
				result.Agents = append(result.Agents, model.AgentInstallation{
					ID: agent.ID, Name: agent.Name, SkillPath: agent.SkillPath,
					InstructionsPath: agent.InstructionsPath,
					SkillState:       fileState(agent.SkillPath), GuidanceState: fileState(agent.InstructionsPath),
				})
			}
			return output.AgentInstallations(cmd.OutOrStdout(), outputFormat, result)
		},
	}
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

func fileState(path string) string {
	if path == "" {
		return "not_configured"
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "missing"
	}
	if err != nil || info.IsDir() {
		return "unavailable"
	}
	return "present"
}
