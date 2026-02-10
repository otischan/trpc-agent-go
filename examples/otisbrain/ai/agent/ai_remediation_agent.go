package agent

import (
	"context"
	"fmt"
	"log"
	"strings"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// AIEnhancedRemediationAgent implements the AI-enhanced remediation functionality
type AIEnhancedRemediationAgent struct {
	config *config.Config
	runner runner.Runner
	stopCh chan struct{}
}

// NewAIEnhancedRemediationAgent creates a new AI-enhanced remediation agent
func NewAIEnhancedRemediationAgent(cfg *config.Config) (*AIEnhancedRemediationAgent, error) {
	// Create model instance
	modelInstance := openai.New(cfg.LLM.Model)

	// Create tools
	remediationTool := createRemediationTool()

	// Create AI agent
	agent := llmagent.New("ai-remediation",
		llmagent.WithModel(modelInstance),
		llmagent.WithTools([]tool.Tool{remediationTool}),
		llmagent.WithInstruction(`You are an AI assistant specialized in recommending remediation actions for Kubernetes cluster issues. 
Based on the provided problem description and cluster state, suggest the most appropriate remediation action. 
Consider the severity of the issue, potential impact on users, and success probability of different approaches.`),
	)

	// Create runner
	run := runner.NewRunner("ai-remediation-app", agent)

	return &AIEnhancedRemediationAgent{
		config: cfg,
		runner: run,
		stopCh: make(chan struct{}),
	}, nil
}

// createRemediationTool creates a tool for AI to recommend remediation actions
func createRemediationTool() tool.Tool {
	return function.NewFunctionTool(
		recommendRemediation,
		function.WithName("recommend_remediation"),
		function.WithDescription("Recommend the best remediation action based on the issue and cluster state."),
	)
}

// recommendRemediation recommends a remediation action based on the issue
func recommendRemediation(ctx context.Context, req recommendRemediationReq) (recommendRemediationRsp, error) {
	log.Printf("Recommending remediation for issue: %s in resource: %s/%s",
		req.Issue, req.ResourceType, req.ResourceName)

	// In a real implementation, this would analyze the issue and recommend remediation
	recommendation := fmt.Sprintf("Recommended remediation for %s '%s/%s': %s",
		req.ResourceType, req.ResourceName, req.Namespace, req.PossibleActions[0])

	return recommendRemediationRsp{
		Recommendation: recommendation,
		Action:         req.PossibleActions[0],
		Confidence:     0.85, // Placeholder confidence score
		Status:         "recommended",
	}, nil
}

// recommendRemediationReq represents the request for recommending remediation
type recommendRemediationReq struct {
	Issue           string   `json:"issue" jsonschema:"description=Issue description,required"`
	ResourceType    string   `json:"resource_type" jsonschema:"description=Type of Kubernetes resource,required"`
	ResourceName    string   `json:"resource_name" jsonschema:"description=Name of the resource,required"`
	Namespace       string   `json:"namespace" jsonschema:"description=Namespace of the resource,required"`
	PossibleActions []string `json:"possible_actions" jsonschema:"description=List of possible remediation actions,required"`
	Context         string   `json:"context" jsonschema:"description=Additional context for remediation"`
}

// recommendRemediationRsp represents the response from recommending remediation
type recommendRemediationRsp struct {
	Recommendation string  `json:"recommendation"`
	Action         string  `json:"action"`
	Confidence     float64 `json:"confidence"`
	Status         string  `json:"status"`
}

// Start starts the AI-enhanced remediation agent
func (air *AIEnhancedRemediationAgent) Start(ctx context.Context) error {
	log.Println("Starting AI-enhanced remediation agent...")

	// Remediation agent will be called on-demand rather than continuously running
	return nil
}

// RecommendRemediation recommends a remediation action based on the issue
func (air *AIEnhancedRemediationAgent) RecommendRemediation(
	issue, resourceType, resourceName, namespace string,
	possibleActions []string, ctxStr string) (string, float64, error) {

	log.Printf("Recommending AI remediation for issue: %s in %s %s/%s",
		issue, resourceType, resourceName, namespace)

	// Prepare the message for the AI
	message := model.NewUserMessage(
		fmt.Sprintf("Issue: %s\nResource: %s %s/%s\nPossible Actions: %v\nContext: %s\n\n"+
			"Based on this information, recommend the best remediation action. "+
			"Explain why this action is recommended and estimate the confidence level.",
			issue, resourceType, resourceName, namespace, possibleActions, ctxStr))

	// Run the AI remediation recommendation
	ctx := context.Background()
	events, err := air.runner.Run(ctx, "ai-user", "ai-remediation-session", message)
	if err != nil {
		return "", 0.0, fmt.Errorf("failed to run AI remediation recommendation: %w", err)
	}

	// Process the AI response
	var response strings.Builder
	for event := range events {
		if event.Object == "chat.completion.chunk" {
			response.WriteString(event.Response.Choices[0].Delta.Content)
		}
	}

	result := response.String()
	log.Printf("AI Remediation Recommendation: %s", result)

	// Save the recommendation result to AI logs
	air.saveRecommendationResult(issue, resourceType, resourceName, namespace, result)

	// For now, return a high confidence as a placeholder
	return result, 0.9, nil
}

// saveRecommendationResult saves the AI remediation recommendation to the AI logs
func (air *AIEnhancedRemediationAgent) saveRecommendationResult(
	issue, resourceType, resourceName, namespace, result string) {

	// In a real implementation, this would write to the AI remediation log file
	log.Printf("AI Remediation Recommendation saved to logs/ai/ai_analysis.log")
	log.Printf("Issue: %s", issue)
	log.Printf("Resource: %s %s/%s", resourceType, resourceName, namespace)
	log.Printf("Recommendation: %s", result)
}

// Stop stops the AI-enhanced remediation agent
func (air *AIEnhancedRemediationAgent) Stop() {
	close(air.stopCh)
}
