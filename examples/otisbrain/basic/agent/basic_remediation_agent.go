package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// BasicRemediationAgent handles automatic remediation tasks
type BasicRemediationAgent struct {
	clientset *kubernetes.Clientset
	namespace string
	config    *config.Config
	stopCh    chan struct{}
}

// NewBasicRemediationAgent creates a new basic remediation agent
func NewBasicRemediationAgent(clientset *kubernetes.Clientset, namespace string, cfg *config.Config) *BasicRemediationAgent {
	return &BasicRemediationAgent{
		clientset: clientset,
		namespace: namespace,
		config:    cfg,
		stopCh:    make(chan struct{}),
	}
}

// Start starts the basic remediation agent
func (bra *BasicRemediationAgent) Start(ctx context.Context) error {
	log.Printf("BasicRemediationAgent started for namespace: %s", bra.namespace)

	return nil
}

// Stop stops the basic remediation agent
func (bra *BasicRemediationAgent) Stop() {
	close(bra.stopCh)
}

// PerformRemediation performs remediation based on the issue detected
func (bra *BasicRemediationAgent) PerformRemediation(issueType, resourceName, namespace string) error {
	if bra.config.Basic.DryRun {
		log.Printf("[DRY RUN] Would perform remediation for %s: %s/%s", issueType, namespace, resourceName)
		return nil
	}

	switch issueType {
	case "pod_crash_loop":
		return bra.restartPod(resourceName, namespace)
	case "deployment_unavailable":
		return bra.restartDeployment(resourceName, namespace)
	case "high_restart_count":
		return bra.restartPod(resourceName, namespace)
	default:
		log.Printf("Unknown issue type for remediation: %s", issueType)
		return fmt.Errorf("unknown issue type: %s", issueType)
	}
}

// restartPod deletes and recreates a pod to restart it
func (bra *BasicRemediationAgent) restartPod(podName, namespace string) error {
	log.Printf("Restarting pod: %s/%s", namespace, podName)

	// Delete the pod, allowing the controller to recreate it
	err := bra.clientset.CoreV1().Pods(namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod %s/%s: %w", namespace, podName, err)
	}

	log.Printf("Successfully deleted pod: %s/%s", namespace, podName)
	return nil
}

// restartDeployment scales the deployment down and up to restart all pods
func (bra *BasicRemediationAgent) restartDeployment(deploymentName, namespace string) error {
	log.Printf("Restarting deployment: %s/%s", namespace, deploymentName)

	// Get the current deployment
	dep, err := bra.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment %s/%s: %w", namespace, deploymentName, err)
	}

	// Store the original replica count
	originalReplicas := *dep.Spec.Replicas

	// Scale down to 0
	log.Printf("Scaling deployment %s/%s to 0 replicas", namespace, deploymentName)
	dep.Spec.Replicas = &[]int32{0}[0]
	_, err = bra.clientset.AppsV1().Deployments(namespace).Update(context.TODO(), dep, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale down deployment %s/%s: %w", namespace, deploymentName, err)
	}

	// Wait a bit
	time.Sleep(5 * time.Second)

	// Scale back up to original count
	log.Printf("Scaling deployment %s/%s back to %d replicas", namespace, deploymentName, originalReplicas)
	dep.Spec.Replicas = &originalReplicas
	_, err = bra.clientset.AppsV1().Deployments(namespace).Update(context.TODO(), dep, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale up deployment %s/%s: %w", namespace, deploymentName, err)
	}

	log.Printf("Successfully restarted deployment: %s/%s", namespace, deploymentName)
	return nil
}

// checkAndRemediatePodIssues checks for pod issues and performs remediation if needed
func (bra *BasicRemediationAgent) checkAndRemediatePodIssues() error {
	pods, err := bra.clientset.CoreV1().Pods(bra.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		// Check if pod is in CrashLoopBackOff
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason == "CrashLoopBackOff" {
				log.Printf("Detected CrashLoopBackOff for pod %s/%s, initiating remediation", bra.namespace, pod.Name)
				if err := bra.PerformRemediation("pod_crash_loop", pod.Name, bra.namespace); err != nil {
					log.Printf("Failed to remediate pod crash loop for %s/%s: %v", bra.namespace, pod.Name, err)
				}
			}

			// Check for high restart count
			if containerStatus.RestartCount > 5 {
				log.Printf("Detected high restart count (%d) for pod %s/%s, initiating remediation",
					containerStatus.RestartCount, bra.namespace, pod.Name)
				if err := bra.PerformRemediation("high_restart_count", pod.Name, bra.namespace); err != nil {
					log.Printf("Failed to remediate high restart count for %s/%s: %v", bra.namespace, pod.Name, err)
				}
			}
		}
	}

	return nil
}

// checkAndRemediateDeploymentIssues checks for deployment issues and performs remediation if needed
func (bra *BasicRemediationAgent) checkAndRemediateDeploymentIssues() error {
	deployments, err := bra.clientset.AppsV1().Deployments(bra.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	for _, deployment := range deployments.Items {
		// Check if deployment is unavailable
		for _, condition := range deployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentProgressing &&
				condition.Status == corev1.ConditionFalse &&
				condition.Reason == "ProgressDeadlineExceeded" {
				log.Printf("Detected unavailable deployment %s/%s, initiating remediation", bra.namespace, deployment.Name)
				if err := bra.PerformRemediation("deployment_unavailable", deployment.Name, bra.namespace); err != nil {
					log.Printf("Failed to remediate unavailable deployment for %s/%s: %v", bra.namespace, deployment.Name, err)
				}
			}
		}
	}

	return nil
}

// RunRemediationCycle runs a single cycle of remediation checks
func (bra *BasicRemediationAgent) RunRemediationCycle() error {
	if err := bra.checkAndRemediatePodIssues(); err != nil {
		return fmt.Errorf("error checking pod issues: %w", err)
	}

	if err := bra.checkAndRemediateDeploymentIssues(); err != nil {
		return fmt.Errorf("error checking deployment issues: %w", err)
	}

	return nil
}
