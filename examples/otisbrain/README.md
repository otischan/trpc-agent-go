# OtisBrain - K8S集群智能运维代理

## 项目概述

OtisBrain 是一个基于 tRPC-Agent-Go 框架构建的 Kubernetes 集群智能监控和运维系统。该系统采用微服务架构，拆分为五个独立的服务：

1. **监控服务 (monitoring-service)**：持续运行的监控和告警任务，在后台默默守护集群健康
2. **聊天服务 (chat-service)**：提供实时聊天界面，让用户可以随时与系统交互，获取集群信息和执行运维操作
3. **规则引擎服务 (rule-engine-service)**：基于预设规则分析异常并执行自动修复策略
4. **AI决策服务 (ai-decision-service)**：与LLM交互，基于监控数据生成决策
5. **MCP操作服务 (mcp-service)**：统化K8S操作接口

系统能够监控指定命名空间的资源状态，并根据预设规则和AI推理执行智能化运维操作。

## 架构设计

### 整体架构

```
otisbrain/
├── README.md                           # 本文件 - 项目说明文档
├── go.mod                              # Go模块定义
├── go.sum                              # Go依赖校验
├── cmd/                                # 命令行工具目录
│   ├── monitoring-service/             # 监控服务
│   │   └── main.go                     # 监控服务主入口
│   ├── chat-service/                   # 聊天服务
│   │   └── main.go                     # 聊天服务主入口
│   ├── rule-engine-service/            # 规则引擎服务
│   │   └── main.go                     # 规则引擎服务主入口
│   ├── ai-decision-service/            # AI决策服务
│   │   └── main.go                     # AI决策服务主入口
│   └── mcp-service/                    # MCP操作服务
│       └── main.go                     # MCP操作服务主入口
├── basic/                              # 基础监控与运维层
│   ├── monitor/                        # 基础监控模块
│   │   ├── collector.go                # 指标收集器
│   │   ├── events.go                   # 事件监听器
│   │   ├── health_checker.go           # 偵康检查器
│   │   └── metrics.go                  # 指标定义
│   ├── agent/                          # 基础代理
│   │   ├── basic_monitor_agent.go      # 基础K8S监控代理
│   │   ├── basic_alert_agent.go        # 基础告警处理代理
│   │   ├── basic_remediation_agent.go  # 基础自动修复代理
│   │   └── basic_rule_engine_agent.go  # 基础规则引擎代理
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
│   │   ├── decision_agent.go           # AI决策制定代理
│   │   └── ai_decision_agent.go        # AI决策服务代理
│   ├── rules/                          # AI增强规则引擎
│   │   ├── ai_rule_engine.go           # AI增强规则引擎
│   │   └── adaptive_rules.go           # 自适应规则定义
│   └── tools/                          # AI增强工具集
│       ├── ai_analysis_tool.go         # AI分析工具
│       └── predictive_tool.go          # 颌测性维护工具
├── ai/chat/                            # AI聊天界面
│   └── chat.go                         # 交互式聊天界面
├── ai/tools/                           # AI技能系统
│   ├── critical_events.go              # 关键事件检索工具
│   └── register.go                     # 技能注册系统
├── shared/                             # 兲享组件
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
├── resources/                          # 资源文件夹
│   ├── mcpserver/                      # MCP服务器配置
│   │   └── config.yaml                 # MCP服务器配置文件
│   └── skills/                         # AI技能定义
│       ├── get_recent_critical_events.yaml  # 关键事件检索技能
│       └── get_cluster_resources.yaml       # 集群资源检索技能
└── testdata/                           # 测试数据
    └── sample_manifests/               # 示例清单文件
```

### 服务拆分说明

#### 监控服务 (monitoring-service)

此服务专注于后台持续运行的K8S监控和运维功能：

