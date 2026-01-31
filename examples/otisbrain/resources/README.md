# Resources 资源目录

本目录包含 OtisBrain 系统的附加资源：

## 子目录

### mcpserver/
Model Context Protocol 服务器的配置文件，用于实现 AI 系统与外部工具之间的通信。

### skills/
技能定义及相关代码，用于扩展 AI 助手的功能。技能使 AI 能够执行特定任务，如检索集群信息、分析事件和提供建议。

## 技能系统 (Skills System)

技能系统允许 AI 助手通过调用专用函数执行各种任务。每个技能都包含以下组成部分：

- **SKILL.md**: 技能定义文件，包含技能元数据、参数说明和示例命令
- **scripts/**: 包含技能实现脚本的子目录

目前系统包含以下技能：

### 1. get_cluster_resources (获取集群资源)
- **功能**: 获取 Kubernetes 集群的资源使用情况，包括 Pod、Deployment、节点资源使用等
- **参数**: namespace（命名空间）、resource_type（资源类型）
- **用途**: 查询集群资源状态

### 2. get_recent_critical_events (获取近期关键事件)
- **功能**: 获取指定时间范围内的关键异常事件信息
- **参数**: time_range（时间范围）、severity（严重级别）、resource_type（资源类型）
- **用途**: 了解集群最近的问题和异常情况

### 3. safe_file_operations (安全文件操作)
- **功能**: 安全地执行文件操作，包括读取、写入、复制、移动和删除文件
- **参数**: operation（操作类型）、source_path（源路径）、destination_path（目标路径）等
- **用途**: 提供安全的文件操作功能，包含安全检查以防止意外覆盖或删除重要文件

技能系统采用模块化设计，每个技能都是一个独立的文件夹，包含定义文件和脚本文件，确保了技能的模块化和可维护性。