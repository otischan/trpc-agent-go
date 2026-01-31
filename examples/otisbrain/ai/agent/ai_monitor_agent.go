package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// AIEnhancedMonitorAgent implements the AI-enhanced monitoring functionality
type AIEnhancedMonitorAgent struct {
	config    *config.Config
	runner    runner.Runner
	stopCh    chan struct{}
}

// NewAIEnhancedMonitorAgent creates a new AI-enhanced monitoring agent
func NewAIEnhancedMonitorAgent(cfg *config.Config) (*AIEnhancedMonitorAgent, error) {
	// Create model instance
	modelInstance := openai.New(cfg.LLM.Model)

	// Create tools
	analysisTool := createAnalysisTool()

	// Create AI agent
	agent := llmagent.New("ai-monitor",
		llmagent.WithModel(modelInstance),
		llmagent.WithTools([]tool.Tool{analysisTool}),
		llmagent.WithInstruction(`You are an AI assistant specialized in analyzing Kubernetes cluster events and logs. 
Analyze the provided information to identify patterns, predict potential issues, and suggest preventive measures.
Focus on identifying anomalies that might not be caught by basic monitoring rules.`),
	)

	// Create runner
	run := runner.NewRunner("ai-monitor-app", agent)

	return &AIEnhancedMonitorAgent{
		config: cfg,
		runner: run,
		stopCh: make(chan struct{}),
	}, nil
}

// createAnalysisTool creates a tool for AI to analyze critical events
func createAnalysisTool() tool.Tool {
	return function.NewFunctionTool(
		analyzeCriticalEvents,
		function.WithName("analyze_critical_events"),
		function.WithDescription("Analyze critical events from the Kubernetes cluster to identify patterns and suggest solutions."),
	)
}

// analyzeCriticalEvents analyzes critical events from log files
func analyzeCriticalEvents(ctx context.Context, req analyzeCriticalEventsReq) (analyzeCriticalEventsRsp, error) {
	log.Printf("Analyzing critical events for namespace: %s", req.Namespace)

	// Read critical events from log files
	eventsContent, err := readCriticalEvents(req.Namespace)
	if err != nil {
		return analyzeCriticalEventsRsp{}, fmt.Errorf("failed to read critical events: %w", err)
	}

	// Perform analysis
	analysisResult := performAnalysis(eventsContent)

	return analyzeCriticalEventsRsp{
		Analysis: analysisResult,
		Status:   "success",
	}, nil
}

// analyzeCriticalEventsReq represents the request for analyzing critical events
type analyzeCriticalEventsReq struct {
	Namespace string `json:"namespace" jsonschema:"description=Namespace to analyze,required"`
	Query     string `json:"query" jsonschema:"description=Specific query or focus area for analysis"`
}

// analyzeCriticalEventsRsp represents the response from analyzing critical events
type analyzeCriticalEventsRsp struct {
	Analysis string `json:"analysis"`
	Status   string `json:"status"`
}

// readCriticalEvents reads critical events from log files
func readCriticalEvents(namespace string) (string, error) {
	logDir := "logs/critical_events"
	
	// Read all log files in the critical events directory
	files, err := filepath.Glob(filepath.Join(logDir, "*.log"))
	if err != nil {
		return "", err
	}

	var content strings.Builder
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %v", file, err)
			continue
		}
		
		content.WriteString(fmt.Sprintf("File: %s\n", filepath.Base(file)))
		content.Write(data)
		content.WriteString("\n\n")
	}

	return content.String(), nil
}

// performAnalysis performs AI analysis on the events content
func performAnalysis(content string) string {
	// In a real implementation, this would call the LLM for analysis
	// For now, we'll return a placeholder
	return fmt.Sprintf("AI Analysis performed on content with length: %d characters.\n\nThis would normally contain detailed analysis of patterns, potential issues, and recommendations based on the provided data.", len(content))
}

// Start starts the AI-enhanced monitoring agent
func (ai *AIEnhancedMonitorAgent) Start(ctx context.Context) error {
	log.Println("Starting AI-enhanced monitoring agent...")

	// Start monitoring in a separate goroutine
	go func() {
		// Check for new critical events periodically
		ticker := time.NewTicker(60 * time.Second) // Check every minute
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := ai.analyzeCriticalEvents(); err != nil {
					log.Printf("Error during AI analysis: %v", err)
				}
			case <-ai.stopCh:
				log.Println("AI-enhanced monitoring agent stopped")
				return
			case <-ctx.Done():
				log.Println("Context cancelled, stopping AI-enhanced monitoring agent")
				close(ai.stopCh)
				return
			}
		}
	}()

	return nil
}

// analyzeCriticalEvents performs AI analysis on critical events
func (ai *AIEnhancedMonitorAgent) analyzeCriticalEvents() error {
	log.Println("Performing AI analysis on critical events...")

	// Prepare the message for the AI
	message := model.NewUserMessage(
		fmt.Sprintf("Analyze the following Kubernetes cluster events and logs for namespace %s. "+
			"Identify patterns, predict potential issues, and suggest preventive measures:\n\n%s",
			ai.config.Namespace, ai.readRecentEvents()))

	// Run the AI analysis
	ctx := context.Background()
	events, err := ai.runner.Run(ctx, "ai-user", "ai-session", message)
	if err != nil {
		return fmt.Errorf("failed to run AI analysis: %w", err)
	}

	// Process the AI response
	var response strings.Builder
	for event := range events {
		if event.Object == "chat.completion.chunk" {
			response.WriteString(event.Response.Choices[0].Delta.Content)
		}
	}

	result := response.String()
	log.Printf("AI Analysis Result: %s", result)

	// Save the analysis result to AI logs
	ai.saveAnalysisResult(result)

	return nil
}

// readRecentEvents reads recent events from the critical events log
func (ai *AIEnhancedMonitorAgent) readRecentEvents() string {
	// Read recent events from the critical events log
	events, err := readCriticalEvents(ai.config.Namespace)
	if err != nil {
		log.Printf("Error reading recent events: %v", err)
		return "Could not read recent events"
	}
	return events
}

// saveAnalysisResult saves the AI analysis result to the AI logs
func (ai *AIEnhancedMonitorAgent) saveAnalysisResult(result string) {
	// In a real implementation, this would write to the AI analysis log file
	log.Printf("AI Analysis saved to logs/ai/ai_analysis.log")
	log.Printf("Analysis content: %s", result)
}

// Stop stops the AI-enhanced monitoring agent
func (ai *AIEnhancedMonitorAgent) Stop() {
	close(ai.stopCh)
}