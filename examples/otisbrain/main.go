package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	aiagent "trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/ai/agent"
	basicagent "trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/shared/k8sclient"
)

var (
	configPath = flag.String("config", "./config/config.yaml", "Path to the configuration file (default: \"./config/config.yaml\")")
)

// Global logger instance
var basicLogger *basic.BasicLogger

func initLogger(logLevel string) {
	var err error
	basicLogger, err = basic.NewBasicLogger(logLevel)
	if err != nil {
		log.Fatalf("Failed to initialize basic logger: %v", err)
	}
}

func main() {
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger with log level from config
	initLogger(cfg.LogLevel)

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
		log.Println("Received shutdown signal, stopping...")
		cancel()
	}()

	// Initialize log aggregator with configurable interval
	aggregationInterval := time.Duration(cfg.Monitoring.AggregationInterval) * time.Minute
	logAggregator := basic.NewLogAggregatorWithInterval(basicLogger.Logger, basicLogger.GetBasicLogPath(), aggregationInterval)
	logAggregator.Start()
	defer logAggregator.Stop()

	// Initialize and run basic monitoring agent if enabled
	if cfg.Monitoring.EnableMonitor {
		basicLogger.Info("Starting basic monitoring agent...")

		monitorAgent := basicagent.NewBasicMonitorAgent(clientset, cfg.Monitoring.Namespace, cfg, basicLogger.Logger)
		if err := monitorAgent.Start(ctx); err != nil {
			basicLogger.Errorf("Error starting basic monitoring agent: %v", err)
		}
	}

	// Initialize and run basic alert agent if enabled
	if cfg.Monitoring.EnableAlert {
		basicLogger.Info("Starting basic alert agent...")

		alertAgent := basicagent.NewBasicAlertAgent(clientset, cfg.Monitoring.Namespace, cfg, basicLogger.Logger)
		if err := alertAgent.Start(ctx); err != nil {
			basicLogger.Errorf("Error starting basic alert agent: %v", err)
		}
	}

	// Initialize and run basic remediation agent if enabled
	if cfg.Monitoring.EnableRemediation {
		basicLogger.Info("Starting basic remediation agent...")

		remediationAgent := basicagent.NewBasicRemediationAgent(clientset, cfg.Monitoring.Namespace, cfg)

		// Run remediation checks periodically
		go func() {
			ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := remediationAgent.RunRemediationCycle(); err != nil {
						basicLogger.Printf("Error running remediation cycle: %v", err)
					}
				case <-ctx.Done():
					basicLogger.Info("Stopping remediation agent...")
					return
				}
			}
		}()
	}

	// Initialize and run AI-enhanced monitoring agent if decision-making is enabled
	if cfg.Monitoring.EnableDecision && cfg.LLM.Enabled {
		basicLogger.Info("Starting AI-enhanced monitoring agent...")

		aiMonitorAgent, err := aiagent.NewAIEnhancedMonitorAgent(cfg)
		if err != nil {
			basicLogger.Printf("Error creating AI-enhanced monitoring agent: %v", err)
		} else {
			if err := aiMonitorAgent.Start(ctx); err != nil {
				basicLogger.Printf("Error starting AI-enhanced monitoring agent: %v", err)
			}
		}
	}

	// Initialize and run AI-enhanced alert agent if decision-making is enabled
	if cfg.Monitoring.EnableDecision && cfg.LLM.Enabled {
		basicLogger.Info("Starting AI-enhanced alert agent...")

		aiAlertAgent, err := aiagent.NewAIEnhancedAlertAgent(cfg)
		if err != nil {
			basicLogger.Printf("Error creating AI-enhanced alert agent: %v", err)
		} else {
			if err := aiAlertAgent.Start(ctx); err != nil {
				basicLogger.Printf("Error starting AI-enhanced alert agent: %v", err)
			}
		}
	}

	// Wait for context cancellation
	<-ctx.Done()
	basicLogger.Info("Application stopped gracefully")
}