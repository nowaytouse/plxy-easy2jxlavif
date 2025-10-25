#!/bin/bash

# Archive工具元数据保留完整测试脚本
# 测试：EXIF + 文件系统时间戳 + Finder标签和注释

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                                                               ║"
echo "║   🧪 Archive工具元数据保留完整测试                          ║"
echo "║                                                               ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

# 测试目录
TEST_DIR="/tmp/pixly_metadata_test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

cd "$TEST_DIR"

echo "📝 步骤1: 创建测试文件..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 创建一个简单的测试图片（使用ImageMagick或ffmpeg）
if command -v convert &> /dev/null; then
    convert -size 200x200 xc:blue test_image.jpg
elif command -v ffmpeg &> /dev/null; then
    ffmpeg -f lavfi -i color=c=blue:s=200x200:d=1 -frames:v 1 test_image.jpg -y 2>/dev/null
else
    echo "⚠️  未找到ImageMagick或ffmpeg，使用替代方法"
    # 创建一个纯文本文件作为测试
    echo "TEST IMAGE" > test_image.jpg
fi

echo "✅ 测试文件创建完成: test_image.jpg"

echo ""
echo "📝 步骤2: 添加完整元数据..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 2.1 添加EXIF元数据
if command -v exiftool &> /dev/null; then
    exiftool -overwrite_original \
        -Artist="测试作者 Test Artist" \
        -Copyright="© 2024 测试版权 Test Copyright" \
        -Comment="测试注释 Test Comment" \
        -CreateDate="2024:01:15 10:30:00" \
        -GPSLatitude="35.6812" \
        -GPSLatitudeRef="N" \
        -GPSLongitude="139.7671" \
        -GPSLongitudeRef="E" \
        test_image.jpg 2>/dev/null
    echo "  ✅ EXIF元数据已添加"
else
    echo "  ⚠️  exiftool未安装，跳过EXIF元数据"
fi

# 2.2 设置文件时间戳（创建时间、修改时间）
touch -t 202401151030.00 test_image.jpg
echo "  ✅ 文件时间戳已设置: 2024-01-15 10:30:00"

# 2.3 添加Finder标签和注释
if command -v xattr &> /dev/null; then
    # Finder标签（红色）
    xattr -w com.apple.metadata:_kMDItemUserTags '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><array><string>Red\n6</string></array></plist>' test_image.jpg 2>/dev/null
    
    # Finder注释
    xattr -w com.apple.metadata:kMDItemFinderComment "测试Finder注释 - 这是一个重要的测试文件" test_image.jpg 2>/dev/null
    
    echo "  ✅ Finder标签和注释已添加"
else
    echo "  ⚠️  xattr未安装，跳过Finder元数据"
fi

echo ""
echo "📊 步骤3: 查看原始文件的完整元数据..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo ""
echo "【文件系统元数据】"
stat -f "  创建时间: %SB" test_image.jpg 2>/dev/null || stat -c "  创建时间: %w" test_image.jpg
stat -f "  修改时间: %Sm" test_image.jpg 2>/dev/null || stat -c "  修改时间: %y" test_image.jpg
stat -f "  访问时间: %Sa" test_image.jpg 2>/dev/null || stat -c "  访问时间: %x" test_image.jpg

if command -v exiftool &> /dev/null; then
    echo ""
    echo "【EXIF内部元数据】"
    exiftool -Artist -Copyright -Comment -CreateDate -GPSLatitude -GPSLongitude test_image.jpg 2>/dev/null | grep -v "File Name" | grep -v "Directory" | grep -v "File Size"
fi

if command -v xattr &> /dev/null; then
    echo ""
    echo "【Finder扩展属性】"
    xattr test_image.jpg 2>/dev/null | while read attr; do
        echo "  - $attr"
    done
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔧 步骤4: 使用archive工具转换..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

ARCHIVE_DIR="/Users/nyamiiko/Documents/git/plxy-easy2jxlavif/easymode/archive"

