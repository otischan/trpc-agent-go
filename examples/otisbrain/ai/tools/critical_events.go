package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CriticalEvent represents a critical event in the system
type CriticalEvent struct {
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
}

// GetRecentCriticalEventsRequest represents the request parameters
type GetRecentCriticalEventsRequest struct {
	TimeRange   string `json:"time_range"`
	Severity    string `json:"severity"`
	ResourceType string `json:"resource_type"`
}

// GetRecentCriticalEventsResponse represents the response
type GetRecentCriticalEventsResponse struct {
	Events        []CriticalEvent `json:"events"`
	Summary       string          `json:"summary"`
	Recommendations []string       `json:"recommendations"`
}

// GetRecentCriticalEvents retrieves recent critical events based on the specified criteria
func GetRecentCriticalEvents(ctx context.Context, req GetRecentCriticalEventsRequest) (GetRecentCriticalEventsResponse, error) {
	// Determine the time threshold based on the requested time range
	timeThreshold := time.Now()
	switch req.TimeRange {
	case "last_hour":
		timeThreshold = time.Now().Add(-1 * time.Hour)
	case "last_6_hours":
		timeThreshold = time.Now().Add(-6 * time.Hour)
	case "last_24_hours":
		timeThreshold = time.Now().Add(-24 * time.Hour)
	case "custom":
		// Use default of last hour if custom isn't properly handled
		timeThreshold = time.Now().Add(-1 * time.Hour)
	default:
		timeThreshold = time.Now().Add(-1 * time.Hour)
	}

	// Read critical events from the log files
	events, err := readCriticalEventsFromLogs(timeThreshold)
	if err != nil {
		return GetRecentCriticalEventsResponse{}, fmt.Errorf("failed to read critical events: %w", err)
	}

	// Filter events based on severity and resource type if specified
	filteredEvents := filterEvents(events, req.Severity, req.ResourceType)

	// Sort events by timestamp (newest first)
	sort.Slice(filteredEvents, func(i, j int) bool {
		return filteredEvents[i].Timestamp.After(filteredEvents[j].Timestamp)
	})

	// Generate summary
	summary := generateSummary(filteredEvents)

	// Generate recommendations based on the events
	recommendations := generateRecommendations(filteredEvents)

	return GetRecentCriticalEventsResponse{
		Events:        filteredEvents,
		Summary:       summary,
		Recommendations: recommendations,
	}, nil
}

// readCriticalEventsFromLogs reads critical events from log files
func readCriticalEventsFromLogs(after time.Time) ([]CriticalEvent, error) {
	var events []CriticalEvent

	// Define the log directory path
	logDir := filepath.Join("logs", "critical_events")
	
	// Check if the directory exists
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		// If the directory doesn't exist, return empty slice
		return events, nil
	}

	// Read all log files in the critical events directory
	files, err := ioutil.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read critical events directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".log") {
			filePath := filepath.Join(logDir, file.Name())
			fileEvents, err := parseLogFile(filePath, after)
			if err != nil {
				// Log the error but continue processing other files
				continue
			}
			events = append(events, fileEvents...)
		}
	}

	return events, nil
}

// parseLogFile parses a log file and extracts critical events after the specified time
func parseLogFile(filePath string, after time.Time) ([]CriticalEvent, error) {
	var events []CriticalEvent

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file %s: %w", filePath, err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, "CRITICAL") || strings.Contains(line, "ERROR") {
			event, err := parseLogLine(line, after)
			if err != nil {
				continue // Skip unparsable lines
			}
			if !event.Timestamp.Before(after) {
				events = append(events, event)
			}
		}
	}

	return events, nil
}

