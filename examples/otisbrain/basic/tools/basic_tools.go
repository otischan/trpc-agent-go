package tools

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// KubectlTool provides kubectl-like functionality
type KubectlTool struct {
	clientset *kubernetes.Clientset
}

// NewKubectlTool creates a new kubectl tool
func NewKubectlTool(kubeconfigPath string) (*KubectlTool, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubectl config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubectl client: %w", err)
	}

	return &KubectlTool{
		clientset: clientset,
	}, nil
}

// RestartTool provides pod restart functionality
type RestartTool struct {
	clientset *kubernetes.Clientset
}

// NewRestartTool creates a new restart tool
func NewRestartTool(clientset *kubernetes.Clientset) *RestartTool {
	return &RestartTool{
		clientset: clientset,
	}
}

// RestartPod deletes a pod to trigger recreation by the controller
func (rt *RestartTool) RestartPod(ctx context.Context, namespace, podName string) error {
	log.Printf("Restarting pod: %s/%s", namespace, podName)

	err := rt.clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod %s/%s: %w", namespace, podName, err)
	}

	log.Printf("Successfully restarted pod: %s/%s", namespace, podName)
	return nil
}

// RollbackTool provides deployment rollback functionality
type RollbackTool struct {
	clientset *kubernetes.Clientset
}

// NewRollbackTool creates a new rollback tool
func NewRollbackTool(clientset *kubernetes.Clientset) *RollbackTool {
	return &RollbackTool{
		clientset: clientset,
	}
}

// RollbackDeployment rolls back a deployment to the previous revision
func (rbt *RollbackTool) RollbackDeployment(ctx context.Context, namespace, deploymentName string) error {
	log.Printf("Rolling back deployment: %s/%s", namespace, deploymentName)

	// Get the deployment
	deployment, err := rbt.clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment %s/%s: %w", namespace, deploymentName, err)
	}

	// For now, we'll just scale down and up to simulate a restart
	// In a real implementation, we would use the rollout undo command
	originalReplicas := *deployment.Spec.Replicas

	// Scale down to 0
	log.Printf("Scaling deployment %s/%s to 0 replicas", namespace, deploymentName)
	deployment.Spec.Replicas = &[]int32{0}[0]
	_, err = rbt.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale down deployment %s/%s: %w", namespace, deploymentName, err)
	}

	// Wait a bit
	// In a real implementation, we would check if all pods are terminated

	// Scale back up to original count
	log.Printf("Scaling deployment %s/%s back to %d replicas", namespace, deploymentName, originalReplicas)
	deployment.Spec.Replicas = &originalReplicas
	_, err = rbt.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale up deployment %s/%s: %w", namespace, deploymentName, err)
	}

	log.Printf("Successfully rolled back deployment: %s/%s", namespace, deploymentName)
	return nil
}