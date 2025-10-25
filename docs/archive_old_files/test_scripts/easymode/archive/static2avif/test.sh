#!/bin/bash

# static2avif 测试脚本

echo "========================================="
echo "static2avif 测试脚本"
echo "========================================="

# 创建测试目录
echo "📂 创建测试目录..."
mkdir -p test/input
mkdir -p test/output

# 创建测试用的JPEG文件（如果系统支持）
echo "🧪 创建测试文件..."

# 创建一个简单的测试JPEG文件（如果ImageMagick可用）
if command -v convert &> /dev/null
then
    echo "🔄 使用ImageMagick创建测试JPEG..."
    # 创建一个简单的JPEG图像
    convert -size 100x100 xc:red -quality 90 test/input/test.jpg
    
    if [ $? -eq 0 ]; then
        echo "✅ 测试JPEG文件创建成功"
    else
        echo "⚠️  测试JPEG文件创建失败"
    fi
else
    echo "⚠️  ImageMagick未安装，跳过测试文件创建"
    echo "💡 提示: 您可以手动将一些静态图片放入 test/input 目录进行测试"
fi

echo "🚀 运行static2avif工具..."
echo "🔧 命令: ./bin/static2avif -input test/input -output test/output -dry-run"

./bin/static2avif -input test/input -output test/output -dry-run

echo ""
echo "✅ 测试脚本执行完成"
echo "💡 要实际转换文件，请运行:"
echo "   ./bin/static2avif -input test/input -output test/output"