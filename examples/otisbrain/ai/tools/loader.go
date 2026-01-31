package tools

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// SkillDefinition represents the structure of a skill defined in YAML
type SkillDefinition struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Version     string                 `yaml:"version"`
	Author      string                 `yaml:"author"`
	Category    string                 `yaml:"category"`
	Tags        []string               `yaml:"tags"`
	Inputs      []SkillInput           `yaml:"inputs"`
	Outputs     []SkillOutput          `yaml:"outputs"`
	Execution   SkillExecution         `yaml:"execution"`
	Metadata    map[string]interface{} `yaml:"metadata"`
}

// SkillInput represents an input parameter for a skill
type SkillInput struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default"`
	Description string `yaml:"description"`
}

// SkillOutput represents an output parameter for a skill
type SkillOutput struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// SkillExecution defines how a skill is executed
type SkillExecution struct {
	Type       string                 `yaml:"type"`
	Function   string                 `yaml:"function"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

// SkillLoader handles loading skills from YAML or SKILL.md files
type SkillLoader struct {
	skillDir string
}

// NewSkillLoader creates a new skill loader
func NewSkillLoader(skillDir string) *SkillLoader {
	return &SkillLoader{
		skillDir: skillDir,
	}
}

// LoadSkills loads all skills from the skill directory
func (sl *SkillLoader) LoadSkills() (map[string]SkillDefinition, error) {
	skills := make(map[string]SkillDefinition)

	// Check if skill directory exists
	if _, err := os.Stat(sl.skillDir); os.IsNotExist(err) {
		return skills, fmt.Errorf("skill directory does not exist: %s", sl.skillDir)
	}

	// Walk through the skill directory
	err := filepath.Walk(sl.skillDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Process YAML files
		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".yaml") || strings.HasSuffix(strings.ToLower(path), ".yml")) {
			skillDef, err := sl.loadSkillFromYAMLFile(path)
			if err != nil {
				return fmt.Errorf("failed to load skill from %s: %w", path, err)
			}

			skills[skillDef.Name] = skillDef
		}

		// Process SKILL.md files
		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".skill.md") || strings.HasSuffix(strings.ToLower(path), "skill.md")) {
			skillDef, err := sl.loadSkillFromMarkdownFile(path)
			if err != nil {
				return fmt.Errorf("failed to load skill from %s: %w", path, err)
			}

			skills[skillDef.Name] = skillDef
		}

		return nil
	})

	return skills, err
}

// loadSkillFromYAMLFile loads a skill definition from a YAML file
func (sl *SkillLoader) loadSkillFromYAMLFile(filePath string) (SkillDefinition, error) {
	var skillDef SkillDefinition

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return skillDef, fmt.Errorf("failed to read skill file: %w", err)
	}

	err = yaml.Unmarshal(data, &skillDef)
	if err != nil {
		return skillDef, fmt.Errorf("failed to parse skill YAML: %w", err)
	}

	return skillDef, nil
}

// loadSkillFromMarkdownFile loads a skill definition from a SKILL.md file
func (sl *SkillLoader) loadSkillFromMarkdownFile(filePath string) (SkillDefinition, error) {
	var skillDef SkillDefinition

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return skillDef, fmt.Errorf("failed to read skill file: %w", err)
	}

	content := string(data)

	// Extract frontmatter (between --- delimiters) - more flexible regex to handle different line endings
	frontmatterRegex := regexp.MustCompile(`(?s)^---[\r\n]+((?:.|[\r\n])*?)---[\r\n]+((?:.|[\r\n])*)$`)
	matches := frontmatterRegex.FindStringSubmatch(content)

	if len(matches) < 3 {
		// Try alternative regex for cases where there might be spaces or different formatting
		altFrontmatterRegex := regexp.MustCompile(`(?s)^[\r\n]*---[\r\n]*(.*?)---[\r\n]*(.*)$`)
		matches = altFrontmatterRegex.FindStringSubmatch(content)
		if len(matches) < 3 {
			return skillDef, fmt.Errorf("invalid SKILL.md format: missing frontmatter")
		}
	}

	// Parse the YAML frontmatter
	yamlData := matches[1]
	err = yaml.Unmarshal([]byte(yamlData), &skillDef)
	if err != nil {
		return skillDef, fmt.Errorf("failed to parse skill frontmatter YAML: %w", err)
	}

	// Extract command examples from the content
	body := matches[2]
	skillDef.Execution = SkillExecution{
		Type:     "shell_command",
		Function: skillDef.Name, // Use the skill name as function name for compatibility
		Parameters: map[string]interface{}{
			"command": sl.extractCommandsFromMarkdown(body),
		},
	}

	return skillDef, nil
}

// extractCommandsFromMarkdown extracts command examples from the markdown body
func (sl *SkillLoader) extractCommandsFromMarkdown(body string) string {
	// Look for command examples in the markdown
	// This is a simplified version - in practice, you might want more sophisticated parsing

	// Find command sections (typically indented or in code blocks)
	lines := strings.Split(body, "\n")
	var commands []string

	for i, line := range lines {
		// Look for "Command:" followed by the actual command on the next line(s)
		if strings.Contains(line, "Command:") {
			// Get the next non-empty line which should contain the command
			for j := i + 1; j < len(lines); j++ {
				cmdLine := strings.TrimSpace(lines[j])
				// Skip empty lines and markdown formatting
				if cmdLine == "" || strings.HasPrefix(cmdLine, "```") {
					continue
				}
				// If it looks like a command (doesn't start with numbers or special markdown chars)
				if !strings.HasPrefix(cmdLine, "1)") && !strings.HasPrefix(cmdLine, "2)") &&
				   !strings.HasPrefix(cmdLine, "-") && !strings.HasPrefix(cmdLine, "#") {
					// Replace parameter placeholders with template syntax
					templatedCmd := sl.convertParameterPlaceholders(cmdLine)
					commands = append(commands, templatedCmd)
					break
				}
			}
		}
	}

	if len(commands) > 0 {
		// Return the first command found, or join multiple commands with semicolons
		return strings.Join(commands, "; ")
	}

	// Default fallback command
	return "echo 'No command found in skill definition'"
}

// convertParameterPlaceholders converts parameter placeholders to template syntax
func (sl *SkillLoader) convertParameterPlaceholders(command string) string {
	// This is a simple conversion - in a real implementation, you might want more sophisticated parsing
	// Replace common parameter patterns with template syntax
	command = strings.ReplaceAll(command, "{{ inputs.time_range }}", "{{.time_range}}")
	command = strings.ReplaceAll(command, "{{ inputs.severity }}", "{{.severity}}")
	command = strings.ReplaceAll(command, "{{ inputs.resource_type }}", "{{.resource_type}}")
	command = strings.ReplaceAll(command, "{{ inputs.namespace }}", "{{.namespace}}")

	return command
}

// ValidateSkill validates a skill definition
func (sl *SkillLoader) ValidateSkill(skillDef SkillDefinition) error {
	if skillDef.Name == "" {
		return fmt.Errorf("skill name is required")
	}

	if skillDef.Description == "" {
		return fmt.Errorf("skill description is required")
	}

	if skillDef.Execution.Type == "" {
		return fmt.Errorf("skill execution type is required")
	}

	return nil
}