---
name: get_recent_critical_events
description: 获取近期关键异常事件信息
---

Overview

获取指定时间范围内的关键异常事件信息，帮助用户了解集群最近的问题和异常情况。

Parameters

- time_range: 时间范围 (last_hour, last_6_hours, last_24_hours, default: last_hour)
- severity: 严重级别 (critical, warning, info, all, default: all)
- resource_type: 资源类型 (pod, deployment, service, all, default: all)

Examples

1) 获取最近的关键事件摘要

   Command:

   bash scripts/get_critical_events.sh all all all

2) 根据时间范围参数获取事件

   Command:

   bash scripts/get_critical_events.sh "{{ inputs.time_range }}" "{{ inputs.severity }}" "{{ inputs.resource_type }}"

Output Files

- None (outputs to stdout)