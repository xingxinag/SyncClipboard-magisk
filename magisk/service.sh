#!/system/bin/sh
# Clipboard Whitelist Module - Service Script
# 在系统启动时运行

MODDIR=${0%/*}

# 等待系统完全启动
while [ "$(getprop sys.boot_completed)" != "1" ]; do
    sleep 1
done

# 额外等待确保所有服务已启动
sleep 5

# 执行剪贴板白名单配置
sh $MODDIR/clipboard_whitelist.sh

# 记录日志
log -t ClipboardWhitelist "剪贴板白名单模块已激活"
