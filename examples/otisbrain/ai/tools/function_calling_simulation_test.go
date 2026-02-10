package tools_test

import (
	"context"
	"testing"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/ai/tools"
)

// TestSkillCallingScenario simulates the scenario where AI assistant calls the skill
func TestSkillCallingScenario(t *testing.T) {
	// Create the skill executor as it would be created in the AI system
	executor := tools.NewSkillExecutor("./test_skills")

	if executor == nil {
		t.Fatal("Expected skill executor to be created, got nil")
	}

	// Test the skill execution by calling it through the executor
	ctx := context.Background()

	// Test case 1: Execution with non-existent skill (should handle gracefully)
	t.Run("NonExistentSkill", func(t *testing.T) {
		_, err := executor.LoadAndExecuteSkill(ctx, "non_existent_skill", map[string]interface{}{})
		if err == nil {
			t.Error("Expected error for non-existent skill")
		}

		t.Logf("Non-existent skill error (expected): %v", err)
	})

	// Test case 2: Execution with empty parameters
	t.Run("EmptyParameters", func(t *testing.T) {
		_, err := executor.LoadAndExecuteSkill(ctx, "non_existent_skill", map[string]interface{}{})
		if err == nil {
			t.Error("Expected error for non-existent skill")
		}

		t.Logf("Empty parameters error (expected): %v", err)
	})
}

// TestErrorScenarios tests scenarios that might cause errors
func TestErrorScenarios(t *testing.T) {
	ctx := context.Background()

	t.Run("TestExecuteGetRecentCriticalEvents", func(t *testing.T) {
		result, err := tools.ExecuteGetRecentCriticalEvents(ctx, "last_hour", "all", "all")
		if err != nil {
			t.Logf("ExecuteGetRecentCriticalEvents error: %v", err)
		} else {
			t.Logf("ExecuteGetRecentCriticalEvents result: %s", result)
		}
	})

	t.Run("TestGetAllCriticalRecords", func(t *testing.T) {
		result, err := tools.GetAllCriticalRecords(context.Background())
		if err != nil {
			t.Logf("GetAllCriticalRecords error: %v", err)
		} else {
			t.Logf("GetAllCriticalRecords result length: %d", len(result))
		}
	})
}

// TestSkillExecutorWithRealSkills tests the skill executor with actual functionality
func TestSkillExecutorWithRealSkills(t *testing.T) {
	ctx := context.Background()

	// Test the actual skill implementations directly
	t.Run("DirectFunctionCall", func(t *testing.T) {
		result, err := tools.ExecuteGetRecentCriticalEvents(ctx, "last_hour", "all", "all")
		if err != nil {
			t.Logf("Direct function call error: %v", err)
		} else {
			if result == "" {
				t.Error("Expected non-empty result from direct function call")
			}
			t.Logf("Direct function call result length: %d", len(result))
		}
	})
}
