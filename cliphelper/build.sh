#!/bin/bash
# SyncClipboard Helper APK 构建脚本
# 需要: Android SDK, Java 8+

set -e

echo "=== 构建 SyncClipboard Helper APK ==="

cd "$(dirname "$0")"

# 检查环境
if [ -z "$ANDROID_HOME" ]; then
    echo "错误: 未设置 ANDROID_HOME 环境变量"
    echo "请安装 Android SDK 并设置 ANDROID_HOME"
    exit 1
fi

# 清理
echo "清理旧构建..."
rm -rf app/build

# 构建
echo "构建 APK..."
./gradlew assembleRelease

# 输出
APK_PATH="app/build/outputs/apk/release/app-release-unsigned.apk"
if [ -f "$APK_PATH" ]; then
    echo "✓ APK 构建成功: $APK_PATH"
    
    # 签名（使用 debug keystore）
    echo "签名 APK..."
    $ANDROID_HOME/build-tools/*/apksigner sign --ks ~/.android/debug.keystore \
        --ks-pass pass:android --key-pass pass:android "$APK_PATH"
    
    # 复制到模块目录
    cp "$APK_PATH" ../module/system/priv-app/SyncClipboardHelper/SyncClipboardHelper.apk
    echo "✓ APK 已复制到模块目录"
else
    echo "✗ APK 构建失败"
    exit 1
fi

echo "=== 构建完成 ==="
