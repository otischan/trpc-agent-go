# OtisBrain - K8S集群智能运维代理

OtisBrain 是一个基于 tRPC-Agent-Go 框架构建的 Kubernetes 集群智能监控和运维系统。该系统采用微服务架构，拆分为五个独立的服务：

1. **监控服务 (monitoring-service)**：持续运行的监控和告警任务
2. **聊天服务 (chat-service)**：提供实时聊天界面
3. **规则引擎服务 (rule-engine-service)**：基于预设规则分析异常并执行自动修复策略
4. **AI决策服务 (ai-decision-service)**：与LLM交互，基于监控数据生成决策
5. **MCP操作服务 (mcp-service)**：统化K8S操作接口

系统能够监控指定命名空间的资源状态，并根据预设规则和AI推理执行智能化运维操作。

## 快速开始

### 监控服务配置

监控服务的配置文件位于 `config/config.yaml`，包含以下主要配置项：

```yaml
log_level: info                    # 日志级别 (debug, info, warn, error)
metrics_port: 8080                 # 暴露Prometheus指标的端口
namespace: default                 # 监控的目标命名空间 (已弃用，请使用 monitoring.namespaces)
kubeconfig: ""                     # K8S集群认证配置文件路径

llm:                               # 大语言模型配置（仅用于AI增强功能）
  model: qwen-plus                 # 使用的AI模型 (如 gpt-4o-mini, qwen-plus, etc.)
  api_key: ""                      # AI模型API密钥 (对于Qwen，将使用DASHSCOPE_API_KEY环境变量)
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"  # AI模型API基础URL
  enabled: true                    # 是否启用AI功能
  variant: "qwen"                  # 模型变体 (openai, qwen, deepseek, hunyuan)

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

# 或者使用构建脚本
./build.sh
```

### 配置 Qwen 模型

要使用阿里云的 Qwen 模型，您需要：

1. 在阿里云控制台获取 DashScope API 密钥
2. 设置环境变量：
   ```bash
   export DASHSCOPE_API_KEY="your-api-key-here"
   ```
3. 确保 config/config.yaml 中的 LLM 配置如下：
   ```yaml
   llm:
     model: qwen-plus                 # 使用的Qwen模型
     api_key: ""                      # 将使用DASHSCOPE_API_KEY环境变量
     base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
     enabled: true
     variant: "qwen"                  # 指定为Qwen变体
   ```

## 文档目录

- [架构设计](docs/ARCHITECTURE.md) - 详细的架构设计、服务拆分、设计理念
- [配置说明](docs/CONFIGURATION.md) - 详细的配置说明、各服务配置选项
- [监控功能](docs/MONITORING.md) - 监控范围、OOM分析机制、内存监控增强
- [日志管理](docs/LOGGING.md) - 日志管理与运行输出说明
- [技能系统](docs/SKILLS.md) - 技能系统架构、开发经验、使用说明
- [安全配置](docs/SECURITY.md) - 安全考虑、RBAC权限配置
- [部署指南](docs/DEPLOYMENT.md) - 部署方式、运维指南、扩展性说明