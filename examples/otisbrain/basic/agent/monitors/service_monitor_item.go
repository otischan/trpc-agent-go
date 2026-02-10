package monitors

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent/common"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// ServiceMonitorItem 监控 Service 状态的实现
type ServiceMonitorItem struct{}

func (s *ServiceMonitorItem) GetName() string {
	return "service-monitor"
}

func (s *ServiceMonitorItem) GetConfigKey() string {
	return "monitoring.enable_service_monitoring"
}

func (s *ServiceMonitorItem) IsEnabled(config *config.Config) bool {
	return config.Monitoring.EnableMonitorResources
}

func (s *ServiceMonitorItem) Monitor(ctx context.Context, params common.MonitorParams) error {
	services, err := params.Clientset.CoreV1().Services(params.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services in namespace %s: %w", params.Namespace, err)
	}

	for _, service := range services.Items {
		params.Logger.Debugf("Service %s/%s - Type: %s", params.Namespace, service.Name, service.Spec.Type)

		// Services typically don't have status conditions like pods/deployments
		// But we can still log their existence and configuration
	}

	return nil
}
