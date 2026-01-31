package tools_test

import (
	"context"
	"testing"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/ai/tools"
)

func TestGetRecentCriticalEvents(t *testing.T) {
	ctx := context.Background()
	
	req := tools.GetRecentCriticalEventsRequest{
		TimeRange:    "last_hour",
		Severity:     "all",
		ResourceType: "all",
	}

	result, err := tools.GetRecentCriticalEvents(ctx, req)
	if err != nil {
		t.Errorf("GetRecentCriticalEvents returned error: %v", err)
	}

	// Check that we get a meaningful response
	if result.Summary == "" {
		t.Error("Expected summary to be populated")
	}

	// The events list might be empty depending on the environment,
	// but the function should still return without error
	t.Logf("Retrieved %d events", len(result.Events))
	t.Logf("Summary: %s", result.Summary)
}

func TestGetRecentCriticalEventsJSON(t *testing.T) {
	// Test with empty JSON (this might trigger the "unexpected end of JSON input" error)
	emptyJSON := "{}"
	
	result, err := tools.GetRecentCriticalEventsJSON(emptyJSON)
	if err != nil {
		t.Logf("GetRecentCriticalEventsJSON with empty JSON returned error: %v", err)
		// This is expected behavior for invalid JSON
	}

	// Test with valid JSON
	validJSON := `{"time_range":"last_hour","severity":"all","resource_type":"all"}`
	
	result, err = tools.GetRecentCriticalEventsJSON(validJSON)
	if err != nil {
		t.Errorf("GetRecentCriticalEventsJSON with valid JSON returned error: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result for valid JSON")
	}

	t.Logf("Valid JSON test result: %s", result[:min(len(result), 100)]) // Log first 100 chars
}

func TestFormatCriticalEventsResult(t *testing.T) {
	// Test with empty result
	emptyResult := tools.GetRecentCriticalEventsResponse{}
	formatted := tools.FormatCriticalEventsResult(emptyResult)
	
	if formatted == "" {
		t.Error("Expected non-empty formatted result even for empty events")
	}

	// Test with sample data
	sampleResult := tools.GetRecentCriticalEventsResponse{
		Events: []tools.CriticalEvent{
			{
				Type:      "pod",
				Name:      "test-pod",
				Namespace: "default",
				Timestamp: tools.CriticalEvent{}.Timestamp, // Will be zero time
				Severity:  "critical",
				Message:   "Test critical event",
			},
		},
		Summary:       "Test summary",
		Recommendations: []string{"Test recommendation"},
	}
	
	formatted = tools.FormatCriticalEventsResult(sampleResult)
	if formatted == "" {
		t.Error("Expected non-empty formatted result for sample data")
	}

	t.Logf("Sample data formatted result length: %d", len(formatted))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}