package basic

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// LogAggregator handles aggregation of logs every 10 minutes
type LogAggregator struct {
	logger          *logrus.Logger
	basicLogPath    string
	interval        time.Duration
	stopCh          chan struct{}
	mutex           sync.Mutex
	lastProcessedAt time.Time
}

// NewLogAggregator creates a new log aggregator with default 10 minute interval
func NewLogAggregator(logger *logrus.Logger, basicLogPath string) *LogAggregator {
	return &LogAggregator{
		logger:          logger,
		basicLogPath:    basicLogPath,
		interval:        10 * time.Minute, // 10 minutes as specified in requirements
		stopCh:          make(chan struct{}),
		lastProcessedAt: time.Now().Add(-10 * time.Minute), // Initialize to 10 minutes ago
	}
}

// NewLogAggregatorWithInterval creates a new log aggregator with custom interval
func NewLogAggregatorWithInterval(logger *logrus.Logger, basicLogPath string, interval time.Duration) *LogAggregator {
	return &LogAggregator{
		logger:          logger,
		basicLogPath:    basicLogPath,
		interval:        interval,
		stopCh:          make(chan struct{}),
		lastProcessedAt: time.Now().Add(-interval), // Initialize to interval ago
	}
}

// Start starts the log aggregation process
func (la *LogAggregator) Start() {
	go la.run()
}

