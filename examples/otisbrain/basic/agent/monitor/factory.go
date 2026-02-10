package monitor

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// MonitorFactory 监控代理工厂
type MonitorFactory struct{}

// CreateMonitorAgent 根据配置创建监控代理
func (f *MonitorFactory) CreateMonitorAgent(
	clientset *kubernetes.Clientset,
	metricsClient *metricsv.Clientset,
	cfg *config.Config,
	logger *logrus.Logger,
) (*BaseMonitorAgent, error) {
	// 从配置中获取命名空间列表
	namespaces := cfg.Monitoring.Namespaces

	// 如果没有指定命名空间，直接返回错误
	if len(namespaces) == 0 {
		return nil, fmt.Errorf("no namespaces specified in configuration, monitoring requires at least one namespace to be specified")
	}

	// 如果指定了 "all"，则获取所有命名空间
	if len(namespaces) == 1 && namespaces[0] == "all" {
		allNamespaces, err := f.getAllNamespaces(clientset)
		if err != nil {
			return nil, fmt.Errorf("failed to get all namespaces: %w", err)
		}
		namespaces = allNamespaces
	}

	return NewBaseMonitorAgent(clientset, metricsClient, namespaces, cfg, logger)
}

// getAllNamespaces 获取所有命名空间
func (f *MonitorFactory) getAllNamespaces(clientset *kubernetes.Clientset) ([]string, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var nsList []string
	for _, ns := range namespaces.Items {
		nsList = append(nsList, ns.Name)
	}

	if len(nsList) == 0 {
		return nil, fmt.Errorf("no namespaces found in cluster")
	}

	return nsList, nil
}
