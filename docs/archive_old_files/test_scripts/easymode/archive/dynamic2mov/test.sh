#!/bin/bash

echo "🧪 测试gif2av1mov工具"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 创建测试目录
TEST_DIR="/tmp/gif2av1mov_test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# 创建测试GIF (使用ffmpeg)
echo "1️⃣ 创建测试GIF..."
if command -v ffmpeg &> /dev/null; then
    ffmpeg -f lavfi -i testsrc=duration=2:size=320x240:rate=10 -pix_fmt rgb24 test.gif -y 2>/dev/null
    echo "✅ 测试GIF创建成功"
else
    echo "❌ ffmpeg未安装"
    exit 1
fi

# 添加元数据
echo "2️⃣ 添加元数据..."
exiftool -overwrite_original -Artist="GIF Test Artist" -Comment="Test GIF" test.gif 2>/dev/null
touch -t 202401151030.00 test.gif

echo ""
echo "📊 原始文件:"
stat -f "  创建: %SB, 修改: %Sm" test.gif
exiftool -Artist test.gif 2>/dev/null | grep "Artist"
ls -lh test.gif

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3️⃣ 测试gif2av1mov转换..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 使用echo模拟拖入
echo "$TEST_DIR" | /Users/nyamiiko/Documents/git/plxy-easy2jxlavif/easymode/archive/gif2av1mov/bin/gif2av1mov-darwin-arm64 2>&1 | head -40

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 验证结果..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [ -f test.mov ]; then
    echo "✅ 转换成功！"
    echo ""
    echo "转换后文件:"
    stat -f "  创建: %SB, 修改: %Sm" test.mov
    exiftool -Artist test.mov 2>/dev/null | grep "Artist"
    ls -lh test.mov
    
    echo ""
    ORIG_TIME=$(stat -f "%Sm" test.gif)
    NEW_TIME=$(stat -f "%Sm" test.mov)
    
    if [ "$ORIG_TIME" = "$NEW_TIME" ]; then
        echo "🎉 时间戳保留成功！"
    else
        echo "⚠️  时间戳对比:"
        echo "  原始: $ORIG_TIME"
        echo "  转换: $NEW_TIME"
    fi
    
    echo ""
    echo "📊 文件大小对比:"
    ORIG_SIZE=$(ls -lh test.gif | awk '{print $5}')
    NEW_SIZE=$(ls -lh test.mov | awk '{print $5}')
    echo "  GIF: $ORIG_SIZE"
    echo "  MOV: $NEW_SIZE"
    
    echo ""
    echo "🎬 验证视频可播放:"
    if ffprobe -v error -show_entries format=duration,format_name -of default=noprint_wrappers=1:nokey=1 test.mov 2>/dev/null; then
        echo "  ✅ 视频格式正常"
    else
        echo "  ❌ 视频格式异常"
    fi
else
    echo "❌ 转换失败"
fi

echo ""
echo "测试目录: $TEST_DIR"
echo "在Finder中验证: open $TEST_DIR"

