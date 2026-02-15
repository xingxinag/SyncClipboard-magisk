#!/system/bin/sh
# SyncClipboard Service Script
# Runs on boot to start the clipboard sync service

MODDIR=${0%/*}
CONFIG_DIR="/data/adb/syncclipboard"
LOG_FILE="$CONFIG_DIR/clipserver.log"
PID_FILE="$CONFIG_DIR/clipserver.pid"

# 日志函数
log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

# Wait for boot to complete
log "Waiting for boot to complete..."
until [ "$(getprop sys.boot_completed)" = 1 ]; do
  sleep 1
done

# Additional wait for system stability
sleep 10

log "SyncClipboard Service Starting..."

# 检测架构
ARCH=$(getprop ro.product.cpu.abi)
case "$ARCH" in
  arm64*) BINARY_PATH="$MODDIR/bin/arm64-v8a/clipserver" ;;
  armeabi*) BINARY_PATH="$MODDIR/bin/armeabi-v7a/clipserver" ;;
  x86_64*) BINARY_PATH="$MODDIR/bin/x86_64/clipserver" ;;
  x86*) BINARY_PATH="$MODDIR/bin/x86/clipserver" ;;
  *)
    log "ERROR: Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

log "Detected architecture: $ARCH"
log "Binary path: $BINARY_PATH"

# Check if clipserver binary exists
if [ ! -f "$BINARY_PATH" ]; then
  log "ERROR: clipserver binary not found at $BINARY_PATH"
  exit 1
fi

# 确保二进制文件可执行
chmod 755 "$BINARY_PATH"

# Ensure config directory exists
mkdir -p "$CONFIG_DIR"
chmod 755 "$CONFIG_DIR"

# 如果配置文件不存在，创建默认配置
if [ ! -f "$CONFIG_DIR/config.json" ]; then
  log "Creating default config..."
  cat > "$CONFIG_DIR/config.json" << 'EOF'
{
  "webdav_url": "",
  "webdav_username": "",
  "webdav_password": "",
  "sync_interval": 60,
  "enabled": false
}
EOF
  chmod 644 "$CONFIG_DIR/config.json"
fi

# 停止旧的实例（如果存在）
if [ -f "$PID_FILE" ]; then
  OLD_PID=$(cat "$PID_FILE")
  if kill -0 "$OLD_PID" 2>/dev/null; then
    log "Stopping old instance (PID: $OLD_PID)..."
    kill "$OLD_PID" 2>/dev/null
    sleep 2
  fi
  rm -f "$PID_FILE"
fi

# 设置环境变量
export PATH="/system/bin:/system/xbin:$PATH"

# Start clipserver
log "Starting clipserver..."
nohup "$BINARY_PATH" \
  -port 8964 \
  -config "$CONFIG_DIR/config.json" \
  >> "$LOG_FILE" 2>&1 &

CLIPSERVER_PID=$!
echo $CLIPSERVER_PID > "$PID_FILE"
log "clipserver started with PID: $CLIPSERVER_PID"

# Verify service is running
sleep 3
if kill -0 $CLIPSERVER_PID 2>/dev/null; then
  log "Service started successfully"
  log "WebUI available at: http://localhost:8964"
  
  # 测试 HTTP 服务是否响应
  sleep 2
  if command -v curl >/dev/null 2>&1; then
    if curl -s http://localhost:8964/health >/dev/null 2>&1; then
      log "HTTP service is responding"
    else
      log "WARNING: HTTP service not responding"
    fi
  fi
else
  log "ERROR: Service failed to start"
  rm -f "$PID_FILE"
  exit 1
fi

log "Service initialization complete"
