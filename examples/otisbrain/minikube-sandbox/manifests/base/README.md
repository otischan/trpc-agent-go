# 测试部署文件

这些YAML文件用于测试OtisBrain监控系统的功能，特别是在minikube环境中验证OOM（Out of Memory）检测能力。

## 文件说明

### busybox-deployment.yaml
原始的busybox测试部署，用于基本功能验证。

### memory-oom-test-deployment.yaml
逐步消耗内存的测试部署，用于测试内存监控功能。

### quick-oom-test-deployment.yaml
快速触发OOM的测试部署（使用Python3，但可能在某些镜像中不可用）。

### shell-oom-test-final.yaml
使用纯shell命令的内存消耗测试，不依赖外部工具。

### stress-oom-test.yaml
最终版的OOM测试部署，使用多种方法确保触发OOM事件，这是最可靠的测试方案。

## 使用方法

```bash
# 应用测试部署
kubectl apply -f stress-oom-test.yaml

# 查看Pod状态
kubectl get pods -l app=stress-oom-test

# 查看Pod详细信息
kubectl describe pods -l app=stress-oom-test

# 删除测试部署
kubectl delete -f stress-oom-test.yaml
```

这些测试部署主要用于验证OtisBrain监控系统是否能够正确检测和记录OOMKilled事件。