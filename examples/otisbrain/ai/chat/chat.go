package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	"trpc.group/trpc-go/trpc-agent-go/skill"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/ai/tools"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/utils"
)

// AIChat manages the AI chat interface for the OtisBrain system
type AIChat struct {
	modelName      string
	streaming      bool
	runner         runner.Runner
	userID         string
	sessionID      string
	variant        string
	config         *config.Config
	logger         *basic.BasicLogger
	skillRepo      skill.Repository
}

// NewAIChat creates a new AI chat instance
func NewAIChat(cfg *config.Config, logger *basic.BasicLogger) *AIChat {
	return &AIChat{
		modelName: cfg.LLM.Model,
		streaming: true, // Always use streaming for better UX
		variant:   "openai", // Default to openai variant
		config:    cfg,
		logger:    logger,
	}
}

// Run starts the AI chat interface in a goroutine
func (c *AIChat) Run(ctx context.Context) error {
	if err := c.setup(ctx); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}
	
	// Close runner when chat ends
	defer c.runner.Close()
	
	return c.startChat(ctx)
}

// setup builds the runner with a model, skills, and the in-memory session store.
func (c *AIChat) setup(_ context.Context) error {
	// Create model instance
	modelInstance := openai.New(c.modelName,
		openai.WithVariant(openai.Variant(c.variant)),
		openai.WithAPIKey(c.config.LLM.APIKey),
		openai.WithBaseURL(c.config.LLM.BaseURL),
	)

	sessionService := sessioninmemory.NewSessionService()

	// Skills repository - dynamically load skills from resources directory
	// Use project root from environment variable to ensure correct path resolution
	projectRoot := os.Getenv("OTISBRAIN_PROJECT_ROOT")
	if projectRoot == "" {
		projectRoot = "."  // fallback to current directory
	}
	skillsRoot := filepath.Join(projectRoot, "resources", "skills")
	repo, err := skill.NewFSRepository(skillsRoot)
	if err != nil {
		c.logger.Printf("Warning: Could not initialize skills repository: %v", err)
		c.logger.Printf("Attempting to initialize skills executor as fallback...")

		// Initialize skills executor as fallback mechanism using the same path
		skillExecutor := tools.NewSkillExecutor(skillsRoot)
		availableSkills, err := skillExecutor.GetAvailableSkills()
		if err != nil {
			c.logger.Printf("Fallback skills initialization also failed: %v", err)
		} else {
			c.logger.Printf("Fallback skills initialized successfully. Available skills: %d", len(availableSkills))
		}
		c.skillRepo = nil  // Explicitly set to nil when repo initialization fails
	} else {
		c.logger.Printf("Skills repository initialized from: %s", skillsRoot)
		c.skillRepo = repo  // Store the repository for later use
	}

	genConfig := model.GenerationConfig{
		MaxTokens:   utils.IntPtr(2000),
		Temperature: utils.FloatPtr(0.7),
		Stream:      c.streaming,
	}

	// Create LLM agent with skills support
	llmAgentOptions := []llmagent.Option{
		llmagent.WithModel(modelInstance),
		llmagent.WithDescription("An AI assistant for the OtisBrain K8S monitoring system."),
		llmagent.WithInstruction(`You are an AI assistant for the OtisBrain K8S monitoring system.
			Help users understand cluster status, investigate issues, and suggest solutions.
			Use tools when helpful to access monitoring data. You can load and run skills to access critical monitoring data.`),
		llmagent.WithGenerationConfig(genConfig),
		llmagent.WithEnableParallelTools(false), // Disable parallel tools for simplicity
	}

	// Add skills support if repository is available
	if repo != nil {
		llmAgentOptions = append(llmAgentOptions, llmagent.WithSkills(repo))
	}

	llmAgent := llmagent.New(
		"otisbrain-assistant",
		llmAgentOptions...,
	)

	c.runner = runner.NewRunner(
		"otisbrain-chat",
		llmAgent,
		runner.WithSessionService(sessionService),
	)

	c.userID = "otisbrain-user"
	c.sessionID = fmt.Sprintf("session-%d", time.Now().Unix())

	// Log that chat is ready
	c.logger.Info("AI Chat ready! Session: ", c.sessionID)
	return nil
}

// startChat begins the chat interface
func (c *AIChat) startChat(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("👤 You: ")
		if !scanner.Scan() {
			break
		}
		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}
		if userInput == "/exit" {
			fmt.Println("👋 Goodbye!")
			return nil
		}
		if err := c.processMessage(ctx, userInput); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
		fmt.Println()
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("input scanner error: %w", err)
	}
	return nil
}

// processMessage sends a message to the AI and processes the response
func (c *AIChat) processMessage(ctx context.Context, userMessage string) error {
	// Handle special management commands
	lowerMsg := strings.ToLower(strings.TrimSpace(userMessage))

	// Handle list skills command
	if lowerMsg == "list skills" || lowerMsg == "/list skills" || lowerMsg == "/skills" {
		return c.listSkills(ctx)
	}

	// Handle help command
	if lowerMsg == "help" || lowerMsg == "/help" {
		return c.showHelp()
	}

	// Handle other management commands as needed
	message := model.NewUserMessage(userMessage)
	requestID := uuid.New().String()
	eventChan, err := c.runner.Run(ctx, c.userID, c.sessionID, message, agent.WithRequestID(requestID))
	if err != nil {
		return fmt.Errorf("failed to run agent: %w", err)
	}
	return c.processResponse(eventChan)
}

// processResponse handles the response from the AI
func (c *AIChat) processResponse(eventChan <-chan *event.Event) error {
	fmt.Print("🤖 Assistant: ")

	var (
		fullContent       string
		toolCallsDetected bool
		assistantStarted  bool
	)

	for evt := range eventChan {
		if err := c.handleEvent(evt, &toolCallsDetected, &assistantStarted, &fullContent); err != nil {
			return err
		}
		if evt.IsFinalResponse() {
			fmt.Printf("\n")
			break
		}
	}
	return nil
}

