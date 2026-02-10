package monitor

import (
	"sync"
)

// MonitorRegistry 负责管理所有监控项
type MonitorRegistry struct {
	items map[string]MonitorItem
	mutex sync.RWMutex
}

func NewMonitorRegistry() *MonitorRegistry {
	return &MonitorRegistry{
		items: make(map[string]MonitorItem),
	}
}

func (mr *MonitorRegistry) Register(item MonitorItem) {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()

	mr.items[item.GetName()] = item
}

func (mr *MonitorRegistry) Get(name string) (MonitorItem, bool) {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	item, exists := mr.items[name]
	return item, exists
}

func (mr *MonitorRegistry) GetAll() []MonitorItem {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	var items []MonitorItem
	for _, item := range mr.items {
		items = append(items, item)
	}
	return items
}
