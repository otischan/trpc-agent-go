package monitors

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent/common"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// MemoryMonitorItem 监控内存使用情况的实现
type MemoryMonitorItem struct{}

func (m *MemoryMonitorItem) GetName() string {
	return "memory-monitor"
}

func (m *MemoryMonitorItem) GetConfigKey() string {
	return "monitoring.enable_memory_monitoring"
}

func (m *MemoryMonitorItem) IsEnabled(config *config.Config) bool {
	return config.Monitoring.MemoryMonitoring.Enabled &&
		config.Monitoring.MemoryMonitoring.BasicCollection.Enabled
}

func (m *MemoryMonitorItem) Monitor(ctx context.Context, params common.MonitorParams) error {
	// Initialize memory collector if memory monitoring is enabled
	if params.Config.Monitoring.MemoryMonitoring.Enabled &&
		params.Config.Monitoring.MemoryMonitoring.BasicCollection.Enabled {
		retentionDays := params.Config.Monitoring.MemoryMonitoring.BasicCollection.RetentionDays
		if retentionDays <= 0 {
			retentionDays = 30 // default to 30 days
		}

		memoryCollector := basic.NewMemoryCollector(
			params.Clientset,
			params.MetricsClient,
			params.Logger,
			retentionDays,
		)

		if err := memoryCollector.CollectMemoryMetrics(params.Namespace); err != nil {
			return fmt.Errorf("error collecting memory metrics: %w", err)
		}
	}

	return nil
}
