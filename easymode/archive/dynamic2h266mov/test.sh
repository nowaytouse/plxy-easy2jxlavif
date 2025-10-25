#!/bin/bash
# dynamic2h266mov测试脚本（H.266编码）

set -e

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                                                               ║"
echo "║   🧪 dynamic2h266mov测试（H.266编码）                       ║"
echo "║                                                               ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

# 检查FFmpeg版本
echo "🔍 检查FFmpeg版本..."
FFMPEG_VERSION=$(ffmpeg -version 2>/dev/null | head -1)
echo "$FFMPEG_VERSION"

# 检查H.266支持
echo ""
echo "🔍 检查H.266/VVC支持..."
if ffmpeg -codecs 2>/dev/null | grep -q "vvc"; then
    echo "✅ H.266/VVC编码器支持确认"
    ffmpeg -codecs 2>/dev/null | grep vvc
else
    echo "❌ 系统不支持H.266/VVC"
    echo "需要: FFmpeg 8.0+"
    echo "升级: brew upgrade ffmpeg"
    exit 1
fi

# 创建测试目录
TEST_DIR="/tmp/dynamic2h266mov_test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📝 准备测试文件..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 创建测试GIF
    ffmpeg -f lavfi -i testsrc=duration=2:size=320x240:rate=10 -pix_fmt rgb24 test.gif -y 2>/dev/null
exiftool -overwrite_original -Artist="H.266 Test" -Comment="Test H.266 Encoding" test.gif 2>/dev/null
touch -t 202401151030.00 test.gif

echo "✅ 测试GIF创建成功"
stat -f "  创建时间: %SB%n  修改时间: %Sm" test.gif
ls -lh test.gif

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎬 开始H.266转换..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 运行转换（使用非交互模式）
/Users/nyamiiko/Documents/git/plxy-easy2jxlavif/easymode/archive/dynamic2h266mov/bin/dynamic2h266mov-darwin-arm64 \
  -dir "$TEST_DIR" \
  -workers 1 \
  2>&1 | grep -E "H.266|动图转MOV成功|空间节省|文件系统元数据已保留|处理完成"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 验证转换结果..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ -f test.mov ]; then
    echo "✅ MOV文件创建成功！"
    echo ""
    
    echo "文件大小对比:"
    ORIG_SIZE=$(ls -lh test.gif | awk '{print $5}')
    NEW_SIZE=$(ls -lh test.mov | awk '{print $5}')
    echo "  原始GIF: $ORIG_SIZE"
    echo "  转换MOV: $NEW_SIZE"
    
    echo ""
    echo "时间戳验证:"
    stat -f "  创建: %SB%n  修改: %Sm" test.mov
    
    echo ""
    echo "EXIF元数据:"
    exiftool -Artist -Comment test.mov 2>/dev/null
    
    echo ""
    echo "视频信息:"
    ffprobe -v quiet -show_entries stream=codec_name,width,height,duration -of default=noprint_wrappers=1 test.mov 2>/dev/null
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # 检查时间戳是否保留
    ORIG_TIME=$(stat -f "%Sm" test.gif)
    NEW_TIME=$(stat -f "%Sm" test.mov)
    
    if [ "$ORIG_TIME" = "$NEW_TIME" ]; then
        echo "🎉 时间戳保留成功！"
    else
        echo "❌ 时间戳未保留"
        echo "  原始: $ORIG_TIME"
        echo "  转换: $NEW_TIME"
    fi
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "🎊 测试完成！"
    echo ""
    echo "查看结果: open $TEST_DIR"
else
    echo "❌ MOV文件未创建，转换失败"
    exit 1
fi
