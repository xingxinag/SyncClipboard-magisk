#!/system/bin/sh
# Clipboard Whitelist Module - Core Script
# 解除应用后台读取剪贴板的限制

LOGTAG="ClipboardWhitelist"

# 日志函数
log_info() {
    log -t $LOGTAG "$1"
    echo "[INFO] $1"
}

log_error() {
    log -t $LOGTAG "[ERROR] $1"
    echo "[ERROR] $1"
}

# 白名单应用列表
# 可以在这里添加需要允许后台读取剪贴板的应用包名
WHITELIST_APPS=(
    "com.example.syncclipboard"  # SyncClipboard 应用（示例）
    "com.github.jericx.syncclipboard"  # SyncClipboard（如果有的话）
    # 在这里添加更多应用包名
    # "com.your.app.packagename"
)

# 自定义白名单文件路径
CUSTOM_WHITELIST="/data/adb/clipboard_whitelist.txt"

# 读取自定义白名单
read_custom_whitelist() {
    if [ -f "$CUSTOM_WHITELIST" ]; then
        log_info "读取自定义白名单: $CUSTOM_WHITELIST"
        while IFS= read -r line; do
            # 忽略空行和注释行
            if [ -n "$line" ] && [ "${line#\#}" = "$line" ]; then
                WHITELIST_APPS+=("$line")
            fi
        done < "$CUSTOM_WHITELIST"
    fi
}

# 为应用授予后台读取剪贴板权限
grant_clipboard_permission() {
    local package="$1"
    
    # 检查应用是否已安装
    if ! pm list packages | grep -q "^package:${package}$"; then
        log_info "应用未安装，跳过: $package"
        return 1
    fi
    
    log_info "正在为应用授予权限: $package"
    
    # 使用 appops 命令设置权限
    # READ_CLIPBOARD 是在 Android 10 中引入的
    appops set $package READ_CLIPBOARD allow 2>/dev/null
    
    if [ $? -eq 0 ]; then
        log_info "✓ 成功授权: $package"
        return 0
    else
        log_error "✗ 授权失败: $package"
        return 1
    fi
}

# 主函数
main() {
    log_info "====================================="
    log_info "剪贴板白名单模块启动"
    log_info "====================================="
    
    # 检查 Android 版本
    SDK_VERSION=$(getprop ro.build.version.sdk)
    log_info "Android SDK 版本: $SDK_VERSION"
    
    if [ $SDK_VERSION -lt 29 ]; then
        log_info "Android 版本低于 10，不需要剪贴板权限管理"
        return 0
    fi
    
    # 读取自定义白名单
    read_custom_whitelist
    
    # 统计
    local success_count=0
    local fail_count=0
    local skip_count=0
    
    # 为白名单中的应用授权
    for app in "${WHITELIST_APPS[@]}"; do
        if grant_clipboard_permission "$app"; then
            success_count=$((success_count + 1))
        else
            if pm list packages | grep -q "^package:${app}$"; then
                fail_count=$((fail_count + 1))
            else
                skip_count=$((skip_count + 1))
            fi
        fi
    done
    
    log_info "====================================="
    log_info "授权完成"
    log_info "成功: $success_count, 失败: $fail_count, 跳过: $skip_count"
    log_info "====================================="
    
    # 创建示例自定义白名单文件（如果不存在）
    if [ ! -f "$CUSTOM_WHITELIST" ]; then
        cat > "$CUSTOM_WHITELIST" << 'EOF'
# 剪贴板白名单配置文件
# 每行一个应用包名
# 以 # 开头的行为注释

# 示例：
# com.example.myapp
# com.github.yourapp

EOF
        chmod 644 "$CUSTOM_WHITELIST"
        log_info "已创建示例配置文件: $CUSTOM_WHITELIST"
    fi
}

# 执行主函数
main
