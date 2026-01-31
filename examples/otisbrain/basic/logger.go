package basic

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// BasicLogger wraps logrus logger with custom output to basic path
type BasicLogger struct {
	*logrus.Logger
	basicLogPath string
}

// NewBasicLogger creates a new logger that outputs to the basic path
func NewBasicLogger(logLevel string) (*BasicLogger, error) {
	// Create basic logs directory if it doesn't exist
	basicLogPath := "logs/basic"
	if err := os.MkdirAll(basicLogPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create basic logs directory: %w", err)
	}

	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Create file for basic logs
	logFileName := filepath.Join(basicLogPath, fmt.Sprintf("basic_%s.log", time.Now().Format("20060102_150405")))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open basic log file: %w", err)
	}

	// Set output to both file and console
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger.SetOutput(multiWriter)

	// Set formatter
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: time.RFC3339,
		FullTimestamp:   true,
	})

	// Add aggregation hook
	aggregationHook := NewAggregationHook(basicLogPath)
	logger.AddHook(aggregationHook)

	return &BasicLogger{
		Logger:       logger,
		basicLogPath: basicLogPath,
	}, nil
}

// WriteCriticalEvent writes critical events to a dedicated file in the basic path
func (bl *BasicLogger) WriteCriticalEvent(objType, objName, eventType, message string) {
	bl.Errorf("CRITICAL_EVENT_LOG - Type: %s, Name: %s, Event: %s, Message: %s",
		objType, objName, eventType, message)

	// Also write to a dedicated critical events file
	criticalLogPath := filepath.Join(bl.basicLogPath, "critical_events.log")
	criticalLogFile, err := os.OpenFile(criticalLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer criticalLogFile.Close()

		// Create a log entry with the desired fields and message
		entry := &logrus.Entry{
			Logger: bl.Logger,
			Data: logrus.Fields{
				"type":    objType,
				"name":    objName,
				"event":   eventType,
				"message": message,
			},
			Level: logrus.ErrorLevel,
			Time:  time.Now(),
			Message: "CRITICAL_EVENT_LOG",
		}

		serialized, _ := bl.Logger.Formatter.Format(entry)
		criticalLogFile.Write(serialized)
	}
}

// WriteCriticalEventForAggregation writes critical events in a format suitable for aggregation
func (bl *BasicLogger) WriteCriticalEventForAggregation(objType, objName, eventType, message string) {
	// Format the log entry for easy parsing during aggregation
	logEntry := fmt.Sprintf("CRITICAL|%s|%s|%s|%s|%s\n",
		time.Now().Format(time.RFC3339),
		objType,
		objName,
		eventType,
		strings.ReplaceAll(message, "|", "_"), // Replace pipe character to avoid parsing issues
	)

	// Write to a dedicated aggregation-ready file
	aggregationLogPath := filepath.Join(bl.basicLogPath, "for_aggregation.log")
	file, err := os.OpenFile(aggregationLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer file.Close()
		file.WriteString(logEntry)
	}
}

// GetBasicLogPath returns the path to basic logs
func (bl *BasicLogger) GetBasicLogPath() string {
	return bl.basicLogPath
}

// GetLogFiles returns all log files in the basic path that match the given pattern
func (bl *BasicLogger) GetLogFiles(pattern string) ([]string, error) {
	patternPath := filepath.Join(bl.basicLogPath, pattern)
	files, err := filepath.Glob(patternPath)
	if err != nil {
		return nil, err
	}
	
	// Filter out directories
	var logFiles []string
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			logFiles = append(logFiles, file)
		}
	}
	
	return logFiles, nil
}

// ParseTimeFromFileName extracts timestamp from log filename
func (bl *BasicLogger) ParseTimeFromFileName(filename string) (time.Time, error) {
	base := filepath.Base(filename)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// Expected format: basic_YYYYMMDD_HHMMSS.log
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid filename format: %s", filename)
	}

	timestampStr := parts[1] // YYYYMMDD_HHMMSS

	// Parse the timestamp
	return time.Parse("20060102_150405", timestampStr)
}