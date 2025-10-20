#!/bin/bash

# 表情包测试脚本
# 用于测试easymode工具的功能

echo "🎭 开始表情包测试..."

# 创建测试目录
TEST_DIR="/tmp/emoji_test"
ORIGINAL_DIR="/Users/nyamiiko/Documents/git/easy2jxlavif-beta/表情包"

echo "📁 创建测试目录: $TEST_DIR"
mkdir -p "$TEST_DIR"

# 复制部分表情包文件进行测试
echo "📋 复制测试文件..."
find "$ORIGINAL_DIR" -name "*.jpg" -o -name "*.jpeg" -o -name "*.png" -o -name "*.gif" -o -name "*.webp" | head -20 | while read file; do
    cp "$file" "$TEST_DIR/"
done

echo "✅ 测试文件准备完成"
echo "📊 测试文件数量: $(find "$TEST_DIR" -type f | wc -l)"

# 测试静态图片转AVIF
echo "🖼️  测试静态图片转AVIF..."
cd /Users/nyamiiko/Documents/git/easy2jxlavif-beta/easymode/static2avif
./static2avif -input "$TEST_DIR" -output "$TEST_DIR/static_avif" -workers 4 -quality 80

# 测试动态图片转AVIF
echo "🎬 测试动态图片转AVIF..."
cd /Users/nyamiiko/Documents/git/easy2jxlavif-beta/easymode/dynamic2avif
./dynamic2avif -input "$TEST_DIR" -output "$TEST_DIR/dynamic_avif" -workers 4 -quality 80

# 测试静态图片转JXL
echo "🖼️  测试静态图片转JXL..."
cd /Users/nyamiiko/Documents/git/easy2jxlavif-beta/easymode/static2jxl
go run main.go -input "$TEST_DIR" -output "$TEST_DIR/static_jxl" -workers 4

# 测试动态图片转JXL
echo "🎬 测试动态图片转JXL..."
cd /Users/nyamiiko/Documents/git/easy2jxlavif-beta/easymode/dynamic2jxl
go run main.go -input "$TEST_DIR" -output "$TEST_DIR/dynamic_jxl" -workers 4

echo "🎉 测试完成!"
echo "📁 测试结果目录: $TEST_DIR"
echo "📊 结果统计:"
echo "   原始文件: $(find "$TEST_DIR" -maxdepth 1 -type f | wc -l)"
echo "   静态AVIF: $(find "$TEST_DIR/static_avif" -type f 2>/dev/null | wc -l)"
echo "   动态AVIF: $(find "$TEST_DIR/dynamic_avif" -type f 2>/dev/null | wc -l)"
echo "   静态JXL: $(find "$TEST_DIR/static_jxl" -type f 2>/dev/null | wc -l)"
echo "   动态JXL: $(find "$TEST_DIR/dynamic_jxl" -type f 2>/dev/null | wc -l)"
