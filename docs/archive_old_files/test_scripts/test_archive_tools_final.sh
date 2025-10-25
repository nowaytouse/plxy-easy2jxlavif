#!/bin/bash
echo "==================================================================="
echo "  归档工具实际转换功能验证"
echo "==================================================================="
echo ""

TEST_DIR="/tmp/pixly_archive_test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR/input_static" "$TEST_DIR/input_dynamic" "$TEST_DIR/output"

# 创建测试文件
echo "创建测试文件..."
ffmpeg -f lavfi -i "color=c=blue:s=200x200" -frames:v 1 "$TEST_DIR/input_static/test.png" -y 2>&1 > /dev/null
ffmpeg -f lavfi -i "color=c=red:s=100x100:d=1" -r 10 -t 1 "$TEST_DIR/input_dynamic/test.gif" -y 2>&1 > /dev/null

echo "✅ 测试文件已创建"
echo "   静态图片: $TEST_DIR/input_static/test.png"
echo "   动态图片: $TEST_DIR/input_dynamic/test.gif"
echo ""

# 测试1: static2jxl
echo "-------------------------------------------------------------------"
echo "测试1: static2jxl (PNG → JPEG XL)"
echo "-------------------------------------------------------------------"
cd easymode/archive/static2jxl
./bin/static2jxl-darwin-arm64 -dir "$TEST_DIR/input_static" -output "$TEST_DIR/output" 2>&1 | tail -15

if [ -f "$TEST_DIR/output/test.jxl" ]; then
    echo ""
    echo "✅ static2jxl 转换成功!"
    echo "   输入: $(ls -lh $TEST_DIR/input_static/test.png | awk '{print $5}')"
    echo "   输出: $(ls -lh $TEST_DIR/output/test.jxl | awk '{print $5}')"
else
    echo "❌ 转换失败 - 未找到输出文件"
fi

echo ""
echo "-------------------------------------------------------------------"
echo "测试2: dynamic2mov (GIF → H.265 MOV)"
echo "-------------------------------------------------------------------"
cd ../dynamic2mov
./bin/dynamic2mov-darwin-arm64 -dir "$TEST_DIR/input_dynamic" -output "$TEST_DIR/output" --codec h265 2>&1 | tail -15

if [ -f "$TEST_DIR/output/test.mov" ]; then
    echo ""
    echo "✅ dynamic2mov 转换成功!"
    echo "   输入: $(ls -lh $TEST_DIR/input_dynamic/test.gif | awk '{print $5}')"
    echo "   输出: $(ls -lh $TEST_DIR/output/test.mov | awk '{print $5}')"
    CODEC=$(ffprobe -v quiet -show_streams -select_streams v:0 "$TEST_DIR/output/test.mov" | grep codec_name | cut -d= -f2)
    echo "   编码器: $CODEC"
else
    echo "❌ 转换失败 - 未找到输出文件"
fi

echo ""
echo "==================================================================="
echo "测试结果:"
echo "==================================================================="
ls -lh "$TEST_DIR/output/" 2>/dev/null
echo ""
echo "测试目录: $TEST_DIR"
