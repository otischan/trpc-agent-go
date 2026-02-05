package basic

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// MemoryDataPoint represents a single memory usage data point
type MemoryDataPoint struct {
	Timestamp   time.Time
	PodName     string
	Namespace   string
	Container   string
	MemoryUsage int64 // in bytes
}

// MemoryCollector collects memory usage metrics
type MemoryCollector struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
	logger        *logrus.Logger
	retentionDays int
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, logger *logrus.Logger, retentionDays int) *MemoryCollector {
	return &MemoryCollector{
		clientset:     clientset,
		metricsClient: metricsClient,
		logger:        logger,
		retentionDays: retentionDays,
	}
}

// getMemoryDataFilePath returns the file path for storing memory data for a specific pod/container
func (mc *MemoryCollector) getMemoryDataFilePath(namespace, podName, container string) string {
	// Create directory structure: logs/memory_usage/{namespace}/{podName}/{container}/
	dirPath := filepath.Join("logs", "memory_usage", namespace, podName, container)

	// Ensure directory exists
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		mc.logger.Errorf("Failed to create directory %s: %v", dirPath, err)
		return ""
	}

	// Return the file path for this pod/container
	return filepath.Join(dirPath, "memory_usage.json")
}

// CollectMemoryMetrics collects memory metrics for all pods in a namespace
func (mc *MemoryCollector) CollectMemoryMetrics(namespace string) error {
	// Log before calling metrics API
	mc.logger.Infof("Attempting to collect memory metrics for namespace: %s", namespace)

	// Get pod metrics from metrics server with improved retry logic
	var podMetrics *v1beta1.PodMetricsList
	var err error

	// Retry up to 3 times with exponential backoff
	for i := 0; i < 3; i++ {
		mc.logger.Debugf("Making attempt %d to get pod metrics for namespace %s", i+1, namespace)
		podMetrics, err = mc.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(context.TODO(), metav1.ListOptions{})
		if err == nil {
			mc.logger.Debugf("Successfully retrieved pod metrics for namespace %s on attempt %d", namespace, i+1)
			break // Success, exit retry loop
		}

		// Log the specific error for debugging
		mc.logger.Errorf("Attempt %d to get pod metrics for namespace %s failed: %v", i+1, namespace, err)

		// Check if the error is related to metrics API not being ready
		errMsg := err.Error()
		if strings.Contains(errMsg, "the server could not find the requested resource") ||
			strings.Contains(errMsg, "unable to handle the request") ||
			strings.Contains(errMsg, "the server is currently unable to handle the request") {
			mc.logger.Infof("Metrics API not ready for namespace %s, attempt %d, retrying in %d seconds...", namespace, i+1, (i+1)*2)
			time.Sleep(time.Duration((i+1)*2) * time.Second) // Exponential backoff
		} else {
			// Different error, don't retry
			mc.logger.Debugf("Different error for namespace %s, not retrying: %v", namespace, err)
			break
		}
	}

	if err != nil {
		mc.logger.Errorf("Failed to get pod metrics for namespace %s after retries: %v", namespace, err)
		return fmt.Errorf("failed to get pod metrics after retries: %w", err)
	}

	// Log how many pods were found
	mc.logger.Debugf("Found %d pods in namespace %s for metrics collection", len(podMetrics.Items), namespace)

	currentTime := time.Now()

	for _, podMetric := range podMetrics.Items {
		for _, container := range podMetric.Containers {
			memoryUsage, ok := container.Usage["memory"]
			if !ok {
				mc.logger.Debugf("No memory usage data for container %s in pod %s/%s", container.Name, namespace, podMetric.Name)
				continue // Skip if memory usage not available
			}

			dataPoint := MemoryDataPoint{
				Timestamp:   currentTime,
				PodName:     podMetric.Name,
				Namespace:   namespace,
				Container:   container.Name,
				MemoryUsage: memoryUsage.Value(),
			}

			// Store the data point in file
			if err := mc.saveMemoryDataPoint(namespace, podMetric.Name, container.Name, dataPoint); err != nil {
				mc.logger.Errorf("Failed to save memory data point for %s/%s/%s: %v", namespace, podMetric.Name, container.Name, err)
				continue
			}

			// Log memory usage for monitoring (changed from Debug to Info level to be visible with info log level)
			mc.logger.Infof("Collected memory metrics for %s/%s/%s: %d bytes at %v",
				namespace, podMetric.Name, container.Name, memoryUsage.Value(), currentTime)
		}
	}

	// Clean up old data files
	mc.cleanupOldDataFiles()

	// Log after completing metrics collection
	mc.logger.Infof("Completed memory metrics collection for namespace: %s, collected data for %d pods", namespace, len(podMetrics.Items))

	return nil
}

