# OtisBrain 安全配置

## 安全考虑

- 使用K8S RBAC进行权限控制
- 所有操作记录审计日志
- 支持操作确认机制（对于高风险操作）
- 加密敏感配置信息

## RBAC权限配置

为了使OtisBrain监控服务正常运行，需要为其配置适当的RBAC权限。以下是所需权限的详细说明：

### 1. 监控服务所需RBAC权限

```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: otisbrain-monitor-sa
  namespace: monitoring  # 根据实际部署命名空间调整
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: otisbrain-monitor-cr
rules:
# Pod相关权限
- apiGroups: [""]
  resources: ["pods", "pods/status", "pods/log"]
  verbs: ["get", "list", "watch"]
# Deployment相关权限
- apiGroups: ["apps"]
  resources: ["deployments", "deployments/status"]
  verbs: ["get", "list", "watch"]
# Service相关权限
- apiGroups: [""]
  resources: ["services", "services/status"]
  verbs: ["get", "list", "watch"]
# 事件相关权限
- apiGroups: [""]
  resources: ["events"]
  verbs: ["get", "list", "watch"]
# 节点相关权限
- apiGroups: [""]
  resources: ["nodes", "nodes/status", "nodes/proxy"]
  verbs: ["get", "list", "watch"]
# PersistentVolumeClaim相关权限
- apiGroups: [""]
  resources: ["persistentvolumeclaims"]
  verbs: ["get", "list", "watch"]
# Namespace相关权限
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "watch"]
# PodMetrics相关权限（用于内存监控）
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods"]
  verbs: ["get", "list"]
# NodeMetrics相关权限（用于内存监控）
- apiGroups: ["metrics.k8s.io"]
  resources: ["nodes"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: otisbrain-monitor-crb
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: otisbrain-monitor-cr
subjects:
- kind: ServiceAccount
  name: otisbrain-monitor-sa
  namespace: monitoring  # 根据实际部署命名空间调整
```

### 2. 权限说明

- **Pod权限**: 监控Pod状态、资源使用情况和日志
- **Deployment权限**: 监控Deployment状态和副本数
- **Service权限**: 监控Service可用性
- **事件权限**: 监控K8S事件流，捕获异常事件
- **节点权限**: 监控节点状态和资源使用
- **PVC权限**: 监控持久卷声明状态
- **Namespace权限**: 监控命名空间状态
- **Metrics权限**: 获取Pod和节点的指标数据（用于内存监控）