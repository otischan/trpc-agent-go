# OtisBrain 监控功能

## 监控范围

### 资源监控
- CPU/内存使用率
- Pod就绪/运行状态
- Deployment副本数
- Service可用性
- 存储卷状态

### 内存监控增强
- **基础内存采集**: 持续采集内存使用量，用于OOM后分析
- **OOM事件检测**: 监控K8S事件中的OOMKilled事件
- **OOM后内存分析**: 当发生OOM时，分析该Pod的历史内存使用模式，计算内存使用的整体趋势斜率
- **内存使用轮廓**: 提供OOM前的内存使用斜率信息，帮助判断是否存在内存泄露趋势

### 应用健康
- 应用响应时间
- 错误率
- 请求成功率
- 自定义业务指标

### 事件监控
- Pod调度失败
- 节点不可用
- 资源配额超限
- 安全事件

## 自动化运维能力

### 自愈能力
- 自动重启故障Pod
- 自动替换不健康节点上的Pod
- 自动扩容应对流量高峰

### 自动优化
- 基于负载自动调整资源限制
- 清理过期或无用资源
- 优化调度策略

### 预测性维护
- 基于趋势预测潜在问题
- 提前执行预防性措施
- 优化资源分配

## 监控系统架构设计

### 核心设计理念

OtisBrain 监控系统采用插件化架构设计，旨在实现以下目标：

1. **易于扩展**: 新增监控项无需修改核心代码
2. **松耦合**: 各监控项相互独立，互不影响
3. **可配置**: 每个监控项可独立启用/禁用
4. **统一管理**: 单一入口管理所有监控项
5. **可测试**: 每个监控项可独立测试

### 核心组件

#### MonitorItem 接口
所有监控项必须实现此接口：

```go
type MonitorItem interface {
    // GetName 返回监控项名称
    GetName() string
    
    // IsEnabled 根据配置判断是否启用此监控项
    IsEnabled(config *config.Config) bool
    
    // Monitor 执行具体的监控逻辑
    Monitor(ctx context.Context, params MonitorParams) error
    
    // GetConfigKey 返回配置中对应的键名
    GetConfigKey() string
}
```

#### MonitorParams 结构
传递给监控项的通用参数：

```go
type MonitorParams struct {
    Clientset     *kubernetes.Clientset
    MetricsClient *metricsv.Clientset
    Namespace     string
    Config        *config.Config
    Logger        *logrus.Logger
}
```

#### MonitorRegistry 注册中心
管理所有监控项的注册和获取：

```go
type MonitorRegistry struct {
    items map[string]MonitorItem
}
```

#### MonitorExecutor 执行器
负责执行所有启用的监控项：

```go
type MonitorExecutor struct {
    registry *MonitorRegistry
    params   MonitorParams
}
```

#### BaseMonitorAgent 统一监控代理
支持单命名空间和多命名空间监控的统一实现：

```go
type BaseMonitorAgent struct {
    clientset     *kubernetes.Clientset
    metricsClient *metricsv.Clientset
    namespaces    []string           // 支持单个或多个命名空间
    config        *config.Config
    logger        *logrus.Logger
    stopCh        chan struct{}
    registry      *MonitorRegistry
    interval      time.Duration
}
```

### 目录结构

```
basic/
├── agent/
│   ├── monitor/
│   │   ├── registry.go          # 监控项注册中心
│   │   ├── executor.go          # 监控执行器
│   │   ├── interfaces.go        # MonitorItem 接口定义
│   │   ├── base_agent.go        # 统一的基础监控代理
│   │   └── factory.go           # 监控代理工厂
│   ├── monitors/                # 具体监控项实现
│   │   ├── pod_monitor_item.go  # Pod 监控项
│   │   ├── deployment_monitor_item.go  # Deployment 监控项
│   │   ├── service_monitor_item.go     # Service 监控项
│   │   ├── event_monitor_item.go       # 事件监控项
│   │   ├── memory_monitor_item.go      # 内存监控项
│   │   └── base_monitor_item.go        # 基础监控项抽象
│   └── utils/                   # 监控工具函数
│       ├── logger.go            # 日志工具
│       └── helpers.go           # 辅助函数
```

### 统一监控代理设计

BaseMonitorAgent 统一了单命名空间和多命名空间的监控逻辑：

1. **参数验证**: 构造函数严格验证命名空间参数，要求至少指定一个命名空间
2. **并发执行**: 为每个命名空间启动独立的监控协程
3. **资源管理**: 统一的生命周期管理和资源清理
4. **日志隔离**: 每个命名空间使用独立的日志记录器

### 添加新监控项的步骤

1. 在 `monitors/` 目录下创建新的监控项文件
2. 实现 `MonitorItem` 接口
3. 在 `monitor/registry.go` 或工厂函数中注册该监控项
4. 在配置文件中添加相应的启用选项

### 命名空间管理

监控系统严格要求必须显式指定命名空间，不允许隐式默认行为：

- **参数验证**: 构造函数验证至少提供一个命名空间
- **空值检查**: 拒绝空命名空间名称
- **特殊值处理**: 不允许使用 "all" 作为普通命名空间名称
- **错误处理**: 提供明确的错误信息帮助用户理解问题

## 关键事件聚合机制

系统实现了关键事件的定时聚合机制：

- **聚合间隔**: 通过 `monitoring.aggregation_interval_minutes` 配置，默认为10分钟
- **聚合内容**: 对 `logs/basic` 路径下的日志进行整理、去重
- **输出位置**: 聚合后的异常数据汇总输出到 `logs/critical_record` 路径
- **目的**: 为AI增强能力的agent提供简洁的异常数据，便于读取和分析
- **格式**: 输出简短的异常数据汇总，包含关键事件、Pod状态等信息

## OOM事件内存分析机制

当检测到OOMKilled事件时，系统会自动执行内存使用分析：

- **触发条件**: 检测到K8S事件中的OOMKilled事件
- **分析内容**: 从历史数据中提取该Pod的内存使用记录，计算内存使用的整体趋势斜率
- **输出位置**: 分析结果记录到 `logs/critical_record/memory_analysis_*.txt`
- **目的**: 提供OOM前的内存使用模式，帮助判断是否存在内存泄露趋势
- **输出示例**:
  ```
  # OOM事件内存分析报告
  Pod: my-app-7d5b8c9c4-xl2v9
  Namespace: default
  OOM时间: 2024-01-15T10:30:00Z

  # 内存使用趋势分析
  内存使用斜率: +2.5 MB/小时
  分析时间范围: OOM前30天
  数据点数量: 1440 (每小时1个点)

  # 分析结论
  - 内存在过去30天内总体呈增长趋势
  - 平均每小时增长2.5MB
  ```

## 使用场景

### 监控服务应用场景

1. **持续生产环境监控**: 实时监控关键应用的健康状况，后台持续运行
2. **自动故障恢复**: 在检测到问题时自动执行预定义的修复操作
3. **自动合规性检查**: 定期检查配置是否符合安全标准
4. **关键信息周期汇总**: 按配置指定间隔周期汇总关键异常信息