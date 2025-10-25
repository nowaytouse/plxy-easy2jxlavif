#!/bin/bash

# 修复所有archive工具的文件系统时间戳保留问题
# 问题：exiftool会改变文件修改时间，所以必须在exiftool之后再恢复时间戳

echo "🔧 修复所有archive工具的时间戳保留顺序..."
echo ""

cd /Users/nyamiiko/Documents/git/plxy-easy2jxlavif/easymode/archive

# 需要修复的工具
TOOLS=(
    "dynamic2avif"
    "video2mov"
    "static2jxl"
    "static2avif"
    "dynamic2jxl"
)

for tool in "${TOOLS[@]}"; do
    echo "📦 修复 $tool..."
    
    if [ ! -f "$tool/main.go" ]; then
        echo "  ❌ 文件不存在"
        continue
    fi
    
    # 备份
    cp "$tool/main.go" "$tool/main.go.time_fix_backup"
    
    # 关键修复：
    # 1. 先捕获时间戳（在exiftool之前）
    # 2. 执行exiftool（会改变修改时间）
    # 3. 使用touch恢复时间戳（在exiftool之后）
    
    echo "  ✅ 已创建备份: $tool/main.go.time_fix_backup"
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "⚠️  需要手动修改每个文件，确保顺序："
echo "   1. 先捕获源文件时间戳"  
echo "   2. 执行exiftool复制EXIF"
echo "   3. 执行xattr复制Finder元数据"
echo "   4. 最后执行touch恢复时间戳"
echo ""
echo "static2avif已经修复，可以作为参考模板"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

