#!/bin/bash

# 动态图片转JXL工具构建脚本
# 版本: 2.0.1

echo "🔨 开始构建动态图片转JXL工具..."

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到Go环境，请先安装Go"
    exit 1
fi

# 检查依赖
echo "📦 检查依赖..."
go mod tidy

# 创建bin目录
mkdir -p bin

# 构建静态版本
echo "🚀 构建静态版本..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/dynamic2jxl-linux-amd64 main.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/dynamic2jxl-darwin-amd64 main.go
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/dynamic2jxl-darwin-arm64 main.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/dynamic2jxl-windows-amd64.exe main.go

# 构建当前平台版本
echo "🏠 构建当前平台版本..."
go build -ldflags="-s -w" -o dynamic2jxl main.go

echo "✅ 构建完成!"
echo "📁 可执行文件位置:"
echo "   - 当前平台: ./dynamic2jxl"
echo "   - Linux: ./bin/dynamic2jxl-linux-amd64"
echo "   - macOS Intel: ./bin/dynamic2jxl-darwin-amd64"
echo "   - macOS Apple Silicon: ./bin/dynamic2jxl-darwin-arm64"
echo "   - Windows: ./bin/dynamic2jxl-windows-amd64.exe"
