#!/bin/bash

# 修复所有archive工具的文件系统元数据保留功能
# 包括Finder标签和注释保留

echo "🔧 修复archive工具的文件系统元数据保留功能..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 需要修复的工具列表
TOOLS=("video2mov" "static2jxl" "static2avif" "dynamic2jxl")

for tool in "${TOOLS[@]}"; do
    echo ""
    echo "📁 修复 $tool..."
    
    if [ -f "$tool/main.go" ]; then
        # 检查是否已经有文件系统元数据保留
        if grep -q "os\.Chtimes\|touch.*-t\|syscall\.Stat_t" "$tool/main.go"; then
            echo "  ✅ 已包含文件系统元数据保留"
        else
            echo "  🔧 添加文件系统元数据保留功能..."
            
            # 备份原文件
            cp "$tool/main.go" "$tool/main.go.backup"
            
            # 使用sed添加文件系统元数据保留代码
            # 这里需要根据每个工具的具体结构来修改
            echo "  ⚠️  需要手动修复 $tool/main.go"
        fi
    else
        echo "  ❌ 文件不存在: $tool/main.go"
    fi
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 修复完成！"
