package basic

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AIAgentReader provides functionality for AI agents to read critical records
type AIAgentReader struct {
	criticalRecordPath string
}

// NewAIAgentReader creates a new reader for AI agents
func NewAIAgentReader() *AIAgentReader {
	return &AIAgentReader{
		criticalRecordPath: "logs/critical_record",
	}
}

// ReadLatestCriticalSummary reads the most recent critical summary file
func (r *AIAgentReader) ReadLatestCriticalSummary() (string, error) {
	files, err := r.getSortedSummaryFiles()
	if err != nil {
		return "", fmt.Errorf("failed to get summary files: %w", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no critical summary files found")
	}

	// Read the most recent file
	latestFile := files[len(files)-1]
	content, err := os.ReadFile(latestFile)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", latestFile, err)
	}

	return string(content), nil
}

// ReadCriticalSummariesInRange reads critical summaries within a specific time range
func (r *AIAgentReader) ReadCriticalSummariesInRange(startTime, endTime time.Time) ([]string, error) {
	files, err := r.getSortedSummaryFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get summary files: %w", err)
	}

	var summaries []string
	for _, file := range files {
		// Extract timestamp from filename
		timestamp, err := r.parseTimeFromFilename(filepath.Base(file))
		if err != nil {
			continue
		}

		// Check if file timestamp is within range
		if timestamp.After(startTime) && timestamp.Before(endTime) {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			summaries = append(summaries, string(content))
		}
	}

	return summaries, nil
}

// getSortedSummaryFiles returns all summary files sorted by timestamp
func (r *AIAgentReader) getSortedSummaryFiles() ([]string, error) {
	pattern := filepath.Join(r.criticalRecordPath, "critical_summary_*.txt")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Sort files by timestamp in filename
	sort.Slice(files, func(i, j int) bool {
		timeI, err1 := r.parseTimeFromFilename(filepath.Base(files[i]))
		timeJ, err2 := r.parseTimeFromFilename(filepath.Base(files[j]))

		if err1 != nil || err2 != nil {
			return files[i] < files[j] // fallback to alphabetical sort
		}

		return timeI.Before(timeJ)
	})

	return files, nil
}

// parseTimeFromFilename extracts timestamp from filename like critical_summary_20060102_150405.txt
func (r *AIAgentReader) parseTimeFromFilename(filename string) (time.Time, error) {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Expected format: critical_summary_YYYYMMDD_HHMMSS
	parts := strings.Split(name, "_")
	if len(parts) < 3 {
		return time.Time{}, fmt.Errorf("invalid filename format: %s", filename)
	}

	// Extract the timestamp part (should be the last two elements after splitting by _)
	dateTimeStr := parts[len(parts)-2] + "_" + parts[len(parts)-1]

	// Parse the timestamp
	return time.Parse("20060102_150405", dateTimeStr)
}

// GetAllCriticalRecords returns all critical records as a single string
func (r *AIAgentReader) GetAllCriticalRecords() (string, error) {
	files, err := r.getSortedSummaryFiles()
	if err != nil {
		return "", fmt.Errorf("failed to get summary files: %w", err)
	}

	var allContent strings.Builder
	for i, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if i > 0 {
			allContent.WriteString("\n" + strings.Repeat("=", 80) + "\n\n")
		}

		allContent.WriteString(string(content))
	}

	if allContent.Len() == 0 {
		return "", fmt.Errorf("no critical records found")
	}

	return allContent.String(), nil
}
