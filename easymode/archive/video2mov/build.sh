#!/bin/bash
echo "🔨 构建 video2mov..."
go build -o bin/video2mov-darwin-arm64 main.go
if [ $? -eq 0 ]; then
    echo "✅ 构建成功: bin/video2mov-darwin-arm64"
    chmod +x bin/video2mov-darwin-arm64
else
    echo "❌ 构建失败!"
    exit 1
fi
