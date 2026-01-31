package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// ToolManager manages the priority-based execution of different tool types
type ToolManager struct {
	skillExecutor *SkillExecutor
	skillDocs     map[string]string // Cache for skill documentation
	docsMutex     sync.RWMutex
}

// NewToolManager creates a new tool manager with the skill executor
func NewToolManager(skillDir string) *ToolManager {
	return &ToolManager{
		skillExecutor: NewSkillExecutor(skillDir),
		skillDocs:     make(map[string]string),
	}
}

// ExecuteToolByPriority executes tools based on the priority order:
// 1. MCP tools (handled externally)
// 2. Skill documentation lookup (skill_load/skill_list_doc)
// 3. Direct skill execution (skill_run equivalent)
func (tm *ToolManager) ExecuteToolByPriority(ctx context.Context, toolName string, params map[string]interface{}) (string, error) {
	// Check if this is a documentation request
	if strings.HasPrefix(toolName, "skill_load") || strings.HasPrefix(toolName, "skill_list_doc") {
		return tm.handleDocRequest(ctx, toolName, params)
	}
	
	// Check if this is a direct skill execution request
	if strings.HasPrefix(toolName, "skill_run") || strings.HasPrefix(toolName, "execute_") {
		return tm.handleDirectSkillExecution(ctx, toolName, params)
	}
	
	// Default to skill execution for backward compatibility
	return tm.skillExecutor.LoadAndExecuteSkill(ctx, toolName, params)
}

// handleDocRequest handles documentation requests
func (tm *ToolManager) handleDocRequest(ctx context.Context, toolName string, params map[string]interface{}) (string, error) {
	switch {
	case strings.Contains(toolName, "skill_list_doc"):
		return tm.listSkillDocs(ctx)
	case strings.Contains(toolName, "skill_load"):
		skillName, exists := params["skill_name"].(string)
		if !exists {
			return "", fmt.Errorf("skill_name parameter required for skill_load")
		}
		return tm.loadSkillDoc(ctx, skillName)
	default:
		return "", fmt.Errorf("unknown documentation request: %s", toolName)
	}
}

// handleDirectSkillExecution handles direct skill execution (skill_run equivalent)
func (tm *ToolManager) handleDirectSkillExecution(ctx context.Context, toolName string, params map[string]interface{}) (string, error) {
	// Extract skill name from tool name (e.g., "skill_run_get_cluster_resources" -> "get_cluster_resources")
	var skillName string
	if strings.HasPrefix(toolName, "skill_run_") {
		skillName = strings.TrimPrefix(toolName, "skill_run_")
	} else if strings.HasPrefix(toolName, "execute_") {
		skillName = strings.TrimPrefix(toolName, "execute_")
	} else {
		skillName = toolName
	}
	
	// Execute the skill directly
	return tm.skillExecutor.LoadAndExecuteSkill(ctx, skillName, params)
}

// listSkillDocs returns documentation for all available skills
func (tm *ToolManager) listSkillDocs(ctx context.Context) (string, error) {
	skills, err := tm.skillExecutor.GetAvailableSkills()
	if err != nil {
		return "", fmt.Errorf("failed to get available skills: %w", err)
	}
	
	var result strings.Builder
	result.WriteString("Available Skills Documentation:\n\n")
	
	for _, skill := range skills {
		doc, err := tm.getSkillDocumentation(ctx, skill.Name)
		if err != nil {
			result.WriteString(fmt.Sprintf("Skill: %s (Error loading documentation: %v)\n\n", skill.Name, err))
		} else {
			result.WriteString(doc)
			result.WriteString("\n")
		}
	}
	
	return result.String(), nil
}

// loadSkillDoc loads documentation for a specific skill
func (tm *ToolManager) loadSkillDoc(ctx context.Context, skillName string) (string, error) {
	// Check cache first
	tm.docsMutex.RLock()
	doc, exists := tm.skillDocs[skillName]
	tm.docsMutex.RUnlock()
	
	if exists {
		return doc, nil
	}
	
	// Load skill definition to get documentation
	skills, err := tm.skillExecutor.GetAvailableSkills()
	if err != nil {
		return "", fmt.Errorf("failed to load skills: %w", err)
	}
	
	skillDef, exists := skills[skillName]
	if !exists {
		return "", fmt.Errorf("skill '%s' not found", skillName)
	}
	
	doc, err = tm.formatSkillDocumentation(skillDef)
	if err != nil {
		return "", fmt.Errorf("failed to format skill documentation: %w", err)
	}
	
	// Cache the documentation
	tm.docsMutex.Lock()
	tm.skillDocs[skillName] = doc
	tm.docsMutex.Unlock()
	
	return doc, nil
}

// getSkillDocumentation gets the documentation for a skill
func (tm *ToolManager) getSkillDocumentation(ctx context.Context, skillName string) (string, error) {
	skills, err := tm.skillExecutor.GetAvailableSkills()
	if err != nil {
		return "", fmt.Errorf("failed to load skills: %w", err)
	}
	
	skillDef, exists := skills[skillName]
	if !exists {
		return "", fmt.Errorf("skill '%s' not found", skillName)
	}
	
	return tm.formatSkillDocumentation(skillDef)
}

// formatSkillDocumentation formats the skill documentation
func (tm *ToolManager) formatSkillDocumentation(skillDef SkillDefinition) (string, error) {
	var result strings.Builder
	
	result.WriteString(fmt.Sprintf("Skill: %s\n", skillDef.Name))
	result.WriteString(fmt.Sprintf("Description: %s\n", skillDef.Description))
	result.WriteString("Parameters:\n")
	
	for _, input := range skillDef.Inputs {
		result.WriteString(fmt.Sprintf("  - %s: %s\n", input.Name, input.Description))
	}
	
	result.WriteString("Execution Type: ")
	if skillDef.Execution.Type != "" {
		result.WriteString(skillDef.Execution.Type)
	} else {
		result.WriteString("Not specified")
	}
	result.WriteString("\n")
	
	if len(skillDef.Outputs) > 0 {
		result.WriteString("Outputs:\n")
		for _, output := range skillDef.Outputs {
			result.WriteString(fmt.Sprintf("  - %s: %s\n", output.Name, output.Description))
		}
	}
	
	return result.String(), nil
}

// ClearCache clears the documentation cache
func (tm *ToolManager) ClearCache() {
	tm.docsMutex.Lock()
	defer tm.docsMutex.Unlock()
	tm.skillDocs = make(map[string]string)
}