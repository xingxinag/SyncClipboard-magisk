#!/system/bin/sh
# SyncClipboard Post-FS-Data Script
# Runs early in boot process

MODDIR=${0%/*}
CONFIG_DIR="/data/adb/syncclipboard"
HELPER_PKG="com.syncclipboard.helper"

# 创建配置目录
mkdir -p "$CONFIG_DIR"
chmod 755 "$CONFIG_DIR"

# 创建 WebUI 目录的符号链接（如果需要）
if [ ! -d "$CONFIG_DIR/webui" ]; then
  ln -sf "$MODDIR/webui" "$CONFIG_DIR/webui"
fi

# 设置 SELinux 上下文（如果支持）
if command -v chcon >/dev/null 2>&1; then
  chcon -R u:object_r:system_file:s0 "$MODDIR/bin" 2>/dev/null
  chcon -R u:object_r:system_data_file:s0 "$CONFIG_DIR" 2>/dev/null
fi

# 日志
echo "[$(date)] SyncClipboard post-fs-data completed" >> "$CONFIG_DIR/post-fs-data.log"

# 自动授予 helper APK 剪贴板权限（尽力而为，不阻塞启动）
if command -v pm >/dev/null 2>&1; then
  if pm list packages | grep -q "$HELPER_PKG"; then
    cmd package install-existing --user 0 "$HELPER_PKG" 2>/dev/null || true
    pm grant "$HELPER_PKG" android.permission.READ_CLIPBOARD_IN_BACKGROUND 2>/dev/null || true
    pm grant "$HELPER_PKG" com.oppo.permission.safe.CLIPBOARD 2>/dev/null || true
    pm grant "$HELPER_PKG" com.oplus.permission.safe.CLIPBOARD 2>/dev/null || true
    appops set "$HELPER_PKG" READ_CLIPBOARD allow 2>/dev/null || true
    echo "[$(date)] Helper APK permissions granted" >> "$CONFIG_DIR/post-fs-data.log"
  else
    echo "[$(date)] Helper APK not installed yet" >> "$CONFIG_DIR/post-fs-data.log"
  fi
fi