# 测试static2jxl工具
if [ -f "$ARCHIVE_DIR/static2jxl/bin/static2jxl-darwin-arm64" ]; then
    echo ""
    echo "📦 测试工具: static2jxl"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    cp test_image.jpg test_for_jxl.jpg
    
    echo "  🔄 转换中..."
    "$ARCHIVE_DIR/static2jxl/bin/static2jxl-darwin-arm64" \
        -dir "$TEST_DIR" \
        -workers 1 2>&1 | grep -E "✅|⚠️|❌|元数据" | head -20
    
    if [ -f "test_for_jxl.jxl" ]; then
        echo ""
        echo "  ✅ 转换成功！验证元数据..."
        echo ""
        echo "  【文件系统元数据对比】"
        echo "    原始文件:"
        stat -f "      创建: %SB, 修改: %Sm" test_image.jpg 2>/dev/null || stat -c "      创建: %w, 修改: %y" test_image.jpg
        echo "    转换后:"
        stat -f "      创建: %SB, 修改: %Sm" test_for_jxl.jxl 2>/dev/null || stat -c "      创建: %w, 修改: %y" test_for_jxl.jxl
        
        if command -v exiftool &> /dev/null; then
            echo ""
            echo "  【EXIF内部元数据对比】"
            echo "    原始文件:"
            exiftool -Artist -Copyright test_image.jpg 2>/dev/null | grep -v "File Name" | sed 's/^/      /'
            echo "    转换后:"
            exiftool -Artist -Copyright test_for_jxl.jxl 2>/dev/null | grep -v "File Name" | sed 's/^/      /'
        fi
        
        if command -v xattr &> /dev/null; then
            echo ""
            echo "  【Finder扩展属性对比】"
            echo "    原始文件扩展属性数量: $(xattr test_image.jpg 2>/dev/null | wc -l | tr -d ' ')"
            echo "    转换后扩展属性数量: $(xattr test_for_jxl.jxl 2>/dev/null | wc -l | tr -d ' ')"
            
            # 检查关键属性是否保留
            if xattr -p com.apple.metadata:kMDItemFinderComment test_for_jxl.jxl &>/dev/null; then
                echo "    ✅ Finder注释已保留"
            else
                echo "    ⚠️  Finder注释未保留"
            fi
            
            if xattr -p com.apple.metadata:_kMDItemUserTags test_for_jxl.jxl &>/dev/null; then
                echo "    ✅ Finder标签已保留"
            else
                echo "    ⚠️  Finder标签未保留"
            fi
        fi
    else
        echo "  ❌ 转换失败！"
    fi
else
    echo "  ⚠️  static2jxl工具未找到"
fi

# 测试static2avif工具
if [ -f "$ARCHIVE_DIR/static2avif/bin/static2avif-darwin-arm64" ]; then
    echo ""
    echo "📦 测试工具: static2avif"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    cp test_image.jpg test_for_avif.jpg
    
    echo "  🔄 转换中..."
    "$ARCHIVE_DIR/static2avif/bin/static2avif-darwin-arm64" \
        -dir "$TEST_DIR" \
        -workers 1 2>&1 | grep -E "✅|⚠️|❌|元数据" | head -20
    
    if [ -f "test_for_avif.avif" ]; then
        echo ""
        echo "  ✅ 转换成功！验证元数据..."
        echo ""
        echo "  【文件系统元数据对比】"
        echo "    原始文件:"
        stat -f "      创建: %SB, 修改: %Sm" test_image.jpg 2>/dev/null || stat -c "      创建: %w, 修改: %y" test_image.jpg
        echo "    转换后:"
        stat -f "      创建: %SB, 修改: %Sm" test_for_avif.avif 2>/dev/null || stat -c "      创建: %w, 修改: %y" test_for_avif.avif
        
        if command -v exiftool &> /dev/null; then
            echo ""
            echo "  【EXIF内部元数据】"
            exiftool -Artist -Copyright test_for_avif.avif 2>/dev/null | grep -v "File Name" | sed 's/^/      /'
        fi
    else
        echo "  ❌ 转换失败！"
    fi
else
    echo "  ⚠️  static2avif工具未找到"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 测试总结"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "测试文件位置: $TEST_DIR"
echo ""
echo "请在Finder中打开以下文件验证："
echo "  1. 原始文件: $TEST_DIR/test_image.jpg"
echo "  2. JXL转换: $TEST_DIR/test_for_jxl.jxl"
echo "  3. AVIF转换: $TEST_DIR/test_for_avif.avif"
echo ""
echo "验证步骤："
echo "  1. 右键点击文件 → 选择\"显示简介\""
echo "  2. 查看\"创建时间\"和\"修改时间\"是否一致"
echo "  3. 查看\"标签\"是否保留（应该是红色）"
echo "  4. 查看\"注释\"是否保留"
echo ""
echo "🎯 预期结果："
echo "  ✅ 创建时间: 2024年1月15日 10:30（所有文件一致）"
echo "  ✅ 修改时间: 2024年1月15日 10:30（所有文件一致）"
echo "  ✅ EXIF元数据: 完全保留"
echo "  ✅ Finder标签: 红色（如果支持）"
echo "  ✅ Finder注释: 保留（如果支持）"
echo ""