- **K8S资源监控代理**：监控Pod状态、Deployment副本数、Service可用性等资源状态
- **K8S事件监控代理**：监听K8S API的事件流，捕获异常事件
- **多命名空间监控**：支持同时监控多个命名空间（通过配置文件中的namespaces字段指定）
- **告警处理**：根据预定义规则生成告警
- **AI增强监控**：使用AI进行异常模式识别和预测性分析
- **AI增强告警**：AI辅助的告警分类和优先级排序
- **关键信息聚合**：按配置指定间隔周期汇总关键异常信息
- **数据持久化**：将监控数据和事件存储到共享存储中
- **规则引擎与修复**：通过规则引擎服务处理自动修复操作

**特点**：
- 不与 LLM 交互，专注于数据收集和处理
- 高可用性：独立运行，不受 LLM 服务状态影响
- 性能优化：专门处理监控任务，无需承担 LLM 通信开销
- 可扩展性：可根据监控负载独立扩展
- **多命名空间支持**：可同时监控多个命名空间，每个命名空间的日志存储在独立的目录中

#### 聊天服务 (chat-service)

此服务专注于用户交互和AI对话功能：

- **交互式聊天**：提供实时对话界面，用户可以查询集群状态
- **工具调用**：通过聊天界面调用监控和查询工具
- **实时反馈**：即时获取集群信息和运维建议
- **AI交互**：与大语言模型进行对话，提供智能分析和决策
- **技能系统**：集成技能系统，允许AI助手执行特定任务

**特点**：
- 专注于用户交互和 LLM 对话
- 响应性：专门处理用户请求，响应更快
- 资源隔离：LLM 调用不会影响监控任务
- 独立部署：可根据用户访问量独立扩展

#### 规则引擎服务 (rule-engine-service)

此服务专注于基于预设规则分析异常并输出策略：

- **规则匹配**：根据预定义规则匹配监控到的异常
- **策略生成**：基于匹配结果生成修复或调整策略
- **策略执行**：通过MCP服务执行生成的策略
- **规则管理**：支持动态加载和更新规则

**特点**：
- 快速响应：基于预设规则快速生成策略
- 确定性：基于明确规则，行为可预测
- 可扩展性：支持多种类型的规则和策略

#### AI决策服务 (ai-decision-service)

此服务专注于与LLM交互，基于监控数据生成决策：

- **数据聚合**：从监控服务收集异常数据
- **Prompt构建**：将监控数据转化为适合LLM的格式
- **AI交互**：与大语言模型交互，获取决策建议
- **决策执行**：通过MCP服务执行AI生成的策略

**特点**：
- 智能决策：利用AI模型进行复杂分析
- 自适应：能够处理未预见的问题
- 上下文感知：考虑历史数据和环境因素

#### MCP操作服务 (mcp-service)

此服务专注于统化K8S操作接口：

- **接口封装**：统一封装K8S API操作
- **安全控制**：实施权限控制和审计
- **操作抽象**：提供高层操作接口
- **状态同步**：确保操作状态的一致性

**特点**：
- 统一接口：为所有服务提供一致的K8S操作接口
- 安全性：集中权限管理和审计
- 可靠性：确保操作的安全和一致性

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
   - **多命名空间支持**：每个命名空间的日志将存储在独立的子目录中 (`logs/basic/{namespace}/`)

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

5. **多命名空间日志聚合**：
   - 日志聚合器会从所有命名空间的日志目录中收集信息
   - 生成的汇总报告会按命名空间分组显示关键事件
   - 便于管理员跨命名空间分析问题模式

### 设计理念

1. **微服务架构**：五个服务分离，确保各司其职且可独立扩展
2. **多代理协作**：采用多个专业代理协同工作的模式，每个代理负责特定任务
3. **可观察性**：全面的指标、日志和追踪能力
4. **渐进式智能**：基础功能不依赖AI，AI作为增强层提供智能分析能力
5. **弹性设计**：核心监控功能独立运行，不受大模型连接状态影响
6. **安全操作**：最小权限原则和操作审计
7. **可扩展性**：模块化设计便于功能扩展
8. **非侵入式AI增强**：AI层独立运行，不对基础层工作流程造成干扰
9. **代码与Skills完全解耦**：确保核心业务逻辑与AI技能系统相互独立，便于独立演进和维护

