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
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/shared/k8sclient"
)

var (
	kubeconfig   = flag.String("kubeconfig", "", "Path to the kubeconfig file (default: ~/.kube/config)")
	namespace    = flag.String("namespace", "default", "Target namespace to monitor")
	enableMonitor = flag.Bool("enable-monitor", true, "Enable K8S monitoring agent (default: true)")
	enableAlert   = flag.Bool("enable-alert", true, "Enable alert handling agent (default: true)")
	enableRemediation = flag.Bool("enable-remediation", false, "Enable auto-remediation agent (default: false)")
	enableDecision = flag.Bool("enable-decision", false, "Enable decision-making agent (default: false)")
	metricsPort  = flag.Int("metrics-port", 8080, "Port to expose Prometheus metrics (default: 8080)")
	configPath   = flag.String("config", "./config/config.yaml", "Path to the configuration file (default: \"./config/config.yaml\")")
	logLevel     = flag.String("log-level", "info", "Log level (debug, info, warn, error) (default: \"info\")")
	model        = flag.String("model", "gpt-4o-mini", "LLM model to use for AI agents (default: \"gpt-4o-mini\")")
	maxRetry     = flag.Int("max-retry", 3, "Maximum retry attempts for failed operations (default: 3)")
	dryRun       = flag.Bool("dry-run", false, "Run in dry-run mode without performing actual operations (default: false)")
)

func main() {
	flag.Parse()

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Printf("Failed to create logs directory: %v", err)
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Update config with command-line flags
	cfg.Namespace = *namespace
	cfg.Basic.DryRun = *dryRun
	cfg.Basic.MaxRetries = *maxRetry
	cfg.LLM.Model = *model
	cfg.LLM.Enabled = *enableDecision // Enable LLM if decision agent is enabled

	clientset, err := k8sclient.NewClient(*kubeconfig)
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

	// Initialize and run basic monitoring agent if enabled
	if *enableMonitor {
		log.Println("Starting basic monitoring agent...")

		monitorAgent := basicagent.NewBasicMonitorAgent(clientset, *namespace, cfg)
		if err := monitorAgent.Start(ctx); err != nil {
			log.Printf("Error starting basic monitoring agent: %v", err)
		}
	}

	// Initialize and run basic alert agent if enabled
	if *enableAlert {
		log.Println("Starting basic alert agent...")

		alertAgent := basicagent.NewBasicAlertAgent(clientset, *namespace, cfg)
		if err := alertAgent.Start(ctx); err != nil {
			log.Printf("Error starting basic alert agent: %v", err)
		}
	}

	// Initialize and run basic remediation agent if enabled
	if *enableRemediation {
		log.Println("Starting basic remediation agent...")

		remediationAgent := basicagent.NewBasicRemediationAgent(clientset, *namespace, cfg)

		// Run remediation checks periodically
		go func() {
			ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := remediationAgent.RunRemediationCycle(); err != nil {
						log.Printf("Error running remediation cycle: %v", err)
					}
				case <-ctx.Done():
					log.Println("Stopping remediation agent...")
					return
				}
			}
		}()
	}

	// Initialize and run AI-enhanced monitoring agent if decision-making is enabled
	if *enableDecision && cfg.LLM.Enabled {
		log.Println("Starting AI-enhanced monitoring agent...")

		aiMonitorAgent, err := aiagent.NewAIEnhancedMonitorAgent(cfg)
		if err != nil {
			log.Printf("Error creating AI-enhanced monitoring agent: %v", err)
		} else {
			if err := aiMonitorAgent.Start(ctx); err != nil {
				log.Printf("Error starting AI-enhanced monitoring agent: %v", err)
			}
		}
	}

	// Initialize and run AI-enhanced alert agent if decision-making is enabled
	if *enableDecision && cfg.LLM.Enabled {
		log.Println("Starting AI-enhanced alert agent...")

		aiAlertAgent, err := aiagent.NewAIEnhancedAlertAgent(cfg)
		if err != nil {
			log.Printf("Error creating AI-enhanced alert agent: %v", err)
		} else {
			if err := aiAlertAgent.Start(ctx); err != nil {
				log.Printf("Error starting AI-enhanced alert agent: %v", err)
			}
		}
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Application stopped gracefully")
}