package monitors

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

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent/common"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// EventMonitorItem 监控 Kubernetes 事件的实现
type EventMonitorItem struct {
	stopCh chan struct{}
}

func NewEventMonitorItem() *EventMonitorItem {
	return &EventMonitorItem{
		stopCh: make(chan struct{}),
	}
}

func (e *EventMonitorItem) GetName() string {
	return "event-monitor"
}

func (e *EventMonitorItem) GetConfigKey() string {
	return "monitoring.enable_event_monitoring"
}

func (e *EventMonitorItem) IsEnabled(config *config.Config) bool {
	return config.Monitoring.EnableMonitorEvents
}

func (e *EventMonitorItem) Monitor(ctx context.Context, params common.MonitorParams) error {
	// Start watching events in a separate goroutine
	go func() {
		err := e.watchEvents(ctx, params)
		if err != nil {
			params.Logger.Errorf("Error watching events: %v", err)
		}
	}()

	return nil
}

// watchEvents watches for Kubernetes events and processes them
func (e *EventMonitorItem) watchEvents(ctx context.Context, params common.MonitorParams) error {
	// Use a more robust way to create the watcher with proper error handling
	var watcher watch.Interface
	var err error

	// Retry mechanism for creating the watcher
	for i := 0; i < 3; i++ {
		watcher, err = params.Clientset.CoreV1().Events(params.Namespace).Watch(context.TODO(), metav1.ListOptions{})
		if err == nil {
			break
		}

		params.Logger.Errorf("Attempt %d to create event watcher failed: %v", i+1, err)
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to create event watcher after retries: %w", err)
	}
	defer watcher.Stop()

	params.Logger.Infof("Started watching events for namespace: %s", params.Namespace)

	for {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Error {
				params.Logger.Errorf("Error watching events: %v", event.Object)
				continue
			}

			if event.Object == nil {
				continue
			}

			switch obj := event.Object.(type) {
			case *corev1.Event:
				e.processEvent(obj, params.Logger)

				// Check if this is an OOMKilled event and perform memory analysis if enabled
				if obj.Reason == "OOMKilled" && params.Config.Monitoring.MemoryMonitoring.OOMAnalysis.Enabled {
					params.Logger.Infof("Detected OOMKilled event: %s/%s - %s", obj.Namespace, obj.InvolvedObject.Name, obj.Message)
					e.handleOOMEvent(obj, params)
				}
			}
		case <-e.stopCh:
			params.Logger.Info("EventMonitorItem stopped")
			return nil
		case <-ctx.Done():
			params.Logger.Info("Context cancelled, stopping EventMonitorItem")
			return nil
		}
	}
}

// handleOOMEvent handles OOMKilled events by performing memory analysis
func (e *EventMonitorItem) handleOOMEvent(event *corev1.Event, params common.MonitorParams) {
	// Extract pod name and namespace from the event
	podName := event.InvolvedObject.Name
	namespace := event.InvolvedObject.Namespace

	params.Logger.Infof("OOMKilled event detected for pod: %s/%s", namespace, podName)

	// Find the container that was killed (usually in the message)
	containerName := e.extractContainerNameFromMessage(event.Message)

	params.Logger.Infof("Analyzing memory for OOM event - Pod: %s, Container: %s", podName, containerName)

	// Perform memory analysis if memory collector is available
	if params.Config.Monitoring.MemoryMonitoring.Enabled && params.Config.Monitoring.MemoryMonitoring.BasicCollection.Enabled {
		// Initialize memory collector if memory monitoring is enabled
		retentionDays := params.Config.Monitoring.MemoryMonitoring.BasicCollection.RetentionDays
		if retentionDays <= 0 {
			retentionDays = 30 // default to 30 days
		}
		memoryCollector := basic.NewMemoryCollector(params.Clientset, params.MetricsClient, params.Logger, retentionDays)

		maxHistoryDays := params.Config.Monitoring.MemoryMonitoring.OOMAnalysis.MaxHistoryDays
		if maxHistoryDays <= 0 {
			maxHistoryDays = 30 // default to 30 days
		}

		minDataPoints := params.Config.Monitoring.MemoryMonitoring.OOMAnalysis.MinDataPoints
		if minDataPoints <= 0 {
			minDataPoints = 10 // default to 10 data points
		}

		slope, err := memoryCollector.AnalyzeMemoryOnOOM(podName, namespace, containerName, maxHistoryDays)
		if err != nil {
			params.Logger.Errorf("Error analyzing memory for OOM event: %v", err)
			// Even if analysis fails, still record the OOM event
			e.recordOOMEventOnly(podName, namespace, containerName, event.Message, params.Logger)
			return
		}

		// Record the analysis result
		e.recordOOMMemoryAnalysis(podName, namespace, containerName, slope, maxHistoryDays, params.Logger)
	}
}

