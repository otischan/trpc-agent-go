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

// AIDecisionAgent implements the AI decision-making functionality
type AIDecisionAgent struct {
	config *config.Config
	runner runner.Runner
	stopCh chan struct{}
}

// NewAIDecisionAgent creates a new AI decision agent
func NewAIDecisionAgent(cfg *config.Config) (*AIDecisionAgent, error) {
	// Create model instance
	modelInstance := openai.New(cfg.LLM.Model)

	// Create tools
	decisionTool := createDecisionTool()

	// Create AI agent
	agent := llmagent.New("ai-decision",
		llmagent.WithModel(modelInstance),
		llmagent.WithTools([]tool.Tool{decisionTool}),
		llmagent.WithInstruction(`You are an AI decision-maker for Kubernetes cluster operations. 
Based on the provided cluster state, events, and issues, recommend the best course of action.
Consider factors like impact on users, risk level, and urgency when making decisions.
Provide clear, actionable recommendations.`),
	)

	// Create runner
	run := runner.NewRunner("ai-decision-app", agent)

	return &AIDecisionAgent{
		config: cfg,
		runner: run,
		stopCh: make(chan struct{}),
	}, nil
}

// createDecisionTool creates a tool for AI to make decisions
func createDecisionTool() tool.Tool {
	return function.NewFunctionTool(
		makeDecision,
		function.WithName("make_decision"),
		function.WithDescription("Make a decision based on cluster state and issues to recommend the best course of action."),
	)
}

// makeDecision makes a decision based on cluster state and issues
func makeDecision(ctx context.Context, req makeDecisionReq) (makeDecisionRsp, error) {
	log.Printf("Making decision for issue: %s in namespace: %s", req.Issue, req.Namespace)

	// In a real implementation, this would analyze the issue and make a decision
	decision := fmt.Sprintf("Recommended action for issue '%s' in namespace '%s': %s",
		req.Issue, req.Namespace, req.SuggestedAction)

	return makeDecisionRsp{
		Decision: decision,
		Action:   req.SuggestedAction,
		Status:   "recommended",
	}, nil
}

// makeDecisionReq represents the request for making a decision
type makeDecisionReq struct {
	Issue           string `json:"issue" jsonschema:"description=Issue or problem to decide on,required"`
	Namespace       string `json:"namespace" jsonschema:"description=Namespace where the issue occurs,required"`
	SuggestedAction string `json:"suggested_action" jsonschema:"description=Suggested action to take,required"`
	Context         string `json:"context" jsonschema:"description=Additional context for the decision"`
}

// makeDecisionRsp represents the response from making a decision
type makeDecisionRsp struct {
	Decision string `json:"decision"`
	Action   string `json:"action"`
	Status   string `json:"status"`
}

// Start starts the AI decision agent
func (aid *AIDecisionAgent) Start(ctx context.Context) error {
	log.Println("Starting AI decision agent...")

	// Decision agent will be called on-demand rather than continuously running
	return nil
}

// MakeDecision makes a decision based on the provided issue
func (aid *AIDecisionAgent) MakeDecision(issue, namespace, suggestedAction, ctxStr string) (string, error) {
	log.Printf("Making AI decision for issue: %s in namespace: %s", issue, namespace)

	// Prepare the message for the AI
	message := model.NewUserMessage(
		fmt.Sprintf("Issue: %s\nNamespace: %s\nSuggested Action: %s\nContext: %s\n\n"+
			"Based on this information, provide a detailed recommendation for the best course of action. "+
			"Consider potential risks, impact on users, and alternative approaches.",
			issue, namespace, suggestedAction, ctxStr))

	// Run the AI decision-making
	ctx := context.Background()
	events, err := aid.runner.Run(ctx, "ai-user", "ai-decision-session", message)
	if err != nil {
		return "", fmt.Errorf("failed to run AI decision-making: %w", err)
	}

	// Process the AI response
	var response strings.Builder
	for event := range events {
		if event.Object == "chat.completion.chunk" {
			response.WriteString(event.Response.Choices[0].Delta.Content)
		}
	}

	result := response.String()
	log.Printf("AI Decision Result: %s", result)

	// Save the decision result to AI logs
	aid.saveDecisionResult(issue, result)

	return result, nil
}

// saveDecisionResult saves the AI decision result to the AI logs
func (aid *AIDecisionAgent) saveDecisionResult(issue, result string) {
	// In a real implementation, this would write to the AI decision log file
	log.Printf("AI Decision saved to logs/ai/ai_decision.log")
	log.Printf("Issue: %s", issue)
	log.Printf("Decision content: %s", result)
}

// Stop stops the AI decision agent
func (aid *AIDecisionAgent) Stop() {
	close(aid.stopCh)
}
