package monitor

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent/common"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent/monitors"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// BaseMonitorAgent 统一的监控代理，支持单/多命名空间
type BaseMonitorAgent struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
	namespaces    []string // 支持单个或多个命名空间
	config        *config.Config
	logger        *logrus.Logger
	stopCh        chan struct{}
	registry      *MonitorRegistry
	interval      time.Duration
}

// NewBaseMonitorAgent 创建监控代理，要求必须提供至少一个命名空间
func NewBaseMonitorAgent(
	clientset *kubernetes.Clientset,
	metricsClient *metricsv.Clientset,
	namespaces []string,
	cfg *config.Config,
	logger *logrus.Logger,
) (*BaseMonitorAgent, error) {
	// 验证命名空间参数
	if len(namespaces) == 0 {
		return nil, fmt.Errorf("at least one namespace must be specified for monitoring")
	}

	// 验证命名空间名称的有效性
	for _, ns := range namespaces {
		if ns == "" {
			return nil, fmt.Errorf("namespace name cannot be empty")
		}
		if ns == "all" {
			return nil, fmt.Errorf("'all' is not allowed as a namespace name, use specific namespaces or handle 'all' at a higher level")
		}
	}

	agent := &BaseMonitorAgent{
		clientset:     clientset,
		metricsClient: metricsClient,
		namespaces:    namespaces,
		config:        cfg,
		logger:        logger,
		stopCh:        make(chan struct{}),
		registry:      setupMonitorRegistry(cfg),
		interval:      time.Duration(cfg.Basic.IntervalSeconds) * time.Second,
	}

	return agent, nil
}

// setupMonitorRegistry 初始化并注册所有监控项
func setupMonitorRegistry(cfg *config.Config) *MonitorRegistry {
	registry := NewMonitorRegistry()

	// 注册所有内置监控项
	registry.Register(&monitors.PodMonitorItem{})
	registry.Register(&monitors.DeploymentMonitorItem{})
	registry.Register(&monitors.ServiceMonitorItem{})
	registry.Register(&monitors.EventMonitorItem{})
	registry.Register(&monitors.MemoryMonitorItem{})

	return registry
}

// Start 启动监控代理
func (bma *BaseMonitorAgent) Start(ctx context.Context) error {
	bma.logger.Infof("BaseMonitorAgent started for namespaces: %v", bma.namespaces)

	// 为每个命名空间启动监控协程
	var wg sync.WaitGroup

	for _, namespace := range bma.namespaces {
		wg.Add(1)
		go func(ns string) {
			defer wg.Done()
			bma.startNamespaceMonitoring(ctx, ns)
		}(namespace)
	}

	// 监听停止信号
	go func() {
		<-ctx.Done()
		bma.logger.Info("Context cancelled, stopping BaseMonitorAgent")
		close(bma.stopCh)
		wg.Wait() // 等待所有监控协程结束
	}()

	return nil
}

// startNamespaceMonitoring 为指定命名空间启动监控
func (bma *BaseMonitorAgent) startNamespaceMonitoring(ctx context.Context, namespace string) {
	// 为每个命名空间创建专用的执行器和参数
	params := common.MonitorParams{
		Clientset:     bma.clientset,
		MetricsClient: bma.metricsClient,
		Namespace:     namespace,
		Config:        bma.config,
		Logger:        bma.createNamespaceLogger(namespace),
	}

	executor := NewMonitorExecutor(bma.registry, params)

	ticker := time.NewTicker(bma.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := executor.Execute(ctx); err != nil {
				params.Logger.Errorf("Error during monitoring: %v", err)
			}
		case <-bma.stopCh:
			params.Logger.Info("Stopped monitoring for namespace: ", namespace)
			return
		case <-ctx.Done():
			params.Logger.Info("Context cancelled for namespace: ", namespace)
			return
		}
	}
}

// createNamespaceLogger 为指定命名空间创建专用日志记录器
func (bma *BaseMonitorAgent) createNamespaceLogger(namespace string) *logrus.Logger {
	logPath := filepath.Join("logs", "basic", namespace)
	logger, err := basic.NewBasicLoggerWithCustomPath(bma.config.LogLevel, logPath)
	if err != nil {
		bma.logger.Errorf("Failed to create namespace-specific logger for %s: %v", namespace, err)
		return bma.logger // 回退到原始日志记录器
	}
	return logger.Logger
}
