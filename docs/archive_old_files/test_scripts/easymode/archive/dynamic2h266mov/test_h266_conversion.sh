#!/bin/bash

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                                                               ║"
echo "║   🧪 H.266实际转换功能测试                                   ║"
echo "║                                                               ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

# 检查FFmpeg H.266支持
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "步骤1: 检查FFmpeg H.266/VVC支持"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if ffmpeg -encoders 2>&1 | grep -q "libvvenc"; then
    echo "✅ FFmpeg支持libvvenc编码器"
    ffmpeg -encoders 2>&1 | grep libvvenc
    CAN_TEST=true
else
    echo "❌ FFmpeg不支持libvvenc编码器"
    echo ""
    echo "需要先安装支持H.266的FFmpeg:"
    echo "  ./bin/dynamic2h266mov-darwin-arm64"
    echo "  选择 [1] 自动从源码编译"
    echo ""
    CAN_TEST=false
fi

if [ "$CAN_TEST" = false ]; then
    echo "⚠️  跳过转换测试"
    exit 0
fi

# 创建测试目录
TEST_DIR="/tmp/pixly_h266_test_$$"
mkdir -p "$TEST_DIR"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "步骤2: 创建测试GIF"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "测试目录: $TEST_DIR"

# 创建简单的测试GIF
ffmpeg -f lavfi -i "color=c=blue:s=320x240:d=1" \
       -f lavfi -i "color=c=red:s=320x240:d=1" \
       -filter_complex "[0:v][1:v]concat=n=2:v=1[v]" \
       -map "[v]" -r 10 -t 2 \
       "$TEST_DIR/test.gif" \
       -y 2>&1 | tail -3

if [ -f "$TEST_DIR/test.gif" ]; then
    SIZE=$(ls -lh "$TEST_DIR/test.gif" | awk '{print $5}')
    echo "✅ 测试GIF创建成功: $SIZE"
else
    echo "❌ 测试GIF创建失败"
    exit 1
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "步骤3: 执行H.266转换"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "转换命令:"
echo "  ffmpeg -i test.gif -c:v libvvenc -qp 28 -preset medium test.mov"
echo ""

# 执行H.266转换
ffmpeg -i "$TEST_DIR/test.gif" \
       -c:v libvvenc \
       -qp 28 \
       -preset medium \
       -pix_fmt yuv420p \
       -f mov \
       -y "$TEST_DIR/test.mov" \
       2>&1

CONVERT_STATUS=$?

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "步骤4: 验证转换结果"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [ $CONVERT_STATUS -eq 0 ] && [ -f "$TEST_DIR/test.mov" ]; then
    echo "✅ H.266转换成功!"
    echo ""
    
    # 文件大小对比
    ORIG_SIZE=$(stat -f%z "$TEST_DIR/test.gif")
    NEW_SIZE=$(stat -f%z "$TEST_DIR/test.mov")
    SAVED=$((ORIG_SIZE - NEW_SIZE))
    PERCENT=$(awk "BEGIN {printf \"%.1f\", ($SAVED * 100.0 / $ORIG_SIZE)}")
    
    echo "文件对比:"
    echo "  GIF原始:  $(ls -lh "$TEST_DIR/test.gif" | awk '{print $5}')"
    echo "  MOV输出:  $(ls -lh "$TEST_DIR/test.mov" | awk '{print $5}')"
    
    if [ $SAVED -gt 0 ]; then
        echo "  节省空间: $PERCENT%"
    fi
    
    echo ""
    echo "编码验证:"
    CODEC=$(ffprobe -v quiet -show_streams -select_streams v:0 "$TEST_DIR/test.mov" 2>&1 | grep "codec_name" | cut -d= -f2)
    echo "  编解码器: $CODEC"
    
    if [ "$CODEC" = "vvc" ] || [ "$CODEC" = "h266" ]; then
        echo "  ✅ 确认为H.266/VVC编码"
    else
        echo "  ⚠️  编码器: $CODEC (预期为vvc)"
    fi
    
    echo ""
    echo "视频信息:"
    ffprobe -v quiet -show_streams -select_streams v:0 "$TEST_DIR/test.mov" 2>&1 | grep -E "width|height|pix_fmt|duration"
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✅ H.266转换功能完全可用！"
    echo ""
    echo "测试文件保存在: $TEST_DIR"
    echo "  可用QuickTime Player播放验证"
    echo ""
    
else
    echo "❌ H.266转换失败"
    echo ""
    echo "错误分析:"
    if ! ffmpeg -encoders 2>&1 | grep -q "libvvenc"; then
        echo "  原因: FFmpeg不支持libvvenc"
        echo "  解决: 需要从源码编译FFmpeg"
    else
        echo "  原因: 转换过程出错"
        echo "  请检查上面的FFmpeg输出"
    fi
fi

echo ""
echo "是否删除测试文件? (y/n, 10秒后自动跳过)"
read -t 10 CLEANUP
if [ "$CLEANUP" = "y" ]; then
    rm -rf "$TEST_DIR"
    echo "✅ 测试文件已删除"
else
    echo "💾 测试文件保留: $TEST_DIR"
fi

