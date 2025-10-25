#!/bin/bash

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                                                               ║"
echo "║   🧹 整理备份文件和临时文件                                  ║"
echo "║                                                               ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

# 创建整理目录
ARCHIVE_DIR="archive_old_files"
mkdir -p "$ARCHIVE_DIR/reports"
mkdir -p "$ARCHIVE_DIR/backups"
mkdir -p "$ARCHIVE_DIR/test_scripts"
mkdir -p "$ARCHIVE_DIR/temp_files"

echo "📁 创建整理目录:"
echo "  • $ARCHIVE_DIR/reports - 完成报告和总结"
echo "  • $ARCHIVE_DIR/backups - 备份文件"
echo "  • $ARCHIVE_DIR/test_scripts - 测试脚本"
echo "  • $ARCHIVE_DIR/temp_files - 临时文件"
echo ""

MOVED_COUNT=0

# 移动完成报告和总结
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1. 移动完成报告和总结文件..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

find . -maxdepth 1 -type f \( -name "*完成报告*" -o -name "*总结*" -o -name "*工作总结*" \) | while read file; do
    if [ -f "$file" ]; then
        mv "$file" "$ARCHIVE_DIR/reports/"
        echo "  ✅ $(basename "$file")"
        MOVED_COUNT=$((MOVED_COUNT + 1))
    fi
done

# 移动备份文件
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2. 移动备份文件..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

find . -type f \( -name "*.bak" -o -name "*.backup" -o -name "*_backup" -o -name "*time_fix_backup" \) | while read file; do
    if [ -f "$file" ]; then
        # 保持目录结构
        DIR=$(dirname "$file")
        RELDIR=${DIR#./}
        mkdir -p "$ARCHIVE_DIR/backups/$RELDIR"
        mv "$file" "$ARCHIVE_DIR/backups/$RELDIR/"
        echo "  ✅ $file"
        MOVED_COUNT=$((MOVED_COUNT + 1))
    fi
done

# 移动测试脚本
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3. 移动测试脚本..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

find . -type f \( -name "test*.sh" -o -name "*_test.sh" \) | while read file; do
    if [ -f "$file" ]; then
        DIR=$(dirname "$file")
        RELDIR=${DIR#./}
        mkdir -p "$ARCHIVE_DIR/test_scripts/$RELDIR"
        mv "$file" "$ARCHIVE_DIR/test_scripts/$RELDIR/"
        echo "  ✅ $file"
        MOVED_COUNT=$((MOVED_COUNT + 1))
    fi
done

# 移动旧文档
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4. 移动旧文档和临时文件..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 移动项目根目录的旧报告
find ./docs -type f \( -name "*总结*" -o -name "*完成报告*" -o -name "项目最终总结*" \) | while read file; do
    if [ -f "$file" ]; then
        mv "$file" "$ARCHIVE_DIR/reports/"
        echo "  ✅ $file"
        MOVED_COUNT=$((MOVED_COUNT + 1))
    fi
done

# 移动H.266相关的临时文档
find . -maxdepth 1 -type f \( -name "*H.266*" -o -name "*验证报告*" -o -name "*指南*" \) | while read file; do
    if [ -f "$file" ]; then
        mv "$file" "$ARCHIVE_DIR/reports/"
        echo "  ✅ $(basename "$file")"
        MOVED_COUNT=$((MOVED_COUNT + 1))
    fi
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 整理统计"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "整理后的文件:"
echo ""
echo "  报告文件:"
ls -1 "$ARCHIVE_DIR/reports/" | wc -l | awk '{print "    数量: " $1}'

echo ""
echo "  备份文件:"
find "$ARCHIVE_DIR/backups/" -type f | wc -l | awk '{print "    数量: " $1}'

echo ""
echo "  测试脚本:"
find "$ARCHIVE_DIR/test_scripts/" -type f | wc -l | awk '{print "    数量: " $1}'

echo ""
echo "总大小:"
du -sh "$ARCHIVE_DIR" | awk '{print "  " $1}'

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 整理完成！"
echo ""
echo "备份文件位置: $ARCHIVE_DIR/"
echo ""
echo "如需恢复文件，可从此目录复制回原位置"
echo "如需删除，可运行: rm -rf $ARCHIVE_DIR"
