# OtisBrain 部署指南

## 部署方式

此项目拆分为多个独立的服务，可通过以下方式运行：

- **监控服务**：`./cmd/monitoring-service/monitoring-service`
- **聊天服务**：`./cmd/chat-service/chat-service`
- **规则引擎服务**：`./cmd/rule-engine-service/rule-engine-service`
- **AI决策服务**：`./cmd/ai-decision-service/ai-decision-service`
- **MCP服务**：`./mcp/start-all-servers.sh` (启动所有MCP服务器)
- **集群内部运行**：在K8S集群内部作为独立Pod运行

## 运行程序

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

# 启动MCP服务
cd ../mcp
./start-all-servers.sh start

# 或者使用构建脚本
./build.sh
```

## 扩展性

- 插件化工具系统
- 可配置的规则引擎
- 支持自定义监控指标
- 可扩展的通知渠道

## 未来发展方向

- 集成更多AI模型以提高决策准确性
- 支持多集群管理
- 更丰富的可视化界面
- 机器学习驱动的异常检测
- 更智能的容量规划算法
- 扩展技能系统，增加更多运维操作技能
- 支持MCP协议，实现更广泛的工具集成

## 使用场景

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

### MCP服务应用场景

1. **工具标准化**: 通过MCP协议提供标准化的工具访问接口
2. **服务解耦**: 通过配置驱动实现服务间的松耦合
3. **AI集成**: 为AI模型提供统一的外部工具访问方式
4. **可扩展性**: 支持动态添加新的MCP服务