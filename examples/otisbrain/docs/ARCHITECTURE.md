# OtisBrain 整体架构设计

## 概述

OtisBrain 是一个智能 Kubernetes 编排和监控系统，结合了 AI 驱动的决策能力、全面的集群监控和标准化的工具集成。系统采用微服务架构，通过松耦合设计实现模块间的协作，为 Kubernetes 操作提供智能自动化。

## 设计原则

### 1. 松耦合架构
- 各模块独立部署和管理
- 通过标准化协议进行通信
- 配置驱动，而非代码硬编码

### 2. 微服务设计
- 每个功能单元作为独立服务运行
- 独立的生命周期管理
- 故障隔离，互不影响

### 3. 标准化协议
- 遵循标准 MCP (Model Context Protocol) 协议
- 与不同 AI 模型兼容
- 第三方工具集成友好

### 4. 配置驱动
- 通过配置文件管理服务连接
- 支持动态启用/禁用服务
- 无需修改代码即可调整架构

## 核心模块

### 1. Monitor 模块
Monitor 模块作为整个系统的数据收集和监控中心。

#### 职责：
- 持续监控 Kubernetes 集群状态
- 收集各种指标和事件（pods、nodes、资源、自定义指标）
- 生成结构化日志和持久化指标数据
- 存储数据在组织化的目录结构中（`logs/basic`、`logs/critical_record`、`logs/memory_usage`）

#### 关键组件：
- **基础监控代理**：监控 pod、deployment 和 service 状态
- **多命名空间监控代理**：跨多个命名空间扩展监控
- **内存收集器**：收集并存储随时间变化的内存使用指标
- **事件监控器**：跟踪关键集群事件并生成警报

#### 数据输出：
- 包含集群状态信息的结构化日志文件
- 内存使用时间序列数据
- 用于异常检测的关键事件日志
- 用于分析的聚合指标

### 2. Chat 模块
Chat 模块充当 AI 决策引擎和对话界面。

#### 职责：
- 处理用户的自然语言请求
- 解释用户意图和需求
- 协调其他模块以满足请求
- 基于可用数据生成智能响应
- 通过 MCP 协议编排工具执行

#### 关键组件：
- **自然语言处理**：理解用户查询
- **工具协调**：管理工具选择和执行
- **响应生成**：创建人类可读的响应
- **上下文管理**：维护对话状态

#### 集成点：
- 从 Monitor 模块日志读取数据
- 通过 MCP 协议调用外部工具
- 为集群操作提供用户界面

### 3. MCP (Model Context Protocol) 模块
MCP 模块作为标准化工具集成和协议适配层。

#### 职责：
- 提供标准化的工具发现和执行协议
- 连接 AI 模型与外部工具和服务
- 管理工具生命周期和执行
- 在不紧密耦合的情况下实现灵活的工具集成

#### 关键组件：
- **工具发现**：通过 MCP 协议发现可用工具
- **执行引擎**：执行工具并管理结果
- **协议适配器**：处理 MCP 通信标准
- **工具注册表**：维护可用工具及其功能

#### 集成点：
- 作为 Chat 和外部服务之间的桥梁
- 使 Monitor 数据能够作为可调用工具公开
- 促进与第三方工具的集成

## 架构流程

### 数据流架构
```
用户请求 → Chat 模块 → MCP 协议 → Monitor MCP 服务 → Monitor 数据
     ↑                           ↓
     ←───────────────────────────────
```

### 详细工作流程：

1. **用户查询**：用户提交自然语言请求（例如，"哪些 pods 存在内存泄漏？"）
2. **Chat 处理**：
   - Chat 模块解释请求
   - 识别对内存分析工具的需求
3. **MCP 发现**：
   - Chat 模块根据 `config/config.yaml` 配置连接到 MCP 服务
   - 通过标准 MCP 协议发现可用的 `detect_memory_leak` 工具
4. **Monitor MCP 服务**：
   - 读取 Monitor 模块收集的内存使用数据
   - 对历史数据执行趋势分析
   - 返回分析结果
5. **响应生成**：
   - Chat 模块接收 MCP 服务结果
   - 生成自然语言响应
6. **用户响应**：系统响应 "命名空间 B 中的 Pod A 显示内存泄漏，趋势为 X MB/小时"

## MCP 服务器架构（微服务方法）

