#!/system/bin/sh
# SyncClipboard Uninstall Script

CONFIG_DIR="/data/adb/syncclipboard"

echo "Stopping SyncClipboard service..."

# Kill clipserver if running
if [ -f "$CONFIG_DIR/clipserver.pid" ]; then
  PID=$(cat "$CONFIG_DIR/clipserver.pid")
  if kill -0 $PID 2>/dev/null; then
    kill $PID
    echo "Service stopped (PID: $PID)"
  fi
  rm -f "$CONFIG_DIR/clipserver.pid"
fi

# Ask user if they want to remove config
echo "Do you want to remove configuration files? (y/n)"
echo "Config directory: $CONFIG_DIR"
echo "Note: This will delete your WebDAV accounts and sync settings."

# In Magisk/KSU modules, we can't interactively ask, so we preserve config by default
echo "Preserving configuration files for future reinstallation."
echo "To manually remove, delete: $CONFIG_DIR"

echo "SyncClipboard uninstalled."
