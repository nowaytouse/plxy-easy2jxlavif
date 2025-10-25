#!/bin/bash
echo "==================================================================="
echo "  归档工具实际转换功能测试"
echo "==================================================================="
echo ""

TEST_DIR="/tmp/pixly_test_$$"
mkdir -p "$TEST_DIR"

# 创建测试PNG
echo "创建测试文件..."
ffmpeg -f lavfi -i "color=c=blue:s=200x200" -frames:v 1 "$TEST_DIR/test.png" -y 2>&1 | tail -1

# 创建测试GIF  
ffmpeg -f lavfi -i "color=c=red:s=100x100:d=1" -r 10 -t 1 "$TEST_DIR/test.gif" -y 2>&1 | tail -1

echo "✅ 测试文件创建完成"
echo ""

# 测试static2jxl
echo "-------------------------------------------------------------------"
echo "测试1: static2jxl (PNG → JXL)"
echo "-------------------------------------------------------------------"
cd static2jxl
./bin/static2jxl-darwin-arm64 -dir "$TEST_DIR/test.png" -output "$TEST_DIR" 2>&1 | grep -E "成功|失败|error" | tail -5

if [ -f "$TEST_DIR/test.jxl" ]; then
    echo "✅ static2jxl 转换成功"
    ls -lh "$TEST_DIR/test.png" "$TEST_DIR/test.jxl"
else
    echo "❌ static2jxl 转换失败"
fi
echo ""

# 测试dynamic2mov
echo "-------------------------------------------------------------------"
echo "测试2: dynamic2mov (GIF → H.265 MOV)"
echo "-------------------------------------------------------------------"
cd ../dynamic2mov
./bin/dynamic2mov-darwin-arm64 -dir "$TEST_DIR/test.gif" -output "$TEST_DIR" --codec h265 2>&1 | grep -E "成功|失败|error" | tail -5

if [ -f "$TEST_DIR/test.mov" ]; then
    echo "✅ dynamic2mov 转换成功"
    ls -lh "$TEST_DIR/test.gif" "$TEST_DIR/test.mov"
else
    echo "❌ dynamic2mov 转换失败"
fi
echo ""

# 总结
echo "==================================================================="
echo "测试总结:"
echo "==================================================================="
ls -lh "$TEST_DIR"
echo ""
echo "测试文件保存在: $TEST_DIR"
echo "可以手动查看验证"
