package common

import (
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// MonitorParams 包含监控所需的通用参数
type MonitorParams struct {
	Clientset     *kubernetes.Clientset
	MetricsClient *metricsv.Clientset
	Namespace     string
	Config        *config.Config
	Logger        *logrus.Logger
}
