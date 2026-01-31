package tools

import (
	"context"
	"fmt"
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
	default:
		return "", fmt.Errorf("unsupported execution type: %s", skillDef.Execution.Type)
	}
}

// executeFunctionCall executes a function call type skill
func (se *SkillExecutor) executeFunctionCall(ctx context.Context, skillDef SkillDefinition, params map[string]interface{}) (string, error) {
	// Execute the appropriate function based on the skill's function name
	switch skillDef.Execution.Function {
	case "getRecentCriticalEvents":
		// Extract parameters with defaults
		timeRange := getStringParam(params, "time_range", "last_hour")
		severity := getStringParam(params, "severity", "all")
		resourceType := getStringParam(params, "resource_type", "all")

		return ExecuteGetRecentCriticalEvents(ctx, timeRange, severity, resourceType)
	case "getAllCriticalRecords":
		return GetAllCriticalRecords(ctx)
	default:
		return "", fmt.Errorf("unknown function '%s'", skillDef.Execution.Function)
	}
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