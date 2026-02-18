#!/bin/bash
# SyncClipboard Universal Module Builder
# Builds a single ZIP package that works on Magisk/KernelSU/APatch

set -e

echo "====================================="
echo "SyncClipboard Universal Module Builder"
echo "====================================="
echo ""

# Version information
VERSION="2.6.9"
VERSION_CODE="20609"
MODULE_NAME="SyncClipboard-magisk"
OUTPUT_ZIP="${MODULE_NAME}_v${VERSION}.zip"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Step 1: Check prerequisites
info "[1/6] 检查构建环境..."

if ! command -v zip &> /dev/null; then
    error "zip 命令未找到，请安装 zip 工具"
fi

if ! command -v go &> /dev/null; then
    warn "Go 未安装，将跳过二进制文件编译"
    warn "请确保 bin/ 目录下已有预编译的二进制文件"
    SKIP_BUILD=true
else
    info "Go 版本: $(go version)"
    SKIP_BUILD=false
fi

# Step 2: Clean old builds
info "[2/6] 清理旧的构建文件..."
rm -f "$OUTPUT_ZIP"
info "清理完成"

# Step 3: Build Go binaries (if Go is available)
if [ "$SKIP_BUILD" = false ]; then
    info "[3/6] 编译 Go 二进制文件..."
    
    cd clipserver
    
    # Build for arm64-v8a
    info "  编译 arm64-v8a..."
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../bin/arm64-v8a/clipserver ./cmd/clipserver
    if [ $? -eq 0 ]; then
        info "  ✓ arm64-v8a 编译成功"
    else
        error "  ✗ arm64-v8a 编译失败"
    fi
    
    # Build for armeabi-v7a
    info "  编译 armeabi-v7a..."
    CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o ../bin/armeabi-v7a/clipserver ./cmd/clipserver
    if [ $? -eq 0 ]; then
        info "  ✓ armeabi-v7a 编译成功"
    else
        error "  ✗ armeabi-v7a 编译失败"
    fi
    
    # Build for x86_64
    info "  编译 x86_64..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/x86_64/clipserver ./cmd/clipserver
    if [ $? -eq 0 ]; then
        info "  ✓ x86_64 编译成功"
    else
        error "  ✗ x86_64 编译失败"
    fi
    
    # Build for x86
    info "  编译 x86..."
    CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o ../bin/x86/clipserver ./cmd/clipserver
    if [ $? -eq 0 ]; then
        info "  ✓ x86 编译成功"
    else
        error "  ✗ x86 编译失败"
    fi
    
    cd ..
else
    info "[3/6] 跳过二进制编译（使用现有文件）"
    
    # Check if binaries exist
    for arch in arm64-v8a armeabi-v7a x86_64 x86; do
        if [ ! -f "bin/$arch/clipserver" ]; then
            error "缺少 $arch 架构的二进制文件: bin/$arch/clipserver"
        fi
    done
fi

# Step 4: Build WebUI (placeholder)
info "[4/6] 准备 WebUI 文件..."

if [ ! -f "webroot/index.html" ]; then
    error "webroot/index.html 不存在"
fi

info "WebUI 文件准备完成"

# Step 5: Create module package
info "[5/6] 创建模块包..."

# Files to include in the ZIP
FILES=(
    "META-INF"
    "bin"
    "webroot"
    "config"
    "module.prop"
    "customize.sh"
    "service.sh"
    "post-fs-data.sh"
    "sepolicy.rule"
    "uninstall.sh"
    "README.md"
)

if [ -d "system" ]; then
    FILES+=("system")
fi

# Create ZIP (quiet mode avoids pipe/grep exit issues under set -e)
zip -qr "$OUTPUT_ZIP" "${FILES[@]}" \
  -x "*.git*" "*.DS_Store" "*/.*" "bin/x86/*" "bin/x86_64/*"

if [ $? -eq 0 ]; then
    info "模块包创建成功"
else
    error "模块包创建失败"
fi

# Step 6: Show summary
info "[6/6] 构建完成!"
echo ""
echo "====================================="
echo "构建摘要"
echo "====================================="
echo "模块名称: $MODULE_NAME"
echo "版本: v$VERSION ($VERSION_CODE)"
echo "输出文件: $OUTPUT_ZIP"
echo "文件大小: $(du -h "$OUTPUT_ZIP" | cut -f1)"
echo ""
echo "支持的环境:"
echo "  ✓ Magisk 26.4+"
echo "  ✓ KernelSU 0.6.6+"
echo "  ✓ APatch 0.10.7+"
echo ""
echo "支持的架构:"
ls -1 bin/ | grep -v "README" | sed 's/^/  ✓ /'
echo ""
echo "====================================="
info "构建成功! 🎉"
echo "====================================="
