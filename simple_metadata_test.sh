#!/bin/bash

echo "🧪 简单元数据保留测试"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 测试目录
TEST_DIR="/tmp/simple_metadata_test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# 创建测试图片
echo "1️⃣ 创建测试图片..."
convert -size 100x100 xc:red test.jpg 2>/dev/null || {
    echo "TEST" > test.jpg
}

# 添加EXIF
echo "2️⃣ 添加EXIF元数据..."
exiftool -overwrite_original \
    -Artist="Test Artist" \
    -Copyright="Test Copyright" \
    -CreateDate="2024:01:15 10:30:00" \
    test.jpg 2>/dev/null

# 设置文件时间
echo "3️⃣ 设置文件时间为2024-01-15 10:30..."
touch -t 202401151030.00 test.jpg

echo ""
echo "📊 原始文件状态:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
stat -f "创建: %SB%n修改: %Sm" test.jpg
exiftool -Artist -Copyright -CreateDate test.jpg 2>/dev/null | grep -v "File Name"

echo ""
echo "4️⃣ 使用static2avif转换..."
/Users/nyamiiko/Documents/git/plxy-easy2jxlavif/easymode/archive/static2avif/bin/static2avif-darwin-arm64 \
    -dir "$TEST_DIR" \
    -workers 1 2>&1 | grep -E "test.avif|✅|文件系统元数据"

echo ""
if [ -f test.avif ]; then
    echo "✅ 转换成功！"
    echo ""
    echo "📊 转换后文件状态:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    stat -f "创建: %SB%n修改: %Sm" test.avif
    exiftool -Artist -Copyright -CreateDate test.avif 2>/dev/null | grep -v "File Name"
    
    echo ""
    echo "📊 对比结果:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    ORIG_TIME=$(stat -f "%Sm" test.jpg)
    NEW_TIME=$(stat -f "%Sm" test.avif)
    
    echo "原始修改时间: $ORIG_TIME"
    echo "转换后修改时间: $NEW_TIME"
    
    if [ "$ORIG_TIME" = "$NEW_TIME" ]; then
        echo ""
        echo "🎉 成功！文件系统时间戳完全保留！"
    else
        echo ""
        echo "❌ 失败！时间戳未保留"
    fi
else
    echo "❌ 转换失败"
fi

echo ""
echo "测试文件位置: $TEST_DIR"
echo "请在Finder中验证：右键 → 显示简介"

