package agent

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// BasicRuleEngineAgent implements the basic rule engine functionality
type BasicRuleEngineAgent struct {
	clientset *kubernetes.Clientset
	namespace string
	config    *config.Config
	logger    *logrus.Logger
	stopCh    chan struct{}
}

// NewBasicRuleEngineAgent creates a new basic rule engine agent
func NewBasicRuleEngineAgent(clientset *kubernetes.Clientset, namespace string, cfg *config.Config, logger *logrus.Logger) *BasicRuleEngineAgent {
	return &BasicRuleEngineAgent{
		clientset: clientset,
		namespace: namespace,
		config:    cfg,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
}

// Start starts the basic rule engine agent
func (bea *BasicRuleEngineAgent) Start(ctx context.Context) error {
	bea.logger.Infof("BasicRuleEngineAgent started for namespace: %s", bea.namespace)

	// Start rule engine in a separate goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(bea.config.RuleEngine.RuleCheckInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := bea.executeRules(); err != nil {
					bea.logger.Errorf("Error during rule execution: %v", err)
				}
			case <-bea.stopCh:
				bea.logger.Info("BasicRuleEngineAgent stopped")
				return
			case <-ctx.Done():
				bea.logger.Info("Context cancelled, stopping BasicRuleEngineAgent")
				close(bea.stopCh)
				return
			}
		}
	}()

	return nil
}

// Stop stops the basic rule engine agent
func (bea *BasicRuleEngineAgent) Stop() {
	close(bea.stopCh)
}

// executeRules executes the defined rules
func (bea *BasicRuleEngineAgent) executeRules() error {
	bea.logger.Debugf("Executing rules for namespace: %s", bea.namespace)

	// TODO: Implement actual rule execution logic
	bea.logger.Debug("Rule execution completed")

	return nil
}