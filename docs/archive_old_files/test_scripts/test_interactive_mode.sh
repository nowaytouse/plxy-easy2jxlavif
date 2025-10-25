#!/bin/bash

echo "🧪 测试static2jxl交互模式"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 创建测试目录
TEST_DIR="/tmp/cli_ui_test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

# 创建测试文件
cd "$TEST_DIR"
convert -size 100x100 xc:green test.jpg 2>/dev/null || echo "TEST" > test.jpg
exiftool -overwrite_original -Artist="CLI Test" -CreateDate="2024:01:15 10:30:00" test.jpg 2>/dev/null
touch -t 202401151030.00 test.jpg

echo "📁 测试文件已创建: $TEST_DIR"
echo ""
echo "原始文件:"
stat -f "  创建: %SB, 修改: %Sm" test.jpg
exiftool -Artist test.jpg 2>/dev/null | grep "Artist"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎨 启动交互模式测试（模拟拖入）..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 使用echo模拟用户输入
echo "$TEST_DIR" | /Users/nyamiiko/Documents/git/plxy-easy2jxlavif/easymode/archive/static2jxl/bin/static2jxl-interactive 2>&1 | head -50

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 验证转换结果..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [ -f "$TEST_DIR/test.jxl" ]; then
    echo "✅ 转换成功！"
    echo ""
    echo "转换后文件:"
    stat -f "  创建: %SB, 修改: %Sm" test.jxl
    exiftool -Artist test.jxl 2>/dev/null | grep "Artist"
    echo ""
    
    ORIG_TIME=$(stat -f "%Sm" test.jpg)
    NEW_TIME=$(stat -f "%Sm" test.jxl)
    
    if [ "$ORIG_TIME" = "$NEW_TIME" ]; then
        echo "🎉 时间戳保留成功！"
    else
        echo "❌ 时间戳未保留"
        echo "  原始: $ORIG_TIME"
        echo "  转换: $NEW_TIME"
    fi
else
    echo "❌ 转换失败"
fi

echo ""
echo "测试目录: $TEST_DIR"
echo "在Finder中验证: open $TEST_DIR"