### 核心模块解耦交互图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           OtisBrain 系统架构                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────┐ │
│  │   聊天服务       │    │   监控服务       │    │      AI增强层          │ │
│  │                 │    │                 │    │                       │ │
│  │  ┌─────────────┐│    │  ┌─────────────┐│    │  ┌───────────────────┐ │ │
│  │  │  Chat UI    ││    │  │  Monitor    ││    │  │  AI Agents        │ │ │
│  │  │             ││    │  │  Agent      ││    │  │                   │ │ │
│  │  │             ││    │  │             ││    │  │  - Decision       │ │ │
│  │  └─────────────┘│    │  │             ││    │  │  - Analysis       │ │ │
│  │        │        │    │  │             ││    │  │  - Predictive     │ │ │
│  │        │        │    │  │             ││    │  │    Maintenance   │ │ │
│  │        ▼        │    │  │             ││    │  │                   │ │ │
│  │  ┌─────────────┐│    │  │             ││    │  └───────────────────┘ │ │
│  │  │ Skills      ││    │  │             ││    │            │           │ │
│  │  │ Registry    ││    │  │             ││    │            │           │ │
│  │  │             ││    │  │             ││    │            ▼           │ │
│  │  │ - Event     ││◄───┼──┤             ││    │  ┌───────────────────┐ │ │
│  │  │   Query     ││    │  │             ││    │  │  AI Tools         │ │ │
│  │  │ - Resource  ││    │  │             ││    │  │                   │ │ │
│  │  │   Query     ││    │  │             ││    │  │  - Analysis       │ │ │
│  │  │ - ...       ││    │  │             ││    │  │  - Prediction     │ │ │
│  │  └─────────────┘│    │  │             ││    │  │  - ...            │ │ │
│  │        │        │    │  │             ││    │  └───────────────────┘ │ │
│  │        │        │    │  │             ││    │                         │ │
│  └─────────────────┘    │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │                         │ │
│                         │  │             ││    │......
```

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

### 监控服务应用场景

1. **持续生产环境监控**: 实时监控关键应用的健康状况，后台持续运行
2. **自动故障恢复**: 在检测到问题时自动执行预定义的修复操作
3. **自动合规性检查**: 定期检查配置是否符合安全标准
4. **关键信息周期汇总**: 按配置指定间隔周期汇总关键异常信息

### 聊天服务应用场景

1. **实时状态查询**: 用户可随时询问集群当前状态
2. **问题诊断咨询**: 与AI助手对话，获取问题诊断和解决建议
3. **运维操作指导**: 获取运维操作建议和最佳实践
4. **异常信息查询**: 查询近期发生的异常事件和处理情况
5. **技能化操作**: 通过自然语言调用预定义技能，如查询关键事件、资源使用情况等

### 规则引擎服务应用场景

1. **自动化策略生成**: 基于预设规则自动分析异常并生成修复策略
2. **快速响应**: 对常见问题快速生成处理方案
3. **确定性决策**: 基于明确规则的可预测行为
4. **规则管理**: 动态加载和更新规则

### AI决策服务应用场景

1. **智能决策**: 利用AI模型进行复杂问题分析
2. **自适应处理**: 处理未预见的问题
3. **上下文感知**: 考虑历史数据和环境因素
4. **高级分析**: 提供深度分析和建议

### MCP操作服务应用场景

1. **统一接口**: 为所有服务提供一致的K8S操作接口
2. **安全控制**: 集中权限管理和审计
3. **操作抽象**: 提供高层操作接口
4. **状态同步**: 确保操作状态的一致性

## 部署方式

此项目拆分为五个独立的二进制程序，可通过以下方式运行：

- **监控服务**：`./cmd/monitoring-service/monitoring-service`
- **聊天服务**：`./cmd/chat-service/chat-service`
- **规则引擎服务**：`./cmd/rule-engine-service/rule-engine-service`
- **AI决策服务**：`./cmd/ai-decision-service/ai-decision-service`
- **MCP操作服务**：`./cmd/mcp-service/mcp-service`
- **集群内部运行**：在K8S集群内部作为独立Pod运行

## 微服务架构优势

1. **高可用性**: 监控服务持续运行，确保集群监控不间断
2. **独立扩展**: 每个服务可独立配置和扩展
3. **故障隔离**: 一个服务故障不会影响其他服务
4. **资源优化**: 每个服务专注于特定任务，资源利用更高效
5. **开发效率**: 团队可以并行开发不同服务
6. **技术灵活性**: 可以为不同服务选择最适合的技术栈
7. **专业分工**: 每个服务专注于特定领域，实现专业化处理

## 未来发展方向

- 集成更多AI模型以提高决策准确性
- 支持多集群管理
- 更丰富的可视化界面
- 机器学习驱动的异常检测
- 更智能的容量规划算法
- 扩展技能系统，增加更多运维操作技能
- 支持MCP协议，实现更广泛的工具集成
```

