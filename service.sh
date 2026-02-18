#!/system/bin/sh
# SyncClipboard service supervisor

MODDIR=${0%/*}
CONFIG_DIR="/data/adb/syncclipboard"
LOG_FILE="$CONFIG_DIR/clipserver.log"
PID_FILE="$CONFIG_DIR/clipserver.pid"
MONITOR_PID_FILE="$CONFIG_DIR/service_monitor.pid"

# 保守熔断策略（用户指定）
MAX_RESTARTS=5
WINDOW_SEC=60
COOLDOWN_SEC=600
RESTART_DELAY_SEC=3

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

is_pid_alive() {
  PID="$1"
  case "$PID" in
    ''|*[!0-9]*) return 1 ;;
  esac
  [ -d "/proc/$PID" ]
}

is_module_disabled() {
  [ -f "$MODDIR/disable" ] || [ -f "$MODDIR/remove" ]
}

resolve_binary() {
  BINARY_PATH="$MODDIR/bin/clipserver"

  if [ ! -f "$BINARY_PATH" ] && [ -f "$MODDIR/bin/clipserver64" ]; then
    BINARY_PATH="$MODDIR/bin/clipserver64"
  fi

  if [ ! -f "$BINARY_PATH" ] && [ -f "$MODDIR/bin/clipserver32" ]; then
    BINARY_PATH="$MODDIR/bin/clipserver32"
  fi
}

ensure_config() {
  mkdir -p "$CONFIG_DIR"
  chmod 755 "$CONFIG_DIR"

  if [ ! -f "$CONFIG_DIR/config.json" ]; then
    log "Creating default config..."
    cat > "$CONFIG_DIR/config.json" << 'EOF'
{
  "accounts": [],
  "active_account_id": "",
  "sync_interval": 60,
  "enabled": false
}
EOF
    chmod 644 "$CONFIG_DIR/config.json"
  else
    if ! grep -q '"accounts"' "$CONFIG_DIR/config.json" 2>/dev/null; then
      log "Migrating legacy config to multi-account format"
      cat > "$CONFIG_DIR/config.json" << 'EOF'
{
  "accounts": [],
  "active_account_id": "",
  "sync_interval": 60,
  "enabled": false
}
EOF
    fi
  fi
}

stop_stale_server() {
  if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE" 2>/dev/null)
    if is_pid_alive "$OLD_PID"; then
      log "Stopping stale clipserver instance: $OLD_PID"
      kill "$OLD_PID" 2>/dev/null
      sleep 2
      kill -9 "$OLD_PID" 2>/dev/null
    fi
    rm -f "$PID_FILE"
  fi
}

start_clipserver() {
  log "Starting clipserver..."
  "$BINARY_PATH" \
    -port 8964 \
    -config "$CONFIG_DIR/config.json" \
    -webroot "$MODDIR/webroot" \
    >> "$LOG_FILE" 2>&1 &

  CLIPSERVER_PID=$!
  echo "$CLIPSERVER_PID" > "$PID_FILE"
  log "clipserver started with PID: $CLIPSERVER_PID"
}

# 确保单实例监控器
if [ -f "$MONITOR_PID_FILE" ]; then
  OLD_MONITOR_PID=$(cat "$MONITOR_PID_FILE" 2>/dev/null)
  if is_pid_alive "$OLD_MONITOR_PID"; then
    exit 0
  fi
fi
echo $$ > "$MONITOR_PID_FILE"

export PATH="/system/bin:/system/xbin:$PATH"

log "Waiting for boot completion..."
until [ "$(getprop sys.boot_completed)" = 1 ]; do
  sleep 1
done

sleep 10

if is_module_disabled; then
  log "Module is disabled/removed, skip start"
  rm -f "$MONITOR_PID_FILE"
  exit 0
fi

ARCH=$(getprop ro.product.cpu.abi)
log "Detected architecture: $ARCH"

resolve_binary
log "Using binary: $BINARY_PATH"

if [ ! -f "$BINARY_PATH" ]; then
  log "ERROR: clipserver binary not found"
  ls -la "$MODDIR/bin/" >> "$LOG_FILE" 2>&1
  rm -f "$MONITOR_PID_FILE"
  exit 1
fi

chmod 755 "$BINARY_PATH"
ensure_config
stop_stale_server

FAIL_COUNT=0
WINDOW_START=0

log "Supervisor started (max_restarts=$MAX_RESTARTS window=${WINDOW_SEC}s cooldown=${COOLDOWN_SEC}s)"

while true; do
  if is_module_disabled; then
    log "Module disabled/removed, supervisor exiting"
    stop_stale_server
    break
  fi

  start_clipserver
  wait "$CLIPSERVER_PID"
  EXIT_CODE=$?
  NOW_TS=$(date +%s)
  rm -f "$PID_FILE"

  log "clipserver exited (pid=$CLIPSERVER_PID, code=$EXIT_CODE)"

  if [ "$WINDOW_START" -eq 0 ] || [ $((NOW_TS - WINDOW_START)) -gt "$WINDOW_SEC" ]; then
    WINDOW_START=$NOW_TS
    FAIL_COUNT=1
  else
    FAIL_COUNT=$((FAIL_COUNT + 1))
  fi

  if [ "$FAIL_COUNT" -ge "$MAX_RESTARTS" ]; then
    log "Circuit breaker opened: $FAIL_COUNT exits in ${WINDOW_SEC}s, cooling down ${COOLDOWN_SEC}s"
    sleep "$COOLDOWN_SEC"
    WINDOW_START=0
    FAIL_COUNT=0
    continue
  fi

  log "Restarting clipserver in ${RESTART_DELAY_SEC}s (attempt=$FAIL_COUNT/$MAX_RESTARTS)"
  sleep "$RESTART_DELAY_SEC"
done

rm -f "$MONITOR_PID_FILE"
log "Supervisor stopped"
