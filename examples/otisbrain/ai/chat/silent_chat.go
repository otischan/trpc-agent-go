package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"trpc.group/trpc-go/trpc-agent-go/agent"
	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	sessioninmemory "trpc.group/trpc-go/trpc-agent-go/session/inmemory"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/utils"
)

// SilentAIChat manages the AI chat interface for the OtisBrain system without printing to screen
type SilentAIChat struct {
	modelName      string
	streaming      bool
	runner         runner.Runner
	userID         string
	sessionID      string
	variant        string
	config         *config.Config
	logger         *basic.BasicLogger
	inputChan      chan string
	responseChan   chan string
}

// NewSilentAIChat creates a new silent AI chat instance
func NewSilentAIChat(cfg *config.Config, logger *basic.BasicLogger) *SilentAIChat {
	return &SilentAIChat{
		modelName: cfg.LLM.Model,
		streaming: true, // Always use streaming for better UX
		variant:   "openai", // Default to openai variant
		config:    cfg,
		logger:    logger,
		inputChan: make(chan string, 10),      // Buffered channel for incoming messages
		responseChan: make(chan string, 10),   // Buffered channel for responses
	}
}

// Run starts the AI chat interface in a goroutine
func (c *SilentAIChat) Run(ctx context.Context) error {
	if err := c.setup(ctx); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}
	
	// Close runner when chat ends
	defer c.runner.Close()
	
	return c.startChat(ctx)
}

// setup builds the runner with a model, tools, and the in-memory session store.
func (c *SilentAIChat) setup(_ context.Context) error {
	// Create model instance
	modelInstance := openai.New(c.modelName, 
		openai.WithVariant(openai.Variant(c.variant)),
		openai.WithAPIKey(c.config.LLM.APIKey),
		openai.WithBaseURL(c.config.LLM.BaseURL),
	)

	sessionService := sessioninmemory.NewSessionService()

	// Create tools for the AI assistant
	// Tool to read critical summaries
	readSummaryTool := function.NewFunctionTool(
		c.readCriticalSummary,
		function.WithName("read_critical_summary"),
		function.WithDescription("Read the latest critical summary from the monitoring system"),
	)

	// Tool to read all critical records
	readAllRecordsTool := function.NewFunctionTool(
		c.readAllCriticalRecords,
		function.WithName("read_all_critical_records"),
		function.WithDescription("Read all critical records from the monitoring system"),
	)

	genConfig := model.GenerationConfig{
		MaxTokens:   utils.IntPtr(2000),
		Temperature: utils.FloatPtr(0.7),
		Stream:      c.streaming,
	}

	llmAgent := llmagent.New(
		"otisbrain-silent-assistant",
		llmagent.WithModel(modelInstance),
		llmagent.WithDescription("A silent AI assistant for OtisBrain K8S monitoring system that logs responses instead of printing to screen."),
		llmagent.WithInstruction(`You are an AI assistant for the OtisBrain K8S monitoring system. 
			Help users understand cluster status, investigate issues, and suggest solutions. 
			Use tools when helpful to access monitoring data. All responses should be logged rather than printed to screen.`),
		llmagent.WithGenerationConfig(genConfig),
		llmagent.WithTools([]tool.Tool{readSummaryTool, readAllRecordsTool}),
		llmagent.WithEnableParallelTools(false), // Disable parallel tools for simplicity
	)

	c.runner = runner.NewRunner(
		"otisbrain-silent-chat",
		llmAgent,
		runner.WithSessionService(sessionService),
	)

	c.userID = "otisbrain-silent-user"
	c.sessionID = fmt.Sprintf("silent-session-%d", time.Now().Unix())

	// Log that chat is ready
	c.logger.Info("Silent AI Chat ready! Session: ", c.sessionID)
	return nil
}

// startChat begins the chat interface without printing to screen
func (c *SilentAIChat) startChat(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Print prompt to screen but don't log it
			fmt.Print("👤 You (AI Console): ")
			if !scanner.Scan() {
				break
			}
			userInput := strings.TrimSpace(scanner.Text())
			if userInput == "" {
				continue
			}
			if userInput == "/exit" {
				c.logger.Info("AI Console closed by user")
				return nil
			}
			
			// Process the message silently (responses go to logs)
			if err := c.processMessage(ctx, userInput); err != nil {
				c.logger.Errorf("Error processing message: %v", err)
				fmt.Printf("❌ Error: %v\n", err) // Still show errors on screen
			}
			fmt.Println() // New line after user input
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("input scanner error: %w", err)
	}
	return nil
}

// processMessage sends a message to the AI and processes the response
func (c *SilentAIChat) processMessage(ctx context.Context, userMessage string) error {
	c.logger.Debugf("Processing user message: %s", userMessage)
	
	message := model.NewUserMessage(userMessage)
	requestID := uuid.New().String()
	eventChan, err := c.runner.Run(ctx, c.userID, c.sessionID, message, agent.WithRequestID(requestID))
	if err != nil {
		return fmt.Errorf("failed to run agent: %w", err)
	}
	
	return c.processResponse(eventChan)
}

