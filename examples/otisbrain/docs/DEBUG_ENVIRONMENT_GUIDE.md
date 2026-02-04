# OtisBrain 调试环境使用指南

## 概述

OtisBrain 调试环境提供了一套完整的开发和测试工具链，用于验证和调试 OtisBrain 监控系统的各项功能。该环境使用 minikube 作为孪生集群，模拟生产环境的 Kubernetes 集群。

## 目录结构

```
/root/workspace/test-env/             # 调试环境根目录
├── debug-logs/                     # 调试日志记录
│   ├── session_YYYYMMDD_N.md      # 调试会话记录
│   └── ...
├── manifests/                      # Kubernetes 部署文件
│   ├── stress-oom-test.yaml       # 压力测试 OOM 场景
│   ├── slow-oom-test.yaml         # 缓慢 OOM 测试场景
│   └── ...
├── scripts/                        # 调试脚本
│   ├── run-monitor-test.sh        # 完整监控测试脚本
│   ├── run-quick-oom-test.sh      # 快速 OOM 测试脚本
│   └── ...
├── monitoring-service              # 监控服务可执行文件
├── monitoring-service-fixed        # 修复后的监控服务可执行文件
└── test-config.yaml               # 测试配置文件
```

## 使用方法

### 1. 环境准备

```bash
# 启动 minikube 集群
minikube start --driver=docker --memory=6g --cpus=4

# 启用 metrics-server
minikube addons enable metrics-server
```

### 2. 部署测试应用

```bash
# 部署 OOM 测试应用
kubectl apply -f /root/workspace/test-env/manifests/slow-oom-test.yaml
```

### 3. 启动监控服务

```bash
# 启动修复后的监控服务
cd /root/workspace/test-env
./monitoring-service-fixed -config ./test-config.yaml
```

### 4. 验证功能

```bash
# 检查监控服务日志
tail -f /root/workspace/test-env/logs/basic/default/basic.log

# 验证指标收集
kubectl top pods

# 检查监控服务生成的日志
ls -la /root/workspace/test-env/logs/
```

## 调试日志记录

### 记录格式

每次调试会话应在 `debug-logs/` 目录下创建一个新的 Markdown 文件，命名格式为 `session_YYYYMMDD_N.md`，其中：

- `YYYYMMDD` 是日期（如 20260204）
- `N` 是当天的会话编号（从1开始）

### 记录内容应包括

1. **调试日期和环境**
2. **问题描述** - 详细描述遇到的问题
3. **问题现象** - 具体的错误信息或异常行为
4. **问题分析** - 对问题原因的分析
5. **解决方案** - 采取的解决措施
6. **修复代码** - 如有代码修改，记录关键部分
7. **验证结果** - 修复后的验证情况
8. **经验总结** - 从此次调试中学到的经验

### 示例

```markdown
# OtisBrain监控服务调试日志

## 调试日期：2026年2月4日
## 调试环境：minikube孪生集群

### 问题描述
监控服务无法成功收集内存指标...

### 问题现象
- 监控服务持续报告错误...
```

## 脚本工具

### 监控测试脚本

```bash
# 运行完整监控测试
./scripts/run-monitor-test.sh

# 运行快速 OOM 测试
./scripts/run-quick-oom-test.sh
```

## 经验总结

### 常见问题及解决方案

1. **Metrics API 不可用**
   - 问题：`the server is currently unable to handle the request`
   - 解决：添加重试机制和指数退避算法

2. **OOM 事件检测**
   - 确保部署的应用有内存限制
   - 监控服务能正确捕获 OOMKilled 事件

3. **日志聚合功能**
   - 验证关键事件被正确记录和聚合
   - 检查聚合报告的准确性

### 最佳实践

1. **渐进式测试**：从简单场景开始，逐步增加复杂度
2. **日志记录**：详细记录每次调试的过程和结果
3. **配置管理**：使用专门的测试配置文件
4. **环境隔离**：确保调试环境与生产环境隔离

## 注意事项

- 调试完成后记得清理测试资源
- 定期备份重要的调试日志
- 保持调试环境的整洁
- 记录的调试日志有助于后续问题排查