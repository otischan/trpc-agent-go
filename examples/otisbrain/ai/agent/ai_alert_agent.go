package agent

import (
	"context"
	"fmt"
	"log"
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

// AIEnhancedAlertAgent implements the AI-enhanced alert processing functionality
type AIEnhancedAlertAgent struct {
	config    *config.Config
	runner    runner.Runner
	stopCh    chan struct{}
}

// NewAIEnhancedAlertAgent creates a new AI-enhanced alert agent
func NewAIEnhancedAlertAgent(cfg *config.Config) (*AIEnhancedAlertAgent, error) {
	// Create model instance
	modelInstance := openai.New(cfg.LLM.Model)

	// Create tools
	alertAnalysisTool := createAlertAnalysisTool()

	// Create AI agent
	agent := llmagent.New("ai-alert",
		llmagent.WithModel(modelInstance),
		llmagent.WithTools([]tool.Tool{alertAnalysisTool}),
		llmagent.WithInstruction(`You are an AI assistant specialized in analyzing Kubernetes alerts and events. 
Classify alerts by severity, correlate related events, and provide context-aware prioritization. 
Identify false positives and group related alerts to reduce noise.`),
	)

	// Create runner
	run := runner.NewRunner("ai-alert-app", agent)

	return &AIEnhancedAlertAgent{
		config: cfg,
		runner: run,
		stopCh: make(chan struct{}),
	}, nil
}

// createAlertAnalysisTool creates a tool for AI to analyze alerts
func createAlertAnalysisTool() tool.Tool {
	return function.NewFunctionTool(
		analyzeAlerts,
		function.WithName("analyze_alerts"),
		function.WithDescription("Analyze alerts and events to classify severity, correlate events, and provide prioritization."),
	)
}

// analyzeAlerts analyzes alerts and provides classification and correlation
func analyzeAlerts(ctx context.Context, req analyzeAlertsReq) (analyzeAlertsRsp, error) {
	log.Printf("Analyzing alerts for namespace: %s", req.Namespace)

	// In a real implementation, this would analyze the alerts
	analysis := fmt.Sprintf("Analysis of %d alerts in namespace %s. Severity distribution: %v", 
		len(req.Alerts), req.Namespace, req.SeverityDistribution)

	return analyzeAlertsRsp{
		Analysis: analysis,
		Status:   "analyzed",
	}, nil
}

// analyzeAlertsReq represents the request for analyzing alerts
type analyzeAlertsReq struct {
	Namespace           string            `json:"namespace" jsonschema:"description=Namespace to analyze,required"`
	Alerts              []string          `json:"alerts" jsonschema:"description=List of alerts to analyze,required"`
	SeverityDistribution map[string]int    `json:"severity_distribution" jsonschema:"description=Map of severity to count"`
	Context             string            `json:"context" jsonschema:"description=Additional context for analysis"`
}

// analyzeAlertsRsp represents the response from analyzing alerts
type analyzeAlertsRsp struct {
	Analysis string `json:"analysis"`
	Status   string `json:"status"`
}

// Start starts the AI-enhanced alert agent
func (aia *AIEnhancedAlertAgent) Start(ctx context.Context) error {
	log.Println("Starting AI-enhanced alert agent...")

	// Start alert analysis in a separate goroutine
	go func() {
		// Check for new alerts periodically
		ticker := time.NewTicker(60 * time.Second) // Check every minute
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := aia.analyzeNewAlerts(); err != nil {
					log.Printf("Error during AI alert analysis: %v", err)
				}
			case <-aia.stopCh:
				log.Println("AI-enhanced alert agent stopped")
				return
			case <-ctx.Done():
				log.Println("Context cancelled, stopping AI-enhanced alert agent")
				close(aia.stopCh)
				return
			}
		}
	}()

	return nil
}

// analyzeNewAlerts performs AI analysis on new alerts
func (aia *AIEnhancedAlertAgent) analyzeNewAlerts() error {
	log.Println("Performing AI analysis on new alerts...")

	// Prepare the message for the AI
	message := model.NewUserMessage(
		fmt.Sprintf("Analyze the following Kubernetes cluster alerts and events for namespace %s. "+
			"Classify by severity, correlate related events, and provide prioritization:\n\n%s",
			aia.config.Namespace, aia.readRecentAlerts()))

	// Run the AI analysis
	ctx := context.Background()
	events, err := aia.runner.Run(ctx, "ai-user", "ai-alert-session", message)
	if err != nil {
		return fmt.Errorf("failed to run AI alert analysis: %w", err)
	}

	// Process the AI response
	var response strings.Builder
	for event := range events {
		if event.Object == "chat.completion.chunk" {
			response.WriteString(event.Response.Choices[0].Delta.Content)
		}
	}

	result := response.String()
	log.Printf("AI Alert Analysis Result: %s", result)

	// Save the analysis result to AI logs
	aia.saveAlertAnalysisResult(result)

	return nil
}

// readRecentAlerts reads recent alerts from the critical events log
func (aia *AIEnhancedAlertAgent) readRecentAlerts() string {
	// For now, return a placeholder
	return fmt.Sprintf("Recent alerts for namespace %s: Pod crash, High CPU usage, Unavailable service", aia.config.Namespace)
}

// saveAlertAnalysisResult saves the AI alert analysis result to the AI logs
func (aia *AIEnhancedAlertAgent) saveAlertAnalysisResult(result string) {
	// In a real implementation, this would write to the AI alert log file
	log.Printf("AI Alert Analysis saved to logs/ai/ai_analysis.log")
	log.Printf("Analysis content: %s", result)
}

// Stop stops the AI-enhanced alert agent
func (aia *AIEnhancedAlertAgent) Stop() {
	close(aia.stopCh)
}