#!/bin/bash

# Pixly v3.1.1 启动脚本
# 双击此文件即可运行

# 获取脚本所在目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# 切换到项目目录
cd "$SCRIPT_DIR"

# 清屏
clear

# 运行Pixly
./pixly_interactive

# 运行结束后等待用户按键
echo ""
echo "按任意键退出..."
read -n 1

