# OtisBrain - K8S集群智能运维代理

## 项目概述

OtisBrain 是一个基于 tRPC-Agent-Go 框架构建的 Kubernetes 集群智能监控和运维系统。该系统能够监控指定命名空间的资源状态，并根据预设规则和AI推理执行智能化运维操作。

## 架构设计

### 核心组件

```
otisbrain/
├── README.md                    # 本文件 - 项目说明文档
├── main.go                      # 主入口程序
├── config/                      # 配置管理
│   ├── config.go                # 配置结构体定义
│   └── config.yaml              # 默认配置文件
├── agent/                       # 智能代理核心逻辑
│   ├── k8s_monitor_agent.go     # K8S监控代理
│   ├── alert_agent.go           # 告警处理代理
│   ├── remediation_agent.go     # 自动修复代理
│   └── decision_agent.go        # 决策制定代理
├── monitor/                     # 监控模块
│   ├── collector.go             # 指标收集器
│   ├── events.go                # 事件监听器
│   ├── health_checker.go        # 健康检查器
│   └── metrics.go               # 指标定义
├── k8sclient/                   # K8S客户端封装
│   ├── client.go                # K8S客户端初始化
│   ├── pod_operations.go        # Pod操作接口
│   ├── deployment_operations.go # Deployment操作接口
│   ├── service_operations.go    # Service操作接口
│   └── namespace_operations.go  # Namespace操作接口
├── tools/                       # 工具集
│   ├── kubectl_tool.go          # Kubectl命令工具
│   ├── alert_tool.go            # 告警工具
│   ├── scaling_tool.go          # 自动扩缩容工具
│   ├── restart_tool.go          # 重启工具
│   └── rollback_tool.go         # 回滚工具
├── rules/                       # 规则引擎
│   ├── rule_engine.go           # 规则引擎核心
│   ├── alert_rules.go           # 告警规则定义
│   ├── remediation_rules.go     # 修复规则定义
│   └── escalation_rules.go      # 升级规则定义
├── storage/                     # 存储层
│   ├── event_log.go             # 事件日志存储
│   └── state_store.go           # 状态存储
├── notification/                # 通知模块
│   ├── notifier.go              # 通知发送器
│   └── templates/               # 通知模板
└── testdata/                    # 测试数据
    └── sample_manifests/        # 示例清单文件
```

### 设计理念

1. **多代理协作**：采用多个专业代理协同工作的模式，每个代理负责特定任务
2. **可观察性**：全面的指标、日志和追踪能力
3. **自动化决策**：基于AI推理的智能决策机制
4. **安全操作**：最小权限原则和操作审计
5. **可扩展性**：模块化设计便于功能扩展

### 核心代理设计

#### 1. K8S监控代理 (k8s_monitor_agent.go)
- 实时监控指定命名空间中的资源状态
- 收集Pod、Deployment、Service等资源的指标
- 检测异常状态并触发告警

#### 2. 告警处理代理 (alert_agent.go)
- 接收来自监控代理的告警信息
- 根据严重程度和类型进行分类
- 执行初步分析并决定后续处理流程

#### 3. 自动修复代理 (remediation_agent.go)
- 根据预定义规则自动执行修复操作
- 包括重启Pod、回滚部署、调整资源配置等
- 记录所有修复操作供审计

#### 4. 决策制定代理 (decision_agent.go)
- 对复杂问题进行深度分析
- 结合历史数据和上下文信息做出决策
- 在必要时请求人工干预

### 监控范围

#### 资源监控
- CPU/内存使用率
- Pod就绪/运行状态
- Deployment副本数
- Service可用性
- 存储卷状态

#### 应用健康
- 应用响应时间
- 错误率
- 请求成功率
- 自定义业务指标

#### 事件监控
- Pod调度失败
- 节点不可用
- 资源配额超限
- 安全事件

### 自动化运维能力

#### 自愈能力
- 自动重启故障Pod
- 自动替换不健康节点上的Pod
- 自动扩容应对流量高峰

#### 自动优化
- 基于负载自动调整资源限制
- 清理过期或无用资源
- 优化调度策略

#### 预测性维护
- 基于趋势预测潜在问题
- 提前执行预防性措施
- 优化资源分配

### 技术栈

- **语言**: Go
- **框架**: tRPC-Agent-Go
- **K8S客户端**: official Kubernetes client-go
- **监控**: Prometheus metrics
- **通知**: Slack, Email, Webhook等
- **存储**: 可选本地文件、Redis或数据库

