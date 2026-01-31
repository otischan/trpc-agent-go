---
name: get_cluster_resources
description: 获取集群资源使用情况
---

Overview

获取 Kubernetes 集群的资源使用情况，包括 Pod、Deployment、节点资源使用等。

Parameters

- namespace: 特定命名空间或"all"表示所有命名空间 (default: all)
- resource_type: 资源类型 (cpu, memory, pods, deployments, all, default: all)

Examples

1) 获取所有命名空间的 Pod 信息

   Command:

   bash scripts/get_cluster_resources.sh all pods

2) 获取特定命名空间的 Deployment 信息

   Command:

   bash scripts/get_cluster_resources.sh "{{ inputs.namespace }}" deployments

3) 获取节点资源使用情况

   Command:

   bash scripts/get_cluster_resources.sh all cpu

4) 获取指定命名空间的 Pod 资源使用情况

   Command:

   bash scripts/get_cluster_resources.sh "{{ inputs.namespace }}" cpu

5) 根据资源类型获取资源信息

   Command:

   bash scripts/get_cluster_resources.sh "{{ inputs.namespace }}" "{{ inputs.resource_type }}"

Output Files

- None (outputs to stdout)