#!/bin/bash

# static2avif 构建脚本

echo "========================================="
echo "static2avif 构建脚本"
echo "========================================="

# 检查Go是否已安装
if ! command -v go &> /dev/null
then
    echo "❌ 错误: 未找到Go命令，请先安装Go"
    exit 1
fi

# 检查FFmpeg是否已安装
if ! command -v ffmpeg &> /dev/null
then
    echo "❌ 错误: 未找到ffmpeg命令，请先安装FFmpeg"
    exit 1
fi

echo "✅ 检查依赖项通过"

# 初始化Go模块
echo "🔄 初始化Go模块..."
go mod tidy

# 构建项目
echo "🔨 构建项目..."
go build -o bin/static2avif main.go

if [ $? -ne 0 ]; then
    echo "❌ 构建失败!"
    exit 1
fi

echo "✅ 构建成功!"
echo "📦 可执行文件位置: bin/static2avif"

# 显示版本信息
echo "ℹ️  显示版本信息:"
./bin/static2avif -h

echo "🎉 构建完成!"