package agent

import (
	"context"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/ai/chat"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// AIConsoleAgent manages the AI console functionality
type AIConsoleAgent struct {
	config *config.Config
	logger *basic.BasicLogger
	chat   *chat.SilentAIChat
	ctx    context.Context
	cancel context.CancelFunc
}

// NewAIConsoleAgent creates a new AI console agent
func NewAIConsoleAgent(cfg *config.Config, logger *basic.BasicLogger) *AIConsoleAgent {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &AIConsoleAgent{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the AI console agent in a goroutine
func (aca *AIConsoleAgent) Start() error {
	aca.logger.Info("Starting AI Console Agent...")
	
	// Create the silent chat instance
	aca.chat = chat.NewSilentAIChat(aca.config, aca.logger)
	
	// Run the chat in a goroutine
	go func() {
		if err := aca.chat.Run(aca.ctx); err != nil {
			aca.logger.Errorf("AI Console error: %v", err)
		}
	}()
	
	aca.logger.Info("AI Console Agent started successfully")
	return nil
}

// Stop stops the AI console agent
func (aca *AIConsoleAgent) Stop() {
	aca.logger.Info("Stopping AI Console Agent...")
	aca.cancel()
}

// IsRunning checks if the AI console agent is running
func (aca *AIConsoleAgent) IsRunning() bool {
	select {
	case <-aca.ctx.Done():
		return false
	default:
		return true
	}
}