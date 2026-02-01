package agent

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// BasicEventMonitorAgent handles Kubernetes event monitoring
type BasicEventMonitorAgent struct {
	clientset *kubernetes.Clientset
	namespace string
	config    *config.Config
	logger    *logrus.Logger
	stopCh    chan struct{}
}

// NewBasicEventMonitorAgent creates a new basic event monitoring agent
func NewBasicEventMonitorAgent(clientset *kubernetes.Clientset, namespace string, cfg *config.Config, logger *logrus.Logger) *BasicEventMonitorAgent {
	return &BasicEventMonitorAgent{
		clientset: clientset,
		namespace: namespace,
		config:    cfg,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
}

// Start starts the basic event monitoring agent
func (bea *BasicEventMonitorAgent) Start(ctx context.Context) error {
	bea.logger.Infof("BasicEventMonitorAgent started for namespace: %s", bea.namespace)

	// Start watching events in a separate goroutine
	go func() {
		err := bea.watchEvents(ctx)
		if err != nil {
			bea.logger.Errorf("Error watching events: %v", err)
		}
	}()

	return nil
}

// Stop stops the basic event monitoring agent
func (bea *BasicEventMonitorAgent) Stop() {
	close(bea.stopCh)
}

// watchEvents watches for Kubernetes events and processes them
func (bea *BasicEventMonitorAgent) watchEvents(ctx context.Context) error {
	watcher, err := bea.clientset.CoreV1().Events(bea.namespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to create event watcher: %w", err)
	}
	defer watcher.Stop()

	bea.logger.Infof("Started watching events for namespace: %s", bea.namespace)

	for {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Error {
				bea.logger.Errorf("Error watching events: %v", event.Object)
				continue
			}

			if event.Object == nil {
				continue
			}

			switch obj := event.Object.(type) {
			case *corev1.Event:
				bea.processEvent(obj)
			}
		case <-bea.stopCh:
			bea.logger.Info("BasicEventMonitorAgent stopped")
			return nil
		case <-ctx.Done():
			bea.logger.Info("Context cancelled, stopping BasicEventMonitorAgent")
			return nil
		}
	}
}

// processEvent processes a Kubernetes event
func (bea *BasicEventMonitorAgent) processEvent(event *corev1.Event) {
	// Determine severity based on event type and reason
	severity := bea.determineSeverity(event)

	// Ignore normal operations
	if severity == "IGNORE" {
		return
	}

	bea.logger.Debugf("Processing event: %s/%s - Reason: %s, Message: %s",
		event.Namespace, event.Name, event.Reason, event.Message)

	// Log the event with severity
	bea.logger.Infof("[%s] Event: %s/%s - Reason: %s, Message: %s",
		severity, event.InvolvedObject.Namespace, event.InvolvedObject.Name,
		event.Reason, event.Message)

	// Write critical events to dedicated log file
	if severity == "CRITICAL" || severity == "ERROR" {
		bea.writeCriticalEvent(&event.InvolvedObject, event.Reason, event.Message)
	}
}

// determineSeverity determines the severity of an event
func (bea *BasicEventMonitorAgent) determineSeverity(event *corev1.Event) string {
	// Define severity mapping based on event reason
	// Only consider abnormal events as critical/error, ignore normal operations
	switch event.Reason {
	case "Failed", "FailedScheduling", "FailedMount", "FailedCreate", "FailedDelete",
	     "Unhealthy", "ImagePullBackOff", "OOMKilled":
		return "CRITICAL"
	case "Warning", "Terminating", "Killing", "Evicted", "BackOff", "CrashLoopBackOff":
		return "ERROR"
	case "Created", "Started", "Pulled", "Scheduled", "Pulling", "Starting", "Start",
	     "ContainerCreating", "SuccessfulAttachVolume", "SuccessfulCreate", "ScalingReplicaSet":
		// Normal operations that should not be logged as events
		return "IGNORE"
	default:
		return "WARNING"
	}
}

// writeCriticalEvent writes critical events to a dedicated log file
func (bea *BasicEventMonitorAgent) writeCriticalEvent(obj *corev1.ObjectReference, eventType, message string) {
	bea.logger.Errorf("CRITICAL_EVENT_LOG - Object: %s/%s (%s), Event: %s, Message: %s",
		obj.Namespace, obj.Name, obj.Kind, eventType, message)

	// Write in format suitable for aggregation
	bea.logger.WithFields(logrus.Fields{
		"namespace": obj.Namespace,
		"objType":   obj.Kind,
		"objName":   obj.Name,
		"eventType": eventType,
		"message":   message,
	}).Error("CRITICAL_EVENT")
}