// processResponse handles the response from the AI and logs it instead of printing to screen
func (c *SilentAIChat) processResponse(eventChan <-chan *event.Event) error {
	var responseBuilder strings.Builder
	responseBuilder.WriteString("🤖 Assistant: ")

	var (
		fullContent       string
		toolCallsDetected bool
		assistantStarted  bool
	)

	for evt := range eventChan {
		if err := c.handleEvent(evt, &toolCallsDetected, &assistantStarted, &fullContent, &responseBuilder); err != nil {
			return err
		}
		if evt.IsFinalResponse() {
			responseBuilder.WriteString("\n")
			break
		}
	}
	
	// Log the complete response instead of printing to screen
	responseText := responseBuilder.String()
	c.logger.Info(responseText)
	
	return nil
}

// handleEvent processes events from the AI
func (c *SilentAIChat) handleEvent(
	evt *event.Event,
	toolCallsDetected *bool,
	assistantStarted *bool,
	fullContent *string,
	responseBuilder *strings.Builder,
) error {
	if evt.Error != nil {
		errorMsg := fmt.Sprintf("\n❌ Error: %s\n", evt.Error.Message)
		c.logger.Error(errorMsg)
		responseBuilder.WriteString(errorMsg)
		return nil
	}
	if c.handleToolCalls(evt, toolCallsDetected, assistantStarted, responseBuilder) {
		return nil
	}
	if c.handleToolResponses(evt, responseBuilder) {
		return nil
	}
	c.handleContent(evt, toolCallsDetected, assistantStarted, fullContent, responseBuilder)
	return nil
}

// handleToolCalls processes tool calls from the AI
func (c *SilentAIChat) handleToolCalls(
	evt *event.Event,
	toolCallsDetected *bool,
	assistantStarted *bool,
	responseBuilder *strings.Builder,
) bool {
	if evt.Response != nil && len(evt.Response.Choices) > 0 && len(evt.Response.Choices[0].Message.ToolCalls) > 0 {
		*toolCallsDetected = true
		if *assistantStarted {
			responseBuilder.WriteString("\n")
		}
		toolCallMsg := "🔧 Callable tool calls initiated:\n"
		responseBuilder.WriteString(toolCallMsg)
		
		for _, toolCall := range evt.Response.Choices[0].Message.ToolCalls {
			callInfo := fmt.Sprintf("   • %s (ID: %s)\n", toolCall.Function.Name, toolCall.ID)
			responseBuilder.WriteString(callInfo)
			if len(toolCall.Function.Arguments) > 0 {
				argsInfo := fmt.Sprintf("     Args: %s\n", string(toolCall.Function.Arguments))
				responseBuilder.WriteString(argsInfo)
			}
		}
		execMsg := "\n🔄 Executing tools...\n"
		responseBuilder.WriteString(execMsg)
		return true
	}
	return false
}

// handleToolResponses processes tool responses
func (c *SilentAIChat) handleToolResponses(evt *event.Event, responseBuilder *strings.Builder) bool {
	if evt.Response == nil || len(evt.Response.Choices) == 0 {
		return false
	}
	hasToolResponse := false
	for _, choice := range evt.Response.Choices {
		if choice.Message.Role == model.RoleTool && choice.Message.ToolID != "" {
			response := fmt.Sprintf("✅ Callable tool response (ID: %s): %s\n",
				choice.Message.ToolID,
				strings.TrimSpace(choice.Message.Content))
			responseBuilder.WriteString(response)
			hasToolResponse = true
		}
	}
	return hasToolResponse
}

// handleContent processes content responses
func (c *SilentAIChat) handleContent(
	evt *event.Event,
	toolCallsDetected *bool,
	assistantStarted *bool,
	fullContent *string,
	responseBuilder *strings.Builder,
) {
	if evt.Response == nil || len(evt.Response.Choices) == 0 {
		return
	}
	content := c.extractContent(evt.Response.Choices[0])
	if content == "" {
		return
	}
	c.displayContent(content, toolCallsDetected, assistantStarted, fullContent, responseBuilder)
}

// extractContent extracts content from response
func (c *SilentAIChat) extractContent(choice model.Choice) string {
	if c.streaming {
		return choice.Delta.Content
	}
	return choice.Message.Content
}

// displayContent adds content to the response builder instead of printing to screen
func (c *SilentAIChat) displayContent(
	content string,
	toolCallsDetected *bool,
	assistantStarted *bool,
	fullContent *string,
	responseBuilder *strings.Builder,
) {
	if !*assistantStarted {
		if *toolCallsDetected {
			responseBuilder.WriteString("\n🤖 Assistant: ")
		}
		*assistantStarted = true
	}
	responseBuilder.WriteString(content)
	*fullContent += content
}


// Tool implementations
func (c *SilentAIChat) readCriticalSummary(ctx context.Context, _ struct{}) (string, error) {
	reader := basic.NewAIAgentReader()
	summary, err := reader.ReadLatestCriticalSummary()
	if err != nil {
		c.logger.Errorf("Error reading critical summary: %v", err)
		return fmt.Sprintf("Error reading critical summary: %v", err), nil
	}
	c.logger.Debugf("Read critical summary: length=%d chars", len(summary))
	return summary, nil
}

func (c *SilentAIChat) readAllCriticalRecords(ctx context.Context, _ struct{}) (string, error) {
	reader := basic.NewAIAgentReader()
	records, err := reader.GetAllCriticalRecords()
	if err != nil {
		c.logger.Errorf("Error reading critical records: %v", err)
		return fmt.Sprintf("Error reading critical records: %v", err), nil
	}
	c.logger.Debugf("Read all critical records: length=%d chars", len(records))
	return records, nil
}