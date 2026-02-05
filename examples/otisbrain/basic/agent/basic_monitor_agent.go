package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
)

// BasicMonitorAgent implements the basic monitoring functionality
type BasicMonitorAgent struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
	namespace     string
	config        *config.Config
	logger        *logrus.Logger
	stopCh        chan struct{}
	memoryCollector *basic.MemoryCollector
}

// NewBasicMonitorAgent creates a new basic monitoring agent
func NewBasicMonitorAgent(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, namespace string, cfg *config.Config, logger *logrus.Logger) *BasicMonitorAgent {
	agent := &BasicMonitorAgent{
		clientset:     clientset,
		metricsClient: metricsClient,
		namespace:     namespace,
		config:        cfg,
		logger:        logger,
		stopCh:        make(chan struct{}),
	}

	// Initialize memory collector if memory monitoring is enabled
	if cfg.Monitoring.MemoryMonitoring.Enabled && cfg.Monitoring.MemoryMonitoring.BasicCollection.Enabled {
		retentionDays := cfg.Monitoring.MemoryMonitoring.BasicCollection.RetentionDays
		if retentionDays <= 0 {
			retentionDays = 30 // default to 30 days
		}
		agent.memoryCollector = basic.NewMemoryCollector(clientset, metricsClient, logger, retentionDays)
	}

	return agent
}

// Start starts the basic monitoring agent
func (bma *BasicMonitorAgent) Start(ctx context.Context) error {
	bma.logger.Infof("BasicMonitorAgent started for namespace: %s", bma.namespace)

	// Start monitoring in a separate goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(bma.config.Basic.IntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := bma.monitor(); err != nil {
					bma.logger.Errorf("Error during monitoring: %v", err)
				}
				// Collect memory metrics if enabled
				if bma.memoryCollector != nil && bma.config.Monitoring.MemoryMonitoring.Enabled {
					if err := bma.memoryCollector.CollectMemoryMetrics(bma.namespace); err != nil {
						bma.logger.Errorf("Error collecting memory metrics: %v", err)
					}
				}
			case <-bma.stopCh:
				bma.logger.Info("BasicMonitorAgent stopped")
				return
			case <-ctx.Done():
				bma.logger.Info("Context cancelled, stopping BasicMonitorAgent")
				close(bma.stopCh)
				return
			}
		}
	}()

	return nil
}

// Stop stops the basic monitoring agent
func (bma *BasicMonitorAgent) Stop() {
	close(bma.stopCh)
}

// monitor performs the actual monitoring work
func (bma *BasicMonitorAgent) monitor() error {
	bma.logger.Debugf("Performing basic monitoring for namespace: %s", bma.namespace)

	// Monitor pods
	if err := bma.monitorPods(); err != nil {
		return fmt.Errorf("error monitoring pods: %w", err)
	}

	// Monitor deployments
	if err := bma.monitorDeployments(); err != nil {
		return fmt.Errorf("error monitoring deployments: %w", err)
	}

	// Monitor services
	if err := bma.monitorServices(); err != nil {
		return fmt.Errorf("error monitoring services: %w", err)
	}

	return nil
}


// monitorPods monitors the status of pods in the namespace
func (bma *BasicMonitorAgent) monitorPods() error {
	pods, err := bma.clientset.CoreV1().Pods(bma.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
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
					bma.logger.Warnf("CRITICAL EVENT: Pod %s/%s has abnormal condition %s with reason: %s",
						bma.namespace, pod.Name, condition.Type, condition.Reason)

					// Write critical event to dedicated log file
					bma.writeCriticalEvent("pod", pod.Name, string(condition.Type), condition.Reason)
				}
			}
		} else if status == corev1.PodFailed || status == corev1.PodUnknown {
			// Log failed or unknown pod states
			bma.logger.Warnf("CRITICAL EVENT: Pod %s/%s status: %s", bma.namespace, pod.Name, status)
			bma.writeCriticalEvent("pod", pod.Name, "PodStatus", string(status))
		}

		// Check for abnormal container statuses - only log critical issues
		for _, containerStatus := range pod.Status.ContainerStatuses {
			// Check for crash loop backoff and other abnormal states
			if containerStatus.State.Waiting != nil {
				waitingReason := containerStatus.State.Waiting.Reason
				if waitingReason == "CrashLoopBackOff" || waitingReason == "ImagePullBackOff" ||
				   waitingReason == "ErrImagePull" || waitingReason == "CreateContainerConfigError" {
					bma.logger.Warnf("CRITICAL EVENT: Container %s in pod %s/%s is in waiting state with reason: %s",
						containerStatus.Name, bma.namespace, pod.Name, waitingReason)

					bma.writeCriticalEvent("container", containerStatus.Name, "WaitingState",
						fmt.Sprintf("%s in pod %s", waitingReason, pod.Name))
				}
			}

			// Check for terminated containers with OOMKilled or other critical reasons
			if containerStatus.State.Terminated != nil {
				terminatedReason := containerStatus.State.Terminated.Reason
				if terminatedReason == "OOMKilled" || terminatedReason == "Error" {
					bma.logger.Warnf("CRITICAL EVENT: Container %s in pod %s/%s was terminated with reason: %s",
						containerStatus.Name, bma.namespace, pod.Name, terminatedReason)

					bma.writeCriticalEvent("container", containerStatus.Name, "Terminated",
						fmt.Sprintf("%s in pod %s", terminatedReason, pod.Name))
				}
			}

			// Check for high restart count (indicating crashes)
			if containerStatus.RestartCount > 5 {
				bma.logger.Warnf("CRITICAL EVENT: Container %s in pod %s/%s has restarted %d times",
					containerStatus.Name, bma.namespace, pod.Name, containerStatus.RestartCount)

				bma.writeCriticalEvent("container", containerStatus.Name, "HighRestartCount",
					fmt.Sprintf("Restarted %d times in pod %s", containerStatus.RestartCount, pod.Name))
			}
		}
	}

	return nil
}

