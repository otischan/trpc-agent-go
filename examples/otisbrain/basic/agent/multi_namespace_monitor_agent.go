package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/client/clientset/versioned"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
)

// MultiNamespaceMonitorAgent implements monitoring across multiple namespaces
type MultiNamespaceMonitorAgent struct {
	clientset     *kubernetes.Clientset
	metricsClient *versioned.Clientset
	namespaces    []string
	config        *config.Config
	logger        *logrus.Logger
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewMultiNamespaceMonitorAgent creates a new multi-namespace monitoring agent
func NewMultiNamespaceMonitorAgent(clientset *kubernetes.Clientset, metricsClient *versioned.Clientset, namespaces []string, cfg *config.Config, logger *logrus.Logger) *MultiNamespaceMonitorAgent {
	return &MultiNamespaceMonitorAgent{
		clientset:     clientset,
		metricsClient: metricsClient,
		namespaces:    namespaces,
		config:        cfg,
		logger:        logger,
		stopCh:        make(chan struct{}),
	}
}

// Start starts the multi-namespace monitoring agent
func (mnma *MultiNamespaceMonitorAgent) Start(ctx context.Context) error {
	mnma.logger.Infof("MultiNamespaceMonitorAgent started for namespaces: %v", mnma.namespaces)

	// Start a separate monitoring routine for each namespace
	for _, namespace := range mnma.namespaces {
		mnma.wg.Add(1)
		go func(ns string) {
			defer mnma.wg.Done()
			mnma.startNamespaceMonitoring(ctx, ns)
		}(namespace)
	}

	return nil
}

// startNamespaceMonitoring starts monitoring for a specific namespace
func (mnma *MultiNamespaceMonitorAgent) startNamespaceMonitoring(ctx context.Context, namespace string) {
	mnma.logger.Infof("Starting monitoring for namespace: %s", namespace)

	// Create a namespace-specific logger that writes to a separate directory
	nsLogger := mnma.createNamespaceLogger(namespace)

	// Start monitoring in a separate goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(mnma.config.Basic.IntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := mnma.monitorNamespace(namespace, nsLogger); err != nil {
					nsLogger.Errorf("Error during monitoring namespace %s: %v", namespace, err)
				}
				// Collect memory metrics if enabled
				if mnma.config.Monitoring.MemoryMonitoring.BasicCollection.Enabled {
					memoryCollector := basic.NewMemoryCollector(mnma.clientset, mnma.metricsClient, nsLogger,
						mnma.config.Monitoring.MemoryMonitoring.BasicCollection.RetentionDays)
					if err := memoryCollector.CollectMemoryMetrics(namespace); err != nil {
						nsLogger.Errorf("Error collecting memory metrics for namespace %s: %v", namespace, err)
					}
				}
			case <-mnma.stopCh:
				nsLogger.Info("MultiNamespaceMonitorAgent stopped for namespace: ", namespace)
				return
			case <-ctx.Done():
				nsLogger.Info("Context cancelled, stopping monitoring for namespace: ", namespace)
				return
			}
		}
	}()
}

// createNamespaceLogger creates a logger specific to a namespace
func (mnma *MultiNamespaceMonitorAgent) createNamespaceLogger(namespace string) *logrus.Logger {
	// Create a logger that writes to a namespace-specific directory
	logPath := filepath.Join("logs", "basic", namespace)
	logger, err := basic.NewBasicLoggerWithCustomPath(mnma.config.LogLevel, logPath)
	if err != nil {
		mnma.logger.Errorf("Failed to create namespace-specific logger for %s: %v", namespace, err)
		// Fallback to the original logger
		return mnma.logger
	}
	return logger.Logger
}

// Stop stops the multi-namespace monitoring agent
func (mnma *MultiNamespaceMonitorAgent) Stop() {
	close(mnma.stopCh)
	mnma.wg.Wait() // Wait for all goroutines to finish
}

// monitorNamespace performs monitoring for a specific namespace
func (mnma *MultiNamespaceMonitorAgent) monitorNamespace(namespace string, logger *logrus.Logger) error {
	logger.Debugf("Performing basic monitoring for namespace: %s", namespace)

	// Monitor pods
	if err := mnma.monitorPods(namespace, logger); err != nil {
		return fmt.Errorf("error monitoring pods in namespace %s: %w", namespace, err)
	}

	// Monitor deployments
	if err := mnma.monitorDeployments(namespace, logger); err != nil {
		return fmt.Errorf("error monitoring deployments in namespace %s: %w", namespace, err)
	}

	// Monitor services
	if err := mnma.monitorServices(namespace, logger); err != nil {
		return fmt.Errorf("error monitoring services in namespace %s: %w", namespace, err)
	}

	return nil
}

