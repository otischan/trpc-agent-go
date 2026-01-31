#!/bin/bash

# 构建脚本用于编译 otisbrain 项目的各个服务

set -e  # 遌输任何命令失败时退出

# 创建目标目录
TARGET_DIR="target"
mkdir -p "$TARGET_DIR"

echo "开始构建 otisbrain 项目..."

# 构建 chat-service
echo "构建 chat-service..."
cd cmd/chat-service
go build -o ../../"$TARGET_DIR"/chat-service .
echo "chat-service 构建完成"
cd ../..

# 构建 monitoring-service
echo "构建 monitoring-service..."
cd cmd/monitoring-service
go build -o ../../"$TARGET_DIR"/monitoring-service .
echo "monitoring-service 构建完成"
cd ../..

echo "所有服务构建完成！"
echo "输出文件位于: $TARGET_DIR"
ls -la "$TARGET_DIR"

echo "构建完成！"