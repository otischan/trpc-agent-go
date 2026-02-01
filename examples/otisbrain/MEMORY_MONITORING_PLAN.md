# OtisBrain 内存监控增强方案

## 方案概述

本方案实现了对OtisBrain监控系统的内存监控功能增强，主要目标是在发生OOMKilled事件时，自动分析该Pod的历史内存使用模式，并计算内存使用的整体趋势斜率。

## 核心设计理念

### 1. 分层监控策略
- **常规采集**: 持续采集内存使用指标，但不进行复杂分析
- **事件触发**: 仅在发生OOMKilled事件时进行深入分析
- **趋势分析**: 通过线性回归算法计算内存使用斜率

### 2. 时间尺度匹配
- **短期聚合**: 每10分钟进行常规异常检测
- **中期分析**: 发生OOM时进行历史数据分析
- **长期趋势**: 基于历史数据判断内存泄露趋势

## 实现细节

### 1. 配置结构
```yaml
monitoring:
  memory_monitoring:
    basic_collection:
      enabled: true                # 是否启用基础内存指标采集
      retention_days: 30           # 内存指标保留天数
    oom_analysis:
      enabled: true                # 是否启用OOM后内存分析
      max_history_days: 30         # OOM分析的最大历史数据天数
      min_data_points: 10          # 最小数据点数量
```

### 2. 核心组件

#### MemoryCollector
- 持续收集Pod内存使用指标
- 存储历史数据用于OOM后分析
- 实现线性回归算法计算内存使用斜率

#### BasicEventMonitorAgent
- 监控K8S事件流中的OOMKilled事件
- 触发内存使用历史分析
- 生成内存分析报告

### 3. 分析算法

#### 线性回归计算内存使用斜率
```go
// 使用时间序列索引作为x值，内存使用量作为y值
// 计算公式: slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
```

#### 斜率解读
- **正斜率**: 内存持续增长（可能存在泄露）
- **负斜率**: 内存逐渐减少
- **零斜率**: 内存使用相对稳定
- **显著性判断**: 斜率绝对值 > 0.1 MB/小时表示值得关注

### 4. 输出格式

当检测到OOM事件时，生成如下格式的分析报告：

```
# OOM事件内存分析报告
Pod: my-app-7d5b8c9c4-xl2v9
Namespace: default
Container: app-container
OOM时间: 2024-01-15T10:30:00Z

# 内存使用趋势分析
内存使用斜率: +2.5 MB/小时
分析时间范围: 过去30天
数据点数量: 1440 (从 2024-01-15T00:00:00Z 到 2024-01-15T10:30:00Z)

# 分析结论
- 内存在过去30天内呈缓慢增长趋势
- 平均每小时增长2.5MB
- 建议检查是否存在内存泄露
```

## RBAC权限要求

```yaml
# Pod相关权限
- resources: ["pods", "pods/status", "pods/log"]
  verbs: ["get", "list", "watch"]
# 事件相关权限
- resources: ["events"]
  verbs: ["get", "list", "watch"]
# PodMetrics相关权限
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods"]
  verbs: ["get", "list"]
```

## 优势

1. **资源高效**: 平时不进行复杂分析，OOM时才计算
2. **精准分析**: 只在真正出现问题时进行深入分析
3. **量化结果**: 提供具体的斜率数值而非阈值判断
4. **长期视角**: 不受固定时间窗口限制，可分析任意长度趋势
5. **运维友好**: 直接提供斜率信息供运维人员判断

## 部署说明

1. 确保K8S集群已部署Metrics Server
2. 配置适当的RBAC权限
3. 在config.yaml中启用memory_monitoring配置
4. 监控服务会在OOM事件发生时自动生成分析报告到logs/critical_record/

## 注意事项

- 需要Metrics Server提供内存指标数据
- 内存数据保留天数影响OOM分析的深度
- 分析结果仅供参考，需结合具体业务场景判断