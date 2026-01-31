package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// BasicMonitorAgent implements the basic monitoring functionality
type BasicMonitorAgent struct {
	clientset *kubernetes.Clientset
	namespace string
	config    *config.Config
	stopCh    chan struct{}
}

// NewBasicMonitorAgent creates a new basic monitoring agent
func NewBasicMonitorAgent(clientset *kubernetes.Clientset, namespace string, cfg *config.Config) *BasicMonitorAgent {
	return &BasicMonitorAgent{
		clientset: clientset,
		namespace: namespace,
		config:    cfg,
		stopCh:    make(chan struct{}),
	}
}

// Start starts the basic monitoring agent
func (bma *BasicMonitorAgent) Start(ctx context.Context) error {
	log.Printf("BasicMonitorAgent started for namespace: %s", bma.namespace)

	// Start monitoring in a separate goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(bma.config.Basic.IntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := bma.monitor(); err != nil {
					log.Printf("Error during monitoring: %v", err)
				}
			case <-bma.stopCh:
				log.Println("BasicMonitorAgent stopped")
				return
			case <-ctx.Done():
				log.Println("Context cancelled, stopping BasicMonitorAgent")
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
	log.Printf("Performing basic monitoring for namespace: %s", bma.namespace)

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
		// Check pod status
		status := pod.Status.Phase
		log.Printf("Pod %s/%s status: %s", bma.namespace, pod.Name, status)

		// Record critical events related to pods
		for _, condition := range pod.Status.Conditions {
			if condition.Status == corev1.ConditionFalse {
				log.Printf("CRITICAL EVENT: Pod %s/%s has condition %s with reason: %s",
					bma.namespace, pod.Name, condition.Type, condition.Reason)

				// Write critical event to dedicated log file
				bma.writeCriticalEvent("pod", pod.Name, string(condition.Type), condition.Reason)
			}
		}

		// Check container statuses
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if !containerStatus.Ready {
				log.Printf("CRITICAL EVENT: Container %s in pod %s/%s is not ready", 
					containerStatus.Name, bma.namespace, pod.Name)
				
				bma.writeCriticalEvent("container", containerStatus.Name, "NotReady", 
					fmt.Sprintf("In pod %s", pod.Name))
			}
			
			if containerStatus.RestartCount > 5 {
				log.Printf("WARNING: Container %s in pod %s/%s has restarted %d times", 
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

		log.Printf("Deployment %s/%s - Replicas: %d/%d Ready: %d Updated: %d Available: %d", 
			bma.namespace, deployment.Name, 
			deployment.Status.Replicas, *replicas,
			readyReplicas, updatedReplicas, availableReplicas)

		// Check deployment conditions
		for _, condition := range deployment.Status.Conditions {
			if condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionUnknown {
				log.Printf("CRITICAL EVENT: Deployment %s/%s has condition %s with reason: %s - %s",
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
		log.Printf("Service %s/%s - Type: %s", bma.namespace, service.Name, service.Spec.Type)
		
		// Services typically don't have status conditions like pods/deployments
		// But we can still log their existence and configuration
	}

	return nil
}

// writeCriticalEvent writes critical events to a dedicated log file
func (bma *BasicMonitorAgent) writeCriticalEvent(objType, objName, eventType, message string) {
	// In a real implementation, this would write to the critical_events log file
	// For now, we'll just log it
	log.Printf("CRITICAL_EVENT_LOG - Type: %s, Name: %s, Event: %s, Message: %s, Time: %s", 
		objType, objName, eventType, message, time.Now().Format(time.RFC3339))
}