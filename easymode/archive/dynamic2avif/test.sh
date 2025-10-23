#!/bin/bash

# dynamic2avif 测试脚本

echo "========================================="
echo "dynamic2avif 测试脚本"
echo "========================================="

# 创建测试目录
echo "📂 创建测试目录..."
mkdir -p test/input
mkdir -p test/output

# 创建测试用的GIF文件（如果系统支持）
echo "🧪 创建测试文件..."

# 创建一个简单的测试GIF文件（如果ImageMagick可用）
if command -v convert &> /dev/null
then
    echo "🔄 使用ImageMagick创建测试GIF..."
    # 创建一个简单的动画GIF
    convert -size 100x100 xc:red -morph 10 -duplicate 1,-2-1 \
            -set delay 10 -loop 0 test/input/test.gif
    
    if [ $? -eq 0 ]; then
        echo "✅ 测试GIF文件创建成功"
    else
        echo "⚠️  测试GIF文件创建失败"
    fi
else
    echo "⚠️  ImageMagick未安装，跳过测试文件创建"
    echo "💡 提示: 您可以手动将一些动态图片放入 test/input 目录进行测试"
fi

echo "🚀 运行dynamic2avif工具..."
echo "🔧 命令: ./bin/dynamic2avif -input test/input -output test/output -dry-run"

./bin/dynamic2avif -input test/input -output test/output -dry-run

echo ""
echo "✅ 测试脚本执行完成"
echo "💡 要实际转换文件，请运行:"
echo "   ./bin/dynamic2avif -input test/input -output test/output"