// monitorDeployments monitors the status of deployments in the namespace
func (bma *BasicMonitorAgent) monitorDeployments() error {
	deployments, err := bma.clientset.AppsV1().Deployments(bma.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	for _, deployment := range deployments.Items {
		// Check deployment status
		replicas := deployment.Spec.Replicas
		readyReplicas := deployment.Status.ReadyReplicas
		updatedReplicas := deployment.Status.UpdatedReplicas
		availableReplicas := deployment.Status.AvailableReplicas

		bma.logger.Debugf("Deployment %s/%s - Replicas: %d/%d Ready: %d Updated: %d Available: %d",
			bma.namespace, deployment.Name,
			deployment.Status.Replicas, *replicas,
			readyReplicas, updatedReplicas, availableReplicas)

		// Check deployment conditions
		for _, condition := range deployment.Status.Conditions {
			if condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionUnknown {
				bma.logger.Warnf("CRITICAL EVENT: Deployment %s/%s has condition %s with reason: %s - %s",
					bma.namespace, deployment.Name, condition.Type, condition.Reason, condition.Message)

				bma.writeCriticalEvent("deployment", deployment.Name, string(condition.Type),
					fmt.Sprintf("%s - %s", condition.Reason, condition.Message))
			}
		}
	}

	return nil
}

// monitorServices monitors the status of services in the namespace
func (bma *BasicMonitorAgent) monitorServices() error {
	services, err := bma.clientset.CoreV1().Services(bma.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	for _, service := range services.Items {
		bma.logger.Debugf("Service %s/%s - Type: %s", bma.namespace, service.Name, service.Spec.Type)

		// Services typically don't have status conditions like pods/deployments
		// But we can still log their existence and configuration
	}

	return nil
}

// writeCriticalEvent writes critical events to a dedicated log file
func (bma *BasicMonitorAgent) writeCriticalEvent(objType, objName, eventType, message string) {
	bma.logger.Errorf("CRITICAL_EVENT_LOG - Namespace: %s, Type: %s, Name: %s, Event: %s, Message: %s",
		bma.namespace, objType, objName, eventType, message)

	// Write in format suitable for aggregation
	bma.logger.WithFields(logrus.Fields{
		"namespace": bma.namespace,
		"objType":   objType,
		"objName":   objName,
		"eventType": eventType,
		"message":   message,
	}).Error("CRITICAL_EVENT")
	
	// Write critical event to dedicated file in the logger's path
	// Since we're passing the logger from BasicLogger.Logger, we need to handle critical events properly
	// Create a more direct approach by using the namespace to write to the correct path
	logPath := filepath.Join("logs", "basic", bma.namespace)
	criticalLogPath := filepath.Join(logPath, "critical_events.log")
	
	criticalLogFile, err := os.OpenFile(criticalLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer criticalLogFile.Close()

		// Create a log entry with the desired fields and message
		entry := &logrus.Entry{
			Logger: bma.logger,
			Data: logrus.Fields{
				"type":    objType,
				"name":    objName,
				"event":   eventType,
				"message": message,
			},
			Level: logrus.ErrorLevel,
			Time:  time.Now(),
			Message: "CRITICAL_EVENT_LOG",
		}

		serialized, _ := bma.logger.Formatter.Format(entry)
		criticalLogFile.Write(serialized)
	}
}