## 快速开始

### 监控服务配置

监控服务的配置文件位于 `config/config.yaml`，包含以下主要配置项：

```yaml
log_level: info                    # 日志级别 (debug, info, warn, error)
metrics_port: 8080                 # 暴露Prometheus指标的端口
namespace: default                 # 监控的目标命名空间 (已弃用，请使用 monitoring.namespaces)
kubeconfig: ""                     # K8S集群认证配置文件路径

llm:                               # 大语言模型配置（仅用于AI增强功能）
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
  enable_monitor_resources: true   # 是否启用K8S资源监控代理
  enable_monitor_events: true      # 是否启用K8S事件监控代理
  namespaces: ["default", "kube-system", "monitoring"] # 监控的命名空间列表 (支持多个命名空间)
  kubeconfig: ""                   # K8S集群认证配置文件路径
  metrics_port: 8080               # 暴露指标的端口
  aggregation_interval_minutes: 10 # 日志聚合间隔（分钟），即每10分钟对关键事件进行一次汇总
```

**多命名空间监控说明**：
- `namespaces` 字段，支持同时监控多个命名空间
- 每个命名空间的日志将存储在独立的目录中 (`logs/basic/{namespace}/`)
- 设置为 `["all"]` 可以监控集群中的所有命名空间
- 日志聚合器会从所有命名空间的日志目录中收集信息并生成汇总报告

### 聊天服务配置

聊天服务的配置文件位于 `config/config.yaml`，包含以下主要配置项：

```yaml
log_level: info                    # 日志级别 (debug, info, warn, error)
metrics_port: 8081                 # 暴露Prometheus指标的端口

llm:                               # 大语言模型配置
  model: gpt-4o-mini               # 使用的AI模型
  api_key: ""                      # AI模型API密钥
  base_url: ""                     # AI模型API基础URL
  enabled: true                    # 是否启用AI功能

mcp:                               # MCP服务器配置
  servers:
    - name: "k8s-monitoring-mcp"
      enabled: false
      transport: "streamable_http"
      server_url: "http://localhost:3000/mcp"
      timeout: 10
      headers: {}

skills:                            # 技能系统配置
  directory: "../../resources/skills" # 技能定义目录
  auto_reload: true                # 是否自动重载技能定义

chat:                              # 聊天界面配置
  enable_streaming: true           # 是否启用流式响应
  max_tokens: 2000                 # 最大响应token数
  temperature: 0.7                 # AI响应温度
```

### 规则引擎服务配置

规则引擎服务的配置文件位于 `config/config.yaml`，包含以下主要配置项：

