package tools

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
)

// Actual skill implementations that connect to the real functionality

// ExecuteGetRecentCriticalEvents executes the get recent critical events skill with real functionality
func ExecuteGetRecentCriticalEvents(ctx context.Context, timeRange, severity, resourceType string) (string, error) {
	// Use the AIAgentReader to get critical events
	reader := basic.NewAIAgentReader()

	// For now, we'll use the existing reader functions
	// In a real implementation, we would apply the filters based on the parameters
	summary, err := reader.ReadLatestCriticalSummary()
	if err != nil {
		return "", fmt.Errorf("error reading critical summary: %w", err)
	}

	return summary, nil
}

// GetAllCriticalRecords executes the get all critical records skill with real functionality
func GetAllCriticalRecords(ctx context.Context) (string, error) {
	// Use the AIAgentReader to get all critical records
	reader := basic.NewAIAgentReader()
	
	records, err := reader.GetAllCriticalRecords()
	if err != nil {
		return "", fmt.Errorf("error reading critical records: %w", err)
	}

	return records, nil
}