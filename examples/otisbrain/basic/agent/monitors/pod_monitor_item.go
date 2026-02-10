package monitors

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent/common"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// PodMonitorItem 监控 Pod 状态的实现
type PodMonitorItem struct{}

func (p *PodMonitorItem) GetName() string {
	return "pod-monitor"
}

func (p *PodMonitorItem) GetConfigKey() string {
	return "monitoring.enable_pod_monitoring"
}

func (p *PodMonitorItem) IsEnabled(config *config.Config) bool {
	// 假设配置中有这个字段，这里使用 EnableMonitorResources 作为示例
	// 实际实现中需要根据具体配置结构来判断
	return config.Monitoring.EnableMonitorResources
}

func (p *PodMonitorItem) Monitor(ctx context.Context, params common.MonitorParams) error {
	pods, err := params.Clientset.CoreV1().Pods(params.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods in namespace %s: %w", params.Namespace, err)
	}

	for _, pod := range pods.Items {
		// Only focus on abnormal pod conditions, ignore normal startup phases
		status := pod.Status.Phase
		if status == corev1.PodRunning || status == corev1.PodPending {
			// Skip logging normal startup activities like pulling, starting containers
			// Only check for abnormal conditions
			for _, condition := range pod.Status.Conditions {
				// Only record abnormal conditions
				if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse &&
					(condition.Reason == "PodCompleted" || condition.Reason == "ContainersNotReady") {
					params.Logger.Warnf("CRITICAL EVENT: Pod %s/%s has abnormal condition %s with reason: %s",
						params.Namespace, pod.Name, condition.Type, condition.Reason)

					// Write critical event to dedicated log file
					common.WriteCriticalEvent(params.Namespace, "pod", pod.Name, string(condition.Type), condition.Reason, params.Logger)
				}
			}
		} else if status == corev1.PodFailed || status == corev1.PodUnknown {
			// Log failed or unknown pod states
			params.Logger.Warnf("CRITICAL EVENT: Pod %s/%s status: %s", params.Namespace, pod.Name, status)
			common.WriteCriticalEvent(params.Namespace, "pod", pod.Name, "PodStatus", string(status), params.Logger)
		}

		// Check for abnormal container statuses - only log critical issues
		for _, containerStatus := range pod.Status.ContainerStatuses {
			// Check for crash loop backoff and other abnormal states
			if containerStatus.State.Waiting != nil {
				waitingReason := containerStatus.State.Waiting.Reason
				if waitingReason == "CrashLoopBackOff" || waitingReason == "ImagePullBackOff" ||
					waitingReason == "ErrImagePull" || waitingReason == "CreateContainerConfigError" {
					params.Logger.Warnf("CRITICAL EVENT: Container %s in pod %s/%s is in waiting state with reason: %s",
						containerStatus.Name, params.Namespace, pod.Name, waitingReason)

					common.WriteCriticalEvent(params.Namespace, "container", containerStatus.Name, "WaitingState",
						fmt.Sprintf("%s in pod %s", waitingReason, pod.Name), params.Logger)
				}
			}

			// Check for terminated containers with OOMKilled or other critical reasons
			if containerStatus.State.Terminated != nil {
				terminatedReason := containerStatus.State.Terminated.Reason
				if terminatedReason == "OOMKilled" || terminatedReason == "Error" {
					params.Logger.Warnf("CRITICAL EVENT: Container %s in pod %s/%s was terminated with reason: %s",
						containerStatus.Name, params.Namespace, pod.Name, terminatedReason)

					common.WriteCriticalEvent(params.Namespace, "container", containerStatus.Name, "Terminated",
						fmt.Sprintf("%s in pod %s", terminatedReason, pod.Name), params.Logger)
				}
			}

			// Check for high restart count (indicating crashes)
			if containerStatus.RestartCount > 5 {
				params.Logger.Warnf("CRITICAL EVENT: Container %s in pod %s/%s has restarted %d times",
					containerStatus.Name, params.Namespace, pod.Name, containerStatus.RestartCount)

				common.WriteCriticalEvent(params.Namespace, "container", containerStatus.Name, "HighRestartCount",
					fmt.Sprintf("Restarted %d times in pod %s", containerStatus.RestartCount, pod.Name), params.Logger)
			}
		}
	}

	return nil
}
