//
// Monitor-MCP-Server provides MCP tools for analyzing OtisBrain monitor module logs
//

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	mcp "trpc.group/trpc-go/trpc-mcp-go"
)

// PodMonitoringInput represents input for pod monitoring
type PodMonitoringInput struct {
	Namespace string `json:"namespace" jsonschema:"required,description=The namespace to monitor"`
	PodName   string `json:"pod_name,omitempty" jsonschema:"description=Optional pod name to monitor specific pod, if not provided all pods in namespace will be monitored"`
}

// PodMonitoringOutput represents output for pod monitoring
type PodMonitoringOutput struct {
	Pods  []PodInfo `json:"pods"`
	Error string    `json:"error,omitempty"`
}

// PodInfo represents pod information
type PodInfo struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	Phase        string            `json:"phase"`
	Restarts     int32             `json:"restarts"`
	Conditions   []ConditionInfo   `json:"conditions"`
	MemoryUsage  *MemoryUsageInfo  `json:"memory_usage,omitempty"`
	CPUUsage     *CPUUsageInfo     `json:"cpu_usage,omitempty"`
	StartTime    *time.Time        `json:"start_time,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	LastEvent    string            `json:"last_event,omitempty"`
	Namespace    string            `json:"namespace"`
}

// ConditionInfo represents condition information
type ConditionInfo struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// MemoryUsageInfo represents memory usage information
type MemoryUsageInfo struct {
	CurrentBytes int64   `json:"current_bytes"`
	LimitBytes   int64   `json:"limit_bytes,omitempty"`
	Percentage   float64 `json:"percentage"`
}

// CPUUsageInfo represents CPU usage information
type CPUUsageInfo struct {
	CurrentMillicores int64 `json:"current_millicores"`
	LimitMillicores   int64 `json:"limit_millicores,omitempty"`
}

// ClusterSummaryInput represents input for cluster summary
type ClusterSummaryInput struct {
	Namespaces []string `json:"namespaces,omitempty" jsonschema:"description=Optional list of namespaces to check, if not provided all namespaces will be checked"`
}

// ClusterSummaryOutput represents output for cluster summary
type ClusterSummaryOutput struct {
	Summary SummaryInfo `json:"summary"`
	Nodes   []NodeInfo  `json:"nodes"`
	Pods    []PodSummary `json:"pods"`
	Error   string      `json:"error,omitempty"`
}

// SummaryInfo represents summary information
type SummaryInfo struct {
	TotalNodes     int      `json:"total_nodes"`
	ReadyNodes     int      `json:"ready_nodes"`
	TotalPods      int      `json:"total_pods"`
	RunningPods    int      `json:"running_pods"`
	FailedPods     int      `json:"failed_pods"`
	CriticalEvents int      `json:"critical_events"`
	OverallStatus  string   `json:"overall_status"`
	RecentIssues   []string `json:"recent_issues"`
}

// NodeInfo represents node information
type NodeInfo struct {
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	Ready      bool            `json:"ready"`
	Conditions []ConditionInfo `json:"conditions"`
}

// PodSummary represents pod summary
type PodSummary struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Phase     string `json:"phase"`
}

// MemoryLeakDetectionInput represents input for memory leak detection
type MemoryLeakDetectionInput struct {
	Namespace string `json:"namespace" jsonschema:"required,description=The namespace containing the pod"`
	PodName   string `json:"pod_name" jsonschema:"required,description=The name of the pod to analyze"`
	Container string `json:"container" jsonschema:"required,description=The name of the container to analyze"`
	Hours     int    `json:"hours,omitempty" jsonschema:"description=Number of hours to analyze (default: 24)"`
}

// MemoryLeakDetectionOutput represents output for memory leak detection
type MemoryLeakDetectionOutput struct {
	PodName        string  `json:"pod_name"`
	Container      string  `json:"container"`
	SlopeMBPerHour float64 `json:"slope_mb_per_hour"`
	IsLeaking      bool    `json:"is_leaking"`
	Confidence     string  `json:"confidence"`
	Trend          string  `json:"trend"`
	MaxMemory      int64   `json:"max_memory_bytes"`
	AvgMemory      int64   `json:"avg_memory_bytes"`
	Error          string  `json:"error,omitempty"`
}

// RecentEventsInput represents input for getting recent events
type RecentEventsInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"description=Optional namespace to filter events, if not provided events from all namespaces will be returned"`
	Limit     int    `json:"limit,omitempty" jsonschema:"description=Maximum number of events to return (default: 50)"`
}

// RecentEventsOutput represents output for getting recent events
type RecentEventsOutput struct {
	Events []EventInfo `json:"events"`
	Error  string      `json:"error,omitempty"`
}

// EventInfo represents event information
type EventInfo struct {
	ObjectName    string    `json:"object_name"`
	ObjectKind    string    `json:"object_kind"`
	Type          string    `json:"type"`
	Reason        string    `json:"reason"`
	Message       string    `json:"message"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	Count         int32     `json:"count"`
}

// ResourceUtilizationInput represents input for resource utilization
type ResourceUtilizationInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"description=Optional namespace to filter resources, if not provided resources from all namespaces will be returned"`
}

// ResourceUtilizationOutput represents output for resource utilization
type ResourceUtilizationOutput struct {
	Namespace       string            `json:"namespace"`
	NodeUtilization []NodeUtilization `json:"node_utilization"`
	PodUtilization  []PodUtilization  `json:"pod_utilization"`
	Error           string            `json:"error,omitempty"`
}

// NodeUtilization represents node resource utilization
type NodeUtilization struct {
	NodeName    string        `json:"node_name"`
	CPUUsage    ResourceUsage `json:"cpu_usage"`
	MemoryUsage ResourceUsage `json:"memory_usage"`
}

