#!/bin/bash
# Build script for Clipboard Whitelist Module
# 用于打包 Magisk 和 KernelSU 模块的构建脚本

set -e

echo "====================================="
echo "Clipboard Whitelist Module Builder"
echo "====================================="
echo ""

# 版本信息
VERSION="1.0.0"
VERSION_CODE="10000"

# 清理旧的构建文件
echo "[1/4] 清理旧的构建文件..."
rm -f clipboard_whitelist_magisk.zip clipboard_whitelist_kernelsu.zip

# 构建 Magisk 版本
echo "[2/4] 构建 Magisk 版本..."
cd magisk
zip -r ../clipboard_whitelist_magisk.zip . -x "*.git*" "*.DS_Store"
cd ..
echo "✓ Magisk 版本构建完成: clipboard_whitelist_magisk.zip"

# 构建 KernelSU 版本
echo "[3/4] 构建 KernelSU 版本..."
cd kernelsu
zip -r ../clipboard_whitelist_kernelsu.zip . -x "*.git*" "*.DS_Store"
cd ..
echo "✓ KernelSU 版本构建完成: clipboard_whitelist_kernelsu.zip"

# 显示文件信息
echo "[4/4] 构建完成!"
echo ""
echo "生成的文件:"
ls -lh clipboard_whitelist_*.zip
echo ""
echo "====================================="
echo "构建成功!"
echo "版本: v$VERSION ($VERSION_CODE)"
echo "====================================="
