# OtisBrain 架构设计

## 整体架构

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
│   │   ├── health_checker.go           # 健康检查器
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
│       └── predictive_tool.go          # 预测性维护工具
├── ai/chat/                            # AI聊天界面
│   └── chat.go                         # 交互式聊天界面
├── ai/tools/                           # AI技能系统
│   ├── critical_events.go              # 关键事件检索工具
│   └── register.go                     # 技能注册系统
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
├── resources/                          # 资源文件夹
│   ├── mcpserver/                      # MCP服务器配置
│   │   └── config.yaml                 # MCP服务器配置文件
│   └── skills/                         # AI技能定义
│       ├── get_recent_critical_events.yaml  # 关键事件检索技能
│       └── get_cluster_resources.yaml       # 集群资源检索技能
└── testdata/                           # 测试数据
    └── sample_manifests/               # 示例清单文件
```

## 服务拆分说明

### 监控服务 (monitoring-service)

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

### 聊天服务 (chat-service)

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

### 规则引擎服务 (rule-engine-service)

此服务专注于基于预设规则分析异常并输出策略：

- **规则匹配**：根据预定义规则匹配监控到的异常
- **策略生成**：基于匹配结果生成修复或调整策略
- **策略执行**：通过MCP服务执行生成的策略
- **规则管理**：支持动态加载和更新规则

**特点**：
- 快速响应：基于预设规则快速生成策略
- 确定性：基于明确规则，行为可预测
- 可扩展性：支持多种类型的规则和策略

### AI决策服务 (ai-decision-service)

此服务专注于与LLM交互，基于监控数据生成决策：

- **数据聚合**：从监控服务收集异常数据
- **Prompt构建**：将监控数据转化为适合LLM的格式
- **AI交互**：与大语言模型交互，获取决策建议
- **决策执行**：通过MCP服务执行AI生成的策略

**特点**：
- 智能决策：利用AI模型进行复杂分析
- 自适应：能够处理未预见的问题
- 上下文感知：考虑历史数据和环境因素

### MCP操作服务 (mcp-service)

此服务专注于统化K8S操作接口：

- **接口封装**：统一封装K8S API操作
- **安全控制**：实施权限控制和审计
- **操作抽象**：提供高层操作接口
- **状态同步**：确保操作状态的一致性

**特点**：
- 统一接口：为所有服务提供一致的K8S操作接口
- 安全性：集中权限管理和审计
- 可靠性：确保操作的安全和一致性

## 设计理念

1. **微服务架构**：五个服务分离，确保各司其职且可独立扩展
2. **多代理协作**：采用多个专业代理协同工作的模式，每个代理负责特定任务
3. **可观察性**：全面的指标、日志和追踪能力
4. **渐进式智能**：基础功能不依赖AI，AI作为增强层提供智能分析能力
5. **弹性设计**：核心监控功能独立运行，不受大模型连接状态影响
6. **安全操作**：最小权限原则和操作审计
7. **可扩展性**：模块化设计便于功能扩展
8. **非侵入式AI增强**：AI层独立运行，不对基础层工作流程造成干扰
9. **代码与Skills完全解耦**：确保核心业务逻辑与AI技能系统相互独立，便于独立演进和维护

## 核心代理设计

### 1. K8S监控代理 (k8s_monitor_agent.go)
- 实时监控指定命名空间中的资源状态
- 收集Pod、Deployment、Service等资源的指标
- 检测异常状态并触发告警
- 独立运行，不依赖大模型连接
- 基础功能：基于预设规则检测常见问题（如Pod崩溃、资源不足等）
- 增强功能：当大模型可用时，进行智能异常检测和根因分析

### 2. 告警处理代理 (alert_agent.go)
- 接收来自监控代理的告警信息
- 根据严重程度和类型进行分类
- 执行初步分析并决定后续处理流程

### 3. 自动修复代理 (remediation_agent.go)
- 根据预定义规则自动执行修复操作
- 包括重启Pod、回滚部署、调整资源配置等
- 记录所有修复操作供审计

### 4. 决策制定代理 (decision_agent.go)
- 对复杂问题进行深度分析（可选功能，需要大模型支持）
- 结合历史数据和上下文信息做出决策
- 当大模型不可用时，使用预定义规则进行基本决策
- 在必要时请求人工干预

## 微服务架构优势

1. **高可用性**: 监控服务持续运行，确保集群监控不间断
2. **独立扩展**: 每个服务可独立配置和扩展
3. **故障隔离**: 一个服务故障不会影响其他服务
4. **资源优化**: 每个服务专注于特定任务，资源利用更高效
5. **开发效率**: 团队可以并行开发不同服务
6. **技术灵活性**: 可以为不同服务选择最适合的技术栈
7. **专业分工**: 每个服务专注于特定领域，实现专业化处理