// PodUtilization represents pod resource utilization
type PodUtilization struct {
	PodName     string        `json:"pod_name"`
	Namespace   string        `json:"namespace"`
	CPUUsage    ResourceUsage `json:"cpu_usage"`
	MemoryUsage ResourceUsage `json:"memory_usage"`
}

// ResourceUsage represents resource usage
type ResourceUsage struct {
	Used         string  `json:"used"`
	Limit        string  `json:"limit,omitempty"`
	UsagePercent float64 `json:"usage_percent"`
}

// AggregatedLogAnalysisInput represents input for aggregated log analysis
type AggregatedLogAnalysisInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"description=Optional namespace to analyze, if not provided all namespaces will be analyzed"`
	Hours     int    `json:"hours,omitempty" jsonschema:"description=Number of hours to analyze (default: 24)"`
}

// AggregatedLogAnalysisOutput represents output for aggregated log analysis
type AggregatedLogAnalysisOutput struct {
	Namespace        string              `json:"namespace"`
	AnalysisPeriod   string              `json:"analysis_period"`
	TotalEvents      int                 `json:"total_events"`
	CriticalEvents   int                 `json:"critical_events"`
	ErrorEvents      int                 `json:"error_events"`
	Warnings         int                 `json:"warnings"`
	TopIssues        []IssueSummary      `json:"top_issues"`
	PodHealthSummary map[string]PodHealth `json:"pod_health_summary"`
	Error            string              `json:"error,omitempty"`
}

// IssueSummary represents a summary of an issue
type IssueSummary struct {
	IssueType string   `json:"issue_type"`
	Count     int      `json:"count"`
	Pods      []string `json:"pods"`
	Severity  string   `json:"severity"`
}

// PodHealth represents the health status of a pod
type PodHealth struct {
	Status        string `json:"status"` // Healthy, Warning, Unhealthy
	LastErrorTime string `json:"last_error_time"`
	ErrorCount    int    `json:"error_count"`
	EventType     string `json:"event_type"`
}

// MonitorMCPServer holds the log directory path
type MonitorMCPServer struct {
	logDir string
}

// NewMonitorMCPServer creates a new monitor MCP server
func NewMonitorMCPServer(logDir string) *MonitorMCPServer {
	return &MonitorMCPServer{
		logDir: logDir,
	}
}

// parseBasicLogs parses basic logs to extract pod information
func (s *MonitorMCPServer) parseBasicLogs(logFile, targetPod string) ([]PodInfo, error) {
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	podMap := make(map[string]*PodInfo)

	// Regular expressions to extract pod information from different log formats
	// Format 1: CRITICAL EVENT: Pod default/slow-oom-test-84ddbbc84-5hdl4 has abnormal condition Ready with reason: ContainersNotReady
	podConditionRegex := regexp.MustCompile(`CRITICAL EVENT: Pod ([^/]+)/([^ ]+) has abnormal condition ([^ ]+) with reason: (.+)`)
	
	// Format 2: CRITICAL EVENT: Container slow-oom-container in pod default/slow-oom-test-84ddbbc84-5hdl4 was terminated with reason: OOMKilled
	containerTerminatedRegex := regexp.MustCompile(`CRITICAL EVENT: Container ([^ ]+) in pod ([^/]+)/([^ ]+) was terminated with reason: (.+)`)
	
	// Format 3: CRITICAL EVENT: Container slow-oom-container in pod default/slow-oom-test-84ddbbc84-5hdl4 has restarted (\d+) times
	containerRestartRegex := regexp.MustCompile(`CRITICAL EVENT: Container ([^ ]+) in pod ([^/]+)/([^ ]+) has restarted (\d+) times`)
	
	// Format 4: CRITICAL EVENT: Deployment default/slow-oom-test has condition Available with reason: (.+)
	deploymentConditionRegex := regexp.MustCompile(`CRITICAL EVENT: Deployment ([^/]+)/([^ ]+) has condition ([^ ]+) with reason: (.+)`)
	
	// Format 5: logrus structured format: msg=CRITICAL_EVENT eventType=Ready message=ContainersNotReady namespace=default objName=slow-oom-test-84ddbbc84-5hdl4 objType=pod
	logrusCriticalEventRegex := regexp.MustCompile(`msg=CRITICAL_EVENT eventType=([^ ]+) message=([^\s]+) namespace=([^ ]+) objName=([^ ]+) objType=([^ \s]+)`)

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Match logrus structured format
		logrusMatches := logrusCriticalEventRegex.FindStringSubmatch(line)
		if len(logrusMatches) >= 6 {
			eventType := logrusMatches[1]
			message := logrusMatches[2]
			namespace := logrusMatches[3]
			objName := logrusMatches[4]
			objType := logrusMatches[5]

			if targetPod != "" && targetPod != objName {
				continue
			}

			// Handle different object types
			if strings.ToLower(objType) == "pod" || strings.ToLower(objType) == "pods" {
				key := fmt.Sprintf("%s/%s", namespace, objName)
				if _, exists := podMap[key]; !exists {
					podMap[key] = &PodInfo{
						Name:      objName,
						Status:    "Unhealthy",
						Phase:     "Unstable",
						Namespace: namespace,
					}
				}

				// Add condition based on event type
				podMap[key].Conditions = append(podMap[key].Conditions, ConditionInfo{
					Type:    eventType,
					Status:  "True",
					Reason:  message,
					Message: fmt.Sprintf("Pod condition: %s with reason: %s", eventType, message),
				})
			} else if strings.ToLower(objType) == "container" {
				// Find the pod that contains this container
				for _, pod := range podMap {
					if pod.Name == objName || strings.Contains(pod.Name, objName) {
						pod.Conditions = append(pod.Conditions, ConditionInfo{
							Type:    eventType,
							Status:  "True",
							Reason:  message,
							Message: fmt.Sprintf("Container %s condition: %s with reason: %s", objName, eventType, message),
						})
						break
					}
				}
				
				// If pod doesn't exist yet, create it
				if targetPod != "" {
					key := fmt.Sprintf("%s/%s", namespace, targetPod)
					if _, exists := podMap[key]; !exists {
						podMap[key] = &PodInfo{
							Name:      targetPod,
							Status:    "Unhealthy",
							Phase:     "Unstable",
							Namespace: namespace,
						}
					}
					podMap[key].Conditions = append(podMap[key].Conditions, ConditionInfo{
						Type:    eventType,
						Status:  "True",
						Reason:  message,
						Message: fmt.Sprintf("Container %s condition: %s with reason: %s", objName, eventType, message),
					})
				}
			}
		}

		// Match plain text format
		podConditionMatches := podConditionRegex.FindStringSubmatch(line)
		if len(podConditionMatches) >= 5 {
			namespace := podConditionMatches[1]
			podName := podConditionMatches[2]
			conditionType := podConditionMatches[3]
			reason := podConditionMatches[4]

			if targetPod != "" && targetPod != podName {
				continue
			}

			key := fmt.Sprintf("%s/%s", namespace, podName)
			if _, exists := podMap[key]; !exists {
				podMap[key] = &PodInfo{
					Name:      podName,
					Status:    "Unhealthy",
					Phase:     "Unstable",
					Namespace: namespace,
				}
			}

			// Add condition
			podMap[key].Conditions = append(podMap[key].Conditions, ConditionInfo{
				Type:    conditionType,
				Status:  "True",
				Reason:  reason,
				Message: fmt.Sprintf("Pod condition: %s with reason: %s", conditionType, reason),
			})
		}

		containerTerminatedMatches := containerTerminatedRegex.FindStringSubmatch(line)
		if len(containerTerminatedMatches) >= 5 {
			containerName := containerTerminatedMatches[1]
			namespace := containerTerminatedMatches[2]
			podName := containerTerminatedMatches[3]
			reason := containerTerminatedMatches[4]

			if targetPod != "" && targetPod != podName {
				continue
			}

			key := fmt.Sprintf("%s/%s", namespace, podName)
			if _, exists := podMap[key]; !exists {
				podMap[key] = &PodInfo{
					Name:      podName,
					Status:    "Unhealthy",
					Phase:     "Unstable",
					Namespace: namespace,
				}
			}

			// Add termination condition
			podMap[key].Conditions = append(podMap[key].Conditions, ConditionInfo{
				Type:    "Terminated",
				Status:  "True",
				Reason:  reason,
				Message: fmt.Sprintf("Container %s terminated with reason: %s", containerName, reason),
			})
		}

		containerRestartMatches := containerRestartRegex.FindStringSubmatch(line)
		if len(containerRestartMatches) >= 5 {
			containerName := containerRestartMatches[1]
			namespace := containerRestartMatches[2]
			podName := containerRestartMatches[3]
			restartCount, _ := strconv.Atoi(containerRestartMatches[4])

			if targetPod != "" && targetPod != podName {
				continue
			}

			key := fmt.Sprintf("%s/%s", namespace, podName)
			if _, exists := podMap[key]; !exists {
				podMap[key] = &PodInfo{
					Name:      podName,
					Status:    "Unhealthy",
					Phase:     "Unstable",
					Namespace: namespace,
					Restarts:  int32(restartCount),
				}
			} else {
				podMap[key].Restarts = int32(restartCount)
			}

			// Add restart condition
			podMap[key].Conditions = append(podMap[key].Conditions, ConditionInfo{
				Type:    "HighRestartCount",
				Status:  "True",
				Reason:  "FrequentRestarts",
				Message: fmt.Sprintf("Container %s has restarted %d times", containerName, restartCount),
			})
		}

		deploymentConditionMatches := deploymentConditionRegex.FindStringSubmatch(line)
		if len(deploymentConditionMatches) >= 5 {
			// Deployment events are associated with pods in the deployment
			// For now, we'll just log that we found a deployment event
			// In a real implementation, we might want to link deployments to their pods
		}
	}

	// Convert map to slice
	pods := make([]PodInfo, 0, len(podMap))
	for _, pod := range podMap {
		// Set status based on conditions
		if len(pod.Conditions) > 0 {
			pod.Status = "Unhealthy"
			pod.Phase = "Unstable"
		} else {
			pod.Status = "Healthy"
			pod.Phase = "Running"
		}
		pods = append(pods, *pod)
	}

	return pods, nil
}

// parseCriticalEvents parses critical events from the log file
func (s *MonitorMCPServer) parseCriticalEvents(logFile string) ([]EventInfo, error) {
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	events := []EventInfo{}

	// Regular expression to parse logrus format: time="..." level=error msg=CRITICAL_EVENT_LOG event=BackOff message="..." name=... type=Pod
	logrusRegex := regexp.MustCompile(`time="([^"]+)" level=error msg=CRITICAL_EVENT_LOG event=([^ ]+) message="([^"]*)" name=([^ ]+) type=([^ \s]+)`)

	// Regular expression to parse the aggregation format: CRITICAL|timestamp|namespace|type|name|event|message
	aggregationFormatRegex := regexp.MustCompile(`^CRITICAL\|([^|]+)\|([^|]+)\|([^|]+)\|([^|]+)\|([^|]+)\|(.*)$`)

	for _, line := range lines {
		// Check if it's in the aggregation format first (CRITICAL|timestamp|namespace|type|name|event|message)
		aggMatches := aggregationFormatRegex.FindStringSubmatch(line)
		if len(aggMatches) >= 7 {
			event := EventInfo{
				ObjectKind: aggMatches[3], // type
				ObjectName: aggMatches[4], // name
				Type:       aggMatches[5], // event
				Message:    aggMatches[6], // message
				Count:      1,
			}

			// Parse timestamp
			timestamp, err := time.Parse(time.RFC3339, aggMatches[1])
			if err != nil {
				timestamp = time.Now() // fallback
			}
			event.FirstSeen = timestamp
			event.LastSeen = timestamp

			events = append(events, event)
			continue
		}

		if strings.Contains(line, "CRITICAL_EVENT_LOG") {
			event := EventInfo{
				Message: line,
				Count:   1,
			}

			// Try to parse logrus format
			matches := logrusRegex.FindStringSubmatch(line)
			if len(matches) >= 6 {
				// Parse timestamp
				timestamp, err := time.Parse("2006-01-02T15:04:05Z07:00", matches[1]) // RFC3339 format
				if err != nil {
					// Try alternative format from the logs
					timestamp, err = time.Parse("2006-01-02T15:04:05-07:00", matches[1])
					if err != nil {
						timestamp = time.Now() // fallback
					}
				}
				event.FirstSeen = timestamp
				event.LastSeen = timestamp

				event.Type = matches[2]      // eventType
				event.Message = matches[3]   // message
				event.ObjectName = matches[4] // name
				event.ObjectKind = matches[5] // type
			} else {
				// Fallback to basic extraction
				event.FirstSeen = time.Now()
				event.LastSeen = time.Now()

				// Extract object information from plain text
				if strings.Contains(line, "Pod") {
					event.ObjectKind = "Pod"
					// Extract pod name from log line
					parts := strings.Split(line, " ")
					for i, part := range parts {
						if part == "Pod" && i+1 < len(parts) {
							objPart := parts[i+1]
							if strings.Contains(objPart, "/") {
								objParts := strings.Split(objPart, "/")
								if len(objParts) >= 2 {
									event.ObjectName = objParts[1]
								}
							}
							break
						}
					}
				} else if strings.Contains(line, "container") {
					event.ObjectKind = "Container"
					// Extract container name
					if idx := strings.Index(line, "container"); idx != -1 {
						remainder := line[idx+len("container")+1:]
						parts := strings.Fields(remainder)
						if len(parts) > 0 {
							event.ObjectName = parts[0]
						}
					}
				} else if strings.Contains(line, "deployment") {
					event.ObjectKind = "Deployment"
					// Extract deployment name
					if idx := strings.Index(line, "deployment"); idx != -1 {
						remainder := line[idx+len("deployment")+1:]
						parts := strings.Fields(remainder)
						if len(parts) > 0 {
							event.ObjectName = parts[0]
						}
					}
				}
			}

			events = append(events, event)
		}
	}

	return events, nil
}

// parseMemoryUsage parses memory usage data from JSON file
func (s *MonitorMCPServer) parseMemoryUsage(jsonFile string) ([]struct {
	Timestamp   time.Time `json:"timestamp"`
	MemoryUsage int64     `json:"memory_usage"`
}, error) {
	content, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		return nil, err
	}

	// Define a temporary struct that matches the JSON format
	var rawJsonData []map[string]interface{}
	if err := json.Unmarshal(content, &rawJsonData); err != nil {
		return nil, err
	}

	// Convert to the expected struct format
	var data []struct {
		Timestamp   time.Time `json:"timestamp"`
		MemoryUsage int64     `json:"memory_usage"`
	}

	for _, item := range rawJsonData {
		tsStr, ok := item["Timestamp"].(string)
		if !ok {
			continue
		}
		
		// Parse the timestamp string to time.Time
		timestamp, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			// Try alternative format from the logs
			timestamp, err = time.Parse("2006-01-02T15:04:05-07:00", tsStr)
			if err != nil {
				continue
			}
		}

		memUsage, ok := item["MemoryUsage"].(float64) // JSON numbers are float64
		if !ok {
			continue
		}

		data = append(data, struct {
			Timestamp   time.Time `json:"timestamp"`
			MemoryUsage int64     `json:"memory_usage"`
		}{
			Timestamp:   timestamp,
			MemoryUsage: int64(memUsage),
		})
	}

	return data, nil
}

// handleMonitorPods handles the monitor_pods tool
func (s *MonitorMCPServer) handleMonitorPods(ctx context.Context, req *mcp.CallToolRequest, input PodMonitoringInput) (PodMonitoringOutput, error) {
	// Read pod information from the log directory
	podInfos := []PodInfo{}

	// Look for logs in the namespace directory
	namespaceDir := filepath.Join(s.logDir, "basic", input.Namespace)
	if _, err := os.Stat(namespaceDir); os.IsNotExist(err) {
		return PodMonitoringOutput{Error: fmt.Sprintf("Namespace %s not found in logs", input.Namespace)}, nil
	}

	// Read basic logs for the namespace
	logFile := filepath.Join(namespaceDir, "basic.log")
	if _, err := os.Stat(logFile); err == nil {
		// Parse the log file to extract pod information
		podInfos, err = s.parseBasicLogs(logFile, input.PodName)
		if err != nil {
			return PodMonitoringOutput{Error: fmt.Sprintf("Error parsing basic logs: %v", err)}, nil
		}
	}

	// Also check critical events
	criticalEventsFile := filepath.Join(namespaceDir, "critical_events.log")
	if _, err := os.Stat(criticalEventsFile); err == nil {
		// Add critical events to pod information
		events, err := s.parseCriticalEvents(criticalEventsFile)
		if err != nil {
			log.Printf("Error parsing critical events: %v", err)
		} else {
			// Associate events with pods
			for i := range podInfos {
				for _, event := range events {
					if strings.Contains(event.Message, podInfos[i].Name) {
						podInfos[i].LastEvent = event.Message
						break
					}
				}
			}
		}
	}

	// Read memory usage data if available
	memoryDir := filepath.Join(s.logDir, "memory_usage", input.Namespace)
	if input.PodName != "" {
		// If specific pod is requested, try to get memory data for it
		podMemoryDir := filepath.Join(memoryDir, input.PodName)
		if _, err := os.Stat(podMemoryDir); err == nil {
			// Look for memory data files in the pod directory
			entries, err := ioutil.ReadDir(podMemoryDir)
			if err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						containerName := entry.Name()
						memoryFile := filepath.Join(podMemoryDir, containerName, "memory_usage.json")
						if _, err := os.Stat(memoryFile); err == nil {
							// Parse memory usage data
							usage, err := s.parseMemoryUsage(memoryFile)
							if err == nil && len(usage) > 0 {
								// Update the pod info with memory usage
								for i := range podInfos {
									if podInfos[i].Name == input.PodName {
										// Just use the latest memory usage for simplicity
										latestUsage := usage[len(usage)-1]
										podInfos[i].MemoryUsage = &MemoryUsageInfo{
											CurrentBytes: latestUsage.MemoryUsage,
											Percentage:   0, // Calculate percentage if limit is known
										}
										break
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return PodMonitoringOutput{
		Pods: podInfos,
	}, nil
}

// handleGetClusterSummary handles the get_cluster_summary tool
func (s *MonitorMCPServer) handleGetClusterSummary(ctx context.Context, req *mcp.CallToolRequest, input ClusterSummaryInput) (ClusterSummaryOutput, error) {
	// Count total namespaces in the logs directory
	logDir := filepath.Join(s.logDir, "basic")
	entries, err := ioutil.ReadDir(logDir)
	if err != nil {
		return ClusterSummaryOutput{Error: fmt.Sprintf("Failed to read log directory: %v", err)}, nil
	}

	var namespaces []string
	for _, entry := range entries {
		if entry.IsDir() {
			namespaces = append(namespaces, entry.Name())
		}
	}

	// If specific namespaces are requested, filter them
	if len(input.Namespaces) > 0 {
		namespaces = input.Namespaces
	}

	// Analyze each namespace
	totalPods := 0
	runningPods := 0
	failedPods := 0
	criticalEvents := 0
	recentIssues := []string{}

	for _, namespace := range namespaces {
		namespaceDir := filepath.Join(logDir, namespace)
		if _, err := os.Stat(namespaceDir); os.IsNotExist(err) {
			continue
		}

		// Count pods from basic logs
		logFile := filepath.Join(namespaceDir, "basic.log")
		if _, err := os.Stat(logFile); err == nil {
			pods, err := s.parseBasicLogs(logFile, "")
			if err == nil {
				totalPods += len(pods)
				for _, pod := range pods {
					switch pod.Phase {
					case "Running":
						runningPods++
					case "Failed", "Unknown":
						failedPods++
					}
				}
			}
		}

		// Count critical events
		criticalEventsFile := filepath.Join(namespaceDir, "critical_events.log")
		if _, err := os.Stat(criticalEventsFile); err == nil {
			events, err := s.parseCriticalEvents(criticalEventsFile)
			if err == nil {
				criticalEvents += len(events)
				// Collect recent issues
				for _, event := range events {
					if len(recentIssues) < 10 { // Limit to 10 recent issues
						recentIssues = append(recentIssues, event.Message)
					}
				}
			}
		}
	}

	// Calculate overall status
	overallStatus := "healthy"
	if failedPods > totalPods*10/100 { // If failed pods exceed 10% of total
		overallStatus = "warning"
	}
	if criticalEvents > 10 { // If there are many critical events
		overallStatus = "critical"
	}

	// For simplicity, we'll return fixed node counts
	// In a real implementation, we would read node data from logs
	summary := SummaryInfo{
		TotalNodes:     3, // Assume 3 nodes
		ReadyNodes:     3, // Assume all nodes are ready
		TotalPods:      totalPods,
		RunningPods:    runningPods,
		FailedPods:     failedPods,
		CriticalEvents: criticalEvents,
		OverallStatus:  overallStatus,
		RecentIssues:   recentIssues,
	}

	return ClusterSummaryOutput{
		Summary: summary,
		Nodes:   []NodeInfo{}, // Placeholder - would read from node logs in real implementation
		Pods:    []PodSummary{}, // Placeholder - would aggregate from all namespaces
	}, nil
}

// handleDetectMemoryLeak handles the detect_memory_leak tool
func (s *MonitorMCPServer) handleDetectMemoryLeak(ctx context.Context, req *mcp.CallToolRequest, input MemoryLeakDetectionInput) (MemoryLeakDetectionOutput, error) {
	if input.Hours <= 0 {
		input.Hours = 24 // Default to 24 hours
	}

	// Read memory usage data for the specified pod/container
	memoryDir := filepath.Join(s.logDir, "memory_usage", input.Namespace, input.PodName, input.Container)
	memoryFile := filepath.Join(memoryDir, "memory_usage.json")

	if _, err := os.Stat(memoryFile); os.IsNotExist(err) {
		return MemoryLeakDetectionOutput{Error: fmt.Sprintf("No memory data found for pod %s/%s container %s", input.Namespace, input.PodName, input.Container)}, nil
	}

	// Parse memory usage data
	usageData, err := s.parseMemoryUsage(memoryFile)
	if err != nil {
		return MemoryLeakDetectionOutput{Error: fmt.Sprintf("Failed to parse memory data: %v", err)}, nil
	}

	if len(usageData) < 2 {
		return MemoryLeakDetectionOutput{Error: "Insufficient data points for analysis (need at least 2)"}, nil
	}

	// Calculate memory growth trend
	// For simplicity, we'll calculate a basic linear trend
	var totalIncrease int64
	var count int
	for i := 1; i < len(usageData); i++ {
		increase := usageData[i].MemoryUsage - usageData[i-1].MemoryUsage
		totalIncrease += increase
		count++
	}

	var avgIncrease int64
	if count > 0 {
		avgIncrease = totalIncrease / int64(count)
	}

	// Calculate average and max memory usage
	var totalMem, maxMem int64
	for _, data := range usageData {
		totalMem += data.MemoryUsage
		if data.MemoryUsage > maxMem {
			maxMem = data.MemoryUsage
		}
	}
	avgMem := totalMem / int64(len(usageData))

	// Calculate slope in MB per hour
	// Simplified calculation - in reality, we'd need to consider the time intervals
	timeDiff := usageData[len(usageData)-1].Timestamp.Sub(usageData[0].Timestamp)
	hoursDiff := timeDiff.Hours()
	if hoursDiff <= 0 {
		hoursDiff = 1 // Prevent division by zero
	}
	slopeMBPerHour := float64(avgIncrease) / 1024 / 1024 // Convert to MB

	isLeaking := slopeMBPerHour > 10.0 // If growing more than 10MB per hour, consider leaking
	confidence := "low"
	if slopeMBPerHour > 50.0 {
		confidence = "high"
	} else if slopeMBPerHour > 20.0 {
		confidence = "medium"
	}

	trend := "stable"
	if slopeMBPerHour > 1.0 {
		trend = "increasing"
	} else if slopeMBPerHour < -1.0 {
		trend = "decreasing"
	}

	return MemoryLeakDetectionOutput{
		PodName:        input.PodName,
		Container:      input.Container,
		SlopeMBPerHour: slopeMBPerHour,
		IsLeaking:      isLeaking,
		Confidence:     confidence,
		Trend:          trend,
		MaxMemory:      maxMem,
		AvgMemory:      avgMem,
	}, nil
}

// handleGetRecentEvents handles the get_recent_events tool
func (s *MonitorMCPServer) handleGetRecentEvents(ctx context.Context, req *mcp.CallToolRequest, input RecentEventsInput) (RecentEventsOutput, error) {
	if input.Limit <= 0 {
		input.Limit = 50 // Default limit
	}

	events := []EventInfo{}

	// Determine which namespaces to check
	var namespaces []string
	if input.Namespace != "" {
		namespaces = []string{input.Namespace}
	} else {
		// Get all namespaces from basic logs directory
		logDir := filepath.Join(s.logDir, "basic")
		entries, err := ioutil.ReadDir(logDir)
		if err != nil {
			return RecentEventsOutput{Error: fmt.Sprintf("Failed to read log directory: %v", err)}, nil
		}
		for _, entry := range entries {
			if entry.IsDir() {
				namespaces = append(namespaces, entry.Name())
			}
		}
	}

	// Collect events from all relevant namespaces
	for _, namespace := range namespaces {
		criticalEventsFile := filepath.Join(s.logDir, "basic", namespace, "critical_events.log")
		if _, err := os.Stat(criticalEventsFile); err == nil {
			namespaceEvents, err := s.parseCriticalEvents(criticalEventsFile)
			if err == nil {
				events = append(events, namespaceEvents...)
			}
		}
	}

	// Limit the number of events returned
	if len(events) > input.Limit {
		events = events[len(events)-input.Limit:]
	}

	return RecentEventsOutput{
		Events: events,
	}, nil
}

// handleGetResourceUtilization handles the get_resource_utilization tool
func (s *MonitorMCPServer) handleGetResourceUtilization(ctx context.Context, req *mcp.CallToolRequest, input ResourceUtilizationInput) (ResourceUtilizationOutput, error) {
	// Read memory usage data
	memoryDir := filepath.Join(s.logDir, "memory_usage")

	var podUtilizations []PodUtilization

	// Get namespaces to scan
	var namespaces []string
	if input.Namespace != "" {
		namespaces = []string{input.Namespace}
	} else {
		// Scan all namespaces in memory usage directory
		entries, err := ioutil.ReadDir(memoryDir)
		if err != nil {
			return ResourceUtilizationOutput{Error: fmt.Sprintf("Failed to read memory usage directory: %v", err)}, nil
		}
		for _, entry := range entries {
			if entry.IsDir() {
				namespaces = append(namespaces, entry.Name())
			}
		}
	}

	// Collect pod utilization data
	for _, namespace := range namespaces {
		namespaceDir := filepath.Join(memoryDir, namespace)
		if _, err := os.Stat(namespaceDir); os.IsNotExist(err) {
			continue
		}

		// Scan pods in this namespace
		podEntries, err := ioutil.ReadDir(namespaceDir)
		if err != nil {
			continue
		}

		for _, podEntry := range podEntries {
			if !podEntry.IsDir() {
				continue
			}

			podName := podEntry.Name()

			// Scan containers in this pod
			containerEntries, err := ioutil.ReadDir(filepath.Join(namespaceDir, podName))
			if err != nil {
				continue
			}

			for _, containerEntry := range containerEntries {
				if !containerEntry.IsDir() {
					continue
				}

				containerName := containerEntry.Name()

				// Read memory usage data for this container
				memoryFile := filepath.Join(namespaceDir, podName, containerName, "memory_usage.json")
				if _, err := os.Stat(memoryFile); err == nil {
					usageData, err := s.parseMemoryUsage(memoryFile)
					if err == nil && len(usageData) > 0 {
						// Use the latest memory usage data
						latestUsage := usageData[len(usageData)-1]

						utilization := PodUtilization{
							PodName:   podName,
							Namespace: namespace,
							MemoryUsage: ResourceUsage{
								Used:         strconv.FormatInt(latestUsage.MemoryUsage, 10),
								UsagePercent: 0, // Calculate percentage if limit is known
							},
							// CPU data would be read from similar files if available
							CPUUsage: ResourceUsage{
								Used:         "N/A", // Placeholder - would read from CPU logs if available
								UsagePercent: 0,
							},
						}
						podUtilizations = append(podUtilizations, utilization)
					}
				}
			}
		}
	}

	return ResourceUtilizationOutput{
		Namespace:      input.Namespace,
		PodUtilization: podUtilizations,
		NodeUtilization: []NodeUtilization{}, // Placeholder - would read from node logs in real implementation
	}, nil
}

// parseAggregatedLogEvents parses events from the for_aggregation.log file
func (s *MonitorMCPServer) parseAggregatedLogEvents(logFile string, timeThreshold time.Time) ([]EventInfo, error) {
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	events := []EventInfo{}

	// Regular expression to parse the aggregation format: CRITICAL|timestamp|namespace|type|name|event|message
	aggregationFormatRegex := regexp.MustCompile(`^CRITICAL\|([^|]+)\|([^|]+)\|([^|]+)\|([^|]+)\|([^|]+)\|(.*)$`)

	for _, line := range lines {
		// Check if it's in the aggregation format (CRITICAL|timestamp|namespace|type|name|event|message)
		aggMatches := aggregationFormatRegex.FindStringSubmatch(line)
		if len(aggMatches) >= 7 {
			// Parse timestamp
			timestamp, err := time.Parse(time.RFC3339, aggMatches[1])
			if err != nil {
				continue // Skip invalid timestamps
			}

			// Only include events within the time threshold
			if timestamp.Before(timeThreshold) {
				continue
			}

			event := EventInfo{
				ObjectKind: aggMatches[3], // type
				ObjectName: aggMatches[4], // name
				Type:       aggMatches[5], // event
				Message:    aggMatches[6], // message
				FirstSeen:  timestamp,
				LastSeen:   timestamp,
				Count:      1,
			}

			events = append(events, event)
		}
	}

	return events, nil
}

// generateTopIssuesSummary generates a summary of the most common issues
func (s *MonitorMCPServer) generateTopIssuesSummary(podHealth map[string]PodHealth) []IssueSummary {
	// Count occurrences of each event type
	eventTypeCount := make(map[string]int)
	eventTypePods := make(map[string][]string) // Track which pods had each event type

	for podName, health := range podHealth {
		eventTypeCount[health.EventType]++
		if _, exists := eventTypePods[health.EventType]; !exists {
			eventTypePods[health.EventType] = []string{}
		}
		eventTypePods[health.EventType] = append(eventTypePods[health.EventType], podName)
	}

	// Convert to slice and sort by count
	type eventTypeCountPair struct {
		eventType string
		count     int
	}
	
	var pairs []eventTypeCountPair
	for eventType, count := range eventTypeCount {
		pairs = append(pairs, eventTypeCountPair{eventType, count})
	}
	
	// Sort by count in descending order
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	// Take top 5 issues
	topCount := len(pairs)
	if topCount > 5 {
		topCount = 5
	}

	var topIssues []IssueSummary
	for i := 0; i < topCount; i++ {
		pair := pairs[i]
		severity := "Medium"
		if strings.Contains(strings.ToLower(pair.eventType), "error") || 
		   strings.Contains(strings.ToLower(pair.eventType), "terminated") || 
		   strings.Contains(strings.ToLower(pair.eventType), "oom") {
			severity = "High"
		} else if strings.Contains(strings.ToLower(pair.eventType), "warning") {
			severity = "Low"
		}
		
		topIssues = append(topIssues, IssueSummary{
			IssueType: pair.eventType,
			Count:     pair.count,
			Pods:      eventTypePods[pair.eventType],
			Severity:  severity,
		})
	}

	return topIssues
}

// handleAggregatedLogAnalysis handles the analyze_aggregated_logs tool
func (s *MonitorMCPServer) handleAggregatedLogAnalysis(ctx context.Context, req *mcp.CallToolRequest, input AggregatedLogAnalysisInput) (AggregatedLogAnalysisOutput, error) {
	if input.Hours <= 0 {
		input.Hours = 24 // Default to 24 hours
	}

	// Determine which namespaces to check
	var namespaces []string
	if input.Namespace != "" {
		namespaces = []string{input.Namespace}
	} else {
		// Get all namespaces from basic logs directory
		logDir := filepath.Join(s.logDir, "basic")
		entries, err := ioutil.ReadDir(logDir)
		if err != nil {
			return AggregatedLogAnalysisOutput{Error: fmt.Sprintf("Failed to read log directory: %v", err)}, nil
		}
		for _, entry := range entries {
			if entry.IsDir() {
				namespaces = append(namespaces, entry.Name())
			}
		}
	}

	// Calculate time threshold
	timeThreshold := time.Now().Add(-time.Duration(input.Hours) * time.Hour)

	// Initialize analysis results
	analysis := AggregatedLogAnalysisOutput{
		Namespace:        input.Namespace,
		AnalysisPeriod:   fmt.Sprintf("Last %d hours", input.Hours),
		PodHealthSummary: make(map[string]PodHealth),
	}

	// Process aggregated logs for each namespace
	for _, namespace := range namespaces {
		aggregatedLogFile := filepath.Join(s.logDir, "basic", namespace, "for_aggregation.log")
		if _, err := os.Stat(aggregatedLogFile); err == nil {
			events, err := s.parseAggregatedLogEvents(aggregatedLogFile, timeThreshold)
			if err == nil {
				analysis.TotalEvents += len(events)
				
				// Categorize events and build summaries
				for _, event := range events {
					// Update pod health summary
					if _, exists := analysis.PodHealthSummary[event.ObjectName]; !exists {
						analysis.PodHealthSummary[event.ObjectName] = PodHealth{
							Status:     "Healthy",
							ErrorCount: 0,
						}
					}
					
					podHealth := analysis.PodHealthSummary[event.ObjectName]
					podHealth.ErrorCount++
					podHealth.EventType = event.Type
					podHealth.LastErrorTime = event.FirstSeen.Format(time.RFC3339)
					
					// Determine severity based on event type
					if strings.Contains(strings.ToLower(event.Type), "error") || 
					   strings.Contains(strings.ToLower(event.Type), "terminated") || 
					   strings.Contains(strings.ToLower(event.Message), "oom") {
						podHealth.Status = "Unhealthy"
						analysis.CriticalEvents++
					} else if strings.Contains(strings.ToLower(event.Type), "warning") {
						if podHealth.Status == "Healthy" {
							podHealth.Status = "Warning"
						}
						analysis.Warnings++
					} else {
						if podHealth.Status == "Healthy" {
							analysis.ErrorEvents++
						}
					}
					
					analysis.PodHealthSummary[event.ObjectName] = podHealth
				}
			}
		}
	}

	// Generate top issues summary
	analysis.TopIssues = s.generateTopIssuesSummary(analysis.PodHealthSummary)

	return analysis, nil
}

func main() {
	// Get log directory from command line argument or use default
	logDir := "/root/workspace/test-env/logs" // Default log directory
	if len(os.Args) > 2 {
		logDir = os.Args[2]
	}

	// Get port from command line argument or use default
	port := 3001 // Default port
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &port)
	}

	// Create MCP streamable HTTP server.
	server := mcp.NewServer("monitor-mcp-server", "1.0.0", mcp.WithServerAddress(fmt.Sprintf(":%d", port)))

	// Create server instance
	monitorServer := NewMonitorMCPServer(logDir)

	// Create monitor_pods tool
	monitorPodsTool := mcp.NewTool(
		"monitor_pods",
		mcp.WithDescription("Monitor the status and resource usage of pods in a Kubernetes namespace by reading data collected by the otisbrain monitor module. Can monitor a specific pod or all pods in the namespace."),
		mcp.WithInputStruct[PodMonitoringInput](),
	)
	server.RegisterTool(monitorPodsTool, mcp.NewTypedToolHandler(
		func(ctx context.Context, req *mcp.CallToolRequest, input PodMonitoringInput) (PodMonitoringOutput, error) {
			return monitorServer.handleMonitorPods(ctx, req, input)
		},
	))

	// Create get_cluster_summary tool
	clusterSummaryTool := mcp.NewTool(
		"get_cluster_summary",
		mcp.WithDescription("Get the recent important events and status summary of the Kubernetes cluster by analyzing data collected by the otisbrain monitor module, including node status, pod status, and critical events."),
		mcp.WithInputStruct[ClusterSummaryInput](),
	)
	server.RegisterTool(clusterSummaryTool, mcp.NewTypedToolHandler(
		func(ctx context.Context, req *mcp.CallToolRequest, input ClusterSummaryInput) (ClusterSummaryOutput, error) {
			return monitorServer.handleGetClusterSummary(ctx, req, input)
		},
	))

	// Create detect_memory_leak tool
	memoryLeakTool := mcp.NewTool(
		"detect_memory_leak",
		mcp.WithDescription("Detect potential memory leaks in a specific pod/container by analyzing memory usage trends from data collected by the otisbrain monitor module."),
		mcp.WithInputStruct[MemoryLeakDetectionInput](),
	)
	server.RegisterTool(memoryLeakTool, mcp.NewTypedToolHandler(
		func(ctx context.Context, req *mcp.CallToolRequest, input MemoryLeakDetectionInput) (MemoryLeakDetectionOutput, error) {
			return monitorServer.handleDetectMemoryLeak(ctx, req, input)
		},
	))

	// Create get_recent_events tool
	recentEventsTool := mcp.NewTool(
		"get_recent_events",
		mcp.WithDescription("Get recent events in the Kubernetes cluster from data collected by the otisbrain monitor module, optionally filtered by namespace."),
		mcp.WithInputStruct[RecentEventsInput](),
	)
	server.RegisterTool(recentEventsTool, mcp.NewTypedToolHandler(
		func(ctx context.Context, req *mcp.CallToolRequest, input RecentEventsInput) (RecentEventsOutput, error) {
			return monitorServer.handleGetRecentEvents(ctx, req, input)
		},
	))

	// Create get_resource_utilization tool
	resourceUtilizationTool := mcp.NewTool(
		"get_resource_utilization",
		mcp.WithDescription("Get resource utilization for nodes and pods in the cluster from data collected by the otisbrain monitor module, optionally filtered by namespace."),
		mcp.WithInputStruct[ResourceUtilizationInput](),
	)
	server.RegisterTool(resourceUtilizationTool, mcp.NewTypedToolHandler(
		func(ctx context.Context, req *mcp.CallToolRequest, input ResourceUtilizationInput) (ResourceUtilizationOutput, error) {
			return monitorServer.handleGetResourceUtilization(ctx, req, input)
		},
	))

	// Create analyze_aggregated_logs tool
	analyzeLogsTool := mcp.NewTool(
		"analyze_aggregated_logs",
		mcp.WithDescription("Analyze aggregated logs from the monitor module to identify patterns, top issues, and health status of pods over a specified time period."),
		mcp.WithInputStruct[AggregatedLogAnalysisInput](),
	)
	server.RegisterTool(analyzeLogsTool, mcp.NewTypedToolHandler(
		func(ctx context.Context, req *mcp.CallToolRequest, input AggregatedLogAnalysisInput) (AggregatedLogAnalysisOutput, error) {
			return monitorServer.handleAggregatedLogAnalysis(ctx, req, input)
		},
	))

	// Start HTTP server.
	fmt.Printf("Starting Monitor MCP Server on port %d, reading logs from: %s\n", port, logDir)
	fmt.Printf("Monitor MCP Server ready with 6 tools based on otisbrain monitor data:\n")
	fmt.Printf("- monitor_pods: Reads pod status from basic logs\n")
	fmt.Printf("- get_cluster_summary: Analyzes cluster health from collected logs\n")
	fmt.Printf("- detect_memory_leak: Analyzes memory trends from memory_usage logs\n")
	fmt.Printf("- get_recent_events: Retrieves critical events from event logs\n")
	fmt.Printf("- get_resource_utilization: Gets resource usage from memory_usage logs\n")
	fmt.Printf("- analyze_aggregated_logs: Analyzes aggregated logs for patterns and issues\n")

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}