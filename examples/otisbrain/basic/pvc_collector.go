package basic

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// PVCDataPoint represents a single PVC usage data point
type PVCDataPoint struct {
	Timestamp     time.Time
	PVCName       string
	Namespace     string
	CapacityBytes int64
	UsedBytes     int64
	UsagePercent  float64
	Status        string // Bound, Pending, Lost, etc.
}

// PVCUsageInfo contains information about a PVC and the pods using it
type PVCUsageInfo struct {
	PVCDataPoint
	UsingPods []string // Pods using this PVC
}

// PVCCollector collects PVC usage metrics
type PVCCollector struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
	logger        *logrus.Logger
	dataStore     map[string][]PVCDataPoint // key: namespace/pvc-name
	storeMutex    sync.RWMutex
	interval      time.Duration
	threshold     int
	maxPodsDisplay int
	retentionDays int
}

// NewPVCCollector creates a new PVC collector
func NewPVCCollector(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, logger *logrus.Logger, 
	intervalSeconds, thresholdPercent, maxPodsDisplay, retentionDays int) *PVCCollector {
	return &PVCCollector{
		clientset:      clientset,
		metricsClient:  metricsClient,
		logger:         logger,
		dataStore:      make(map[string][]PVCDataPoint),
		interval:       time.Duration(intervalSeconds) * time.Second,
		threshold:      thresholdPercent,
		maxPodsDisplay: maxPodsDisplay,
		retentionDays:  retentionDays,
	}
}

// CollectPVCMetrics collects PVC metrics for all PVCs in a namespace
func (pc *PVCCollector) CollectPVCMetrics(namespace string) error {
	// Get PVCs from the namespace
	pvcs, err := pc.clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get PVCs: %w", err)
	}

	currentTime := time.Now()

	for _, pvc := range pvcs.Items {
		// Get capacity from PVC status or spec
		var capacityBytes int64
		if capacity, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
			capacityBytes = capacity.Value()
		} else if capacity, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
			capacityBytes = capacity.Value()
		} else {
			pc.logger.Debugf("Could not determine capacity for PVC %s/%s", namespace, pvc.Name)
			continue
		}

		// For now, we'll set used bytes to 0 and usage percent to 0
		// In a real implementation, we would get this from storage metrics if available
		usedBytes := int64(0)
		usagePercent := float64(0)

		// Try to get actual usage from metrics if available
		// Note: Storage metrics availability depends on the storage provider
		// This is a simplified implementation
		if capacityBytes > 0 {
			// In a real implementation, we would query storage metrics here
			// For now, we'll simulate getting usage data
			// In practice, this would come from storage provider metrics or external monitoring
		}

		dataPoint := PVCDataPoint{
			Timestamp:     currentTime,
			PVCName:       pvc.Name,
			Namespace:     namespace,
			CapacityBytes: capacityBytes,
			UsedBytes:     usedBytes,
			UsagePercent:  usagePercent,
			Status:        string(pvc.Status.Phase),
		}

		// Store the data point
		key := fmt.Sprintf("%s/%s", namespace, pvc.Name)
		pc.storeMutex.Lock()
		pc.dataStore[key] = append(pc.dataStore[key], dataPoint)
		pc.storeMutex.Unlock()
	}

	// Clean up old data points
	pc.cleanupOldData()

	return nil
}

// cleanupOldData removes data points older than retention period
func (pc *PVCCollector) cleanupOldData() {
	pc.storeMutex.Lock()
	defer pc.storeMutex.Unlock()

	cutoffTime := time.Now().AddDate(0, 0, -pc.retentionDays)

	for key, dataPoints := range pc.dataStore {
		var cleaned []PVCDataPoint
		for _, dp := range dataPoints {
			if dp.Timestamp.After(cutoffTime) {
				cleaned = append(cleaned, dp)
			}
		}
		pc.dataStore[key] = cleaned
	}
}

// GetPVCUsageInfo gets PVC usage info along with pods using the PVC
func (pc *PVCCollector) GetPVCUsageInfo(namespace, pvcName string, maxDays int) (*PVCUsageInfo, error) {
	// Get PVC usage history
	history := pc.GetPVCHistory(namespace, pvcName, maxDays)

	if len(history) == 0 {
		return nil, fmt.Errorf("no data found for PVC %s/%s", namespace, pvcName)
	}

	// Get the most recent data point
	latest := history[len(history)-1]

	// Find pods using this PVC
	usingPods, err := pc.GetPodsUsingPVC(namespace, pvcName)
	if err != nil {
		pc.logger.Errorf("Error finding pods using PVC %s/%s: %v", namespace, pvcName, err)
		// Continue with empty pod list
	}

	// Limit the number of pods displayed
	if len(usingPods) > pc.maxPodsDisplay {
		usingPods = usingPods[:pc.maxPodsDisplay]
	}

	return &PVCUsageInfo{
		PVCDataPoint: latest,
		UsingPods:    usingPods,
	}, nil
}

// GetPVCHistory returns PVC usage history for a specific PVC
func (pc *PVCCollector) GetPVCHistory(namespace, pvcName string, maxDays int) []PVCDataPoint {
	pc.storeMutex.RLock()
	defer pc.storeMutex.RUnlock()

	key := fmt.Sprintf("%s/%s", namespace, pvcName)
	dataPoints := pc.dataStore[key]

	// Filter by time if needed
	cutoffTime := time.Now().AddDate(0, 0, -maxDays)
	var filtered []PVCDataPoint
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

// GetPodsUsingPVC finds pods that are using a specific PVC
func (pc *PVCCollector) GetPodsUsingPVC(namespace, pvcName string) ([]string, error) {
	pods, err := pc.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	var usingPods []string
	for _, pod := range pods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pvcName {
				usingPods = append(usingPods, pod.Name)
				break // A pod might have multiple volumes, but we only want to add it once
			}
		}
	}

	// Sort for consistent output
	sort.Strings(usingPods)

	return usingPods, nil
}

// CheckPVCUsage checks all PVCs in a namespace for usage exceeding threshold
func (pc *PVCCollector) CheckPVCUsage(namespace string) ([]PVCUsageInfo, error) {
	pvcs, err := pc.clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get PVCs: %w", err)
	}

	var highUsagePVCs []PVCUsageInfo

	for _, pvc := range pvcs.Items {
		// Get PVC usage info with associated pods
		usageInfo, err := pc.GetPVCUsageInfo(namespace, pvc.Name, 1) // Check last day of data
		if err != nil {
			pc.logger.Debugf("Could not get usage info for PVC %s/%s: %v", namespace, pvc.Name, err)
			continue
		}

		// In a real implementation, we would have actual usage data
		// For now, we'll simulate checking against a threshold
		// Since we don't have real usage data from metrics, we'll check for other conditions
		// such as status or simulate based on some criteria

		// Check if PVC status is problematic (not bound)
		if usageInfo.Status != "Bound" {
			highUsagePVCs = append(highUsagePVCs, *usageInfo)
			continue
		}

		// In a real implementation with actual usage data, we would check:
		// if usageInfo.UsagePercent >= float64(pc.threshold) {
		//     highUsagePVCs = append(highUsagePVCs, *usageInfo)
		// }

		// For now, we'll just return PVCs with issues (non-bound status)
		// In a production implementation, we would need to get actual usage data
		// from storage metrics or other sources
	}

	return highUsagePVCs, nil
}