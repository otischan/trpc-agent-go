package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolHandler handles tools according to the new priority order
type ToolHandler struct {
	chat *AIChat
}

// NewToolHandler creates a new tool handler
func NewToolHandler(chat *AIChat) *ToolHandler {
	return &ToolHandler{
		chat: chat,
	}
}

// HandleTool handles a tool call according to the priority order:
// 1. First try MCP tools if available
// 2. For skills, first load and read the complete skill documentation using skill_load/skill_list_doc before making decisions
// 3. Then execute skills directly using skill_run when appropriate
func (th *ToolHandler) HandleTool(ctx context.Context, toolName string, args json.RawMessage) (string, error) {
	// Parse arguments
	var params map[string]interface{}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to parse tool arguments: %w", err)
		}
	} else {
		params = make(map[string]interface{})
	}

	// Check if this is an MCP-related tool first
	// For MCP tools, we rely on the framework's built-in MCP tool handling
	// So we'll return an error to indicate this should be handled elsewhere if it's an MCP tool
	if th.isMCPRelatedTool(toolName) {
		// MCP tools are handled by the framework's MCP toolsets
		return "", fmt.Errorf("MCP tool '%s' should be handled by framework", toolName)
	}

	// Check if this is a documentation request
	if th.isDocRequest(toolName) {
		return th.chat.ExecuteToolByPriority(ctx, toolName, params)
	}

	// Check if this is a direct skill execution request
	if th.isSkillRunRequest(toolName) {
		return th.chat.ExecuteToolByPriority(ctx, toolName, params)
	}

	// For regular skills, we should first load documentation before execution
	// This enforces the new priority order: load docs first, then execute
	if th.isRegularSkill(toolName) {
		// First, load the skill documentation
		docParams := map[string]interface{}{
			"skill_name": toolName,
		}

		_, err := th.chat.ExecuteToolByPriority(ctx, "skill_load_"+toolName, docParams)
		if err != nil {
			// If documentation loading fails, we can still try to execute the skill
			// but log the issue
			fmt.Printf("Warning: Could not load documentation for skill '%s': %v\n", toolName, err)
		} else {
			// Documentation loaded successfully, now execute the skill
			fmt.Printf("Loaded documentation for skill '%s', proceeding with execution\n", toolName)
		}

		// Now execute the skill with the original parameters
		return th.chat.ExecuteToolByPriority(ctx, "skill_run_"+toolName, params)
	}

	// Default to direct execution if it doesn't match any specific pattern
	return th.chat.ExecuteToolByPriority(ctx, toolName, params)
}

// isMCPRelatedTool checks if the tool is related to MCP
func (th *ToolHandler) isMCPRelatedTool(toolName string) bool {
	// Check if any MCP toolsets contain this tool
	for _, toolSet := range th.chat.mcpToolSets {
		if toolSet != nil {
			// We can't directly check if a tool exists in the toolset without initializing it
			// So we'll use naming convention as a heuristic
			toolNameLower := strings.ToLower(toolName)
			if strings.Contains(toolNameLower, "mcp") || strings.Contains(toolNameLower, "k8s") {
				return true
			}
		}
	}
	return false
}

// isDocRequest checks if this is a documentation request
func (th *ToolHandler) isDocRequest(toolName string) bool {
	return strings.HasPrefix(toolName, "skill_load") ||
		strings.HasPrefix(toolName, "skill_list_doc") ||
		strings.Contains(toolName, "doc") ||
		strings.Contains(toolName, "describe")
}

// isSkillRunRequest checks if this is a direct skill execution request
func (th *ToolHandler) isSkillRunRequest(toolName string) bool {
	return strings.HasPrefix(toolName, "skill_run") ||
		strings.HasPrefix(toolName, "execute_") ||
		strings.HasPrefix(toolName, "run_")
}

// isRegularSkill checks if this is a regular skill that should follow the doc-first approach
func (th *ToolHandler) isRegularSkill(toolName string) bool {
	// Check if this matches known skill names
	// We'll assume any skill that isn't explicitly a doc or run request is a regular skill
	return !th.isDocRequest(toolName) && !th.isSkillRunRequest(toolName) && !th.isMCPRelatedTool(toolName)
}
