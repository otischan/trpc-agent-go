# OtisBrain - K8S集群智能运维代理

## 项目概述

OtisBrain 是一个基于 tRPC-Agent-Go 框架构建的 Kubernetes 集群智能监控和运维系统。该系统分为两个层次：

1. **基础监控与运维层**：在无须连接大语言模型的情况下，提供核心的K8S资源监控、事件监听和基于预定义规则的自动化运维能力
2. **智能增强层**：在能够连接大语言模型时，提供高级的异常检测、根因分析和智能决策能力

系统能够监控指定命名空间的资源状态，并根据预设规则和AI推理执行智能化运维操作。

## 架构设计

### 整体架构

```
otisbrain/
├── README.md                           # 本文件 - 项目说明文档
├── main.go                             # 主入口程序
├── config/                             # 配置管理
│   ├── config.go                       # 配置结构体定义
│   └── config.yaml                     # 默认配置文件
├── basic/                              # 基础监控与运维层
│   ├── monitor/                        # 基础监控模块
│   │   ├── collector.go                # 指标收集器
│   │   ├── events.go                   # 事件监听器
│   │   ├── health_checker.go           # 健康检查器
│   │   └── metrics.go                  # 指标定义
│   ├── agent/                          # 基础代理
│   │   ├── basic_monitor_agent.go      # 基础K8S监控代理
│   │   ├── basic_alert_agent.go        # 基础告警处理代理
│   │   └── basic_remediation_agent.go  # 基础自动修复代理
│   ├── rules/                          # 基础规则引擎
│   │   ├── basic_rule_engine.go        # 基础规则引擎核心
│   │   ├── alert_rules.go              # 基础告警规则定义
│   │   └── remediation_rules.go        # 基础修复规则定义
│   └── tools/                          # 基础工具集
│       ├── kubectl_tool.go             # Kubectl命令工具
│       ├── restart_tool.go             # 重启工具
│       └── rollback_tool.go            # 回滚工具
├── ai/                                 # 智能增强层
│   ├── agent/                          # AI代理
│   │   ├── ai_monitor_agent.go         # AI增强监控代理
│   │   ├── ai_alert_agent.go           # AI增强告警处理代理
│   │   ├── ai_remediation_agent.go     # AI增强自动修复代理
│   │   └── decision_agent.go           # AI决策制定代理
│   ├── rules/                          # AI增强规则引擎
│   │   ├── ai_rule_engine.go           # AI增强规则引擎
│   │   └── adaptive_rules.go           # 自适应规则定义
│   └── tools/                          # AI增强工具集
│       ├── ai_analysis_tool.go         # AI分析工具
│       └── predictive_tool.go          # 预测性维护工具
├── shared/                             # 共享组件
│   ├── k8sclient/                      # K8S客户端封装
│   │   ├── client.go                   # K8S客户端初始化
│   │   ├── pod_operations.go           # Pod操作接口
│   │   ├── deployment_operations.go    # Deployment操作接口
│   │   ├── service_operations.go       # Service操作接口
│   │   └── namespace_operations.go     # Namespace操作接口
│   ├── storage/                        # 存储层
│   │   ├── event_log.go                # 事件日志存储
│   │   └── state_store.go              # 状态存储
│   └── notification/                   # 通知模块
│       ├── notifier.go                 # 通知发送器
│       └── templates/                  # 通知模板
├── logs/                               # 日志目录（运行时创建）
│   ├── basic/                          # 基础层日志
│   │   ├── monitor.log                 # 基础监控日志
│   │   ├── alert.log                   # 基础告警日志
│   │   └── remediation.log             # 基础修复日志
│   ├── ai/                             # 智能增强层日志
│   │   ├── ai_monitor.log              # AI监控分析日志
│   │   ├── ai_decision.log             # AI决策日志
│   │   └── ai_analysis.log             # AI分析日志
│   └── critical_events/                # 关键信息日志
│       ├── events.log                  # 重要事件日志
│       ├── pod_status_changes.log      # Pod状态变更日志
│       └── anomaly_detections.log      # 异常检测日志
└── testdata/                           # 测试数据
    └── sample_manifests/               # 示例清单文件
```

