# OtisBrain MCP 服务架构设计

## 概述

OtisBrain MCP 服务是一套独立的 Model Context Protocol 服务器集合，为 AI 模型提供标准化的工具访问接口。每个 MCP 服务器都是独立的二进制文件，通过标准 MCP 协议与 Chat 模块通信。

## 架构设计

### 1. 独立 MCP 服务器模式

#### 目录结构
```
mcp/
├── run-mcp-config.yaml          # MCP 服务配置文件
├── start-all-servers.sh         # 启动脚本
├── monitor-mcp-server/          # 监控 MCP 服务器
│   ├── monitor-mcp-server       # 二进制文件
│   ├── config.yaml              # 服务器配置
│   └── Dockerfile               # 容器化配置
├── k8s-operation-mcp-server/    # K8s 操作 MCP 服务器
│   ├── k8s-operation-mcp-server # 二进制文件
│   ├── config.yaml
│   └── Dockerfile
└── ...                          # 其他 MCP 服务器
```

#### 配置文件 (run-mcp-config.yaml)
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

### 2. Monitor MCP 服务器实现

#### 核心功能
Monitor MCP 服务器提供基于 Monitor 模块收集数据的 AI 可访问工具：

1. **monitor_pods**：读取 pod 状态和资源使用情况
2. **get_cluster_summary**：分析集群健康状况
3. **detect_memory_leak**：检测内存泄漏趋势
4. **get_recent_events**：获取最近事件
5. **get_resource_utilization**：获取资源利用率

#### 数据源
- 直接读取 Monitor 模块生成的日志文件
- 访问时间序列内存使用数据
- 分析关键事件日志
- 聚合资源利用率指标

#### 通信协议
- 标准 MCP v1alpha1 协议
- HTTP 传输
- JSON 格式数据交换
- CORS 支持跨域访问

### 3. 启动管理

#### 启动脚本 (start-all-servers.sh)
```bash
#!/bin/bash
# 解析 run-mcp-config.yaml 并启动所有启用的 MCP 服务器
# 每个服务器作为独立进程运行，暴露自己的端点
# Chat 模块通过标准 MCP 协议调用这些端点
```

#### 进程管理
- 每个 MCP 服务器作为独立进程运行
- PID 文件管理进程生命周期
- 日志文件记录服务器状态
- 健康检查确保服务可用性

## 通信模式

### Chat ↔ MCP 服务器
```
Chat Agent → HTTP/MCP Protocol → Independent MCP Server → Monitor Data
     ↑                                        ↓
     ←──────────────────────────────────────────────
```

### 详细工作流程：
1. **工具发现**：Chat 模块通过标准 MCP 协议发现可用工具
2. **工具调用**：Chat 模块调用特定 MCP 服务器的工具
3. **数据读取**：MCP 服务器读取 Monitor 模块收集的数据
4. **结果返回**：MCP 服务器返回处理结果给 Chat 模块

## 优势

### 1. 真正的松耦合
- 每个 MCP 服务器独立部署
- 独立的生命周期管理
- 故障隔离，互不影响

### 2. 可扩展性
- 轻松添加新的 MCP 服务器
- 按需启用/禁用特定服务
- 独立扩展不同服务

### 3. 标准化协议
- 遵循标准 MCP 协议
- 与不同 AI 模型兼容
- 第三方工具集成友好

### 4. 资源效率
- Monitor 数据复用，避免重复收集
- 独立进程，资源隔离
- 按需启动，节省资源

## 部署模式

### 单机部署
- 所有 MCP 服务器在同一主机运行
- 通过不同端口区分服务
- 适合开发和测试环境

### 容器化部署
- 每个 MCP 服务器独立容器
- 通过 Kubernetes 服务暴露
- 适合生产环境

### 混合部署
- Monitor MCP 服务器独立部署（关键服务）
- K8s Operation MCP 服务器独立部署
- 根据需要添加其他 MCP 服务器
- 平衡性能和资源利用

## 安全考虑

- **访问控制**：MCP 服务器验证输入并清理输出
- **数据隐私**：Monitor 日志在本地处理，不对外暴露
- **资源限制**：MCP 服务器实施超时和资源约束
- **认证授权**：敏感操作的可选认证层

## 监控和运维

- **健康检查**：每个 MCP 服务器提供健康检查端点
- **日志管理**：独立的日志文件便于调试
- **性能监控**：响应时间和资源使用监控
- **配置热更新**：支持配置动态更新

## 未来扩展

- **插件架构**：支持第三方 MCP 服务器
- **自动扩缩容**：基于负载的自动扩缩容
- **多租户支持**：支持多用户环境
- **自定义工具开发**：允许用户创建特定领域工具
- **更多专用 MCP 服务器**：可根据需要扩展更多专用 MCP 服务