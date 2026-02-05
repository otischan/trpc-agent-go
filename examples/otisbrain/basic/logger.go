package basic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// BasicLogger wraps logrus logger with custom output to basic path
type BasicLogger struct {
	*logrus.Logger
	basicLogPath string
}

// NewBasicLogger creates a new logger that outputs to the basic path
func NewBasicLogger(logLevel string) (*BasicLogger, error) {
	return NewBasicLoggerWithNamespace(logLevel, "default")
}

// NewBasicLoggerWithNamespace creates a new logger that outputs to a namespace-specific path
func NewBasicLoggerWithNamespace(logLevel, namespace string) (*BasicLogger, error) {
	// Create basic logs directory if it doesn't exist
	basicLogPath := filepath.Join("logs", "basic", namespace)
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

	// Set output to file with rotation (for background tasks)
	logFileName := filepath.Join(basicLogPath, "basic.log")
	logger.SetOutput(&lumberjack.Logger{
		Filename:   logFileName,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	})

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

// NewBasicLoggerWithCustomPath creates a new basic logger with a custom path
func NewBasicLoggerWithCustomPath(logLevel string, customPath string) (*BasicLogger, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(customPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Set output to file with rotation (for background tasks)
	logFileName := filepath.Join(customPath, "basic.log")
	logger.SetOutput(&lumberjack.Logger{
		Filename:   logFileName,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	})

	// Set formatter
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: time.RFC3339,
		FullTimestamp:   true,
	})

	// Add aggregation hook
	aggregationHook := NewAggregationHook(customPath)
	logger.AddHook(aggregationHook)

	return &BasicLogger{
		Logger:       logger,
		basicLogPath: customPath,
	}, nil
}

// NewConsoleLogger creates a new logger that outputs only to console (for chat interface)
func NewConsoleLogger(logLevel string) (*logrus.Logger, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set output to console only
	logger.SetOutput(os.Stdout)

	// Set formatter for console
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: time.RFC3339,
		FullTimestamp:   true,
		ForceColors:     true, // Enable colors for console output
	})

	return logger, nil
}

// NewFileOnlyLogger creates a new logger that outputs only to files (for background tasks)
func NewFileOnlyLogger(logLevel string, logPath string, fileName string) (*logrus.Logger, error) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set output to file with rotation
	logFilePath := filepath.Join(logPath, fileName)
	logger.SetOutput(&lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	})

	// Set formatter
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: time.RFC3339,
		FullTimestamp:   true,
	})

	return logger, nil
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
			Level:   logrus.ErrorLevel,
			Time:    time.Now(),
			Message: "CRITICAL_EVENT_LOG",
		}

		serialized, _ := bl.Logger.Formatter.Format(entry)
		criticalLogFile.Write(serialized)
	}
}

// WriteCriticalEventForAggregation writes critical events in a format suitable for aggregation
func (bl *BasicLogger) WriteCriticalEventForAggregation(objType, objName, eventType, message string) {
	// Extract namespace from basicLogPath (logs/basic/{namespace}/)
	namespace := ExtractNamespaceFromPath(bl.basicLogPath)

	// Format the log entry for easy parsing during aggregation
	logEntry := fmt.Sprintf("CRITICAL|%s|%s|%s|%s|%s|%s\n",
		time.Now().Format(time.RFC3339),
		namespace,
		objType,
		objName,
		eventType,
		strings.ReplaceAll(message, "|", "_"), // Replace pipe character to avoid parsing issues
	)

	// Write to a dedicated aggregation-ready file
	aggregationLogPath := filepath.Join(bl.basicLogPath, "for_aggregation.log")
	file, err := os.OpenFile(aggregationLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		bl.Logger.Errorf("Failed to open aggregation log file %s: %v", aggregationLogPath, err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(logEntry); err != nil {
		bl.Logger.Errorf("Failed to write to aggregation log file %s: %v", aggregationLogPath, err)
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