// handleEvent processes events from the AI
func (c *AIChat) handleEvent(
	evt *event.Event,
	toolCallsDetected *bool,
	assistantStarted *bool,
	fullContent *string,
) error {
	if evt.Error != nil {
		fmt.Printf("\n❌ Error: %s\n", evt.Error.Message)
		return nil
	}
	if c.handleToolCalls(evt, toolCallsDetected, assistantStarted) {
		return nil
	}
	if c.handleToolResponses(evt) {
		return nil
	}
	c.handleContent(evt, toolCallsDetected, assistantStarted, fullContent)
	return nil
}

// handleToolCalls processes tool calls from the AI
func (c *AIChat) handleToolCalls(
	evt *event.Event,
	toolCallsDetected *bool,
	assistantStarted *bool,
) bool {
	if evt.Response != nil && len(evt.Response.Choices) > 0 && len(evt.Response.Choices[0].Message.ToolCalls) > 0 {
		*toolCallsDetected = true
		if *assistantStarted {
			fmt.Printf("\n")
		}
		fmt.Printf("🔧 Callable tool calls initiated:\n")
		for _, toolCall := range evt.Response.Choices[0].Message.ToolCalls {
			fmt.Printf("   • %s (ID: %s)\n", toolCall.Function.Name, toolCall.ID)
			if len(toolCall.Function.Arguments) > 0 {
				fmt.Printf("     Args: %s\n", string(toolCall.Function.Arguments))
			}
		}
		fmt.Printf("\n🔄 Executing tools...\n")
		return true
	}
	return false
}

// handleToolResponses processes tool responses
func (c *AIChat) handleToolResponses(evt *event.Event) bool {
	if evt.Response == nil || len(evt.Response.Choices) == 0 {
		return false
	}
	hasToolResponse := false
	for _, choice := range evt.Response.Choices {
		if choice.Message.Role == model.RoleTool && choice.Message.ToolID != "" {
			fmt.Printf("✅ Callable tool response (ID: %s): %s\n",
				choice.Message.ToolID,
				strings.TrimSpace(choice.Message.Content))
			hasToolResponse = true
		}
	}
	return hasToolResponse
}

// handleContent processes content responses
func (c *AIChat) handleContent(
	evt *event.Event,
	toolCallsDetected *bool,
	assistantStarted *bool,
	fullContent *string,
) {
	if evt.Response == nil || len(evt.Response.Choices) == 0 {
		return
	}
	content := c.extractContent(evt.Response.Choices[0])
	if content == "" {
		return
	}
	c.displayContent(content, toolCallsDetected, assistantStarted, fullContent)
}

// extractContent extracts content from response
func (c *AIChat) extractContent(choice model.Choice) string {
	if c.streaming {
		return choice.Delta.Content
	}
	return choice.Message.Content
}

// displayContent displays content to the user
func (c *AIChat) displayContent(
	content string,
	toolCallsDetected *bool,
	assistantStarted *bool,
	fullContent *string,
) {
	if !*assistantStarted {
		if *toolCallsDetected {
			fmt.Printf("\n🤖 Assistant: ")
		}
		*assistantStarted = true
	}
	fmt.Print(content)
	*fullContent += content
}

// listSkills lists all available skills
func (c *AIChat) listSkills(ctx context.Context) error {
	fmt.Println("📋 Available Skills:")

	// Since the repository doesn't have a List method, use our custom skill executor
	fmt.Println("  🔄 Using fallback method to load skills...")

	// Use project root from environment variable to ensure correct path resolution
	projectRoot := os.Getenv("OTISBRAIN_PROJECT_ROOT")
	if projectRoot == "" {
		projectRoot = "."  // fallback to current directory
	}
	skillsPath := filepath.Join(projectRoot, "resources", "skills")
	skillExecutor := tools.NewSkillExecutor(skillsPath)
	availableSkills, err := skillExecutor.GetAvailableSkills()
	if err != nil {
		fmt.Printf("  ❌ Could not load skills: %v\n", err)
	} else {
		if len(availableSkills) == 0 {
			fmt.Println("  No skills available")
		} else {
			for _, skill := range availableSkills {
				fmt.Printf("  • %s: %s\n", skill.Name, skill.Description)
			}
		}
	}

	fmt.Println("")
	return nil
}

// showHelp shows help information
func (c *AIChat) showHelp() error {
	fmt.Println("📖 OtisBrain AI Assistant Help:")
	fmt.Println("  • Ask questions about your Kubernetes cluster")
	fmt.Println("  • Request information about cluster resources")
	fmt.Println("  • Ask for recent critical events")
	fmt.Println("  • Type 'list skills' to see available tools")
	fmt.Println("  • Type '/exit' to quit")
	fmt.Println("")
	return nil
}

// getSkillsRepository returns the stored skills repository
func (c *AIChat) getSkillsRepository() skill.Repository {
	return c.skillRepo
}

// getRecentCriticalEvents retrieves recent critical events based on user request
func (c *AIChat) getRecentCriticalEvents(ctx context.Context, _ struct{}) (string, error) {
	// Default to last hour if no specific time range is provided
	req := tools.GetRecentCriticalEventsRequest{
		TimeRange:    "last_hour",
		Severity:     "all",
		ResourceType: "all",
	}

	result, err := tools.GetRecentCriticalEvents(ctx, req)
	if err != nil {
		return fmt.Sprintf("Error retrieving critical events: %v", err), nil
	}

	return tools.FormatCriticalEventsResult(result), nil
}

