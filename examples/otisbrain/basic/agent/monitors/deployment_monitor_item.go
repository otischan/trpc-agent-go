package monitors

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent/common"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// DeploymentMonitorItem 监控 Deployment 状态的实现
type DeploymentMonitorItem struct{}

func (d *DeploymentMonitorItem) GetName() string {
	return "deployment-monitor"
}

func (d *DeploymentMonitorItem) GetConfigKey() string {
	return "monitoring.enable_deployment_monitoring"
}

func (d *DeploymentMonitorItem) IsEnabled(config *config.Config) bool {
	return config.Monitoring.EnableMonitorResources
}

func (d *DeploymentMonitorItem) Monitor(ctx context.Context, params common.MonitorParams) error {
	deployments, err := params.Clientset.AppsV1().Deployments(params.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments in namespace %s: %w", params.Namespace, err)
	}

	for _, deployment := range deployments.Items {
		// Check deployment status
		replicas := deployment.Spec.Replicas
		readyReplicas := deployment.Status.ReadyReplicas
		updatedReplicas := deployment.Status.UpdatedReplicas
		availableReplicas := deployment.Status.AvailableReplicas

		params.Logger.Debugf("Deployment %s/%s - Replicas: %d/%d Ready: %d Updated: %d Available: %d",
			params.Namespace, deployment.Name,
			deployment.Status.Replicas, *replicas,
			readyReplicas, updatedReplicas, availableReplicas)

		// Check deployment conditions
		for _, condition := range deployment.Status.Conditions {
			if condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionUnknown {
				params.Logger.Warnf("CRITICAL EVENT: Deployment %s/%s has condition %s with reason: %s - %s",
					params.Namespace, deployment.Name, condition.Type, condition.Reason, condition.Message)

				common.WriteCriticalEvent(params.Namespace, "deployment", deployment.Name, string(condition.Type),
					fmt.Sprintf("%s - %s", condition.Reason, condition.Message), params.Logger)
			}
		}
	}

	return nil
}
