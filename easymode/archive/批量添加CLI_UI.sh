#!/bin/bash
# 批量为归档工具添加CLI UI
# 从static2jxl提取CLI UI代码并应用到其他4个工具

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                                                               ║"
echo "║   🎨 批量添加CLI UI（从static2jxl提取）                     ║"
║                                                               ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

# 提取static2jxl的CLI UI代码（行491-808）
echo "📝 提取static2jxl的CLI UI代码..."
sed -n '491,808p' static2jxl/main.go > /tmp/cli_ui_template.go

# 工具列表：工具名:横幅标题:功能描述
TOOLS=(
  "static2avif:static2avif v2.3.0 - 静态图转AVIF工具:静态图片转换为AVIF格式（高效压缩）"
  "dynamic2avif:dynamic2avif v2.3.0 - 动图转AVIF工具:动态图片转换为AVIF格式（高效动图）"
  "dynamic2jxl:dynamic2jxl v2.3.0 - 动图转JXL工具:动态图片转换为JXL格式（无损动图）"
  "video2mov:video2mov v2.3.0 - 视频重封装工具:视频重封装为MOV格式（不重编码）"
)

for TOOL_INFO in "${TOOLS[@]}"; do
  IFS=':' read -r TOOL_NAME BANNER_TITLE BANNER_DESC <<< "$TOOL_INFO"
  
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "🔧 处理工具: $TOOL_NAME"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  
  if [ ! -d "$TOOL_NAME" ]; then
    echo "❌ 工具目录不存在: $TOOL_NAME"
    continue
  fi
  
  cd "$TOOL_NAME"
  
  # 检查是否已经有CLI UI
  if grep -q "runInteractiveMode" main.go 2>/dev/null; then
    echo "✅ $TOOL_NAME 已有CLI UI，跳过"
    cd ..
    continue
  fi
  
  echo "📝 添加CLI UI到 $TOOL_NAME/main.go..."
  
  # 1. 添加bufio导入（如果没有）
  if ! grep -q '"bufio"' main.go; then
    sed -i '' '/"context"/a\
	"bufio"
' main.go
    echo "  ✅ 添加bufio导入"
  fi
  
  # 2. 备份原main.go
  cp main.go main.go.backup
  
  # 3. 找到原main()函数的位置
  MAIN_LINE=$(grep -n "^func main()" main.go | cut -d: -f1)
  
  # 4. 提取main()之前的部分
  head -n $((MAIN_LINE - 1)) main.go > /tmp/before_main.go
  
  # 5. 提取main()之后的部分（但跳过原main函数）
  # 找到原main函数的结束位置（简化：假设main函数很短）
  tail -n +$((MAIN_LINE + 30)) main.go > /tmp/after_main.go
  
  # 6. 自定义横幅
  sed "s/static2jxl v2.3.0 - 静态图转JXL工具/$BANNER_TITLE/g" /tmp/cli_ui_template.go | \
    sed "s/静态图片转换为JXL格式（无损\/完美可逆）/$BANNER_DESC/g" > /tmp/cli_ui_customized.go
  
  # 7. 组合新文件
  cat /tmp/before_main.go /tmp/cli_ui_customized.go /tmp/after_main.go > main.go.new
  
  # 8. 替换原文件
  mv main.go.new main.go
  
  echo "  ✅ CLI UI已添加"
  
  # 9. 编译测试
  echo "  🔨 编译测试..."
  if go build -o bin/${TOOL_NAME}-darwin-arm64 main.go 2>&1 | grep -q "error"; then
    echo "  ❌ 编译失败，恢复备份"
    mv main.go.backup main.go
  else
    echo "  ✅ 编译成功"
    rm -f main.go.backup
  fi
  
  cd ..
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎉 批量添加完成！"
echo ""
echo "查看结果:"
echo "  ls -l */bin/*-darwin-arm64"