// Stop stops the log aggregation process
func (la *LogAggregator) Stop() {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	// Check if already closed to prevent panic
	select {
	case <-la.stopCh:
		// Channel already closed
		return
	default:
		// Channel not closed yet
		close(la.stopCh)
	}
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

	// Use the last processed time as the start time to avoid gaps/overlaps
	la.mutex.Lock()
	startTime := la.lastProcessedAt
	endTime := time.Now()
	la.lastProcessedAt = endTime
	la.mutex.Unlock()

	// Read aggregation-ready log files from all namespace directories
	aggregationFiles, err := la.getLogFilesForAggregation(startTime, endTime)
	if err != nil {
		la.logger.Errorf("Failed to get aggregation files: %v", err)
		return
	}

	// Also look for log files in namespace-specific directories
	namespaceLogFiles, err := la.getNamespaceLogFilesForAggregation(startTime, endTime)
	if err != nil {
		la.logger.Errorf("Failed to get namespace log files: %v", err)
		// Continue with just the basic aggregation files
	} else {
		aggregationFiles = append(aggregationFiles, namespaceLogFiles...)
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

// getNamespaceLogFilesForAggregation gets log files from namespace-specific directories
func (la *LogAggregator) getNamespaceLogFilesForAggregation(startTime, endTime time.Time) ([]string, error) {
	var files []string

	// Look for basic logs in namespace-specific directories
	logsDir := filepath.Join("logs", "basic")

	// Read all directories in the logs/basic directory
	dirEntries, err := os.ReadDir(logsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs directory: %w", err)
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			// Look for log files in namespace-specific directory
			nsLogPath := filepath.Join(logsDir, entry.Name())

			// Look for the aggregation-ready log file in the namespace directory
			// Prioritize for_aggregation.log as it contains properly formatted events
			nsAggregationPattern := filepath.Join(nsLogPath, "for_aggregation.log")
			if _, err := os.Stat(nsAggregationPattern); err == nil {
				files = append(files, nsAggregationPattern)
			} else {
				// Only fall back to basic.log if for_aggregation.log doesn't exist
				// This avoids double-processing of events
				nsBasicPattern := filepath.Join(nsLogPath, "basic.log")
				if _, err := os.Stat(nsBasicPattern); err == nil {
					files = append(files, nsBasicPattern)
				}
			}
		}
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
		la.logger.Errorf("Failed to open file %s: %v", filePath, err)
		return nil, err
	}
	defer file.Close()

	var events []CriticalEvent
	scanner := bufio.NewScanner(file)

	// Extract namespace from file path
	namespace := ExtractNamespaceFromPath(filePath)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Expected format: CRITICAL|timestamp|namespace|type|name|event|message
		// This is the primary format for critical events and should be prioritized
		parts := strings.Split(line, "|")
		if len(parts) >= 7 {
			if parts[0] == "CRITICAL" {
				timestampStr := parts[1]
				timestamp, err := time.Parse(time.RFC3339, timestampStr)
				if err != nil {
					la.logger.Warnf("Failed to parse timestamp '%s' in file %s at line %d: %v", timestampStr, filePath, lineNum, err)
					continue
				}

				// Check if the event falls within our time window
				if timestamp.Before(startTime) || timestamp.After(endTime) {
					continue
				}

				// Use namespace from the log entry if available, otherwise use extracted namespace
				eventNamespace := parts[2]
				if eventNamespace == "" {
					eventNamespace = namespace
				}

				event := CriticalEvent{
					Timestamp: timestamp,
					Namespace: eventNamespace,
					Type:      parts[3],
					Name:      parts[4],
					Event:     parts[5],
					Message:   parts[6],
				}

				events = append(events, event)
			}
			// Skip processing logrus format if it's a properly formatted CRITICAL line
			continue
		}

		// Only process logrus format logs for files that are known to contain them
		// Skip this for for_aggregation.log files which use the | delimiter format
		if strings.HasSuffix(filePath, "for_aggregation.log") {
			continue
		}

		// Handle logrus format logs that might appear in basic.log files
		// Only attempt to parse lines that look like they might be critical events
		if strings.Contains(line, "CRITICAL_EVENT") || strings.Contains(line, "CRITICAL") || strings.Contains(line, "ERROR") {
			event, err := parseLogrusLine(line, namespace)
			if err != nil {
				// Only log warning for lines that appear to be critical events but fail to parse
				if strings.Contains(line, "CRITICAL_EVENT") {
					la.logger.Warnf("Failed to parse logrus line in file %s at line %d: %v", filePath, lineNum, err)
				}
			} else if !event.Timestamp.Before(startTime) && !event.Timestamp.After(endTime) {
				events = append(events, event)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		la.logger.Errorf("Error reading file %s: %v", filePath, err)
		return events, err
	}

	return events, nil
}

// parseLogrusLine parses a logrus-formatted log line
func parseLogrusLine(line, defaultNamespace string) (CriticalEvent, error) {
	event := CriticalEvent{
		Namespace: defaultNamespace,
	}

	// Check if the line contains CRITICAL_EVENT to identify critical events
	if !strings.Contains(line, "CRITICAL_EVENT") {
		return CriticalEvent{}, fmt.Errorf("not a critical event log line")
	}

	// Parse logrus structured format: time="..." level=info msg="..."
	// Extract timestamp from time field
	timeStart := strings.Index(line, `time="`)
	if timeStart != -1 {
		timeStart += len(`time="`)
		timeEnd := strings.Index(line[timeStart:], `"`)
		if timeEnd != -1 {
			timeStr := line[timeStart : timeStart+timeEnd]
			timestamp, err := time.Parse("2006-01-02T15:04:05Z07:00", timeStr)
			if err != nil {
				// Try RFC3339 format
				timestamp, err = time.Parse(time.RFC3339, timeStr)
			}
			if err == nil {
				event.Timestamp = timestamp
			}
		}
	}

	// Extract namespace from the log fields - supports both "namespace":"value" and namespace=value formats
	nsStart := strings.Index(line, `namespace":"`)
	if nsStart != -1 {
		nsStart += len(`namespace":"`)
		nsEnd := strings.Index(line[nsStart:], `"`)
		if nsEnd != -1 {
			event.Namespace = line[nsStart : nsStart+nsEnd]
		}
	} else {
		// Try the format: namespace=value
		nsStart = strings.Index(line, "namespace=")
		if nsStart != -1 {
			nsStart += len("namespace=")
			// Find next space or end of line
			nsEnd := strings.IndexAny(line[nsStart:], " \n\t\r")
			if nsEnd == -1 {
				nsEnd = len(line) - nsStart
			}
			event.Namespace = strings.Trim(line[nsStart:nsStart+nsEnd], `"`)
		}
	}

	// Extract type from the log fields - supports both "objType":"value" and objType=value formats
	typeStart := strings.Index(line, `objType":"`)
	if typeStart != -1 {
		typeStart += len(`objType":"`)
		typeEnd := strings.Index(line[typeStart:], `"`)
		if typeEnd != -1 {
			event.Type = line[typeStart : typeStart+typeEnd]
		}
	} else {
		// Try the format: objType=value
		typeStart = strings.Index(line, "objType=")
		if typeStart != -1 {
			typeStart += len("objType=")
			// Find next space or end of line
			typeEnd := strings.IndexAny(line[typeStart:], " \n\t\r")
			if typeEnd == -1 {
				typeEnd = len(line) - typeStart
			}
			event.Type = strings.Trim(line[typeStart:typeStart+typeEnd], `"`)
		}
	}

	// Extract name from the log fields - supports both "objName":"value" and objName=value formats
	nameStart := strings.Index(line, `objName":"`)
	if nameStart != -1 {
		nameStart += len(`objName":"`)
		nameEnd := strings.Index(line[nameStart:], `"`)
		if nameEnd != -1 {
			event.Name = line[nameStart : nameStart+nameEnd]
		}
	} else {
		// Try the format: objName=value
		nameStart = strings.Index(line, "objName=")
		if nameStart != -1 {
			nameStart += len("objName=")
			// Find next space or end of line
			nameEnd := strings.IndexAny(line[nameStart:], " \n\t\r")
			if nameEnd == -1 {
				nameEnd = len(line) - nameStart
			}
			event.Name = strings.Trim(line[nameStart:nameStart+nameEnd], `"`)
		}
	}

	// Extract event from the log fields - supports both "eventType":"value" and eventType=value formats
	eventStart := strings.Index(line, `eventType":"`)
	if eventStart != -1 {
		eventStart += len(`eventType":"`)
		eventEnd := strings.Index(line[eventStart:], `"`)
		if eventEnd != -1 {
			event.Event = line[eventStart : eventStart+eventEnd]
		}
	} else {
		// Try the format: eventType=value
		eventStart = strings.Index(line, "eventType=")
		if eventStart != -1 {
			eventStart += len("eventType=")
			// Find next space or end of line
			eventEnd := strings.IndexAny(line[eventStart:], " \n\t\r")
			if eventEnd == -1 {
				eventEnd = len(line) - eventStart
			}
			event.Event = strings.Trim(line[eventStart:eventStart+eventEnd], `"`)
		}
	}

	// Extract message from the log fields - supports both "message":"value" and message="value" formats
	msgStart := strings.Index(line, `message":"`)
	if msgStart != -1 {
		msgStart += len(`message":"`)
		msgEnd := strings.Index(line[msgStart:], `"`)
		if msgEnd != -1 {
			event.Message = line[msgStart : msgStart+msgEnd]
		}
	} else {
		// Try the format: message="value"
		msgStart = strings.Index(line, `message="`)
		if msgStart != -1 {
			msgStart += len(`message="`)
			msgEnd := strings.Index(line[msgStart:], `"`)
			if msgEnd != -1 {
				event.Message = line[msgStart : msgStart+msgEnd]
			}
		} else {
			// Try the format: msg="value"
			msgStart = strings.Index(line, `msg="`)
			if msgStart != -1 {
				msgStart += len(`msg="`)
				msgEnd := strings.Index(line[msgStart:], `"`)
				if msgEnd != -1 {
					event.Message = line[msgStart : msgStart+msgEnd]
				}
			}
		}
	}

	// If we have at least a timestamp and a message, consider it a valid event
	if !event.Timestamp.IsZero() && (event.Message != "" || event.Event != "") {
		return event, nil
	}

	// If we couldn't parse the structured log, try to extract basic info from the message
	// Check if the line contains ERROR or CRITICAL keywords
	if strings.Contains(line, "ERROR") || strings.Contains(line, "CRITICAL") {
		// Extract message from the msg field if possible
		msgStart = strings.Index(line, `msg="`)
		if msgStart != -1 {
			msgStart += len(`msg="`)
			msgEnd := strings.Index(line[msgStart:], `"`)
			if msgEnd != -1 {
				event.Message = line[msgStart : msgStart+msgEnd]
				// If we have a message and timestamp, return the event
				if !event.Timestamp.IsZero() {
					return event, nil
				}
			}
		}
	}

	return CriticalEvent{}, fmt.Errorf("could not parse log line")
}

// CriticalEvent represents a critical event parsed from logs
type CriticalEvent struct {
	Timestamp time.Time
	Namespace string
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

	// Group events by namespace first
	eventsByNamespace := make(map[string][]CriticalEvent)
	for _, event := range events {
		eventsByNamespace[event.Namespace] = append(eventsByNamespace[event.Namespace], event)
	}

	// Process each namespace
	for namespace, nsEvents := range eventsByNamespace {
		summary.WriteString(fmt.Sprintf("## Namespace: %s\n", namespace))

		// Group events by type for easier analysis
		eventsByType := make(map[string][]CriticalEvent)
		for _, event := range nsEvents {
			eventsByType[event.Type] = append(eventsByType[event.Type], event)
		}

		// Write grouped events
		for eventType, typeEvents := range eventsByType {
			summary.WriteString(fmt.Sprintf("### %s Events\n", strings.Title(strings.ToLower(eventType))))

			// Group by name within type
			eventsByName := make(map[string][]CriticalEvent)
			for _, event := range typeEvents {
				eventsByName[event.Name] = append(eventsByName[event.Name], event)
			}

			for name, nameEvents := range eventsByName {
				summary.WriteString(fmt.Sprintf("#### %s\n", name))

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
		summary.WriteString("\n")
	}

	// Add a section for AI analysis
	summary.WriteString("# AI Analysis Recommendations\n")
	summary.WriteString("- Review the above critical events for patterns or recurring issues\n")
	summary.WriteString("- Investigate root causes of high-frequency events\n")
	summary.WriteString("- Consider adjusting alert thresholds if appropriate\n")
	summary.WriteString("- Plan remediation actions for persistent issues\n")
	summary.WriteString("- Compare event patterns across different namespaces\n")

	return summary.String()
}
