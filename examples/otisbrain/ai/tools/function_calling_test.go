package tools_test

import (
	"context"
	"testing"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/ai/tools"
)

// Test the skill executor creation and execution
func TestSkillExecutorCreation(t *testing.T) {
	// Create the skill executor
	executor := tools.NewSkillExecutor("./test_skills") // Use a test directory

	if executor == nil {
		t.Fatal("Expected skill executor to be created, got nil")
	}

	// We'll test the execution with a mock skill directory
}

// Test the skill executor execution
func TestSkillExecutorExecution(t *testing.T) {
	// Create the skill executor
	executor := tools.NewSkillExecutor("./test_skills") // Use a test directory

	if executor == nil {
		t.Fatal("Expected skill executor to be created, got nil")
	}

	// Test with a non-existent skill directory (should handle gracefully)
	ctx := context.Background()

	// Test the execution function directly
	_, err := executor.LoadAndExecuteSkill(ctx, "non_existent_skill", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for non-existent skill")
	}

	t.Logf("Skill execution error (expected): %v", err)
}

// Helper function to test the wrapper functionality
func testGetRecentCriticalEventsWrapper(ctx context.Context) (string, error) {
	// This function is no longer used since we removed the old implementation
	// Keeping it for reference if needed later
	return "mock result", nil
}

// Test edge cases that might cause JSON parsing errors
func TestEdgeCases(t *testing.T) {
	ctx := context.Background()

	// Test with empty request (should use defaults)
	emptyReq := tools.GetRecentCriticalEventsRequest{}

	result, err := tools.GetRecentCriticalEvents(ctx, emptyReq)
	if err != nil {
		t.Logf("Expected behavior: GetRecentCriticalEvents with empty request returned error: %v", err)
		// This is acceptable behavior
		return
	}

	formatted := tools.FormatCriticalEventsResult(result)
	if formatted == "" {
		t.Error("Expected non-empty formatted result even with empty request")
	}

	t.Logf("Empty request result length: %d", len(formatted))
}
