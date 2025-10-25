#!/bin/bash
# 批量为归档工具添加CLI UI功能
# 基于static2jxl模板

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                                                               ║"
echo "║   🎨 批量为归档工具添加CLI UI                               ║"
echo "║                                                               ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

# 工具列表
TOOLS=(
  "static2avif:静态图→AVIF:JPG/PNG→AVIF:AVIF"
  "dynamic2avif:动图→AVIF:GIF/WebP→AVIF:AVIF"
  "dynamic2jxl:动图→JXL:GIF/WebP→JXL:JXL"
  "video2mov:视频→MOV:视频重封装:MOV"
)

# 处理每个工具
for TOOL_INFO in "${TOOLS[@]}"; do
  IFS=':' read -r TOOL_NAME TOOL_DESC TOOL_FORMAT TOOL_EXT <<< "$TOOL_INFO"
  
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "🔧 处理工具: $TOOL_NAME ($TOOL_DESC)"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  
  if [ ! -d "$TOOL_NAME" ]; then
    echo "❌ 工具目录不存在: $TOOL_NAME"
    continue
  fi
  
  cd "$TOOL_NAME"
  
  # 检查main.go是否已经有CLI UI
  if grep -q "runInteractiveMode" main.go 2>/dev/null; then
    echo "✅ $TOOL_NAME 已经有CLI UI，跳过"
    cd ..
    continue
  fi
  
  echo "📝 需要手动添加CLI UI到 $TOOL_NAME/main.go"
  echo "   使用static2jxl作为模板复制以下功能:"
  echo "   1. 添加 bufio 导入"
  echo "   2. 修改 main() 函数检测无参数"
  echo "   3. 添加 runInteractiveMode() 函数"
  echo "   4. 添加 runNonInteractiveMode_WithOpts() 函数"
  echo "   5. 添加 promptForDirectory() 等8个辅助函数"
  echo ""
  
  cd ..
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎉 检查完成！"
echo ""
echo "建议: 使用AI助手手动复制static2jxl的CLI UI代码到各工具"
echo "这样可以确保正确性并进行必要的调整"

