#!/bin/bash

# 闷茶子测试脚本
# 用于测试优化后的Pixly主程序功能

echo "🎭 开始闷茶子测试..."

# 设置测试目录
TEST_DIR="/Users/nyamiiko/Documents/git/easy2jxlavif-beta/闷茶子"
ORIGINAL_DIR="/Users/nyamiiko/Documents/git/easy2jxlavif-beta/闷茶子"

echo "📁 测试目录: $TEST_DIR"

# 检查目录是否存在
if [ ! -d "$TEST_DIR" ]; then
    echo "❌ 错误: 测试目录不存在: $TEST_DIR"
    exit 1
fi

# 统计原始文件
echo "📊 原始文件统计:"
echo "   总文件数: $(find "$TEST_DIR" -type f | wc -l)"
echo "   JPG文件: $(find "$TEST_DIR" -name "*.jpg" -o -name "*.jpeg" | wc -l)"
echo "   PNG文件: $(find "$TEST_DIR" -name "*.png" | wc -l)"
echo "   GIF文件: $(find "$TEST_DIR" -name "*.gif" | wc -l)"

# 计算总大小
TOTAL_SIZE=$(find "$TEST_DIR" -type f -exec du -ch {} + | tail -1 | cut -f1)
echo "   总大小: $TOTAL_SIZE"

echo ""
echo "🔍 开始智能扫描测试..."

# 测试优化后的主程序
cd /Users/nyamiiko/Documents/git/easy2jxlavif-beta

# 检查优化后的主程序是否存在
if [ ! -f "main_optimized.go" ]; then
    echo "❌ 错误: 优化后的主程序不存在"
    exit 1
fi

# 构建优化后的主程序
echo "🔨 构建优化后的主程序..."
go build -o pixly_optimized main_optimized.go
if [ $? -ne 0 ]; then
    echo "❌ 构建失败"
    exit 1
fi

echo "✅ 构建成功"

# 测试智能扫描功能
echo ""
echo "🧠 测试智能扫描功能..."
./pixly_optimized -dir "$TEST_DIR" -debug -non-interactive -format auto
SCAN_RESULT=$?

if [ $SCAN_RESULT -eq 0 ]; then
    echo "✅ 智能扫描测试通过"
else
    echo "❌ 智能扫描测试失败"
fi

# 测试JXL转换
echo ""
echo "🖼️  测试JXL转换..."
./pixly_optimized -dir "$TEST_DIR" -format jxl -non-interactive -quality high
JXL_RESULT=$?

if [ $JXL_RESULT -eq 0 ]; then
    echo "✅ JXL转换测试通过"
else
    echo "❌ JXL转换测试失败"
fi

# 测试AVIF转换
echo ""
echo "🎬 测试AVIF转换..."
./pixly_optimized -dir "$TEST_DIR" -format avif -non-interactive -quality medium
AVIF_RESULT=$?

if [ $AVIF_RESULT -eq 0 ]; then
    echo "✅ AVIF转换测试通过"
else
    echo "❌ AVIF转换测试失败"
fi

# 测试表情包模式
echo ""
echo "😊 测试表情包模式..."
./pixly_optimized -dir "$TEST_DIR" -sticker -non-interactive -format auto
STICKER_RESULT=$?

if [ $STICKER_RESULT -eq 0 ]; then
    echo "✅ 表情包模式测试通过"
else
    echo "❌ 表情包模式测试失败"
fi

# 统计转换结果
echo ""
echo "📊 转换结果统计:"
echo "   JXL文件: $(find "$TEST_DIR" -name "*.jxl" | wc -l)"
echo "   AVIF文件: $(find "$TEST_DIR" -name "*.avif" | wc -l)"

# 计算压缩效果
if [ -f "$TEST_DIR"/*.jxl ]; then
    JXL_SIZE=$(find "$TEST_DIR" -name "*.jxl" -exec du -ch {} + | tail -1 | cut -f1)
    echo "   JXL总大小: $JXL_SIZE"
fi

if [ -f "$TEST_DIR"/*.avif ]; then
    AVIF_SIZE=$(find "$TEST_DIR" -name "*.avif" -exec du -ch {} + | tail -1 | cut -f1)
    echo "   AVIF总大小: $AVIF_SIZE"
fi

# 检查状态数据库
echo ""
echo "🗄️  检查状态数据库..."
if [ -f ~/.pixly/state.db ]; then
    echo "✅ 状态数据库已创建"
    echo "   数据库大小: $(du -h ~/.pixly/state.db | cut -f1)"
else
    echo "⚠️  状态数据库未找到"
fi

# 生成测试报告
echo ""
echo "📝 生成测试报告..."
REPORT_FILE="test_report_menchazi_$(date +%Y%m%d_%H%M%S).txt"

cat > "$REPORT_FILE" << EOF
闷茶子测试报告
================
测试时间: $(date)
测试目录: $TEST_DIR

原始文件统计:
- 总文件数: $(find "$TEST_DIR" -type f | wc -l)
- JPG文件: $(find "$TEST_DIR" -name "*.jpg" -o -name "*.jpeg" | wc -l)
- PNG文件: $(find "$TEST_DIR" -name "*.png" | wc -l)
- GIF文件: $(find "$TEST_DIR" -name "*.gif" | wc -l)
- 总大小: $TOTAL_SIZE

测试结果:
- 智能扫描: $([ $SCAN_RESULT -eq 0 ] && echo "通过" || echo "失败")
- JXL转换: $([ $JXL_RESULT -eq 0 ] && echo "通过" || echo "失败")
- AVIF转换: $([ $AVIF_RESULT -eq 0 ] && echo "通过" || echo "失败")
- 表情包模式: $([ $STICKER_RESULT -eq 0 ] && echo "通过" || echo "失败")

转换结果:
- JXL文件: $(find "$TEST_DIR" -name "*.jxl" | wc -l)
- AVIF文件: $(find "$TEST_DIR" -name "*.avif" | wc -l)

状态数据库:
- 存在: $([ -f ~/.pixly/state.db ] && echo "是" || echo "否")
- 大小: $([ -f ~/.pixly/state.db ] && du -h ~/.pixly/state.db | cut -f1 || echo "N/A")
EOF

echo "✅ 测试报告已生成: $REPORT_FILE"

# 清理临时文件
echo ""
echo "🧹 清理临时文件..."
rm -f pixly_optimized

echo "🎉 闷茶子测试完成!"
echo "📊 总体结果:"
echo "   智能扫描: $([ $SCAN_RESULT -eq 0 ] && echo "✅ 通过" || echo "❌ 失败")"
echo "   JXL转换: $([ $JXL_RESULT -eq 0 ] && echo "✅ 通过" || echo "❌ 失败")"
echo "   AVIF转换: $([ $AVIF_RESULT -eq 0 ] && echo "✅ 通过" || echo "❌ 失败")"
echo "   表情包模式: $([ $STICKER_RESULT -eq 0 ] && echo "✅ 通过" || echo "❌ 失败")"
