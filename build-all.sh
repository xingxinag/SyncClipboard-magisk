#!/bin/bash
# 完整构建脚本：Go 服务 + APK + Magisk 模块

set -e

echo "=== SyncClipboard Magisk 模块完整构建 ==="

# 1. 构建 Go 服务
echo "1. 构建 Go 服务..."
cd clipserver
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../module/clipserver ./cmd/clipserver
cd ..
echo "✓ Go 服务构建完成"

# 2. 构建 APK（如果有 Android SDK）
if [ -n "$ANDROID_HOME" ] && [ -f "cliphelper/gradlew" ]; then
    echo "2. 构建 APK..."
    cd cliphelper
    ./gradlew assembleRelease
    
    APK_PATH="app/build/outputs/apk/release/app-release-unsigned.apk"
    if [ -f "$APK_PATH" ]; then
        # 创建系统应用目录
        mkdir -p ../module/system/priv-app/SyncClipboardHelper
        
        # 签名并复制
        $ANDROID_HOME/build-tools/*/apksigner sign --ks ~/.android/debug.keystore \
            --ks-pass pass:android --key-pass pass:android "$APK_PATH"
        cp "$APK_PATH" ../module/system/priv-app/SyncClipboardHelper/SyncClipboardHelper.apk
        echo "✓ APK 构建并集成完成"
    fi
    cd ..
else
    echo "2. 跳过 APK 构建（未找到 Android SDK 或 gradlew）"
    echo "   注意：没有 APK，剪贴板功能将降级到共享文件模式"
fi

# 3. 打包模块
echo "3. 打包 Magisk 模块..."
cd module
zip -r ../SyncClipboard-magisk.zip .
cd ..
echo "✓ 模块打包完成: SyncClipboard-magisk.zip"

echo "=== 构建完成 ==="
ls -lh SyncClipboard-magisk.zip