### 功能分层

#### 基础监控与运维层 (无需LLM)

此层提供核心的K8S监控和运维功能，即使在无法连接大语言模型的情况下也能正常运行：

- **基础监控代理**：监控Pod状态、Deployment副本数、Service可用性等
- **事件监听**：监听K8S API的事件流，捕获异常事件
- **预定义规则引擎**：基于硬编码规则进行异常检测
- **基础修复工具**：执行预定义的修复操作（如重启Pod、调整副本数）

#### 智能增强层 (需要LLM连接)

此层在基础层之上提供高级功能，需要连接大语言模型：

- **智能监控代理**：使用AI进行异常模式识别和预测性分析
- **智能告警处理**：AI辅助的告警分类和优先级排序
- **智能决策代理**：复杂问题的根因分析和解决方案推荐
- **自适应规则引擎**：AI驱动的动态规则调整

### 日志管理与运行输出

程序运行时会自动创建 `logs` 目录，用于存放不同层级的日志和关键信息：

#### 日志目录结构

```
logs/
├── basic/                          # 基础层日志
│   ├── monitor.log                 # 基础监控日志
│   ├── alert.log                   # 基础告警日志
│   └── remediation.log             # 基础修复日志
├── ai/                             # 智能增强层日志
│   ├── ai_monitor.log              # AI监控分析日志
│   ├── ai_decision.log             # AI决策日志
│   └── ai_analysis.log             # AI分析日志
└── critical_events/                # 关键信息日志
    ├── events.log                  # 重要事件日志
    ├── pod_status_changes.log      # Pod状态变更日志
    └── anomaly_detections.log      # 异常检测日志
```

#### 日志处理机制

1. **基础层日志处理**：
   - 程序启动时自动创建 `logs/basic/` 目录
   - 基础监控代理记录Pod状态、资源使用情况等信息
   - 事件监听器捕获并记录关键事件，包括对象名称、时间戳、事件类型等
   - 告警和修复操作的日志也记录在此目录

2. **关键信息提取**：
   - 基础层从K8S API采集的关键信息（如事件、Pod状态变化）会被整理并存储在 `logs/critical_events/` 目录
   - 每条记录包含：对象类型、对象名称、命名空间、时间戳、事件详情等关键字段
   - 这些信息格式化存储，便于AI层读取和分析

3. **AI增强层日志处理**：
   - AI层不会干预基础层的正常工作
   - AI代理在单独的goroutine中运行，定期从 `logs/critical_events/` 目录读取基础层生成的关键信息
   - AI分析结果记录在 `logs/ai/` 目录中，包括分析过程、决策依据和建议操作
   - AI层的日志与基础层完全隔离，确保基础功能不受AI处理的影响

4. **日志轮转与清理**：
   - 系统会自动管理日志文件大小，当日志文件达到一定大小时会进行轮转
   - 旧的日志文件会被压缩归档，保留最近的活动日志以节省磁盘空间

### 设计理念

1. **分层架构**：基础功能与智能增强分离，确保核心功能始终可用
2. **多代理协作**：采用多个专业代理协同工作的模式，每个代理负责特定任务
3. **可观察性**：全面的指标、日志和追踪能力
4. **渐进式智能**：基础功能不依赖AI，AI作为增强层提供智能分析能力
5. **弹性设计**：核心监控功能独立运行，不受大模型连接状态影响
6. **安全操作**：最小权限原则和操作审计
7. **可扩展性**：模块化设计便于功能扩展
8. **非侵入式AI增强**：AI层独立运行，不对基础层工作流程造成干扰

### 核心代理设计

#### 1. K8S监控代理 (k8s_monitor_agent.go)
- 实时监控指定命名空间中的资源状态
- 收集Pod、Deployment、Service等资源的指标
- 检测异常状态并触发告警
- 独立运行，不依赖大模型连接
- 基础功能：基于预设规则检测常见问题（如Pod崩溃、资源不足等）
- 增强功能：当大模型可用时，进行智能异常检测和根因分析

