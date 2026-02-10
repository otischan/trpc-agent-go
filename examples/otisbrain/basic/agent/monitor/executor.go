package monitor

import (
	"context"
	"fmt"
	"sync"

	"trpc.group/trpc-go/trpc-agent-go/examples/otisbrain/basic/agent/common"
)

// MonitorExecutor 负责执行注册的监控项
type MonitorExecutor struct {
	registry *MonitorRegistry
	params   common.MonitorParams
}

func NewMonitorExecutor(registry *MonitorRegistry, params common.MonitorParams) *MonitorExecutor {
	return &MonitorExecutor{
		registry: registry,
		params:   params,
	}
}

func (me *MonitorExecutor) Execute(ctx context.Context) error {
	items := me.registry.GetAll()

	var wg sync.WaitGroup
	var errs []error
	var mu sync.Mutex

	for _, item := range items {
		if !item.IsEnabled(me.params.Config) {
			me.params.Logger.Debugf("Skipping disabled monitor item: %s", item.GetName())
			continue
		}

		wg.Add(1)
		go func(monitorItem MonitorItem) {
			defer wg.Done()

			if err := monitorItem.Monitor(ctx, me.params); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("error in monitor item %s: %w", monitorItem.GetName(), err))
				me.params.Logger.Errorf("Error in monitor item %s: %v", monitorItem.GetName(), err)
				mu.Unlock()
			}
		}(item)
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("monitor execution errors: %v", errs)
	}

	return nil
}
