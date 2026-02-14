#!/system/bin/sh
# SyncClipboard Service Script
# Runs on boot to start the clipboard sync service

MODDIR=${0%/*}
CONFIG_DIR="/data/adb/syncclipboard"
LOG_FILE="$CONFIG_DIR/service.log"

# Wait for boot to complete
until [ "$(getprop sys.boot_completed)" = 1 ]; do
  sleep 1
done

# Additional wait for system stability
sleep 10

# Start logging
exec 1>"$LOG_FILE" 2>&1

echo "[$(date)] SyncClipboard Service Starting..."

# Check if clipserver binary exists
if [ ! -f "$MODDIR/bin/clipserver" ]; then
  echo "[$(date)] ERROR: clipserver binary not found!"
  exit 1
fi

# Ensure config directory exists
mkdir -p "$CONFIG_DIR"

# Start clipserver
echo "[$(date)] Starting clipserver..."
"$MODDIR/bin/clipserver" serve \
  --config "$CONFIG_DIR/config.json" \
  --webui "$MODDIR/webui" \
  >> "$LOG_FILE" 2>&1 &

CLIPSERVER_PID=$!
echo "[$(date)] clipserver started with PID: $CLIPSERVER_PID"
echo $CLIPSERVER_PID > "$CONFIG_DIR/clipserver.pid"

# Verify service is running
sleep 2
if kill -0 $CLIPSERVER_PID 2>/dev/null; then
  echo "[$(date)] Service started successfully"
  echo "[$(date)] WebUI available at: http://localhost:8964"
else
  echo "[$(date)] ERROR: Service failed to start"
  exit 1
fi
