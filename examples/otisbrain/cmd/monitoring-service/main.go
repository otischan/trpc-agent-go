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
	"k8s.io/metrics/pkg/client/clientset/versioned"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent/monitor"
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

	// Create metrics client for memory metrics
	kubeConfig, err := k8sclient.GetConfig(cfg.Monitoring.Kubeconfig)
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}
	metricsClient, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("Failed to create metrics client: %v", err)
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

	// Use the new monitoring architecture
	factory := &monitor.MonitorFactory{}
	monitorAgent, err := factory.CreateMonitorAgent(clientset, metricsClient, cfg, fileLogger.Logger)
	if err != nil {
		consoleLogger.Fatalf("Failed to create monitoring agent: %v", err)
	}

	if err := monitorAgent.Start(ctx); err != nil {
		consoleLogger.Errorf("Error starting monitoring agent: %v", err)
	}

	consoleLogger.Info("Monitoring service started successfully with new architecture")

	// Wait for context cancellation
	<-ctx.Done()

	consoleLogger.Info("Monitoring service stopped gracefully")
}