### 安全考虑

- 使用K8S RBAC进行权限控制
- 所有操作记录审计日志
- 支持操作确认机制（对于高风险操作）
- 加密敏感配置信息

### 扩展性

- 插件化工具系统
- 可配置的规则引擎
- 支持自定义监控指标
- 可扩展的通知渠道

## 使用场景

1. **生产环境监控**: 实时监控关键应用的健康状况
2. **自动故障恢复**: 在检测到问题时自动执行修复操作
3. **容量规划**: 基于历史数据预测资源需求
4. **成本优化**: 自动调整资源以优化成本
5. **合规性检查**: 定期检查配置是否符合安全标准

## 部署方式

此项目构建为单个二进制程序，可通过以下方式运行：
- 本地运行：直接在本地机器上运行二进制文件
- 集群内部运行：在K8S集群内部作为Pod运行

## 未来发展方向

- 集成更多AI模型以提高决策准确性
- 支持多集群管理
- 更丰富的可视化界面
- 机器学习驱动的异常检测
- 更智能的容量规划算法
```

## 快速开始

### 命令行参数

程序支持以下命令行参数来控制不同代理的能力：

```
Usage: otisbrain [options]

Options:
  -kubeconfig string
        Path to the kubeconfig file (default: ~/.kube/config)
  -namespace string
        Target namespace to monitor (default: "default")
  -enable-monitor
        Enable K8S monitoring agent (default: true)
  -enable-alert
        Enable alert handling agent (default: true)
  -enable-remediation
        Enable auto-remediation agent (default: false)
  -enable-decision
        Enable decision-making agent (default: false)
  -metrics-port int
        Port to expose Prometheus metrics (default: 8080)
  -config string
        Path to the configuration file (default: "./config/config.yaml")
  -log-level string
        Log level (debug, info, warn, error) (default: "info")
  -model string
        LLM model to use for AI agents (default: "gpt-4o-mini")
  -max-retry int
        Maximum retry attempts for failed operations (default: 3)
  -dry-run
        Run in dry-run mode without performing actual operations (default: false)
```

### 参数说明

- **-kubeconfig**: 指定K8S集群的认证配置文件路径
- **-namespace**: 指定要监控的目标命名空间
- **-enable-monitor**: 启用/禁用K8S监控代理
- **-enable-alert**: 启用/禁用告警处理代理
- **-enable-remediation**: 启用/禁用自动修复代理（高风险操作，建议谨慎启用）
- **-enable-decision**: 启用/禁用决策制定代理（需要AI模型支持）
- **-metrics-port**: 指定暴露Prometheus指标的端口
- **-config**: 指定配置文件路径
- **-log-level**: 设置日志输出级别
- **-model**: 指定用于AI代理的大语言模型
- **-max-retry**: 设置失败操作的最大重试次数
- **-dry-run**: 以演练模式运行，不执行实际的变更操作

### 示例用法

```bash
# 基础监控模式（仅启用监控和告警）
./otisbrain -namespace production -enable-remediation=false -enable-decision=false

# 全功能模式（启用所有代理）
./otisbrain -namespace production -enable-monitor -enable-alert -enable-remediation -enable-decision -model gpt-4o

# 演练模式（测试配置，不执行实际操作）
./otisbrain -namespace staging -dry-run -log-level debug

# 自定义配置
./otisbrain -kubeconfig /custom/path/kubeconfig -namespace myapp -config /etc/otisbrain/config.yaml
```

### 代理能力说明

1. **监控代理 (-enable-monitor)**:
   - 实时监控资源状态
   - 收集性能指标
   - 检测异常状态
   - 此代理默认启用，是其他代理的基础

2. **告警代理 (-enable-alert)**:
   - 处理监控代理产生的告警
   - 分类和优先级排序
   - 发送通知
   - 此代理依赖监控代理，默认启用

3. **修复代理 (-enable-remediation)**:
   - 自动执行预定义的修复操作
   - 如重启故障Pod、调整资源配额等
   - 高风险操作，需要谨慎启用
   - 建议先在非生产环境测试

4. **决策代理 (-enable-decision)**:
   - 使用AI模型进行复杂问题分析
   - 制定高级决策策略
   - 需要配置AI模型访问
   - 适合处理复杂或未知问题