// extractContainerNameFromMessage extracts container name from event message
func (e *EventMonitorItem) extractContainerNameFromMessage(message string) string {
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
	return ""
}

// recordOOMMemoryAnalysis records the OOM memory analysis to critical record
func (e *EventMonitorItem) recordOOMMemoryAnalysis(podName, namespace, containerName string, slope float64, maxHistoryDays int, logger *logrus.Logger) {
	// Create critical_record directory if it doesn't exist
	criticalRecordPath := "logs/critical_record"
	if err := basic.CreateDirectory(criticalRecordPath); err != nil {
		logger.Errorf("Failed to create critical_record directory: %v", err)
		return
	}

	// Prepare the analysis report
	report := e.formatOOMMemoryAnalysisReport(podName, namespace, containerName, slope, maxHistoryDays)

	// Write the report to a file
	filename := filepath.Join(criticalRecordPath, fmt.Sprintf("memory_analysis_%s_%s_%s_%s.txt",
		namespace, podName, containerName, time.Now().Format("20060102_150405")))

	if err := basic.WriteFile(filename, report); err != nil {
		logger.Errorf("Failed to write OOM memory analysis report: %v", err)
		return
	}

	logger.Infof("OOM memory analysis report written to: %s", filename)
}

// recordOOMEventOnly records just the OOM event when analysis fails
func (e *EventMonitorItem) recordOOMEventOnly(podName, namespace, containerName, message string, logger *logrus.Logger) {
	// Create critical_record directory if it doesn't exist
	criticalRecordPath := "logs/critical_record"
	if err := basic.CreateDirectory(criticalRecordPath); err != nil {
		logger.Errorf("Failed to create critical_record directory: %v", err)
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

	if err := basic.WriteFile(filename, report); err != nil {
		logger.Errorf("Failed to write OOM event record: %v", err)
		return
	}

	logger.Infof("OOM event record written to: %s", filename)
}

// formatOOMMemoryAnalysisReport formats the OOM memory analysis report
func (e *EventMonitorItem) formatOOMMemoryAnalysisReport(podName, namespace, containerName string, slope float64, maxHistoryDays int) string {
	// Note: This is a simplified version. In a real implementation, we would need to access
	// the memory collector to get historical data
	history := []basic.MemoryDataPoint{} // Placeholder - would get actual history in real implementation

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

// processEvent processes a Kubernetes event
func (e *EventMonitorItem) processEvent(event *corev1.Event, logger *logrus.Logger) {
	// Determine severity based on event type and reason
	severity := e.determineSeverity(event)

	// Ignore normal operations
	if severity == "IGNORE" {
		return
	}

	logger.Debugf("Processing event: %s/%s - Reason: %s, Message: %s",
		event.Namespace, event.Name, event.Reason, event.Message)

	// Log the event with severity
	logger.Infof("[%s] Event: %s/%s - Reason: %s, Message: %s",
		severity, event.InvolvedObject.Namespace, event.InvolvedObject.Name,
		event.Reason, event.Message)

	// Write critical events to dedicated log file
	if severity == "CRITICAL" || severity == "ERROR" {
		e.writeCriticalEvent(&event.InvolvedObject, event.Reason, event.Message, logger)
	}
}

// determineSeverity determines the severity of an event
func (e *EventMonitorItem) determineSeverity(event *corev1.Event) string {
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
func (e *EventMonitorItem) writeCriticalEvent(obj *corev1.ObjectReference, eventType, message string, logger *logrus.Logger) {
	logger.Errorf("CRITICAL_EVENT_LOG - Object: %s/%s (%s), Event: %s, Message: %s",
		obj.Namespace, obj.Name, obj.Kind, eventType, message)

	// Write in format suitable for aggregation
	logger.WithFields(logrus.Fields{
		"namespace": obj.Namespace,
		"objType":   obj.Kind,
		"objName":   obj.Name,
		"eventType": eventType,
		"message":   message,
	}).Error("CRITICAL_EVENT")

	// Write critical event to dedicated file in the logger's path
	logPath := filepath.Join("logs", "basic", obj.Namespace)
	criticalLogPath := filepath.Join(logPath, "critical_events.log")

	criticalLogFile, err := common.OpenFile(criticalLogPath)
	if err == nil {
		defer criticalLogFile.Close()

		// Create a log entry with the desired fields and message
		entry := &logrus.Entry{
			Logger: logger,
			Data: logrus.Fields{
				"type":    obj.Kind,
				"name":    obj.Name,
				"event":   eventType,
				"message": message,
			},
			Level:   logrus.ErrorLevel,
			Time:    time.Now(),
			Message: "CRITICAL_EVENT_LOG",
		}

		serialized, _ := logger.Formatter.Format(entry)
		criticalLogFile.Write(serialized)
	}
}
