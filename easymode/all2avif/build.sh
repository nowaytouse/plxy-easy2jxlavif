#!/bin/bash

# AVIF 批量转换工具构建脚本
# 作者: AI Assistant

set -e

echo "🔨 开始构建 all2avif..."

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到Go环境"
    exit 1
fi

# 创建bin目录
mkdir -p bin

# 构建应用程序
echo "📦 构建应用程序..."
go build -o bin/all2avif main.go

# 设置执行权限
chmod +x bin/all2avif

# 创建符号链接
if [ -f "all2avif" ]; then
    rm all2avif
fi
ln -s bin/all2avif all2avif

echo "✅ 构建完成!"
echo "📁 可执行文件位置: bin/all2avif"
echo "🔗 符号链接: all2avif"

# 显示版本信息
echo "📋 版本信息:"
./bin/all2avif -h 2>/dev/null || echo "运行 ./bin/all2avif -h 查看帮助信息"

