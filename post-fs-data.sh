#!/system/bin/sh
# SyncClipboard Post-FS-Data Script
# Runs early in boot process

MODDIR=${0%/*}
CONFIG_DIR="/data/adb/syncclipboard"

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
