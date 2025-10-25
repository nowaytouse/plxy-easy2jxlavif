#!/bin/bash

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                                                               ║"
echo "║   🧪 Archive工具完整测试 - 所有5个工具                     ║"
echo "║                                                               ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

# 测试目录
TEST_DIR="/tmp/archive_tools_test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# 创建测试图片
echo "📝 准备测试文件..."
convert -size 100x100 xc:blue test.jpg 2>/dev/null || echo "TEST" > test.jpg

# 添加EXIF元数据
exiftool -overwrite_original \
    -Artist="Archive Test Artist" \
    -Copyright="© 2024 Archive Test" \
    -Comment="Archive Tool Test" \
    -CreateDate="2024:01:15 10:30:00" \
    test.jpg 2>/dev/null

# 设置文件时间
touch -t 202401151030.00 test.jpg

# 添加Finder标签
xattr -w com.apple.metadata:_kMDItemUserTags '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><array><string>Red\n6</string></array></plist>' test.jpg 2>/dev/null

# 添加Finder注释
xattr -w com.apple.metadata:kMDItemFinderComment "Archive工具测试文件" test.jpg 2>/dev/null

echo "✅ 测试文件准备完成"
echo ""
echo "📊 原始文件状态:"
stat -f "  创建: %SB%n  修改: %Sm" test.jpg
exiftool -Artist -Copyright test.jpg 2>/dev/null | grep -v "File Name"
echo ""

ARCHIVE_DIR="/Users/nyamiiko/Documents/git/plxy-easy2jxlavif/easymode/archive"

# 测试工具列表
declare -A TOOLS
TOOLS=(
    ["static2avif"]="test_static2avif.jpg:test_static2avif.avif"
    ["static2jxl"]="test_static2jxl.jpg:test_static2jxl.jxl"
)

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 开始测试所有工具..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 测试static2avif
echo ""
echo "📦 测试工具: static2avif"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cp test.jpg test_static2avif.jpg
touch -t 202401151030.00 test_static2avif.jpg

if [ -f "$ARCHIVE_DIR/static2avif/bin/static2avif-darwin-arm64" ]; then
    echo "  🔄 转换中..."
    "$ARCHIVE_DIR/static2avif/bin/static2avif-darwin-arm64" -dir . -workers 1 2>&1 | grep -E "test_static2avif|文件系统元数据已保留" | tail -3
    
    if [ -f test_static2avif.avif ]; then
        echo ""
        echo "  📊 结果验证:"
        ORIG_TIME=$(stat -f "%Sm" test.jpg)
        NEW_TIME=$(stat -f "%Sm" test_static2avif.avif)
        
        echo "    原始: $ORIG_TIME"
        echo "    转换: $NEW_TIME"
        
        if [ "$ORIG_TIME" = "$NEW_TIME" ]; then
            echo "    ✅ static2avif - 时间戳保留成功！"
        else
            echo "    ❌ static2avif - 时间戳未保留"
        fi
    fi
else
    echo "  ⚠️  工具未找到"
fi

# 测试static2jxl
echo ""
echo "📦 测试工具: static2jxl"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cp test.jpg test_static2jxl.jpg
touch -t 202401151030.00 test_static2jxl.jpg

if [ -f "$ARCHIVE_DIR/static2jxl/bin/static2jxl-darwin-arm64" ]; then
    echo "  🔄 转换中..."
    "$ARCHIVE_DIR/static2jxl/bin/static2jxl-darwin-arm64" -dir . -workers 1 2>&1 | grep -E "test_static2jxl|文件系统元数据已保留" | tail -3
    
    if [ -f test_static2jxl.jxl ]; then
        echo ""
        echo "  📊 结果验证:"
        ORIG_TIME=$(stat -f "%Sm" test.jpg)
        NEW_TIME=$(stat -f "%Sm" test_static2jxl.jxl)
        
        echo "    原始: $ORIG_TIME"
        echo "    转换: $NEW_TIME"
        
        if [ "$ORIG_TIME" = "$NEW_TIME" ]; then
            echo "    ✅ static2jxl - 时间戳保留成功！"
        else
            echo "    ❌ static2jxl - 时间戳未保留"
        fi
    fi
else
    echo "  ⚠️  工具未找到"
fi

# 测试dynamic2avif (使用GIF)
echo ""
echo "📦 测试工具: dynamic2avif"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cp test.jpg test_dynamic2avif.gif
touch -t 202401151030.00 test_dynamic2avif.gif

