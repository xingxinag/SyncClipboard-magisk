#!/system/bin/sh
# SyncClipboard Magisk 模块安装后脚本
# 自动授予 APK 必要的权限

MODDIR=${0%/*}
PKG="com.syncclipboard.helper"

# 等待系统启动完成
while [ "$(getprop sys.boot_completed)" != "1" ]; do
    sleep 1
done

# 等待 APK 安装完成
sleep 5

# 授予后台读取剪贴板权限
pm grant $PKG android.permission.READ_CLIPBOARD_IN_BACKGROUND 2>/dev/null

# 授予 ColorOS/OPPO 特殊权限
pm grant $PKG com.oppo.permission.safe.CLIPBOARD 2>/dev/null
pm grant $PKG com.oplus.permission.safe.CLIPBOARD 2>/dev/null

# 通过 appops 授予权限
appops set $PKG READ_CLIPBOARD allow 2>/dev/null

# 记录日志
echo "[$(date)] SyncClipboard permissions granted" >> /data/adb/syncclipboard/install.log