// parseLogLine parses a single log line to extract critical event information
func parseLogLine(line string, after time.Time) (CriticalEvent, error) {
	var event CriticalEvent
	
	// Parse timestamp from log line (assuming RFC3339 format)
	// Format: time="..." level=... msg="..."
	
	// Extract timestamp
	if idx := strings.Index(line, "time=\""); idx != -1 {
		start := idx + 6 // Length of "time="" 
		if end := strings.Index(line[start:], "\""); end != -1 {
			tsStr := line[start : start+end]
			if ts, err := time.Parse(time.RFC3339, tsStr); err == nil {
				event.Timestamp = ts
			} else {
				// If RFC3339 parsing fails, try other common formats
				if ts, err := time.Parse("2006-01-02T15:04:05Z07:00", tsStr); err == nil {
					event.Timestamp = ts
				} else {
					event.Timestamp = time.Now() // fallback
				}
			}
		}
	} else {
		event.Timestamp = time.Now() // fallback
	}
	
	// Extract message
	if idx := strings.Index(line, "msg=\""); idx != -1 {
		start := idx + 5 // Length of "msg="" 
		if end := strings.Index(line[start:], "\""); end != -1 {
			event.Message = line[start : start+end]
		}
	}
	
	// Determine severity based on the log level
	if strings.Contains(line, "level=error") || strings.Contains(line, "CRITICAL") {
		event.Severity = "critical"
	} else if strings.Contains(line, "level=warning") {
		event.Severity = "warning"
	} else {
		event.Severity = "info"
	}
	
	// Extract resource type and name from message
	event.Type, event.Name, event.Namespace = extractResourceInfo(event.Message)
	
	// Set default values if extraction failed
	if event.Type == "" {
		event.Type = "unknown"
	}
	if event.Name == "" {
		event.Name = "unknown"
	}
	if event.Namespace == "" {
		event.Namespace = "default"
	}
	
	return event, nil
}

// extractResourceInfo extracts resource type, name, and namespace from a message
func extractResourceInfo(message string) (resourceType, name, namespace string) {
	// Look for patterns like "namespace/resource" or "type name in namespace"
	lowerMsg := strings.ToLower(message)
	
	// Extract namespace if present
	if idx := strings.Index(lowerMsg, "namespace:"); idx != -1 {
		end := strings.Index(lowerMsg[idx:], " ") 
		if end == -1 {
			end = len(lowerMsg) - idx
		} else {
			end += idx
		}
		namespace = strings.TrimSpace(lowerMsg[idx+10:end]) // "namespace:" is 10 chars
	}
	
	// Extract resource type and name based on common patterns
	if strings.Contains(lowerMsg, "pod") {
		resourceType = "pod"
		// Look for pod name pattern: "Pod namespace/name" or similar
		if idx := strings.Index(message, "Pod "); idx != -1 {
			remainder := message[idx+4:] // "Pod " is 4 chars
			parts := strings.SplitN(remainder, "/", 2)
			if len(parts) == 2 {
				namespace = parts[0]
				name = strings.SplitN(parts[1], " ", 2)[0]
			}
		}
	} else if strings.Contains(lowerMsg, "deployment") {
		resourceType = "deployment"
		if idx := strings.Index(message, "Deployment "); idx != -1 {
			remainder := message[idx+11:] // "Deployment " is 11 chars
			parts := strings.SplitN(remainder, "/", 2)
			if len(parts) == 2 {
				namespace = parts[0]
				name = strings.SplitN(parts[1], " ", 2)[0]
			}
		}
	} else if strings.Contains(lowerMsg, "service") {
		resourceType = "service"
		if idx := strings.Index(message, "Service "); idx != -1 {
			remainder := message[idx+8:] // "Service " is 8 chars
			parts := strings.SplitN(remainder, "/", 2)
			if len(parts) == 2 {
				namespace = parts[0]
				name = strings.SplitN(parts[1], " ", 2)[0]
			}
		}
	}
	
	return resourceType, name, namespace
}

// filterEvents filters events based on severity and resource type
func filterEvents(events []CriticalEvent, severity, resourceType string) []CriticalEvent {
	if severity == "all" && resourceType == "all" {
		return events
	}
	
	var filtered []CriticalEvent
	for _, event := range events {
		severityMatch := severity == "all" || strings.EqualFold(event.Severity, severity)
		typeMatch := resourceType == "all" || strings.EqualFold(event.Type, resourceType)
		
		if severityMatch && typeMatch {
			filtered = append(filtered, event)
		}
	}
	
	return filtered
}