```yaml
log_level: info                    # 日志级别 (debug, info, warn, error)
metrics_port: 8082                 # 暴露Prometheus指标的端口
namespace: default                 # 监控的目标命名空间

rules:                             # 规则引擎配置
  alert_rules_file: "./rules/alert_rules.yaml"       # 告警规则文件路径
  remediation_rules_file: "./rules/remediation_rules.yaml" # 修复规则文件路径

basic:                             # 基础监控配置
  interval_seconds: 30             # 监控间隔秒数
  max_retries: 3                   # 最大重试次数
  dry_run: false                   # 是否演练模式（不执行实际操作）

rule_engine:                       # 规则引擎配置
  enable_rule_engine: true         # 是否启用规则引擎
  namespace: default               # 监控的命名空间
  metrics_port: 8082               # 暴露指标的端口
  aggregation_interval_minutes: 10 # 日志聚合间隔（分钟），即每10分钟对关键事件进行一次汇总
  rule_check_interval: 60          # 规则检查间隔（秒）
```

### AI决策服务配置

AI决策服务的配置文件位于 `config/config.yaml`，包含以下主要配置项：

```yaml
log_level: info                    # 日志级别 (debug, info, warn, error)
metrics_port: 8083                 # 暴露Prometheus指标的端口

llm:                               # 大语言模型配置
  model: gpt-4o-mini               # 使用的AI模型
  api_key: ""                      # AI模型API密钥
  base_url: ""                     # AI模型API基础URL
  enabled: true                    # 是否启用AI功能

ai_decision:                       # AI决策服务配置
  enable_ai_decision: true         # 是否启用AI决策
  namespace: default               # 监控的命名空间
  metrics_port: 8083               # 暴露指标的端口
  decision_interval: 120           # 决策间隔（秒）
  max_concurrent_requests: 5       # 最大并发请求数
```

### MCP操作服务配置

MCP操作服务的配置文件位于 `config/config.yaml`，包含以下主要配置项：

```yaml
log_level: info                    # 日志级别 (debug, info, warn, error)
metrics_port: 8084                 # 暴露Prometheus指标的端口

mcp:                               # MCP服务器配置
  servers:
    - name: "k8s-operation-mcp"
      enabled: true
      transport: "streamable_http"
      server_url: "http://localhost:3000/mcp"
      timeout: 10
      headers: {}

k8s_operation:                    # K8S操作服务配置
  enable_k8s_operation: true      # 是否启用K8S操作服务
  namespace: default              # 操作的命名空间
  metrics_port: 8084              # 暴露指标的端口
  max_concurrent_operations: 10   # 最大并发操作数
```

### 运行程序

```bash
# 构建监控服务
cd cmd/monitoring-service
go build -o monitoring-service .
./monitoring-service

# 构建聊天服务
cd ../chat-service
go build -o chat-service .
./chat-service

# 构建规则引擎服务
cd ../rule-engine-service
go build -o rule-engine-service .
./rule-engine-service

# 构建AI决策服务
cd ../ai-decision-service
go build -o ai-decision-service .
./ai-decision-service

# 构建MCP操作服务
cd ../mcp-service
go build -o mcp-service .
./mcp-service

# 或者使用Makefile（如果存在）
make build-all

# 或者使用构建脚本
./build.sh
```

### 代理能力说明

1. **K8S资源监控代理 (monitoring.enable_monitor_resources)**:
   - 实时监控资源状态
   - 收集性能指标
   - 检测异常状态
   - 此代理默认启用，是其他代理的基础
   - 独立运行，不依赖大模型连接
   - 基础功能：基于预设规则检测常见问题（如Pod崩溃、资源不足等）
   - 增强功能：当大模型可用时，进行智能异常检测和根因分析

2. **K8S事件监控代理 (monitoring.enable_monitor_events)**:
   - 处理K8S事件流，监听集群事件
   - 分类和优先级排序
   - 发送通知
   - 此代理依赖监控代理，默认启用

