package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
var consoleLogger *logrus.Logger
var fileLogger *basic.BasicLogger

func setProjectRootEnv() {
	// 获取可执行文件路径
	exe, err := os.Executable()
	if err != nil {
		// 如果无法获取可执行文件路径，尝试使用当前工作目录
		currentDir, cwdErr := os.Getwd()
		if cwdErr != nil {
			log.Printf("Warning: Could not determine executable path or current directory: %v", err)
			return
		}
		os.Setenv("OTISBRAIN_PROJECT_ROOT", currentDir)
		return
	}

	// 获取可执行文件所在目录
	exeDir := filepath.Dir(exe)

	// 从可执行文件目录向上查找项目根目录
	projectRoot := findProjectRoot(exeDir)
	if projectRoot != "" {
		os.Setenv("OTISBRAIN_PROJECT_ROOT", projectRoot)
	} else {
		// 如果找不到项目根目录，使用可执行文件所在目录
		os.Setenv("OTISBRAIN_PROJECT_ROOT", exeDir)
	}
}

// findProjectRoot 查找项目根目录
func findProjectRoot(startDir string) string {
	currentDir := startDir

	// 向上遍历目录树，直到找到项目根目录
	for {
		// 检查当前目录是否包含项目标识文件
		if hasProjectMarker(currentDir) {
			return currentDir
		}

		// 到达根目录时停止
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	return "" // 如果没找到，则返回空字符串
}

// hasProjectMarker 检查目录是否包含项目标识文件
func hasProjectMarker(dir string) bool {
	markers := []string{"go.mod", "README.md", "main.go", "config", "resources", "basic", "ai"}

	for _, marker := range markers {
		path := filepath.Join(dir, marker)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

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

	// 设置项目根目录环境变量
	setProjectRootEnv()

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
			// Determine which namespaces to monitor
			var namespaces []string
			if len(cfg.Monitoring.Namespaces) > 0 {
				// Use the new Namespaces field if available
				namespaces = cfg.Monitoring.Namespaces
			} else {
				// Fallback to the old Namespace field for backward compatibility
				namespaces = []string{cfg.Monitoring.Namespace}
			}

			if len(namespaces) == 1 && namespaces[0] == "all" {
				// Special case: monitor all namespaces
				consoleLogger.Info("Monitoring all namespaces")
				monitorAgent := basicagent.NewMultiNamespaceMonitorAgent(clientset, getAllNamespaces(clientset), cfg, fileLogger.Logger)
				if err := monitorAgent.Start(ctx); err != nil {
					consoleLogger.Errorf("Error starting multi-namespace monitoring agent: %v", err)
				}
			} else if len(namespaces) > 1 {
				// Multiple namespaces specified
				consoleLogger.Infof("Monitoring multiple namespaces: %v", namespaces)
				monitorAgent := basicagent.NewMultiNamespaceMonitorAgent(clientset, namespaces, cfg, fileLogger.Logger)
				if err := monitorAgent.Start(ctx); err != nil {
					consoleLogger.Errorf("Error starting multi-namespace monitoring agent: %v", err)
				}
			} else {
				// Single namespace - use the original agent for backward compatibility
				consoleLogger.Infof("Monitoring single namespace: %s", namespaces[0])
				monitorAgent := basicagent.NewBasicMonitorAgent(clientset, namespaces[0], cfg, fileLogger.Logger)
				if err := monitorAgent.Start(ctx); err != nil {
					consoleLogger.Errorf("Error starting basic monitoring agent: %v", err)
				}
			}
		}()
	}

	// Initialize and run basic alert agent if enabled (in background with file-only logger)
	if cfg.Monitoring.EnableAlert {
		// Start in a separate goroutine to avoid logging interfering with chat
		go func() {
			// Determine which namespace to use for alerts
			var alertNamespace string
			if len(cfg.Monitoring.Namespaces) > 0 {
				// Use the first namespace from the list for alerts
				alertNamespace = cfg.Monitoring.Namespaces[0]
			} else {
				alertNamespace = cfg.Monitoring.Namespace
			}

			alertAgent := basicagent.NewBasicAlertAgent(clientset, alertNamespace, cfg, fileLogger.Logger)
			if err := alertAgent.Start(ctx); err != nil {
				consoleLogger.Errorf("Error starting basic alert agent: %v", err)
			}
		}()
	}

	// Initialize and run basic remediation agent if enabled (in background with file-only logger)
	if cfg.Monitoring.EnableRemediation {
		// Determine which namespace to use for remediation
		var remediationNamespace string
		if len(cfg.Monitoring.Namespaces) > 0 {
			// Use the first namespace from the list for remediation
			remediationNamespace = cfg.Monitoring.Namespaces[0]
		} else {
			remediationNamespace = cfg.Monitoring.Namespace
		}

		// Run remediation checks periodically in background
		go func() {
			remediationAgent := basicagent.NewBasicRemediationAgent(clientset, remediationNamespace, cfg)

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
			// Update the config to use the first namespace from the list if multiple are specified
			modifiedCfg := *cfg
			if len(cfg.Monitoring.Namespaces) > 0 {
				modifiedCfg.Monitoring.Namespace = cfg.Monitoring.Namespaces[0]
			}

			aiMonitorAgent, err := aiagent.NewAIEnhancedMonitorAgent(&modifiedCfg)
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
			// Update the config to use the first namespace from the list if multiple are specified
			modifiedCfg := *cfg
			if len(cfg.Monitoring.Namespaces) > 0 {
				modifiedCfg.Monitoring.Namespace = cfg.Monitoring.Namespaces[0]
			}

			aiAlertAgent, err := aiagent.NewAIEnhancedAlertAgent(&modifiedCfg)
			if err != nil {
				consoleLogger.Printf("Error creating AI-enhanced alert agent: %v", err)
			} else {
				if err := aiAlertAgent.Start(ctx); err != nil {
					consoleLogger.Printf("Error starting AI-enhanced alert agent: %v", err)
				}
			}
		}()
	}

	consoleLogger.Info("Monitoring service started successfully")

	// Wait for context cancellation
	<-ctx.Done()

	consoleLogger.Info("Monitoring service stopped gracefully")
}

// getAllNamespaces gets all namespaces in the cluster
func getAllNamespaces(clientset *kubernetes.Clientset) []string {
	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list namespaces: %v", err)
		// Return default namespace as fallback
		return []string{"default"}
	}

	var nsList []string
	for _, ns := range namespaces.Items {
		nsList = append(nsList, ns.Name)
	}
	return nsList
}