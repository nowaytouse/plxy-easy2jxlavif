#!/bin/bash

# 批量为归档工具添加交互CLI UI
# 使用static2jxl的成功模板

echo "🔨 批量为归档工具添加交互CLI UI..."
echo ""

# 从static2jxl提取交互模式代码
TEMPLATE_START_LINE=$(grep -n "^// runInteractiveMode " static2jxl/main.go | cut -d: -f1)
echo "📋 从static2jxl提取交互模式代码（从第${TEMPLATE_START_LINE}行开始）..."

# 提取所有交互模式相关函数
tail -n +${TEMPLATE_START_LINE} static2jxl/main.go > /tmp/interactive_functions.txt

echo "✅ 交互模式代码已提取: /tmp/interactive_functions.txt"
echo ""
echo "需要为以下工具添加交互模式："
echo "  1. static2avif"
echo "  2. dynamic2avif"  
echo "  3. dynamic2jxl"
echo "  4. video2mov"
echo ""

