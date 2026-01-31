package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// BasicAlertAgent handles alert processing
type BasicAlertAgent struct {
	clientset *kubernetes.Clientset
	namespace string
	config    *config.Config
	stopCh    chan struct{}
}

// NewBasicAlertAgent creates a new basic alert agent
func NewBasicAlertAgent(clientset *kubernetes.Clientset, namespace string, cfg *config.Config) *BasicAlertAgent {
	return &BasicAlertAgent{
		clientset: clientset,
		namespace: namespace,
		config:    cfg,
		stopCh:    make(chan struct{}),
	}
}

// Start starts the basic alert agent
func (baa *BasicAlertAgent) Start(ctx context.Context) error {
	log.Printf("BasicAlertAgent started for namespace: %s", baa.namespace)

	// Start watching events in a separate goroutine
	go func() {
		err := baa.watchEvents(ctx)
		if err != nil {
			log.Printf("Error watching events: %v", err)
		}
	}()

	return nil
}

// Stop stops the basic alert agent
func (baa *BasicAlertAgent) Stop() {
	close(baa.stopCh)
}

// watchEvents watches for Kubernetes events and processes them
func (baa *BasicAlertAgent) watchEvents(ctx context.Context) error {
	watcher, err := baa.clientset.CoreV1().Events(baa.namespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to create event watcher: %w", err)
	}
	defer watcher.Stop()

	log.Printf("Started watching events for namespace: %s", baa.namespace)

	for {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Error {
				log.Printf("Error watching events: %v", event.Object)
				continue
			}

			if event.Object == nil {
				continue
			}

			switch obj := event.Object.(type) {
			case *corev1.Event:
				baa.processEvent(obj)
			}
		case <-baa.stopCh:
			log.Println("BasicAlertAgent stopped")
			return nil
		case <-ctx.Done():
			log.Println("Context cancelled, stopping BasicAlertAgent")
			return nil
		}
	}
}

// processEvent processes a Kubernetes event
func (baa *BasicAlertAgent) processEvent(event *corev1.Event) {
	log.Printf("Processing event: %s/%s - Reason: %s, Message: %s", 
		event.Namespace, event.Name, event.Reason, event.Message)

	// Determine severity based on event type and reason
	severity := baa.determineSeverity(event)

	// Log the event with severity
	log.Printf("[%s] Event: %s/%s - Reason: %s, Message: %s", 
		severity, event.InvolvedObject.Namespace, event.InvolvedObject.Name, 
		event.Reason, event.Message)

	// Write critical events to dedicated log file
	if severity == "CRITICAL" || severity == "ERROR" {
		baa.writeCriticalEvent(&event.InvolvedObject, event.Reason, event.Message)
	}
}

// determineSeverity determines the severity of an event
func (baa *BasicAlertAgent) determineSeverity(event *corev1.Event) string {
	// Define severity mapping based on event reason
	switch event.Reason {
	case "Failed", "FailedScheduling", "FailedMount", "FailedCreate", "FailedDelete", 
	     "BackOff", "Unhealthy", "CrashLoopBackOff", "ImagePullBackOff":
		return "CRITICAL"
	case "Warning", "Terminating", "Killing", "Evicted":
		return "ERROR"
	case "Created", "Started", "Pulled", "Scheduled":
		return "INFO"
	default:
		return "WARNING"
	}
}

// writeCriticalEvent writes critical events to a dedicated log file
func (baa *BasicAlertAgent) writeCriticalEvent(obj *corev1.ObjectReference, eventType, message string) {
	// In a real implementation, this would write to the critical_events log file
	// For now, we'll just log it
	log.Printf("CRITICAL_EVENT_LOG - Object: %s/%s (%s), Event: %s, Message: %s, Time: %s", 
		obj.Namespace, obj.Name, obj.Kind, eventType, message, time.Now().Format(time.RFC3339))
}