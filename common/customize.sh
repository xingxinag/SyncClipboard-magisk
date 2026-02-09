#!/system/bin/sh
# Clipboard Whitelist Module - Universal Installer
# Compatible with both Magisk and KernelSU

MODPATH=${0%/*}

# 打印日志函数
ui_print() {
  echo "$1"
}

# 检测运行环境
detect_environment() {
  if [ -n "$KSU" ] || [ -n "$KSU_VER" ]; then
    ui_print "- 检测到 KernelSU 环境"
    ENV_TYPE="kernelsu"
  elif [ -n "$MAGISK_VER" ] || [ -d "/data/adb/magisk" ]; then
    ui_print "- 检测到 Magisk 环境"
    ENV_TYPE="magisk"
  else
    ui_print "! 未检测到 Magisk 或 KernelSU 环境"
    ui_print "! 安装终止"
    exit 1
  fi
}

# 设置权限
set_permissions() {
  ui_print "- 设置文件权限"
  set_perm_recursive $MODPATH 0 0 0755 0644
  set_perm $MODPATH/service.sh 0 0 0755
  set_perm $MODPATH/clipboard_whitelist.sh 0 0 0755
}

# 主安装流程
ui_print "*************************************"
ui_print "  Clipboard Whitelist Module"
ui_print "  剪贴板白名单模块"
ui_print "*************************************"
ui_print ""

ui_print "- 正在检测安装环境..."
detect_environment

ui_print "- 提取模块文件..."
unzip -o "$ZIPFILE" 'system/*' -d $MODPATH >&2

ui_print "- 复制脚本文件..."
cp -f $MODPATH/common/clipboard_whitelist.sh $MODPATH/

set_permissions

ui_print ""
ui_print "- 安装完成!"
ui_print "- 请重启设备以激活模块"
ui_print "- 环境类型: $ENV_TYPE"
ui_print ""
