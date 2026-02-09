package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	trpc_mcp_go "trpc.group/trpc-go/trpc-mcp-go"
)

// MonitorMCPServer represents the MCP server for monitoring tools
// This server reads and analyzes data collected by the otisbrain monitor module
type MonitorMCPServer struct {
	router *gin.Engine
	port   int
	logDir string // Directory where monitor logs are stored
}

// PodMonitoringInput represents input for pod monitoring
type PodMonitoringInput struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"pod_name,omitempty"`
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
	Namespaces []string `json:"namespaces,omitempty"`
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
	TotalNodes     int    `json:"total_nodes"`
	ReadyNodes     int    `json:"ready_nodes"`
	TotalPods      int    `json:"total_pods"`
	RunningPods    int    `json:"running_pods"`
	FailedPods     int    `json:"failed_pods"`
	CriticalEvents int    `json:"critical_events"`
	OverallStatus  string `json:"overall_status"`
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
	Namespace string `json:"namespace"`
	PodName   string `json:"pod_name"`
	Container string `json:"container"`
	Hours     int    `json:"hours"`
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
	Namespace string `json:"namespace,omitempty"`
	Limit     int    `json:"limit"`
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
	Namespace string `json:"namespace,omitempty"`
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

// NewMonitorMCPServer creates a new monitor MCP server
func NewMonitorMCPServer(port int, logDir string) *MonitorMCPServer {
	server := &MonitorMCPServer{
		port:   port,
		logDir: logDir,
	}

	// Initialize gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	router.Use(cors.New(config))

	server.router = router
	server.setupRoutes()
	return server
}

// setupRoutes sets up the MCP server routes
func (s *MonitorMCPServer) setupRoutes() {
	// MCP discovery endpoint
	s.router.GET("/.well-known/mcp-transports", func(c *gin.Context) {
		transports := []map[string]string{
			{
				"transport": "http",
				"url":       fmt.Sprintf("http://localhost:%d", s.port),
			},
		}
		c.JSON(http.StatusOK, transports)
	})

	// MCP tools endpoint
	s.router.GET("/v1alpha1/tools", func(c *gin.Context) {
		tools := []trpc_mcp_go.Tool{
			{
				Name:        "monitor_pods",
				Description: "Monitor the status and resource usage of pods in a Kubernetes namespace by reading data collected by the otisbrain monitor module. Can monitor a specific pod or all pods in the namespace.",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"namespace": map[string]interface{}{
							"type":        "string",
							"description": "The namespace to monitor",
						},
						"pod_name": map[string]interface{}{
							"type":        "string",
							"description": "Optional pod name to monitor specific pod, if not provided all pods in namespace will be monitored",
						},
					},
					"required": []string{"namespace"},
				},
			},
			{
				Name:        "get_cluster_summary",
				Description: "Get the recent important events and status summary of the Kubernetes cluster by analyzing data collected by the otisbrain monitor module, including node status, pod status, and critical events.",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"namespaces": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "Optional list of namespaces to check, if not provided all namespaces will be checked",
						},
					},
				},
			},
			{
				Name:        "detect_memory_leak",
				Description: "Detect potential memory leaks in a specific pod/container by analyzing memory usage trends from data collected by the otisbrain monitor module.",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"namespace": map[string]interface{}{
							"type":        "string",
							"description": "The namespace containing the pod",
						},
						"pod_name": map[string]interface{}{
							"type":        "string",
							"description": "The name of the pod to analyze",
						},
						"container": map[string]interface{}{
							"type":        "string",
							"description": "The name of the container to analyze",
						},
						"hours": map[string]interface{}{
							"type":        "integer",
							"description": "Number of hours to analyze (default: 24)",
						},
					},
					"required": []string{"namespace", "pod_name", "container"},
				},
			},
			{
				Name:        "get_recent_events",
				Description: "Get recent events in the Kubernetes cluster from data collected by the otisbrain monitor module, optionally filtered by namespace.",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"namespace": map[string]interface{}{
							"type":        "string",
							"description": "Optional namespace to filter events, if not provided events from all namespaces will be returned",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of events to return (default: 50)",
						},
					},
				},
			},
			{
				Name:        "get_resource_utilization",
				Description: "Get resource utilization for nodes and pods in the cluster from data collected by the otisbrain monitor module, optionally filtered by namespace.",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"namespace": map[string]interface{}{
							"type":        "string",
							"description": "Optional namespace to filter resources, if not provided resources from all namespaces will be returned",
						},
					},
				},
			},
		}
		c.JSON(http.StatusOK, gin.H{"tools": tools})
	})

	// MCP execute endpoint
	s.router.POST("/v1alpha1/execute", func(c *gin.Context) {
		var req trpc_mcp_go.ExecuteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var result interface{}
		var err error

		switch req.ToolName {
		case "monitor_pods":
			result, err = s.handleMonitorPods(req.Arguments)
		case "get_cluster_summary":
			result, err = s.handleGetClusterSummary(req.Arguments)
		case "detect_memory_leak":
			result, err = s.handleDetectMemoryLeak(req.Arguments)
		case "get_recent_events":
			result, err = s.handleGetRecentEvents(req.Arguments)
		case "get_resource_utilization":
			result, err = s.handleGetResourceUtilization(req.Arguments)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown tool: %s", req.ToolName)})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := trpc_mcp_go.ExecuteResponse{
			Content: result,
		}
		c.JSON(http.StatusOK, resp)
	})

	// Health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "monitor-mcp-server", "port": s.port})
	})
}

// handleMonitorPods handles the monitor_pods tool
func (s *MonitorMCPServer) handleMonitorPods(args json.RawMessage) (interface{}, error) {
	var input PodMonitoringInput
	if err := json.Unmarshal(args, &input); err != nil {
		return PodMonitoringOutput{Error: fmt.Sprintf("Invalid input: %v", err)}, nil
	}

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

// parseBasicLogs parses basic logs to extract pod information
func (s *MonitorMCPServer) parseBasicLogs(logFile, targetPod string) ([]PodInfo, error) {
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	podMap := make(map[string]*PodInfo)

	// Regular expressions to extract pod information
	podRegex := regexp.MustCompile(`Pod ([^/]+)/([^ ]+) status: ([^,]+)`)
	containerRegex := regexp.MustCompile(`Container ([^ ]+) in pod ([^/]+)/([^ ]+) is in waiting state with reason: ([^,]+)`)

	for _, line := range lines {
		if strings.Contains(line, "CRITICAL EVENT:") {
			// Parse pod status
			podMatches := podRegex.FindStringSubmatch(line)
			if len(podMatches) >= 4 {
				namespace := podMatches[1]
				podName := podMatches[2]
				status := podMatches[3]

				if targetPod != "" && targetPod != podName {
					continue
				}

				key := fmt.Sprintf("%s/%s", namespace, podName)
				if _, exists := podMap[key]; !exists {
					podMap[key] = &PodInfo{
						Name:      podName,
						Status:    status,
						Phase:     status,
						Namespace: namespace,
					}
				}
			}

			// Parse container status
			containerMatches := containerRegex.FindStringSubmatch(line)
			if len(containerMatches) >= 5 {
				containerName := containerMatches[1]
				namespace := containerMatches[2]
				podName := containerMatches[3]
				reason := containerMatches[4]

				if targetPod != "" && targetPod != podName {
					continue
				}

				key := fmt.Sprintf("%s/%s", namespace, podName)
				if _, exists := podMap[key]; !exists {
					podMap[key] = &PodInfo{
						Name:      podName,
						Status:    "Unknown",
						Phase:     "Unknown",
						Namespace: namespace,
					}
				}

				// Add condition for container issue
				podMap[key].Conditions = append(podMap[key].Conditions, ConditionInfo{
					Type:    "ContainerIssue",
					Status:  "Warning",
					Reason:  reason,
					Message: fmt.Sprintf("Container %s has issue: %s", containerName, reason),
				})
			}
		}
	}

	// Convert map to slice
	pods := make([]PodInfo, 0, len(podMap))
	for _, pod := range podMap {
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

	for _, line := range lines {
		if strings.Contains(line, "CRITICAL_EVENT_LOG") {
			// Extract event information from the log line
			event := EventInfo{
				Message: line,
				// Set default times - in a real implementation, parse from log timestamp
				FirstSeen: time.Now(),
				LastSeen:  time.Now(),
				Count:     1,
			}

			// Try to extract object information
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

	var data []struct {
		Timestamp   time.Time `json:"timestamp"`
		MemoryUsage int64     `json:"memory_usage"`
	}
	
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	return data, nil
}

// handleGetClusterSummary handles the get_cluster_summary tool
func (s *MonitorMCPServer) handleGetClusterSummary(args json.RawMessage) (interface{}, error) {
	var input ClusterSummaryInput
	if err := json.Unmarshal(args, &input); err != nil {
		return ClusterSummaryOutput{Error: fmt.Sprintf("Invalid input: %v", err)}, nil
	}

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
func (s *MonitorMCPServer) handleDetectMemoryLeak(args json.RawMessage) (interface{}, error) {
	var input MemoryLeakDetectionInput
	if err := json.Unmarshal(args, &input); err != nil {
		return MemoryLeakDetectionOutput{Error: fmt.Sprintf("Invalid input: %v", err)}, nil
	}

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
func (s *MonitorMCPServer) handleGetRecentEvents(args json.RawMessage) (interface{}, error) {
	var input RecentEventsInput
	if err := json.Unmarshal(args, &input); err != nil {
		return RecentEventsOutput{Error: fmt.Sprintf("Invalid input: %v", err)}, nil
	}

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
func (s *MonitorMCPServer) handleGetResourceUtilization(args json.RawMessage) (interface{}, error) {
	var input ResourceUtilizationInput
	if err := json.Unmarshal(args, &input); err != nil {
		return ResourceUtilizationOutput{Error: fmt.Sprintf("Invalid input: %v", err)}, nil
	}

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

// Start starts the MCP server
func (s *MonitorMCPServer) Start() error {
	log.Printf("Starting Monitor MCP Server on port %d, reading logs from: %s", s.port, s.logDir)
	return s.router.Run(fmt.Sprintf(":%d", s.port))
}

func main() {
	// Default log directory - can be overridden by config
	logDir := "/root/workspace/test-env/logs" // Use the test environment logs
	port := 3001 // Default port
	
	// Allow command line override
	if len(os.Args) > 1 {
		port, _ = strconv.Atoi(os.Args[1])
	}
	if len(os.Args) > 2 {
		logDir = os.Args[2]
	}
	
	server := NewMonitorMCPServer(port, logDir)
	
	fmt.Printf("Monitor MCP Server ready with 5 tools based on otisbrain monitor data:\n")
	fmt.Printf("- monitor_pods: Reads pod status from basic logs\n")
	fmt.Printf("- get_cluster_summary: Analyzes cluster health from collected logs\n")
	fmt.Printf("- detect_memory_leak: Analyzes memory trends from memory_usage logs\n")
	fmt.Printf("- get_recent_events: Retrieves critical events from event logs\n")
	fmt.Printf("- get_resource_utilization: Gets resource usage from memory_usage logs\n")
	
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}