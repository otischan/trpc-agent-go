package basic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// AggregationHook is a logrus hook that writes critical events in a format suitable for aggregation
type AggregationHook struct {
	basicLogPath string
}

// NewAggregationHook creates a new aggregation hook
func NewAggregationHook(basicLogPath string) *AggregationHook {
	return &AggregationHook{
		basicLogPath: basicLogPath,
	}
}

// Fire executes the hook when a log entry is made
func (hook *AggregationHook) Fire(entry *logrus.Entry) error {
	// Check if this is a critical event that needs to be aggregated
	if entry.Message == "CRITICAL_EVENT" {
		objType, _ := entry.Data["objType"].(string)
		objName, _ := entry.Data["objName"].(string)
		eventType, _ := entry.Data["eventType"].(string)
		message, _ := entry.Data["message"].(string)
		namespace, _ := entry.Data["namespace"].(string)

		if objType != "" && objName != "" && eventType != "" && message != "" {
			hook.writeCriticalEventForAggregation(namespace, objType, objName, eventType, message, entry.Time)
		}
	}

	return nil
}

// writeCriticalEventForAggregation writes critical events in a format suitable for aggregation
func (hook *AggregationHook) writeCriticalEventForAggregation(namespace, objType, objName, eventType, message string, timestamp time.Time) {
	// Format the log entry for easy parsing during aggregation
	logEntry := fmt.Sprintf("CRITICAL|%s|%s|%s|%s|%s|%s\n",
		timestamp.Format(time.RFC3339),
		namespace,
		objType,
		objName,
		eventType,
		strings.ReplaceAll(message, "|", "_"), // Replace pipe character to avoid parsing issues
	)

	// Write to a dedicated aggregation-ready file
	aggregationLogPath := filepath.Join(hook.basicLogPath, "for_aggregation.log")
	file, err := os.OpenFile(aggregationLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer file.Close()
		file.WriteString(logEntry)
	}
}

// Levels returns the log levels that this hook applies to
func (hook *AggregationHook) Levels() []logrus.Level {
	return logrus.AllLevels
}