### 目录结构
```
mcp/
├── run-mcp-config.yaml          # MCP 服务配置文件
├── start-all-servers.sh         # 启动脚本
├── monitor-mcp-server/          # 监控 MCP 服务器
│   ├── monitor-mcp-server       # 二进制文件
│   ├── config.yaml              # 服务器配置
│   └── Dockerfile               # 容器化配置
├── k8s-operation-mcp-server/    # K8s 操作 MCP 服务器
│   ├── k8s-operation-mcp-server # 二进制文件（待实现）
│   ├── config.yaml
│   └── Dockerfile
└── ...                          # 其他 MCP 服务器
```

### 配置文件 (run-mcp-config.yaml)
```yaml
mcp_servers:
  - name: "monitor-mcp-server"
    enabled: true
    port: 3001
    binary_path: "./monitor-mcp-server/monitor-mcp-server"
    args: ["--log-dir", "/workspace/logs"]
    transport: "http"
    server_url: "http://localhost:3001"
    health_check_path: "/health"
    timeout_seconds: 30

  - name: "k8s-operation-mcp-server"
    enabled: true
    port: 3002
    binary_path: "./k8s-operation-mcp-server/k8s-operation-mcp-server"
    args: []
    transport: "http"
    server_url: "http://localhost:3002"
    health_check_path: "/health"
    timeout_seconds: 30
```

### 启动脚本 (start-all-servers.sh)
```bash
#!/bin/bash
# 解析 run-mcp-config.yaml 并启动所有启用的 MCP 服务器
# 每个服务器作为独立进程运行，暴露自己的端点
# Chat 模块通过标准 MCP 协议调用这些端点
```

### 通信模式
- **Chat Agent** ↔ **配置驱动** ↔ **独立的 MCP 服务器**
- 每个 MCP 服务器作为独立进程运行，有自己的端点
- 通过标准 MCP 协议在 Chat 和 MCP 服务器之间通信
- 真正的微服务架构，松耦合

## 基于 Monitor 的 MCP 服务器设计

### 目的
Monitor MCP 服务器利用现有的 Monitor 模块数据提供 AI 可访问的工具，而无需复制数据收集工作。

### 数据源
- **直接日志读取**：从 Monitor 模块读取结构化日志
- **内存使用分析**：访问时间序列内存数据
- **事件关联**：将关键事件与 pod 状态相结合
- **资源指标**：利用收集的资源利用率数据

### 可用工具
1. **monitor_pods**：从基本日志读取 pod 状态
2. **get_cluster_summary**：从收集的日志分析集群健康状况
3. **detect_memory_leak**：从内存使用日志分析内存趋势
4. **get_recent_events**：从事件日志检索关键事件
5. **get_resource_utilization**：从内存使用日志获取资源使用情况

### 优势
- **解耦**：Monitor 专注于数据收集，MCP 专注于数据服务
- **效率**：重用现有监控基础设施
- **可扩展性**：轻松添加新的监控分析工具
- **一致性**：与 Monitor 模块数据结构保持一致

## 系统集成点

### Monitor ↔ MCP
- MCP 服务器读取 Monitor 生成的日志和指标
- MCP 提供对 Monitor 数据的标准化访问
- 不直接复制数据收集

### Chat ↔ MCP
- Chat 模块通过 `config/config.yaml` 配置发现并连接 MCP 服务
- MCP 处理工具执行并返回结果
- 标准化协议确保兼容性

### Monitor ↔ Chat
- Chat 通过 MCP 抽象读取 Monitor 数据
- 直接日志读取用于快速状态检查
- 通过 MCP 工具间接访问进行分析

## 已实现内容

### Monitor 模块
- [x] 基础监控代理
- [x] 多命名空间监控代理
- [x] 内存收集器
- [x] 事件监控器
- [x] 日志聚合器
- [x] 结构化日志输出

### Chat 模块
- [x] 自然语言处理
- [x] 工具协调
- [x] 响应生成
- [x] 上下文管理
- [x] MCP 协议集成

### MCP 模块
- [x] Monitor MCP 服务器实现
- [x] 标准 MCP 协议支持
- [x] 动态工具发现
- [x] 配置驱动连接
- [x] 启动脚本

## 待实现内容

### Monitor 模块
- [ ] 更多监控指标类型
- [ ] 高级告警规则
- [ ] 性能优化

### Chat 模块
- [ ] 更多内置工具
- [ ] 对话历史管理
- [ ] 用户偏好学习

### MCP 模块
- [ ] K8s Operation MCP 服务器
- [ ] Rule Engine MCP 服务器
- [ ] 更多专用 MCP 服务
- [ ] 服务健康检查
- [ ] 负载均衡

### 系统级
- [ ] 完整的部署方案
- [ ] 安全加固
- [ ] 性能基准测试
- [ ] 生产环境监控
- [ ] 自动扩缩容