// generateSummary generates a summary of the events
func generateSummary(events []CriticalEvent) string {
	if len(events) == 0 {
		return "No critical events detected in the specified time range."
	}
	
	criticalCount := 0
	warningCount := 0
	podIssues := 0
	deploymentIssues := 0
	
	for _, event := range events {
		switch strings.ToLower(event.Severity) {
		case "critical":
			criticalCount++
		case "warning":
			warningCount++
		}
		
		switch strings.ToLower(event.Type) {
		case "pod":
			podIssues++
		case "deployment":
			deploymentIssues++
		}
	}
	
	summary := fmt.Sprintf(
		"Found %d critical events in the specified time range: %d critical severity, %d warning severity. "+
		"Affected resources: %d pods, %d deployments.",
		len(events), criticalCount, warningCount, podIssues, deploymentIssues)
	
	return summary
}

// generateRecommendations generates recommendations based on the events
func generateRecommendations(events []CriticalEvent) []string {
	if len(events) == 0 {
		return []string{"No issues detected. System appears healthy."}
	}
	
	var recommendations []string
	seenRecommendations := make(map[string]bool)
	
	for _, event := range events {
		var rec string
		
		// Generate recommendations based on event type and message
		if strings.Contains(strings.ToLower(event.Message), "crashloopbackoff") {
			rec = fmt.Sprintf("Pod %s in namespace %s is in CrashLoopBackOff. Check application logs and resource limits.", event.Name, event.Namespace)
		} else if strings.Contains(strings.ToLower(event.Message), "oomkilled") {
			rec = fmt.Sprintf("Container in pod %s was killed due to OutOfMemory. Consider increasing memory limits.", event.Name)
		} else if strings.Contains(strings.ToLower(event.Message), "progressdeadlineexceeded") {
			rec = fmt.Sprintf("Deployment %s rollout failed due to progress deadline exceeded. Check deployment configuration and available resources.", event.Name)
		} else if strings.Contains(strings.ToLower(event.Message), "imagepullbackoff") {
			rec = fmt.Sprintf("Pod %s failed to start due to ImagePullBackOff. Verify image name and registry credentials.", event.Name)
		} else if strings.Contains(strings.ToLower(event.Message), "failed") {
			rec = fmt.Sprintf("Resource %s in namespace %s has failed. Investigate the specific error in the logs.", event.Name, event.Namespace)
		}
		
		// Add recommendation if it's unique and not empty
		if rec != "" && !seenRecommendations[rec] {
			recommendations = append(recommendations, rec)
			seenRecommendations[rec] = true
		}
	}
	
	// If no specific recommendations were generated, add generic ones
	if len(recommendations) == 0 {
		recommendations = append(recommendations, 
			"Review the detailed event logs for more information.",
			"Consider checking cluster resource utilization.",
			"Verify application configurations and dependencies.")
	}
	
	return recommendations
}

// FormatCriticalEventsResult formats the critical events result for display to the user
func FormatCriticalEventsResult(result GetRecentCriticalEventsResponse) string {
	if len(result.Events) == 0 {
		return "No critical events were found in the specified time range.\n\n" + result.Summary
	}

	output := fmt.Sprintf("## Critical Events Summary\n%s\n\n", result.Summary)

	output += fmt.Sprintf("## Recent Critical Events (%d total)\n", len(result.Events))
	for i, event := range result.Events {
		output += fmt.Sprintf("%d. **[%s]** %s - %s/%s in namespace %s\n",
			i+1, event.Severity, event.Message, event.Type, event.Name, event.Namespace)
		output += fmt.Sprintf("   *Time:* %s\n\n", event.Timestamp.Format("2006-01-02 15:04:05"))
	}

	if len(result.Recommendations) > 0 {
		output += "## Recommendations\n"
		for i, rec := range result.Recommendations {
			output += fmt.Sprintf("%d. %s\n", i+1, rec)
		}
	}

	return output
}

// GetRecentCriticalEventsJSON is a wrapper that accepts and returns JSON
func GetRecentCriticalEventsJSON(jsonReq string) (string, error) {
	var req GetRecentCriticalEventsRequest
	if err := json.Unmarshal([]byte(jsonReq), &req); err != nil {
		return "", fmt.Errorf("invalid JSON request: %w", err)
	}

	resp, err := GetRecentCriticalEvents(context.Background(), req)
	if err != nil {
		return "", err
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonResp), nil
}