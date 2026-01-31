package basic

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// LogAggregator handles aggregation of logs every 10 minutes
type LogAggregator struct {
	logger       *logrus.Logger
	basicLogPath string
	interval     time.Duration
	stopCh       chan struct{}
}

// NewLogAggregator creates a new log aggregator with default 10 minute interval
func NewLogAggregator(logger *logrus.Logger, basicLogPath string) *LogAggregator {
	return &LogAggregator{
		logger:       logger,
		basicLogPath: basicLogPath,
		interval:     10 * time.Minute, // 10 minutes as specified in requirements
		stopCh:       make(chan struct{}),
	}
}

// NewLogAggregatorWithInterval creates a new log aggregator with custom interval
func NewLogAggregatorWithInterval(logger *logrus.Logger, basicLogPath string, interval time.Duration) *LogAggregator {
	return &LogAggregator{
		logger:       logger,
		basicLogPath: basicLogPath,
		interval:     interval,
		stopCh:       make(chan struct{}),
	}
}

// Start starts the log aggregation process
func (la *LogAggregator) Start() {
	go la.run()
}

// Stop stops the log aggregation process
func (la *LogAggregator) Stop() {
	close(la.stopCh)
}

// run runs the aggregation loop
func (la *LogAggregator) run() {
	ticker := time.NewTicker(la.interval)
	defer ticker.Stop()

	// Perform initial aggregation
	la.aggregateLogs()

	for {
		select {
		case <-ticker.C:
			la.aggregateLogs()
		case <-la.stopCh:
			la.logger.Info("Log aggregator stopped")
			return
		}
	}
}

// aggregateLogs performs the actual aggregation of logs
func (la *LogAggregator) aggregateLogs() {
	la.logger.Info("Starting log aggregation...")

	// Create critical_record directory if it doesn't exist
	criticalRecordPath := "logs/critical_record"
	if err := os.MkdirAll(criticalRecordPath, 0755); err != nil {
		la.logger.Errorf("Failed to create critical_record directory: %v", err)
		return
	}

	// Get logs from the last 10 minutes
	startTime := time.Now().Add(-la.interval)
	endTime := time.Now()

	// Read aggregation-ready log files
	aggregationFiles, err := la.getLogFilesForAggregation(startTime, endTime)
	if err != nil {
		la.logger.Errorf("Failed to get aggregation files: %v", err)
		return
	}

	// Parse and aggregate critical events
	events, err := la.parseCriticalEvents(aggregationFiles, startTime, endTime)
	if err != nil {
		la.logger.Errorf("Failed to parse critical events: %v", err)
		return
	}

	// Deduplicate events
	deduplicatedEvents := la.deduplicateEvents(events)

	// Generate summary
	summary := la.generateSummary(deduplicatedEvents, startTime, endTime)

	// Write summary to critical_record path
	outputFile := filepath.Join(criticalRecordPath, fmt.Sprintf("critical_summary_%s.txt", 
		startTime.Format("20060102_150405")))

	if err := os.WriteFile(outputFile, []byte(summary), 0644); err != nil {
		la.logger.Errorf("Failed to write critical summary to file: %v", err)
		return
	}

	la.logger.Infof("Log aggregation completed. Summary written to: %s", outputFile)
}

// getLogFilesForAggregation gets log files that fall within the specified time range
func (la *LogAggregator) getLogFilesForAggregation(startTime, endTime time.Time) ([]string, error) {
	// Look for aggregation-ready log files
	pattern := filepath.Join(la.basicLogPath, "for_aggregation.log")
	
	// Since we're appending to a single file, we just need to check if it exists
	// and if it has content within our time window
	files := []string{}
	
	if _, err := os.Stat(pattern); err == nil {
		files = append(files, pattern)
	}
	
	return files, nil
}

// parseCriticalEvents parses critical events from log files within the time range
func (la *LogAggregator) parseCriticalEvents(files []string, startTime, endTime time.Time) ([]CriticalEvent, error) {
	var events []CriticalEvent

	for _, file := range files {
		fileEvents, err := la.parseFileForCriticalEvents(file, startTime, endTime)
		if err != nil {
			la.logger.Errorf("Failed to parse file %s: %v", file, err)
			continue
		}
		events = append(events, fileEvents...)
	}

	return events, nil
}

