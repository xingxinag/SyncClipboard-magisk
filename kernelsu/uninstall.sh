#!/system/bin/sh
# Clipboard Whitelist Module - Uninstall Script
# 卸载时执行的清理脚本

LOGTAG="ClipboardWhitelist"

# 日志函数
log_info() {
    log -t $LOGTAG "$1"
    echo "[INFO] $1"
}

log_info "====================================="
log_info "剪贴板白名单模块卸载"
log_info "====================================="

# 清理自定义配置文件（可选）
# 如果想在卸载时保留用户配置，可以注释掉下面这行
CUSTOM_WHITELIST="/data/adb/clipboard_whitelist.txt"

if [ -f "$CUSTOM_WHITELIST" ]; then
    log_info "发现配置文件: $CUSTOM_WHITELIST"
    log_info "如需删除配置文件，请手动执行:"
    log_info "  rm -f $CUSTOM_WHITELIST"
    # 不自动删除配置文件，让用户自行决定
    # rm -f "$CUSTOM_WHITELIST"
fi

log_info "====================================="
log_info "模块已卸载"
log_info "已授予的权限在重启后会保留"
log_info "如需撤销权限，请手动执行:"
log_info "  appops set <包名> READ_CLIPBOARD default"
log_info "====================================="
