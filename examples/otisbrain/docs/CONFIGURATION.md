# OtisBrain 配置说明

## 配置文件结构

OtisBrain 使用 YAML 格式的配置文件，主要配置文件位于 `config/config.yaml`。

## 监控服务配置

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
  memory_monitoring:               # 内存监控配置
    basic_collection:              # 基础内存采集配置
      enabled: true                # 是否启用基础内存指标采集（启用即采集内存使用量）
      retention_days: 30           # 内存指标保留天数（用于OOM后分析）
    oom_analysis:                  # OOM事件分析配置
      enabled: true                # 是否启用OOM后内存分析
      max_history_days: 30         # OOM分析的最大历史数据天数
      min_data_points: 10          # 最小数据点数量（确保分析可靠性）
```

**多命名空间监控说明**：
- `namespaces` 字段，支持同时监控多个命名空间
- 每个命名空间的日志将存储在独立的目录中 (`logs/basic/{namespace}/`)
- 设置为 `["all"]` 可以监控集群中的所有命名空间
- 日志聚合器会从所有命名空间的日志目录中收集信息并生成汇总报告

## 聊天服务配置

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
    - name: "monitor-mcp-server"
      enabled: true
      transport: "http"
      server_url: "http://localhost:3001"
      timeout: 10
      headers: {}

    - name: "k8s-operation-mcp-server"
      enabled: false  # 待实现
      transport: "http"
      server_url: "http://localhost:3002"
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

## 规则引擎服务配置

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

## AI决策服务配置

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

## MCP操作服务配置

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

## 代理能力说明

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