// parseFileForCriticalEvents parses a single file for critical events within the time range
func (la *LogAggregator) parseFileForCriticalEvents(filePath string, startTime, endTime time.Time) ([]CriticalEvent, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []CriticalEvent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		
		// Expected format: CRITICAL|timestamp|type|name|event|message
		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}
		
		if parts[0] != "CRITICAL" {
			continue
		}
		
		timestampStr := parts[1]
		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			continue
		}
		
		// Check if the event falls within our time window
		if timestamp.Before(startTime) || timestamp.After(endTime) {
			continue
		}
		
		event := CriticalEvent{
			Timestamp: timestamp,
			Type:      parts[2],
			Name:      parts[3],
			Event:     parts[4],
			Message:   parts[5],
		}
		
		events = append(events, event)
	}

	return events, scanner.Err()
}

// CriticalEvent represents a critical event parsed from logs
type CriticalEvent struct {
	Timestamp time.Time
	Type      string
	Name      string
	Event     string
	Message   string
}

// deduplicateEvents removes duplicate events based on type, name, event, and message
func (la *LogAggregator) deduplicateEvents(events []CriticalEvent) []CriticalEvent {
	seen := make(map[string]bool)
	var deduplicated []CriticalEvent

	for _, event := range events {
		// Create a unique key for deduplication
		key := fmt.Sprintf("%s:%s:%s:%s", event.Type, event.Name, event.Event, event.Message)
		
		if !seen[key] {
			seen[key] = true
			deduplicated = append(deduplicated, event)
		}
	}

	return deduplicated
}

// generateSummary generates a summary of critical events
func (la *LogAggregator) generateSummary(events []CriticalEvent, startTime, endTime time.Time) string {
	if len(events) == 0 {
		return fmt.Sprintf(
			"# Critical Events Summary\n"+
				"Time Period: %s to %s\n"+
				"Total Critical Events: 0\n"+
				"No critical events detected in this period.\n",
			startTime.Format(time.RFC3339),
			endTime.Format(time.RFC3339),
		)
	}

	// Sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf(
		"# Critical Events Summary\n"+
			"Time Period: %s to %s\n"+
			"Total Critical Events: %d\n\n",
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		len(events),
	))

	// Group events by type for easier analysis
	eventsByType := make(map[string][]CriticalEvent)
	for _, event := range events {
		eventsByType[event.Type] = append(eventsByType[event.Type], event)
	}

	// Write grouped events
	for eventType, typeEvents := range eventsByType {
		summary.WriteString(fmt.Sprintf("## %s Events\n", strings.Title(strings.ToLower(eventType))))

		// Group by name within type
		eventsByName := make(map[string][]CriticalEvent)
		for _, event := range typeEvents {
			eventsByName[event.Name] = append(eventsByName[event.Name], event)
		}

		for name, nameEvents := range eventsByName {
			summary.WriteString(fmt.Sprintf("### %s\n", name))

			// Count occurrences of the same event for this object
			eventCounts := make(map[string]int)
			eventMessages := make(map[string]string)
			eventTimestamps := make(map[string][]string)

			for _, event := range nameEvents {
				key := fmt.Sprintf("%s:%s", event.Event, event.Message)
				eventCounts[key]++
				eventMessages[key] = event.Message
				if eventTimestamps[key] == nil {
					eventTimestamps[key] = []string{}
				}
				eventTimestamps[key] = append(eventTimestamps[key], event.Timestamp.Format("15:04:05"))
			}

			for key, count := range eventCounts {
				parts := strings.Split(key, ":")
				eventType := parts[0]
				message := eventMessages[key]

				// Show first and last timestamps if multiple occurrences
				timestampInfo := ""
				times := eventTimestamps[key]
				if len(times) > 1 {
					timestampInfo = fmt.Sprintf(" (first: %s, last: %s, count: %d)", times[0], times[len(times)-1], count)
				} else {
					timestampInfo = fmt.Sprintf(" (%s)", times[0])
				}

				summary.WriteString(fmt.Sprintf(
					"- **%s**: %s%s\n",
					eventType,
					message,
					timestampInfo,
				))
			}
		}
		summary.WriteString("\n")
	}

	// Add a section for AI analysis
	summary.WriteString("# AI Analysis Recommendations\n")
	summary.WriteString("- Review the above critical events for patterns or recurring issues\n")
	summary.WriteString("- Investigate root causes of high-frequency events\n")
	summary.WriteString("- Consider adjusting alert thresholds if appropriate\n")
	summary.WriteString("- Plan remediation actions for persistent issues\n")

	return summary.String()
}