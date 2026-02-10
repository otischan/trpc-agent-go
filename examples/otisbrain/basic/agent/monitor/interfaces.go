package monitor

import (
	"context"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent/common"
	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/config"
)

// MonitorItem 接口定义监控项的基本行为
type MonitorItem interface {
	// GetName 返回监控项名称
	GetName() string

	// IsEnabled 根据配置判断是否启用此监控项
	IsEnabled(config *config.Config) bool

	// Monitor 执行具体的监控逻辑
	Monitor(ctx context.Context, params common.MonitorParams) error

	// GetConfigKey 返回配置中对应的键名
	GetConfigKey() string
}
