package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

// SkillExecutor handles execution of skills
type SkillExecutor struct {
	loader *SkillLoader
}

// NewSkillExecutor creates a new skill executor
func NewSkillExecutor(skillDir string) *SkillExecutor {
	return &SkillExecutor{
		loader: NewSkillLoader(skillDir),
	}
}

// LoadAndExecuteSkill loads a skill from YAML and executes it
func (se *SkillExecutor) LoadAndExecuteSkill(ctx context.Context, skillName string, params map[string]interface{}) (string, error) {
	// Load all skills
	skills, err := se.loader.LoadSkills()
	if err != nil {
		return "", fmt.Errorf("failed to load skills: %w", err)
	}

	// Find the requested skill
	skillDef, exists := skills[skillName]
	if !exists {
		return "", fmt.Errorf("skill '%s' not found", skillName)
	}

	// Validate the skill
	if err := se.loader.ValidateSkill(skillDef); err != nil {
		return "", fmt.Errorf("invalid skill '%s': %w", skillName, err)
	}

	// Execute the skill based on its execution type
	switch skillDef.Execution.Type {
	case "function_call":
		return se.executeFunctionCall(ctx, skillDef, params)
	case "shell_command":
		return se.executeShellCommand(ctx, skillDef, params)
	default:
		return "", fmt.Errorf("unsupported execution type: %s", skillDef.Execution.Type)
	}
}

// executeFunctionCall executes a function call type skill by converting it to a shell command
func (se *SkillExecutor) executeFunctionCall(ctx context.Context, skillDef SkillDefinition, params map[string]interface{}) (string, error) {
	// For complete decoupling, we'll convert function calls to shell commands
	// This removes the hard-coded function mappings while maintaining functionality

	// Get the command template from the skill definition's function name and parameters
	// This assumes the YAML file contains a command template in the parameters
	commandTemplate, exists := skillDef.Execution.Parameters["command_template"]
	if exists {
		templateStr, ok := commandTemplate.(string)
		if ok {
			// Process the template with the provided parameters
			processedCommand, err := processTemplate(templateStr, params)
			if err != nil {
				return "", fmt.Errorf("failed to process command template: %w", err)
			}

			// Execute the processed command
			return executeShellCommand(processedCommand)
		}
	}

	// If no command template exists, try to find a command parameter
	command, exists := skillDef.Execution.Parameters["command"].(string)
	if exists {
		// Process the template with the provided parameters
		processedCommand, err := processTemplate(command, params)
		if err != nil {
			return "", fmt.Errorf("failed to process command template: %w", err)
		}

		// Execute the processed command
		return executeShellCommand(processedCommand)
	}

	// If no command is found, return an error indicating the skill is misconfigured
	return "", fmt.Errorf("skill '%s' is misconfigured: no command template found", skillDef.Name)
}

// executeShellCommand executes a shell command type skill
func (se *SkillExecutor) executeShellCommand(ctx context.Context, skillDef SkillDefinition, params map[string]interface{}) (string, error) {
	// Get the command template from the skill definition
	commandTemplate, exists := skillDef.Execution.Parameters["command"].(string)
	if !exists {
		return "", fmt.Errorf("command parameter not found in skill execution parameters")
	}

	// Process the template with the provided parameters
	processedCommand, err := processTemplate(commandTemplate, params)
	if err != nil {
		return "", fmt.Errorf("failed to process command template: %w", err)
	}

	// Execute the processed command
	return executeShellCommand(processedCommand)
}

// processTemplate processes a command template with the given parameters
func processTemplate(templateStr string, params map[string]interface{}) (string, error) {
	// First, convert the template syntax from {{.param}} to {{.param}}
	// Our SKILL.md files use {{ inputs.param_name }} syntax
	convertedTemplate := convertToGoTemplateSyntax(templateStr)

	tmpl, err := template.New("command").Parse(convertedTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, params)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// convertToGoTemplateSyntax converts template syntax to Go template format
func convertToGoTemplateSyntax(templateStr string) string {
	// Convert {{ inputs.param_name }} to {{.param_name}}
	re := regexp.MustCompile(`{{\s*inputs\.(\w+)\s*}}`)
	return re.ReplaceAllString(templateStr, "{{.$1}}")
}

// executeShellCommand executes a shell command and returns the output
func executeShellCommand(command string) (string, error) {
	// Split command into parts to handle properly
	cmdParts := strings.Split(command, " ")
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Use the first part as command and rest as arguments
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Return output even if there's an error (e.g., command not found)
		return string(output), nil
	}

	return string(output), nil
}

// getStringParam extracts a string parameter from the params map with a default value
func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, exists := params[key]; exists {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

// GetAvailableSkills returns a list of all available skills loaded from YAML files
func (se *SkillExecutor) GetAvailableSkills() (map[string]SkillDefinition, error) {
	return se.loader.LoadSkills()
}

// GetCriticalEventsForAI is a simplified function that returns only the most relevant information for AI consumption
func GetCriticalEventsForAI(ctx context.Context, timeRange string) (string, error) {
	// This function provides a direct API for retrieving critical events
	// It can be used by external systems or for testing purposes

	// For now, we'll return a mock response
	mockResponse := fmt.Sprintf("Mock response for time range: %s", timeRange)
	return mockResponse, nil
}
