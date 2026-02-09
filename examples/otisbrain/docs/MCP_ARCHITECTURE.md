# OtisBrain MCP 服务架构

## 概述

MCP (Model Context Protocol) 服务是一套独立的工具服务器，为 AI 模型提供标准化的工具访问接口。采用微服务架构，通过配置驱动实现松耦合集成。

## 架构设计

### 核心理念
- **松耦合**：通过配置文件管理服务连接，而非硬编码
- **微服务**：每个 MCP 服务器独立部署和管理
- **标准化**：遵循标准 MCP 协议进行通信
- **可扩展**：易于添加新的 MCP 服务

### 目录结构
```
mcp/
├── run-mcp-config.yaml          # MCP 服务配置文件
├── start-all-servers.sh         # 统一启动脚本
├── monitor-mcp-server/          # 监控 MCP 服务器
│   ├── monitor-mcp-server       # 二进制文件
│   ├── config.yaml              # 服务器配置
│   └── Dockerfile               # 容器化配置
├── k8s-operation-mcp-server/    # K8s 操作 MCP 服务器（待实现）
│   ├── k8s-operation-mcp-server # 二进制文件（待实现）
│   ├── config.yaml
│   └── Dockerfile
└── ...                          # 其他 MCP 服务器
```

## 配置管理

### 服务配置 (run-mcp-config.yaml)
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
    enabled: false  # 待实现
    port: 3002
    binary_path: "./k8s-operation-mcp-server/k8s-operation-mcp-server"
    args: []
    transport: "http"
    server_url: "http://localhost:3002"
    health_check_path: "/health"
    timeout_seconds: 30
```

## 服务管理

### 启动服务
```bash
# 启动所有启用的 MCP 服务器
./start-all-servers.sh start

# 检查服务器状态
./start-all-servers.sh status

# 停止所有 MCP 服务器
./start-all-servers.sh stop

# 重启服务
./start-all-servers.sh restart
```

## 与 Chat 模块集成

Chat 模块通过配置文件与 MCP 服务实现松耦合集成：

### 配置文件 (config/config.yaml)
```yaml
mcp:
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
```

### 集成代码示例
```go
// 在 Chat 模块中根据配置动态初始化 MCP 工具集
for _, server := range config.MCP.Servers {
    if server.Enabled {
        mcpToolSet := mcp.NewMCPToolSet(
            mcp.ConnectionConfig{
                Transport: server.Transport,
                ServerURL: server.ServerURL,
                Timeout:   time.Duration(server.Timeout) * time.Second,
                Headers:   server.Headers,
            },
            mcp.WithName(server.Name),
        )

        if err := mcpToolSet.Init(ctx); err != nil {
            log.Printf("Failed to initialize MCP toolset '%s': %v", server.Name, err)
        } else {
            toolSets = append(toolSets, mcpToolSet)
        }
    }
}
```

## 已实现服务

### Monitor MCP 服务器
提供基于 Monitor 模块收集数据的 AI 可访问工具：

1. **monitor_pods**
   - 功能：监控 Kubernetes 命名空间中 pods 的状态和资源使用情况
   - 输入：namespace（必需），pod_name（可选）
   - 输出：包含 pod 状态、资源使用等信息的结构化数据

2. **get_cluster_summary**
   - 功能：获取 Kubernetes 集群的近期重要事件和状态摘要
   - 输入：namespaces（可选）
   - 输出：集群健康状况摘要

3. **detect_memory_leak**
   - 功能：检测特定 pod/container 中的潜在内存泄漏
   - 输入：namespace、pod_name、container、hours
   - 输出：内存泄漏分析结果

4. **get_recent_events**
   - 功能：获取 Kubernetes 集群中的最近事件
   - 输入：namespace（可选）、limit
   - 输出：最近的集群事件列表

5. **get_resource_utilization**
   - 功能：获取集群中节点和 pod 的资源利用率
   - 输入：namespace（可选）
   - 输出：资源利用率数据

#### 数据源
Monitor MCP 服务器直接读取 Monitor 模块生成的日志和指标数据：

- **基本日志**：`/workspace/logs/basic/{namespace}/basic.log`
- **内存使用**：`/workspace/logs/memory_usage/{namespace}/{pod}/{container}/memory_usage.json`
- **关键事件**：`/workspace/logs/basic/{namespace}/critical_events.log`

## 待实现服务

### K8s Operation MCP 服务器
- [ ] Pod 操作工具（创建、删除、重启）
- [ ] Deployment 管理工具
- [ ] Service 管理工具
- [ ] 配置管理工具
- [ ] 资源调度工具

### Rule Engine MCP 服务器
- [ ] 规则评估工具
- [ ] 自动修复工具
- [ ] 告警管理工具

## 部署模式

### 单机部署
- 所有 MCP 服务器在同一主机运行
- 通过不同端口区分服务
- 适合开发和测试环境

### 容器化部署
- 每个 MCP 服务器独立容器
- 通过 Kubernetes 服务暴露
- 适合生产环境

## 安全考虑

- **访问控制**：MCP 服务器验证输入并清理输出
- **数据隐私**：Monitor 日志在本地处理，不对外暴露
- **资源限制**：MCP 服务器实施超时和资源约束
- **认证授权**：敏感操作的可选认证层

## 运维管理

- **健康检查**：每个 MCP 服务器提供健康检查端点
- **日志管理**：独立的日志文件便于调试
- **性能监控**：响应时间和资源使用监控
- **配置热更新**：支持配置动态更新