package basic

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	dataStore     map[string][]MemoryDataPoint // key: namespace/pod/container
	storeMutex    sync.RWMutex
	retentionDays int
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, logger *logrus.Logger, retentionDays int) *MemoryCollector {
	return &MemoryCollector{
		clientset:     clientset,
		metricsClient: metricsClient,
		logger:        logger,
		dataStore:     make(map[string][]MemoryDataPoint),
		retentionDays: retentionDays,
	}
}

// CollectMemoryMetrics collects memory metrics for all pods in a namespace
func (mc *MemoryCollector) CollectMemoryMetrics(namespace string) error {
	// Get pod metrics from metrics server
	podMetrics, err := mc.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod metrics: %w", err)
	}

	currentTime := time.Now()

	for _, podMetric := range podMetrics.Items {
		for _, container := range podMetric.Containers {
			memoryUsage, ok := container.Usage["memory"]
			if !ok {
				continue // Skip if memory usage not available
			}

			dataPoint := MemoryDataPoint{
				Timestamp:   currentTime,
				PodName:     podMetric.Name,
				Namespace:   namespace,
				Container:   container.Name,
				MemoryUsage: memoryUsage.Value(),
			}

			// Store the data point
			key := fmt.Sprintf("%s/%s/%s", namespace, podMetric.Name, container.Name)
			mc.storeMutex.Lock()
			mc.dataStore[key] = append(mc.dataStore[key], dataPoint)
			mc.storeMutex.Unlock()
		}
	}

	// Clean up old data points
	mc.cleanupOldData()

	return nil
}

// cleanupOldData removes data points older than retention period
func (mc *MemoryCollector) cleanupOldData() {
	mc.storeMutex.Lock()
	defer mc.storeMutex.Unlock()

	cutoffTime := time.Now().AddDate(0, 0, -mc.retentionDays)

	for key, dataPoints := range mc.dataStore {
		var cleaned []MemoryDataPoint
		for _, dp := range dataPoints {
			if dp.Timestamp.After(cutoffTime) {
				cleaned = append(cleaned, dp)
			}
		}
		mc.dataStore[key] = cleaned
	}
}

// GetMemoryHistory returns memory usage history for a specific pod/container
func (mc *MemoryCollector) GetMemoryHistory(namespace, podName, container string, maxDays int) []MemoryDataPoint {
	mc.storeMutex.RLock()
	defer mc.storeMutex.RUnlock()

	key := fmt.Sprintf("%s/%s/%s", namespace, podName, container)
	dataPoints := mc.dataStore[key]

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
		x := float64(i) // Using index as x value for time series
		y := float64(point.MemoryUsage) / (1024 * 1024)      // Convert to MB

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
	return slope // Slope in MB per hour
}

// AnalyzeMemoryOnOOM performs memory analysis when OOM event occurs
func (mc *MemoryCollector) AnalyzeMemoryOnOOM(podName, namespace, container string, maxHistoryDays int) (float64, error) {
	history := mc.GetMemoryHistory(namespace, podName, container, maxHistoryDays)
	
	if len(history) < 2 {
		return 0, fmt.Errorf("insufficient data points for analysis (need at least 2, got %d)", len(history))
	}

	slope := mc.CalculateMemorySlope(history)
	
	// Log the analysis result
	mc.logger.Infof("OOM Memory Analysis for %s/%s/%s - Slope: %.2f MB/hour, Data Points: %d, Time Range: %s to %s",
		namespace, podName, container,
		slope,
		len(history),
		history[0].Timestamp.Format(time.RFC3339),
		history[len(history)-1].Timestamp.Format(time.RFC3339))

	return slope, nil
}