# OtisBrain MCP 服务

MCP (Model Context Protocol) 服务是一套独立的工具服务器，为 AI 模型提供标准化的工具访问接口。

## 快速开始

### 1. 启动 MCP 服务

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

### 2. 配置 MCP 服务

编辑 `run-mcp-config.yaml` 文件来启用/禁用特定的 MCP 服务器：

```yaml
mcp_servers:
  - name: "monitor-mcp-server"
    enabled: true  # 设置为 false 可禁用此服务器
    port: 3001
    binary_path: "./monitor-mcp-server/monitor-mcp-server"
    args: ["--log-dir", "/workspace/logs"]
    transport: "http"
    server_url: "http://localhost:3001/mcp"
```

## 与 Chat 模块集成

Chat 模块通过配置文件与 MCP 服务实现松耦合集成。Chat 模块读取 `config/config.yaml` 中的 MCP 服务器配置来连接相应的服务：

```yaml
# config/config.yaml
mcp:
  servers:
    - name: "monitor-mcp-server"
      enabled: true
      transport: "http"
      server_url: "http://localhost:3001/mcp"
      timeout: 10
      headers: {}

    - name: "k8s-operation-mcp-server"
      enabled: false  # 待实现
      transport: "http"
      server_url: "http://localhost:3002/mcp"
      timeout: 10
      headers: {}
```

## Monitor MCP 服务器

Monitor MCP 服务器提供基于 Monitor 模块收集数据的 AI 可访问工具：

### 可用工具

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

### 数据源

Monitor MCP 服务器直接读取 Monitor 模块生成的日志和指标数据：

- **基本日志**：`/workspace/logs/basic/{namespace}/basic.log`
- **内存使用**：`/workspace/logs/memory_usage/{namespace}/{pod}/{container}/memory_usage.json`
- **关键事件**：`/workspace/logs/basic/{namespace}/critical_events.log`

## 开发新 MCP 服务器

要创建新的 MCP 服务器：

1. 创建新的目录：`mcp/my-new-mcp-server/`
2. 实现 MCP 服务器逻辑
3. 添加到 `run-mcp-config.yaml` 配置中
4. 确保启动脚本支持新服务

## 故障排除

### 检查服务器状态
```bash
./start-all-servers.sh status
```

### 查看日志
```bash
# 查看 Monitor MCP 服务器日志
tail -f logs/monitor-mcp-server.log
```

### 重启服务
```bash
./start-all-servers.sh stop
./start-all-servers.sh start
```

## 架构优势

- **松耦合**：每个 MCP 服务器独立部署和管理
- **可扩展性**：轻松添加新的 MCP 服务器
- **标准化**：遵循标准 MCP 协议
- **资源效率**：Monitor 数据复用，避免重复收集