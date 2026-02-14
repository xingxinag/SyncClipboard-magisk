#!/bin/bash
# SyncClipboard Universal Module Builder
# Builds a single ZIP package that works on Magisk/KernelSU/APatch

set -e

echo "====================================="
echo "SyncClipboard Universal Module Builder"
echo "====================================="
echo ""

# Version information
VERSION="1.0.0"
VERSION_CODE="10000"
MODULE_NAME="SyncClipboard"
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
    
    cd ../clipserver
    
    # Define architectures
    declare -A ARCHS=(
        ["arm64-v8a"]="linux/arm64"
        ["armeabi-v7a"]="linux/arm"
        ["x86_64"]="linux/amd64"
        ["x86"]="linux/386"
    )
    
    for arch in "${!ARCHS[@]}"; do
        info "  编译 $arch..."
        GOOS=linux GOARCH=${ARCHS[$arch]#*/} \
            go build -ldflags="-s -w" \
            -o "../SyncClipboard-magisk/bin/$arch/clipserver" \
            ./cmd/clipserver
        
        if [ $? -eq 0 ]; then
            info "  ✓ $arch 编译成功"
        else
            error "  ✗ $arch 编译失败"
        fi
    done
    
    cd ../SyncClipboard-magisk
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

if [ ! -f "webui/index.html" ]; then
    warn "webui/index.html 不存在，创建占位文件"
    cat > webui/index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SyncClipboard</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-bottom: 20px;
        }
        .status {
            padding: 15px;
            background: #e8f5e9;
            border-left: 4px solid #4caf50;
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔄 SyncClipboard</h1>
        <div class="status">
            <strong>状态:</strong> 服务运行中
        </div>
        <p>WebUI 开发中... 当前为占位页面</p>
        <p>配置文件: /data/adb/syncclipboard/config.json</p>
    </div>
</body>
</html>
EOF
fi

info "WebUI 文件准备完成"

# Step 5: Create module package
info "[5/6] 创建模块包..."

# Files to include in the ZIP
FILES=(
    "META-INF"
    "bin"
    "webui"
    "module.prop"
    "customize.sh"
    "service.sh"
    "uninstall.sh"
    "README.md"
)

# Create ZIP
zip -r "$OUTPUT_ZIP" "${FILES[@]}" -x "*.git*" "*.DS_Store" "*/.*" 2>&1 | grep -v "adding:"

if [ ${PIPESTATUS[0]} -eq 0 ]; then
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
