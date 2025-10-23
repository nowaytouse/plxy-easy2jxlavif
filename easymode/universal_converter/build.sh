#!/bin/bash

# 通用媒体转换工具构建脚本

echo "🔧 构建通用媒体转换工具..."

# 创建bin目录
mkdir -p bin

# 构建主程序
echo "📦 编译主程序..."
go build -o bin/universal_converter main.go

if [ $? -eq 0 ]; then
    echo "✅ 构建成功！"
    echo "📁 可执行文件位置: bin/universal_converter"
    echo ""
    echo "🚀 使用方法:"
    echo "  ./bin/universal_converter -dir <输入目录> -type <转换类型> -mode <处理模式>"
    echo ""
    echo "📋 参数说明:"
    echo "  -type: avif, jxl, mov"
    echo "  -mode: all, static, dynamic, video"
    echo ""
    echo "💡 示例:"
    echo "  ./bin/universal_converter -dir ./images -type jxl -mode all"
    echo "  ./bin/universal_converter -dir ./photos -type avif -mode static"
    echo "  ./bin/universal_converter -dir ./videos -type mov -mode video"
else
    echo "❌ 构建失败！"
    exit 1
fi
