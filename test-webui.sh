#!/bin/bash

echo "================================"
echo "SyncClipboard WebUI 测试脚本"
echo "================================"
echo ""

# 检查后端服务是否运行
echo "1. 检查后端服务..."
if curl -s http://localhost:8964/health > /dev/null 2>&1; then
    echo "   ✓ 后端服务运行正常 (端口 8964)"
else
    echo "   ✗ 后端服务未运行"
    echo "   请先启动后端服务："
    echo "   cd clipserver && go run cmd/clipserver/main.go"
    exit 1
fi

echo ""
echo "2. 测试 API 端点..."

# 测试配置 API
echo -n "   - GET /api/config: "
if curl -s http://localhost:8964/api/config > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
fi

# 测试剪贴板 API
echo -n "   - GET /api/clipboard: "
if curl -s http://localhost:8964/api/clipboard > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
fi

# 测试同步状态 API
echo -n "   - GET /api/sync/status: "
if curl -s http://localhost:8964/api/sync/status > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
fi

echo ""
echo "3. WebUI 文件检查..."
echo -n "   - index.html: "
if [ -f "webroot/index.html" ]; then
    SIZE=$(wc -l < webroot/index.html)
    echo "✓ ($SIZE 行)"
else
    echo "✗"
fi

echo -n "   - 备份文件: "
if [ -f "webroot/index.html.v2.0.0.backup" ]; then
    echo "✓"
else
    echo "✗"
fi

echo ""
echo "================================"
echo "测试完成！"
echo ""
echo "访问 WebUI："
echo "  浏览器: http://localhost:8964"
echo "  或在模块管理器中打开"
echo "================================"