#### 2. 告警处理代理 (alert_agent.go)
- 接收来自监控代理的告警信息
- 根据严重程度和类型进行分类
- 执行初步分析并决定后续处理流程

#### 3. 自动修复代理 (remediation_agent.go)
- 根据预定义规则自动执行修复操作
- 包括重启Pod、回滚部署、调整资源配置等
- 记录所有修复操作供审计

#### 4. 决策制定代理 (decision_agent.go)
- 对复杂问题进行深度分析（可选功能，需要大模型支持）
- 结合历史数据和上下文信息做出决策
- 当大模型不可用时，使用预定义规则进行基本决策
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
- **K8S客户端**: official Kubernetes client-go (核心依赖)
- **监控**: Prometheus metrics
- **AI集成**: OpenAI-compatible API (可选增强功能)
- **通知**: Slack, Email, Webhook等
- **存储**: 可选本地文件、Redis或数据库
- **规则引擎**: 基于预定义规则和AI增强规则
- **日志**: Zap logger with structured logging and log rotation

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

### 基础层应用场景

1. **基础生产环境监控**: 实时监控关键应用的健康状况（无需LLM）
2. **基础故障恢复**: 在检测到问题时自动执行预定义的修复操作（无需LLM）
3. **基础合规性检查**: 定期检查配置是否符合安全标准（无需LLM）

### 智能层应用场景

1. **智能生产环境监控**: 使用AI进行异常模式识别和预测性分析
2. **智能故障恢复**: AI辅助的根因分析和智能修复策略
3. **智能容量规划**: 基于AI的历史数据分析和资源需求预测
4. **智能成本优化**: AI驱动的资源优化策略
5. **智能合规性检查**: AI辅助的安全策略分析和建议

## 部署方式

此项目构建为单个二进制程序，可通过以下方式运行：
- 本地运行：直接在本地机器上运行二进制文件
- 集群内部运行：在K8S集群内部作为Pod运行

## 分层架构优势

1. **高可用性**: 基础监控功能不依赖外部AI服务，确保核心功能始终可用
2. **灵活性**: 用户可根据环境和需求选择启用基础层或完整功能
3. **成本效益**: 在无AI连接时仍可提供核心监控功能，降低运营成本
4. **安全性**: 减少对外部AI服务的依赖，降低安全风险
5. **渐进式部署**: 可先部署基础功能，后续再启用AI增强功能

## 未来发展方向

- 集成更多AI模型以提高决策准确性
- 支持多集群管理
- 更丰富的可视化界面
- 机器学习驱动的异常检测
- 更智能的容量规划算法
```

## 快速开始

### 配置文件

程序的所有参数现在都通过配置文件进行管理。配置文件位于 `config/config.yaml`，包含以下主要配置项：

```yaml
log_level: info                    # 日志级别 (debug, info, warn, error)
metrics_port: 8080                 # 暴露Prometheus指标的端口
namespace: default                 # 监控的目标命名空间
kubeconfig: ""                     # K8S集群认证配置文件路径

llm:                               # 大语言模型配置
  model: gpt-4o-mini               # 使用的AI模型
  api_key: ""                      # AI模型API密钥
  base_url: ""                     # AI模型API基础URL
  enabled: false                   # 是否启用AI功能

rules:                             # 规则引擎配置
  alert_rules_file: "./rules/alert_rules.yaml"       # 告警规则文件路径
  remediation_rules_file: "./rules/remediation_rules.yaml" # 修复规则文件路径

basic:                             # 基础监控配置
  interval_seconds: 30             # 监控间隔秒数
  max_retries: 3                   # 最大重试次数
  dry_run: false                   # 是否演练模式（不执行实际操作）

