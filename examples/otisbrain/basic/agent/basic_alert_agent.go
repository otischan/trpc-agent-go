package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
)

// BasicEventMonitorAgent handles Kubernetes event monitoring
type BasicEventMonitorAgent struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
	namespace     string
	config        *config.Config
	logger        *logrus.Logger
	stopCh        chan struct{}
	memoryCollector *basic.MemoryCollector
}

// NewBasicEventMonitorAgent creates a new basic event monitoring agent
func NewBasicEventMonitorAgent(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, namespace string, cfg *config.Config, logger *logrus.Logger) *BasicEventMonitorAgent {
	agent := &BasicEventMonitorAgent{
		clientset:     clientset,
		metricsClient: metricsClient,
		namespace:     namespace,
		config:        cfg,
		logger:        logger,
		stopCh:        make(chan struct{}),
	}

	// Initialize memory collector if memory monitoring is enabled
	if cfg.Monitoring.MemoryMonitoring.Enabled && cfg.Monitoring.MemoryMonitoring.BasicCollection.Enabled {
		retentionDays := cfg.Monitoring.MemoryMonitoring.BasicCollection.RetentionDays
		if retentionDays <= 0 {
			retentionDays = 30 // default to 30 days
		}
		agent.memoryCollector = basic.NewMemoryCollector(clientset, metricsClient, logger, retentionDays)
	}

	return agent
}

