package commands

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/raithlin/gha/pkg/model"
)

const requiredCapabilitiesMetadata = "gha-required-capabilities"

func validateSkillCapabilities(skill []byte, capabilities *model.Capabilities) ([]string, error) {
	metadata, err := skillFrontmatterMetadata(skill)
	if err != nil {
		return nil, err
	}
	requirements := strings.Split(metadata[requiredCapabilitiesMetadata], ",")
	if len(requirements) == 0 || strings.TrimSpace(metadata[requiredCapabilitiesMetadata]) == "" {
		return nil, fmt.Errorf("skill does not declare %s", requiredCapabilitiesMetadata)
	}
	available := map[string]bool{}
	if capabilities != nil {
		for _, capability := range capabilities.Commands {
			if capability.Status == "available" {
				available[capability.Command] = true
			}
		}
	}
	missingSet := map[string]bool{}
	for _, requirement := range requirements {
		command := strings.TrimSpace(requirement)
		if command == "" {
			return nil, fmt.Errorf("skill declares an empty required capability")
		}
		if !available[command] {
			missingSet[command] = true
		}
	}
	missing := make([]string, 0, len(missingSet))
	for command := range missingSet {
		missing = append(missing, command)
	}
	sort.Strings(missing)
	return missing, nil
}

func skillFrontmatterMetadata(skill []byte) (map[string]string, error) {
	content := strings.TrimPrefix(string(skill), "\ufeff")
	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("skill has no YAML frontmatter")
	}
	content = strings.TrimPrefix(content, "---\n")
	end := strings.Index(content, "\n---")
	if end < 0 {
		return nil, fmt.Errorf("skill YAML frontmatter is not closed")
	}
	var frontmatter struct {
		Metadata map[string]string `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(content[:end]), &frontmatter); err != nil {
		return nil, fmt.Errorf("parse skill YAML frontmatter: %w", err)
	}
	if frontmatter.Metadata == nil {
		return nil, fmt.Errorf("skill does not declare %s", requiredCapabilitiesMetadata)
	}
	return frontmatter.Metadata, nil
}