// monitorPods monitors the status of pods in the specified namespace
func (mnma *MultiNamespaceMonitorAgent) monitorPods(namespace string, logger *logrus.Logger) error {
	pods, err := mnma.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
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
					logger.Warnf("CRITICAL EVENT: Pod %s/%s has abnormal condition %s with reason: %s",
						namespace, pod.Name, condition.Type, condition.Reason)

					// Write critical event to dedicated log file
					mnma.writeCriticalEvent(namespace, "pod", pod.Name, string(condition.Type), condition.Reason, logger)
				}
			}
		} else if status == corev1.PodFailed || status == corev1.PodUnknown {
			// Log failed or unknown pod states
			logger.Warnf("CRITICAL EVENT: Pod %s/%s status: %s", namespace, pod.Name, status)
			mnma.writeCriticalEvent(namespace, "pod", pod.Name, "PodStatus", string(status), logger)
		}

		// Check for abnormal container statuses - only log critical issues
		for _, containerStatus := range pod.Status.ContainerStatuses {
			// Check for crash loop backoff and other abnormal states
			if containerStatus.State.Waiting != nil {
				waitingReason := containerStatus.State.Waiting.Reason
				if waitingReason == "CrashLoopBackOff" || waitingReason == "ImagePullBackOff" ||
				   waitingReason == "ErrImagePull" || waitingReason == "CreateContainerConfigError" {
					logger.Warnf("CRITICAL EVENT: Container %s in pod %s/%s is in waiting state with reason: %s",
						containerStatus.Name, namespace, pod.Name, waitingReason)

					mnma.writeCriticalEvent(namespace, "container", containerStatus.Name, "WaitingState",
						fmt.Sprintf("%s in pod %s", waitingReason, pod.Name), logger)
				}
			}

			// Check for terminated containers with OOMKilled or other critical reasons
			if containerStatus.State.Terminated != nil {
				terminatedReason := containerStatus.State.Terminated.Reason
				if terminatedReason == "OOMKilled" || terminatedReason == "Error" {
					logger.Warnf("CRITICAL EVENT: Container %s in pod %s/%s was terminated with reason: %s",
						containerStatus.Name, namespace, pod.Name, terminatedReason)

					mnma.writeCriticalEvent(namespace, "container", containerStatus.Name, "Terminated",
						fmt.Sprintf("%s in pod %s", terminatedReason, pod.Name), logger)
				}
			}

			// Check for high restart count (indicating crashes)
			if containerStatus.RestartCount > 5 {
				logger.Warnf("CRITICAL EVENT: Container %s in pod %s/%s has restarted %d times",
					containerStatus.Name, namespace, pod.Name, containerStatus.RestartCount)

				mnma.writeCriticalEvent(namespace, "container", containerStatus.Name, "HighRestartCount",
					fmt.Sprintf("Restarted %d times in pod %s", containerStatus.RestartCount, pod.Name), logger)
			}
		}
	}

	return nil
}

// monitorDeployments monitors the status of deployments in the specified namespace
func (mnma *MultiNamespaceMonitorAgent) monitorDeployments(namespace string, logger *logrus.Logger) error {
	deployments, err := mnma.clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments in namespace %s: %w", namespace, err)
	}

	for _, deployment := range deployments.Items {
		// Check deployment status
		replicas := deployment.Spec.Replicas
		readyReplicas := deployment.Status.ReadyReplicas
		updatedReplicas := deployment.Status.UpdatedReplicas
		availableReplicas := deployment.Status.AvailableReplicas

		logger.Debugf("Deployment %s/%s - Replicas: %d/%d Ready: %d Updated: %d Available: %d",
			namespace, deployment.Name,
			deployment.Status.Replicas, *replicas,
			readyReplicas, updatedReplicas, availableReplicas)

		// Check deployment conditions
		for _, condition := range deployment.Status.Conditions {
			if condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionUnknown {
				logger.Warnf("CRITICAL EVENT: Deployment %s/%s has condition %s with reason: %s - %s",
					namespace, deployment.Name, condition.Type, condition.Reason, condition.Message)

				mnma.writeCriticalEvent(namespace, "deployment", deployment.Name, string(condition.Type),
					fmt.Sprintf("%s - %s", condition.Reason, condition.Message), logger)
			}
		}
	}

	return nil
}

// monitorServices monitors the status of services in the specified namespace
func (mnma *MultiNamespaceMonitorAgent) monitorServices(namespace string, logger *logrus.Logger) error {
	services, err := mnma.clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services in namespace %s: %w", namespace, err)
	}

	for _, service := range services.Items {
		logger.Debugf("Service %s/%s - Type: %s", namespace, service.Name, service.Spec.Type)

		// Services typically don't have status conditions like pods/deployments
		// But we can still log their existence and configuration
	}

	return nil
}

// writeCriticalEvent writes critical events to a dedicated log file
func (mnma *MultiNamespaceMonitorAgent) writeCriticalEvent(namespace, objType, objName, eventType, message string, logger *logrus.Logger) {
	logger.Errorf("CRITICAL_EVENT_LOG - Namespace: %s, Type: %s, Name: %s, Event: %s, Message: %s",
		namespace, objType, objName, eventType, message)

	// Write in format suitable for aggregation
	logger.WithFields(logrus.Fields{
		"namespace": namespace,
		"objType":   objType,
		"objName":   objName,
		"eventType": eventType,
		"message":   message,
	}).Error("CRITICAL_EVENT")
}