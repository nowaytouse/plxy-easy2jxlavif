#!/bin/bash

TOOLS=("dynamic2jxl" "static2avif" "static2jxl" "video2mov")

echo "🔨 批量编译测试..."
echo ""

for TOOL in "${TOOLS[@]}"; do
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📦 $TOOL"
    
    cd "$TOOL"
    go mod tidy > /dev/null 2>&1
    
    if go build -o bin/${TOOL}-darwin-arm64 main.go 2>&1 | tee /tmp/${TOOL}_build.log | grep -i error; then
        echo "  ❌ 编译失败，详情: /tmp/${TOOL}_build.log"
        tail -10 /tmp/${TOOL}_build.log
    else
        SIZE=$(ls -lh bin/${TOOL}-darwin-arm64 2>/dev/null | awk '{print $5}')
        echo "  ✅ 编译成功 ($SIZE)"
    fi
    
    cd ..
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 批量编译测试完成！"