if [ -f "$ARCHIVE_DIR/dynamic2avif/bin/dynamic2avif-darwin-arm64" ]; then
    echo "  🔄 转换中..."
    "$ARCHIVE_DIR/dynamic2avif/bin/dynamic2avif-darwin-arm64" -dir . -workers 1 2>&1 | grep -E "test_dynamic2avif|文件系统元数据已保留" | tail -3
    
    if [ -f test_dynamic2avif.avif ]; then
        echo ""
        echo "  📊 结果验证:"
        ORIG_TIME=$(stat -f "%Sm" test.jpg)
        NEW_TIME=$(stat -f "%Sm" test_dynamic2avif.avif)
        
        echo "    原始: $ORIG_TIME"
        echo "    转换: $NEW_TIME"
        
        if [ "$ORIG_TIME" = "$NEW_TIME" ]; then
            echo "    ✅ dynamic2avif - 时间戳保留成功！"
        else
            echo "    ❌ dynamic2avif - 时间戳未保留"
        fi
    fi
else
    echo "  ⚠️  工具未找到"
fi

# 测试dynamic2jxl (使用GIF)
echo ""
echo "📦 测试工具: dynamic2jxl"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cp test.jpg test_dynamic2jxl.gif
touch -t 202401151030.00 test_dynamic2jxl.gif

if [ -f "$ARCHIVE_DIR/dynamic2jxl/bin/dynamic2jxl-darwin-arm64" ]; then
    echo "  🔄 转换中..."
    "$ARCHIVE_DIR/dynamic2jxl/bin/dynamic2jxl-darwin-arm64" -dir . -workers 1 2>&1 | grep -E "test_dynamic2jxl|文件系统元数据已保留" | tail -3
    
    if [ -f test_dynamic2jxl.jxl ]; then
        echo ""
        echo "  📊 结果验证:"
        ORIG_TIME=$(stat -f "%Sm" test.jpg)
        NEW_TIME=$(stat -f "%Sm" test_dynamic2jxl.jxl)
        
        echo "    原始: $ORIG_TIME"
        echo "    转换: $NEW_TIME"
        
        if [ "$ORIG_TIME" = "$NEW_TIME" ]; then
            echo "    ✅ dynamic2jxl - 时间戳保留成功！"
        else
            echo "    ❌ dynamic2jxl - 时间戳未保留"
        fi
    fi
else
    echo "  ⚠️  工具未找到"
fi

# 测试video2mov (需要视频文件)
echo ""
echo "📦 测试工具: video2mov"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if command -v ffmpeg &> /dev/null; then
    echo "  🔄 创建测试视频..."
    ffmpeg -f lavfi -i testsrc=duration=1:size=320x240:rate=1 -pix_fmt yuv420p test_video.mp4 -y 2>/dev/null
    touch -t 202401151030.00 test_video.mp4
    exiftool -overwrite_original -Artist="Video Test" test_video.mp4 2>/dev/null
    
    if [ -f "$ARCHIVE_DIR/video2mov/bin/video2mov-darwin-arm64" ]; then
        echo "  🔄 转换中..."
        "$ARCHIVE_DIR/video2mov/bin/video2mov-darwin-arm64" -dir . -workers 1 2>&1 | grep -E "test_video|文件系统元数据已保留" | tail -3
        
        if [ -f test_video.mov ]; then
            echo ""
            echo "  📊 结果验证:"
            ORIG_TIME=$(stat -f "%Sm" test_video.mp4)
            NEW_TIME=$(stat -f "%Sm" test_video.mov)
            
            echo "    原始: $ORIG_TIME"
            echo "    转换: $NEW_TIME"
            
            if [ "$ORIG_TIME" = "$NEW_TIME" ]; then
                echo "    ✅ video2mov - 时间戳保留成功！"
            else
                echo "    ❌ video2mov - 时间戳未保留"
            fi
        fi
    else
        echo "  ⚠️  工具未找到"
    fi
else
    echo "  ⚠️  ffmpeg未安装，跳过视频测试"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 测试总结"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "测试文件位置: $TEST_DIR"
echo ""
echo "在Finder中验证："
echo "  open $TEST_DIR"
echo ""
echo "查看文件\"显示简介\"应该显示："
echo "  创建时间: 2024年1月15日 星期一 上午10:30"
echo "  修改时间: 2024年1月15日 星期一 上午10:30"
echo ""