3. **修复代理 (rule_engine.enable_rule_engine)**:
   - 基于预设规则自动执行修复操作
   - 通过规则引擎服务实现
   - 如重启故障Pod、调整资源配额等
   - 高风险操作，需要谨慎启用
   - 建议先在非生产环境测试

4. **规则引擎代理 (rule_engine.enable_rule_engine)**:
   - 基于预设规则分析异常
   - 生成修复或调整策略
   - 通过MCP服务执行策略
   - 支持动态规则管理

5. **AI决策代理 (ai_decision.enable_ai_decision)**:
   - 与LLM交互，基于监控数据生成决策
   - 构建适合LLM的prompt格式
   - 通过MCP服务执行AI生成的策略
   - 考虑历史数据和上下文因素

### 关键事件聚合机制

系统实现了关键事件的定时聚合机制：

- **聚合间隔**: 通过 `monitoring.aggregation_interval_minutes` 配置，默认为10分钟
- **聚合内容**: 对 `logs/basic` 路径下的日志进行整理、去重
- **输出位置**: 聚合后的异常数据汇总输出到 `logs/critical_record` 路径
- **目的**: 为AI增强能力的agent提供简洁的异常数据，便于读取和分析
- **格式**: 输出简短的异常数据汇总，包含关键事件、Pod状态等信息

### 技能系统架构

系统采用标准化的技能组织结构，每个技能都是一个独立的文件夹，包含定义文件和脚本文件：

```
skills/
├── get_recent_critical_events/           # 关键事件查询技能
│   ├── SKILL.md                        # 技能定义文件（固定命名）
│   └── scripts/                        # 技能脚本目录
│       └── get_critical_events.sh      # 获取关键事件的shell脚本
└── get_cluster_resources/              # 集群资源查询技能
    ├── SKILL.md                        # 技能定义文件（固定命名）
    └── scripts/                        # 技能脚本目录
        └── get_cluster_resources.sh    # 获取集群资源的shell脚本
```

这种结构确保了技能的模块化和可维护性，每个技能都可以独立开发、测试和部署。

### 技能系统开发经验总结

#### 1. 技能定义格式
- 使用 SKILL.md 文件定义技能，包含 YAML frontmatter 和 Markdown 内容
- frontmatter 部分定义技能元数据（name, description 等）
- body 部分包含参数说明、示例命令和输出说明
- 正确的 frontmatter 格式为：
  ```
  ---
  name: skill-name
  description: skill description
  ---
  ```

#### 2. 路径配置注意事项
- 日志文件实际写入路径：`logs/basic/critical_events.log`
- 聚合后摘要文件路径：`logs/critical_record/critical_summary_YYYYMMDD_HHMMSS.txt`
- 技能脚本只使用聚合后的摘要文件，不回退到原始日志文件
- 使用 `ls -t` 命令按时间排序查找最新的摘要文件
- 聚合文件包含更高质量的信息，仅使用聚合文件进行查询反馈

#### 3. 正则表达式处理
- SKILL.md 文件解析需处理不同操作系统的换行符（LF vs CRLF）
- 使用灵活的正则表达式 `[\r\n]` 来匹配不同换行符
- 确保 frontmatter 解析正则表达式能正确匹配分隔符

#### 4. Shell 脚本编写
- 脚本应包含适当的错误处理和回退机制
- 使用参数化方式处理输入参数
- 脚本需具有可执行权限（chmod +x）
- 脚本路径应相对于技能目录的 scripts 子目录

#### 5. 参数模板处理
- 在 SKILL.md 中使用 `{{ inputs.param_name }}` 语法定义参数占位符
- 系统会自动将此语法转换为 Go 模板语法 `{{.param_name }}`
- 参数在运行时会被实际值替换

#### 6. 技能执行机制
- 技能通过通用的 shell 命令执行机制运行，实现与代码的完全解耦
- 支持 `shell_command` 和 `function_call` 两种执行类型
- 通过 `list skills` 命令可动态查看可用技能列表