// saveMemoryDataPoint saves a memory data point to the corresponding file
func (mc *MemoryCollector) saveMemoryDataPoint(namespace, podName, container string, dataPoint MemoryDataPoint) error {
	filePath := mc.getMemoryDataFilePath(namespace, podName, container)
	if filePath == "" {
		return fmt.Errorf("could not get file path for %s/%s/%s", namespace, podName, container)
	}

	// Load existing data points
	existingData, err := mc.loadMemoryDataFromFile(filePath)
	if err != nil {
		// If file doesn't exist, start with empty slice
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to load existing data: %w", err)
		}
		existingData = []MemoryDataPoint{}
	}

	// Append new data point
	updatedData := append(existingData, dataPoint)

	// Write updated data back to file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print JSON
	if err := encoder.Encode(updatedData); err != nil {
		return fmt.Errorf("failed to encode data to JSON: %w", err)
	}

	return nil
}

// loadMemoryDataFromFile loads memory data points from a file
func (mc *MemoryCollector) loadMemoryDataFromFile(filePath string) ([]MemoryDataPoint, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dataPoints []MemoryDataPoint
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&dataPoints); err != nil {
		return nil, err
	}

	return dataPoints, nil
}

// cleanupOldDataFiles removes old data files based on retention policy
func (mc *MemoryCollector) cleanupOldDataFiles() {
	// For now, we'll skip the cleanup to avoid potential hanging issues
	// TODO: Implement a safer cleanup mechanism
}

// GetMemoryHistory returns memory usage history for a specific pod/container
func (mc *MemoryCollector) GetMemoryHistory(namespace, podName, container string, maxDays int) []MemoryDataPoint {
	filePath := mc.getMemoryDataFilePath(namespace, podName, container)
	if filePath == "" {
		mc.logger.Errorf("Could not get file path for %s/%s/%s", namespace, podName, container)
		return []MemoryDataPoint{}
	}

	// Load data from file
	dataPoints, err := mc.loadMemoryDataFromFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			mc.logger.Errorf("Failed to load memory data for %s/%s/%s: %v", namespace, podName, container, err)
		}
		return []MemoryDataPoint{}
	}

	// Filter by time if needed
	cutoffTime := time.Now().AddDate(0, 0, -maxDays)
	var filtered []MemoryDataPoint
	for _, dp := range dataPoints {
		if dp.Timestamp.After(cutoffTime) {
			filtered = append(filtered, dp)
		}
	}

	// Sort by timestamp
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	return filtered
}

// CalculateMemorySlope calculates the memory usage trend slope using linear regression
func (mc *MemoryCollector) CalculateMemorySlope(dataPoints []MemoryDataPoint) float64 {
	n := len(dataPoints)
	if n < 2 {
		return 0
	}

	// Use indices as x values for time series (to avoid precision issues with large Unix timestamps)
	var sumX, sumY, sumXY, sumXX float64
	for i, point := range dataPoints {
		x := float64(i)                                 // Using index as x value for time series
		y := float64(point.MemoryUsage) / (1024 * 1024) // Convert to MB

		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Linear regression slope formula: a = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	denominator := float64(n)*sumXX - sumX*sumX
	if math.Abs(denominator) < 1e-9 { // Avoid division by zero
		return 0
	}

	slope := (float64(n)*sumXY - sumX*sumY) / denominator
	return slope // Slope in MB per data point interval
}

// AnalyzeMemoryOnOOM performs memory analysis when OOM event occurs
func (mc *MemoryCollector) AnalyzeMemoryOnOOM(podName, namespace, container string, maxHistoryDays int) (float64, error) {
	history := mc.GetMemoryHistory(namespace, podName, container, maxHistoryDays)

	if len(history) < 2 {
		return 0, fmt.Errorf("insufficient data points for analysis (need at least 2, got %d)", len(history))
	}

	slope := mc.CalculateMemorySlope(history)

	// Log the analysis result
	mc.logger.Infof("OOM Memory Analysis for %s/%s/%s - Slope: %.2f MB/data_point, Data Points: %d, Time Range: %s to %s",
		namespace, podName, container,
		slope,
		len(history),
		history[0].Timestamp.Format(time.RFC3339),
		history[len(history)-1].Timestamp.Format(time.RFC3339))

	return slope, nil
}

// GetPodMemoryUsage returns the most recent memory usage for a specific pod/container
func (mc *MemoryCollector) GetPodMemoryUsage(namespace, podName, container string) *MemoryDataPoint {
	filePath := mc.getMemoryDataFilePath(namespace, podName, container)
	if filePath == "" {
		mc.logger.Errorf("Could not get file path for %s/%s/%s", namespace, podName, container)
		return nil
	}

	// Load data from file
	dataPoints, err := mc.loadMemoryDataFromFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			mc.logger.Errorf("Failed to load memory data for %s/%s/%s: %v", namespace, podName, container, err)
		}
		return nil
	}

	if len(dataPoints) == 0 {
		return nil
	}

	// Return the most recent data point
	return &dataPoints[len(dataPoints)-1]
}