monitoring:                        # 监控代理配置
  enable_monitor: true             # 是否启用监控代理
  enable_alert: true               # 是否启用告警代理
  enable_remediation: false        # 是否启用修复代理
  enable_decision: false           # 是否启用决策代理
  namespace: default               # 监控的命名空间
  kubeconfig: ""                   # K8S集群认证配置文件路径
  metrics_port: 8080               # 暴露指标的端口
  aggregation_interval_minutes: 10 # 日志聚合间隔（分钟），即每10分钟对关键事件进行一次汇总
```

### 配置说明

- **log_level**: 控制日志输出级别，支持 debug, info, warn, error
- **metrics_port**: Prometheus指标暴露端口
- **namespace**: 要监控的Kubernetes命名空间
- **kubeconfig**: K8S集群认证配置文件路径
- **llm**: 大语言模型相关配置，包括模型类型、API密钥等
- **rules**: 规则引擎配置，指定告警和修复规则文件位置
- **basic**: 基础监控配置，包括监控间隔、重试次数等
- **monitoring**: 监控代理配置，控制各个代理的启停状态
- **aggregation_interval_minutes**: 关键事件聚合间隔，控制每多少分钟对basic路径下的日志进行整理、去重和汇总

### 示例配置

```yaml
# 生产环境示例配置
log_level: info
metrics_port: 8080
namespace: production

llm:
  model: gpt-4o
  api_key: "your-api-key-here"
  base_url: "https://api.openai.com/v1"
  enabled: true

basic:
  interval_seconds: 15
  max_retries: 5
  dry_run: false

monitoring:
  enable_monitor: true
  enable_alert: true
  enable_remediation: true
  enable_decision: true
  namespace: production
  aggregation_interval_minutes: 10  # 每10分钟聚合一次关键事件
```

```yaml
# 测试环境示例配置
log_level: debug
namespace: test

basic:
  interval_seconds: 30
  max_retries: 3
  dry_run: true  # 演练模式，不执行实际操作

monitoring:
  enable_monitor: true
  enable_alert: true
  enable_remediation: false  # 测试环境禁用自动修复
  enable_decision: false
  aggregation_interval_minutes: 5  # 测试环境缩短聚合间隔
```

### 运行程序

```bash
# 使用默认配置文件运行
./otisbrain

# 指定自定义配置文件
./otisbrain -config /path/to/custom/config.yaml
```

### 代理能力说明

1. **监控代理 (monitoring.enable_monitor)**:
   - 实时监控资源状态
   - 收集性能指标
   - 检测异常状态
   - 此代理默认启用，是其他代理的基础
   - 独立运行，不依赖大模型连接
   - 基础功能：基于预设规则检测常见问题（如Pod崩溃、资源不足等）
   - 增强功能：当大模型可用时，进行智能异常检测和根因分析

2. **告警代理 (monitoring.enable_alert)**:
   - 处理监控代理产生的告警
   - 分类和优先级排序
   - 发送通知
   - 此代理依赖监控代理，默认启用

3. **修复代理 (monitoring.enable_remediation)**:
   - 自动执行预定义的修复操作
   - 如重启故障Pod、调整资源配额等
   - 高风险操作，需要谨慎启用
   - 建议先在非生产环境测试

4. **决策代理 (monitoring.enable_decision)**:
   - 使用AI模型进行复杂问题分析（当AI模型可用时）
   - 当AI模型不可用时，使用预定义规则进行基本决策
   - 制定高级决策策略
   - 需要配置AI模型访问（可选）
   - 适合处理复杂或未知问题

### 关键事件聚合机制

系统实现了关键事件的定时聚合机制：

- **聚合间隔**: 通过 `monitoring.aggregation_interval_minutes` 配置，默认为10分钟
- **聚合内容**: 对 `logs/basic` 路径下的日志进行整理、去重
- **输出位置**: 聚合后的异常数据汇总输出到 `logs/critical_record` 路径
- **目的**: 为AI增强能力的agent提供简洁的异常数据，便于读取和分析
- **格式**: 输出简短的异常数据汇总，包含关键事件、Pod状态等信息