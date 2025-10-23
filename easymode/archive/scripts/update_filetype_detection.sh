#!/bin/bash

# 批量更新所有脚本使用增强的文件类型检测

echo "🔧 开始更新所有脚本的文件类型检测..."

# 更新函数
update_script() {
    local script_name=$1
    local script_dir=$2
    
    echo "📦 更新 $script_name..."
    
    if [ -d "$script_dir" ]; then
        cd "$script_dir"
        
        # 更新go.mod
        go mod edit -replace pixly/utils=../utils
        go mod tidy
        
        # 编译测试
        go build -o "bin/$script_name" main.go
        
        if [ $? -eq 0 ]; then
            echo "✅ $script_name 更新成功"
        else
            echo "❌ $script_name 更新失败"
            return 1
        fi
        
        cd ..
    else
        echo "⚠️  目录不存在: $script_dir"
        return 1
    fi
}

# 更新所有脚本
echo "🚀 开始更新..."

# 1. static2avif
update_script "static2avif" "static2avif"

# 2. static2jxl
update_script "static2jxl" "static2jxl"

# 3. dynamic2avif
update_script "dynamic2avif" "dynamic2avif"

# 4. dynamic2jxl
update_script "dynamic2jxl" "dynamic2jxl"

# 5. video2mov
update_script "video2mov" "video2mov"

# 6. merge_xmp
update_script "merge_xmp" "merge_xmp"

# 7. deduplicate_media
update_script "deduplicate_media" "deduplicate_media"

echo ""
echo "🎉 所有脚本更新完成！"
