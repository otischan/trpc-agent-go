package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	aiagent "trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/ai/agent"
	basicagent "trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/ai/chat"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/shared/k8sclient"
)

var (
	configPath = flag.String("config", "./config/config.yaml", "Path to the configuration file (default: \"./config/config.yaml\")")
)

// Global logger instance for chat interface
var consoleLogger *logrus.Logger
var fileLogger *basic.BasicLogger

func initConsoleLogger(logLevel string) {
	var err error
	consoleLogger, err = basic.NewConsoleLogger(logLevel)
	if err != nil {
		log.Fatalf("Failed to initialize console logger: %v", err)
	}
}

func initFileLogger(logLevel string) {
	var err error
	fileLogger, err = basic.NewBasicLogger(logLevel)
	if err != nil {
		log.Fatalf("Failed to initialize file logger: %v", err)
	}
}

func main() {
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize both loggers with log level from config
	initConsoleLogger(cfg.LogLevel)
	initFileLogger(cfg.LogLevel)

	// Use values from the monitoring config section
	clientset, err := k8sclient.NewClient(cfg.Monitoring.Kubeconfig)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal, stopping...")
		cancel()
	}()

	// Initialize log aggregator with configurable interval
	aggregationInterval := time.Duration(cfg.Monitoring.AggregationInterval) * time.Minute
	logAggregator := basic.NewLogAggregatorWithInterval(fileLogger.Logger, fileLogger.GetBasicLogPath(), aggregationInterval)
	logAggregator.Start()
	defer logAggregator.Stop()

	// Initialize and run basic monitoring agent if enabled (in background with file-only logger)
	if cfg.Monitoring.EnableMonitor {
		// Start in a separate goroutine to avoid logging interfering with chat
		go func() {
			monitorAgent := basicagent.NewBasicMonitorAgent(clientset, cfg.Monitoring.Namespace, cfg, fileLogger.Logger)
			if err := monitorAgent.Start(ctx); err != nil {
				consoleLogger.Errorf("Error starting basic monitoring agent: %v", err)
			}
		}()
	}

	// Initialize and run basic alert agent if enabled (in background with file-only logger)
	if cfg.Monitoring.EnableAlert {
		// Start in a separate goroutine to avoid logging interfering with chat
		go func() {
			alertAgent := basicagent.NewBasicAlertAgent(clientset, cfg.Monitoring.Namespace, cfg, fileLogger.Logger)
			if err := alertAgent.Start(ctx); err != nil {
				consoleLogger.Errorf("Error starting basic alert agent: %v", err)
			}
		}()
	}

	// Initialize and run basic remediation agent if enabled (in background with file-only logger)
	if cfg.Monitoring.EnableRemediation {
		// Run remediation checks periodically in background
		go func() {
			remediationAgent := basicagent.NewBasicRemediationAgent(clientset, cfg.Monitoring.Namespace, cfg)

			ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := remediationAgent.RunRemediationCycle(); err != nil {
						consoleLogger.Printf("Error running remediation cycle: %v", err)
					}
				case <-ctx.Done():
					consoleLogger.Info("Stopping remediation agent...")
					return
				}
			}
		}()
	}

	// Initialize and run AI-enhanced monitoring agent if decision-making is enabled (in background with file-only logger)
	if cfg.Monitoring.EnableDecision && cfg.LLM.Enabled {
		// Start in a separate goroutine to avoid logging interfering with chat
		go func() {
			aiMonitorAgent, err := aiagent.NewAIEnhancedMonitorAgent(cfg)
			if err != nil {
				consoleLogger.Printf("Error creating AI-enhanced monitoring agent: %v", err)
			} else {
				if err := aiMonitorAgent.Start(ctx); err != nil {
					consoleLogger.Printf("Error starting AI-enhanced monitoring agent: %v", err)
				}
			}
		}()
	}

	// Initialize and run AI-enhanced alert agent if decision-making is enabled (in background with file-only logger)
	if cfg.Monitoring.EnableDecision && cfg.LLM.Enabled {
		// Start in a separate goroutine to avoid logging interfering with chat
		go func() {
			aiAlertAgent, err := aiagent.NewAIEnhancedAlertAgent(cfg)
			if err != nil {
				consoleLogger.Printf("Error creating AI-enhanced alert agent: %v", err)
			} else {
				if err := aiAlertAgent.Start(ctx); err != nil {
					consoleLogger.Printf("Error starting AI-enhanced alert agent: %v", err)
				}
			}
		}()
	}

	// Start the foreground chat interface in the main thread
	if cfg.LLM.Enabled {
		// Clear any previous output and present clean chat interface
		fmt.Print("\n\x1b[2J\x1b[H") // ANSI escape codes to clear screen
		fmt.Println("🚀 OtisBrain AI Assistant Started!")
		fmt.Println("Type your questions below. Type '/exit' to quit.\n")

		// Create the interactive chat instance with console logger
		aiChat := chat.NewAIChat(cfg, &basic.BasicLogger{Logger: consoleLogger})

		// Run the chat interface in the main thread (foreground) - this blocks
		if err := aiChat.Run(ctx); err != nil {
			consoleLogger.Errorf("AI Chat error: %v", err)
		}
	} else {
		fmt.Println("LLM is not enabled in config. Chat interface will not start.")
		fmt.Println("Current config enables LLM:", cfg.LLM.Enabled)

		// If chat is not enabled, wait for context cancellation
		<-ctx.Done()
	}

	consoleLogger.Info("Application stopped gracefully")
}