// Start starts the basic event monitoring agent
func (bea *BasicEventMonitorAgent) Start(ctx context.Context) error {
	bea.logger.Infof("BasicEventMonitorAgent started for namespace: %s", bea.namespace)

	// Start watching events in a separate goroutine
	go func() {
		err := bea.watchEvents(ctx)
		if err != nil {
			bea.logger.Errorf("Error watching events: %v", err)
		}
	}()

	// Start periodic memory collection if enabled
	if bea.memoryCollector != nil {
		go func() {
			ticker := time.NewTicker(30 * time.Second) // Collect memory metrics every 30 seconds
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := bea.collectMemoryMetrics(); err != nil {
						bea.logger.Errorf("Error collecting memory metrics: %v", err)
					}
				case <-bea.stopCh:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return nil
}

// Stop stops the basic event monitoring agent
func (bea *BasicEventMonitorAgent) Stop() {
	close(bea.stopCh)
}

// collectMemoryMetrics collects memory metrics for the namespace
func (bea *BasicEventMonitorAgent) collectMemoryMetrics() error {
	if bea.memoryCollector != nil {
		return bea.memoryCollector.CollectMemoryMetrics(bea.namespace)
	}
	return nil
}

// watchEvents watches for Kubernetes events and processes them
func (bea *BasicEventMonitorAgent) watchEvents(ctx context.Context) error {
	// Use a more robust way to create the watcher with proper error handling
	var watcher watch.Interface
	var err error

	// Retry mechanism for creating the watcher
	for i := 0; i < 3; i++ {
		watcher, err = bea.clientset.CoreV1().Events(bea.namespace).Watch(context.TODO(), metav1.ListOptions{})
		if err == nil {
			break
		}

		bea.logger.Errorf("Attempt %d to create event watcher failed: %v", i+1, err)
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to create event watcher after retries: %w", err)
	}
	defer watcher.Stop()

	bea.logger.Infof("Started watching events for namespace: %s", bea.namespace)

	for {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Error {
				bea.logger.Errorf("Error watching events: %v", event.Object)
				continue
			}

			if event.Object == nil {
				continue
			}

			switch obj := event.Object.(type) {
			case *corev1.Event:
				bea.processEvent(obj)

				// Check if this is an OOMKilled event and perform memory analysis if enabled
				if obj.Reason == "OOMKilled" && bea.config.Monitoring.MemoryMonitoring.OOMAnalysis.Enabled {
					bea.logger.Infof("Detected OOMKilled event: %s/%s - %s", obj.Namespace, obj.InvolvedObject.Name, obj.Message)
					bea.handleOOMEvent(obj)
				}
			}
		case <-bea.stopCh:
			bea.logger.Info("BasicEventMonitorAgent stopped")
			return nil
		case <-ctx.Done():
			bea.logger.Info("Context cancelled, stopping BasicEventMonitorAgent")
			return nil
		}
	}
}

// handleOOMEvent handles OOMKilled events by performing memory analysis
func (bea *BasicEventMonitorAgent) handleOOMEvent(event *corev1.Event) {
	if bea.memoryCollector == nil {
		bea.logger.Warn("Memory collector not initialized, skipping OOM analysis")
		return
	}

	// Extract pod name and namespace from the event
	podName := event.InvolvedObject.Name
	namespace := event.InvolvedObject.Namespace

	bea.logger.Infof("OOMKilled event detected for pod: %s/%s", namespace, podName)

	// Find the container that was killed (usually in the message)
	containerName := bea.extractContainerNameFromMessage(event.Message)

	bea.logger.Infof("Analyzing memory for OOM event - Pod: %s, Container: %s", podName, containerName)

	// Perform memory analysis
	maxHistoryDays := bea.config.Monitoring.MemoryMonitoring.OOMAnalysis.MaxHistoryDays
	if maxHistoryDays <= 0 {
		maxHistoryDays = 30 // default to 30 days
	}

	minDataPoints := bea.config.Monitoring.MemoryMonitoring.OOMAnalysis.MinDataPoints
	if minDataPoints <= 0 {
		minDataPoints = 10 // default to 10 data points
	}

	slope, err := bea.memoryCollector.AnalyzeMemoryOnOOM(podName, namespace, containerName, maxHistoryDays)
	if err != nil {
		bea.logger.Errorf("Error analyzing memory for OOM event: %v", err)
		// Even if analysis fails, still record the OOM event
		bea.recordOOMEventOnly(podName, namespace, containerName, event.Message)
		return
	}

	// Record the analysis result
	bea.recordOOMMemoryAnalysis(podName, namespace, containerName, slope, maxHistoryDays)
}

// extractContainerNameFromMessage extracts container name from event message
func (bea *BasicEventMonitorAgent) extractContainerNameFromMessage(message string) string {
	// Common pattern in OOMKilled messages: "Container <container-name> failed"
	if idx := strings.Index(message, "Container "); idx != -1 {
		start := idx + len("Container ")
		end := strings.Index(message[start:], " ")
		if end != -1 {
			return message[start : start+end]
		}
	}
	// If we can't extract from message, return empty string
	// The memory collector will analyze all containers in the pod
	bea.logger.Debugf("Could not extract container name from message: %s", message)
	return ""
}

// recordOOMMemoryAnalysis records the OOM memory analysis to critical record
func (bea *BasicEventMonitorAgent) recordOOMMemoryAnalysis(podName, namespace, containerName string, slope float64, maxHistoryDays int) {
	// Create critical_record directory if it doesn't exist
	criticalRecordPath := "logs/critical_record"
	if err := bea.createDirectory(criticalRecordPath); err != nil {
		bea.logger.Errorf("Failed to create critical_record directory: %v", err)
		return
	}

	// Prepare the analysis report
	report := bea.formatOOMMemoryAnalysisReport(podName, namespace, containerName, slope, maxHistoryDays)

	// Write the report to a file
	filename := filepath.Join(criticalRecordPath, fmt.Sprintf("memory_analysis_%s_%s_%s_%s.txt",
		namespace, podName, containerName, time.Now().Format("20060102_150405")))

	if err := bea.writeFile(filename, report); err != nil {
		bea.logger.Errorf("Failed to write OOM memory analysis report: %v", err)
		return
	}

	bea.logger.Infof("OOM memory analysis report written to: %s", filename)
}

// recordOOMEventOnly records just the OOM event when analysis fails
func (bea *BasicEventMonitorAgent) recordOOMEventOnly(podName, namespace, containerName, message string) {
	// Create critical_record directory if it doesn't exist
	criticalRecordPath := "logs/critical_record"
	if err := bea.createDirectory(criticalRecordPath); err != nil {
		bea.logger.Errorf("Failed to create critical_record directory: %v", err)
		return
	}

	report := fmt.Sprintf(`# OOM事件记录
Pod: %s
Namespace: %s
Container: %s
时间: %s
消息: %s

# 注意
内存分析失败，但事件已记录
`, podName, namespace, containerName, time.Now().Format(time.RFC3339), message)

	// Write the report to a file
	filename := filepath.Join(criticalRecordPath, fmt.Sprintf("oom_event_only_%s_%s_%s_%s.txt",
		namespace, podName, containerName, time.Now().Format("20060102_150405")))

	if err := bea.writeFile(filename, report); err != nil {
		bea.logger.Errorf("Failed to write OOM event record: %v", err)
		return
	}

	bea.logger.Infof("OOM event record written to: %s", filename)
}

// formatOOMMemoryAnalysisReport formats the OOM memory analysis report
func (bea *BasicEventMonitorAgent) formatOOMMemoryAnalysisReport(podName, namespace, containerName string, slope float64, maxHistoryDays int) string {
	history := bea.memoryCollector.GetMemoryHistory(namespace, podName, containerName, maxHistoryDays)

	var dataPointsCount int
	var timeRangeStart, timeRangeEnd time.Time

	if len(history) > 0 {
		dataPointsCount = len(history)
		timeRangeStart = history[0].Timestamp
		timeRangeEnd = history[len(history)-1].Timestamp
	}

	var analysisConclusion string
	if slope > 0 {
		analysisConclusion = fmt.Sprintf("- 内存在过去%d天内总体呈增长趋势\n- 平均每数据点增长%.2fMB", maxHistoryDays, slope)
	} else if slope < 0 {
		analysisConclusion = fmt.Sprintf("- 内存在过去%d天内总体呈下降趋势\n- 平均每数据点减少%.2fMB", maxHistoryDays, -slope)
	} else {
		analysisConclusion = fmt.Sprintf("- 内存在过去%d天内总体保持稳定\n- 平均每数据点变化%.2fMB", maxHistoryDays, slope)
	}

	return fmt.Sprintf(`# OOM事件内存分析报告
Pod: %s
Namespace: %s
Container: %s
OOM时间: %s

# 内存使用趋势分析
内存使用斜率: %.2f MB/数据点
分析时间范围: 过去%d天
数据点数量: %d (从 %s 到 %s)

# 分析结论
%s
`, podName, namespace, containerName, time.Now().Format(time.RFC3339), slope, maxHistoryDays,
		dataPointsCount, timeRangeStart.Format(time.RFC3339), timeRangeEnd.Format(time.RFC3339), analysisConclusion)
}

// createDirectory creates a directory if it doesn't exist
func (bea *BasicEventMonitorAgent) createDirectory(path string) error {
	return basic.CreateDirectory(path)
}

// writeFile writes content to a file
func (bea *BasicEventMonitorAgent) writeFile(filename, content string) error {
	return basic.WriteFile(filename, content)
}

// processEvent processes a Kubernetes event
func (bea *BasicEventMonitorAgent) processEvent(event *corev1.Event) {
	// Determine severity based on event type and reason
	severity := bea.determineSeverity(event)

	// Ignore normal operations
	if severity == "IGNORE" {
		return
	}

	bea.logger.Debugf("Processing event: %s/%s - Reason: %s, Message: %s",
		event.Namespace, event.Name, event.Reason, event.Message)

	// Log the event with severity
	bea.logger.Infof("[%s] Event: %s/%s - Reason: %s, Message: %s",
		severity, event.InvolvedObject.Namespace, event.InvolvedObject.Name,
		event.Reason, event.Message)

	// Write critical events to dedicated log file
	if severity == "CRITICAL" || severity == "ERROR" {
		bea.writeCriticalEvent(&event.InvolvedObject, event.Reason, event.Message)
	}
}

// determineSeverity determines the severity of an event
func (bea *BasicEventMonitorAgent) determineSeverity(event *corev1.Event) string {
	// Define severity mapping based on event reason
	// Only consider abnormal events as critical/error, ignore normal operations
	switch event.Reason {
	case "Failed", "FailedScheduling", "FailedMount", "FailedCreate", "FailedDelete",
	     "Unhealthy", "ImagePullBackOff", "OOMKilled":
		return "CRITICAL"
	case "Warning", "Terminating", "Killing", "Evicted", "BackOff", "CrashLoopBackOff":
		return "ERROR"
	case "Created", "Started", "Pulled", "Scheduled", "Pulling", "Starting", "Start",
	     "ContainerCreating", "SuccessfulAttachVolume", "SuccessfulCreate", "ScalingReplicaSet":
		// Normal operations that should not be logged as events
		return "IGNORE"
	default:
		return "WARNING"
	}
}

// writeCriticalEvent writes critical events to a dedicated log file
func (bea *BasicEventMonitorAgent) writeCriticalEvent(obj *corev1.ObjectReference, eventType, message string) {
	bea.logger.Errorf("CRITICAL_EVENT_LOG - Object: %s/%s (%s), Event: %s, Message: %s",
		obj.Namespace, obj.Name, obj.Kind, eventType, message)

	// Write in format suitable for aggregation
	bea.logger.WithFields(logrus.Fields{
		"namespace": obj.Namespace,
		"objType":   obj.Kind,
		"objName":   obj.Name,
		"eventType": eventType,
		"message":   message,
	}).Error("CRITICAL_EVENT")
}