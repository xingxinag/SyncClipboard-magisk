#!/system/bin/sh
# SyncClipboard - Universal Installer
# Compatible with Magisk/KernelSU/APatch

# shellcheck disable=SC2034
SKIPUNZIP=1

# Minimum version requirements
MIN_KSU_VERSION=10940
MIN_KSUD_VERSION=11575
MIN_MAGISK_VERSION=26402
MIN_APATCH_VERSION=10700
MIN_API_LEVEL=26  # Android 8.0+

# Module configuration
MODULE_ID="syncclipboard"
MODULE_NAME="SyncClipboard"
CONFIG_DIR="/data/adb/syncclipboard"

# ============================================
# Environment Detection
# ============================================

detect_environment() {
  if [ "$BOOTMODE" ] && [ "$KSU" ]; then
    ui_print "- 检测到 KernelSU 环境"
    ui_print "- KernelSU 版本: $KSU_KERNEL_VER_CODE (kernel) + $KSU_VER_CODE (ksud)"
    
    # Validate KernelSU version
    if ! [ "$KSU_KERNEL_VER_CODE" ] || [ "$KSU_KERNEL_VER_CODE" -lt "$MIN_KSU_VERSION" ]; then
      ui_print "*********************************************************"
      ui_print "! KernelSU 内核版本过旧!"
      ui_print "! 请更新 KernelSU 到最新版本"
      abort    "*********************************************************"
    fi
    
    if ! [ "$KSU_VER_CODE" ] || [ "$KSU_VER_CODE" -lt "$MIN_KSUD_VERSION" ]; then
      ui_print "*********************************************************"
      ui_print "! KernelSU Manager 版本过旧!"
      ui_print "! 请更新 KernelSU Manager 到最新版本"
      abort    "*********************************************************"
    fi
    
    # Check for Magisk conflict
    if [ "$(which magisk)" ]; then
      ui_print "*********************************************************"
      ui_print "! 不支持多个 Root 实现共存!"
      ui_print "! 请在安装前卸载 Magisk"
      abort    "*********************************************************"
    fi
    
    ENV_TYPE="kernelsu"
    
  elif [ "$BOOTMODE" ] && [ "$APATCH" ]; then
    ui_print "- 检测到 APatch 环境"
    ui_print "- APatch 版本: $APATCH_VER_CODE"
    
    if ! [ "$APATCH_VER_CODE" ] || [ "$APATCH_VER_CODE" -lt "$MIN_APATCH_VERSION" ]; then
      ui_print "*********************************************************"
      ui_print "! APatch 版本过旧!"
      ui_print "! 请更新 APatch 到最新版本"
      abort    "*********************************************************"
    fi
    
    ENV_TYPE="apatch"
    
  elif [ "$BOOTMODE" ] && [ "$MAGISK_VER_CODE" ]; then
    ui_print "- 检测到 Magisk 环境"
    ui_print "- Magisk 版本: $MAGISK_VER_CODE"
    
    if [ "$MAGISK_VER_CODE" -lt "$MIN_MAGISK_VERSION" ]; then
      ui_print "*********************************************************"
      ui_print "! Magisk 版本过旧!"
      ui_print "! 请更新 Magisk 到最新版本"
      abort    "*********************************************************"
    fi
    
    ENV_TYPE="magisk"
    
  else
    ui_print "*********************************************************"
    ui_print "! 不支持从 Recovery 安装"
    ui_print "! 请从 KernelSU/APatch/Magisk 应用中安装"
    abort    "*********************************************************"
  fi
}

# ============================================
# System Validation
# ============================================

validate_system() {
  # Check Android version
  if [ "$API" -lt "$MIN_API_LEVEL" ]; then
    ui_print "! 不支持的 Android 版本: API $API"
    abort "! 最低支持 Android 8.0 (API 26)"
  else
    ui_print "- Android API 级别: $API"
  fi
  
  # Check architecture
  if [ "$ARCH" != "arm64" ] && [ "$ARCH" != "x64" ] && [ "$ARCH" != "arm" ]; then
    abort "! 不支持的 CPU 架构: $ARCH"
  else
    ui_print "- CPU 架构: $ARCH"
  fi
  
  # Detect 32-bit support
  HAS32BIT=false
  if [ -n "$(getprop ro.product.cpu.abilist32)" ] || [ -n "$(getprop ro.system.product.cpu.abilist32)" ]; then
    HAS32BIT=true
    ui_print "- 检测到 32 位支持"
  fi
}

# ============================================
# File Extraction
# ============================================

extract_files() {
  ui_print "- 提取模块文件..."
  
  # Extract core module files
  unzip -o "$ZIPFILE" 'module.prop' -d "$MODPATH" >&2
  unzip -o "$ZIPFILE" 'service.sh' -d "$MODPATH" >&2
  unzip -o "$ZIPFILE" 'uninstall.sh' -d "$MODPATH" >&2
  
  # Extract post-fs-data.sh if exists
  unzip -o "$ZIPFILE" 'post-fs-data.sh' -d "$MODPATH" >&2 2>/dev/null || true
  
  # Extract SELinux rules if exists
  unzip -o "$ZIPFILE" 'sepolicy.rule' -d "$MODPATH" >&2 2>/dev/null || true
  
  # Extract WebUI
  ui_print "- 提取 Web UI..."
  unzip -o "$ZIPFILE" "webui/*" -d "$MODPATH" >&2
  
  # Extract config template
  unzip -o "$ZIPFILE" "config/*" -d "$MODPATH" >&2 2>/dev/null || true
  
  # Extract architecture-specific binaries
  mkdir -p "$MODPATH/bin"
  
  if [ "$ARCH" = "x64" ]; then
    if [ "$HAS32BIT" = "true" ]; then
      ui_print "- 提取 x86 二进制文件..."
      unzip -o "$ZIPFILE" 'bin/x86/clipserver' -d "$MODPATH" >&2
      mv "$MODPATH/bin/x86/clipserver" "$MODPATH/bin/clipserver32"
    fi
    
    ui_print "- 提取 x86_64 二进制文件..."
    unzip -o "$ZIPFILE" 'bin/x86_64/clipserver' -d "$MODPATH" >&2
    mv "$MODPATH/bin/x86_64/clipserver" "$MODPATH/bin/clipserver64"
    ln -s "./clipserver64" "$MODPATH/bin/clipserver"
    
  elif [ "$ARCH" = "arm64" ]; then
    if [ "$HAS32BIT" = "true" ]; then
      ui_print "- 提取 ARM32 二进制文件..."
      unzip -o "$ZIPFILE" 'bin/armeabi-v7a/clipserver' -d "$MODPATH" >&2
      mv "$MODPATH/bin/armeabi-v7a/clipserver" "$MODPATH/bin/clipserver32"
    fi
    
    ui_print "- 提取 ARM64 二进制文件..."
    unzip -o "$ZIPFILE" 'bin/arm64-v8a/clipserver' -d "$MODPATH" >&2
    mv "$MODPATH/bin/arm64-v8a/clipserver" "$MODPATH/bin/clipserver64"
    ln -s "./clipserver64" "$MODPATH/bin/clipserver"
    
  else  # ARM only
    ui_print "- 提取 ARM32 二进制文件..."
    unzip -o "$ZIPFILE" 'bin/armeabi-v7a/clipserver' -d "$MODPATH" >&2
    mv "$MODPATH/bin/armeabi-v7a/clipserver" "$MODPATH/bin/clipserver"
  fi
  
  # Clean up architecture directories
  rm -rf "$MODPATH/bin/x86" "$MODPATH/bin/x86_64" "$MODPATH/bin/armeabi-v7a" "$MODPATH/bin/arm64-v8a"
}

# ============================================
# Configuration Setup
# ============================================

setup_config() {
  ui_print "- 初始化配置目录..."
  
  # Create config directory
  mkdir -p "$CONFIG_DIR"
  
  # Create default config if not exists
  if [ ! -f "$CONFIG_DIR/config.json" ]; then
    ui_print "- 创建默认配置文件..."
    cat > "$CONFIG_DIR/config.json" << 'EOF'
{
  "webdav_url": "",
  "webdav_username": "",
  "webdav_password": "",
  "sync_interval": 60,
  "enabled": false
}
EOF
  else
    ui_print "- 保留现有配置文件"
  fi
  
  # Set permissions on config directory
  chmod 755 "$CONFIG_DIR"
  chmod 644 "$CONFIG_DIR/config.json"
}

# ============================================
# Permissions
# ============================================

set_permissions() {
  ui_print "- 设置文件权限..."
  
  # Set base permissions
  set_perm_recursive "$MODPATH" 0 0 0755 0644
  
  # Set executable permissions
  set_perm "$MODPATH/service.sh" 0 0 0755
  set_perm "$MODPATH/uninstall.sh" 0 0 0755
  set_perm_recursive "$MODPATH/bin" 0 0 0755 0755
  
  # Set WebUI permissions
  set_perm_recursive "$MODPATH/webui" 0 0 0755 0644
}

# ============================================
# Environment-specific Setup
# ============================================

setup_environment_specific() {
  case "$ENV_TYPE" in
    kernelsu)
      ui_print "- 配置 KernelSU 环境..."
      # 安装 SELinux 规则
      if [ -f "$MODPATH/sepolicy.rule" ]; then
        ui_print "- 安装 SELinux 规则..."
        # KernelSU 会自动加载 sepolicy.rule
      fi
      ;;
    apatch)
      ui_print "- 配置 APatch 环境..."
      # 安装 SELinux 规则
      if [ -f "$MODPATH/sepolicy.rule" ]; then
        ui_print "- 安装 SELinux 规则..."
        # APatch 会自动加载 sepolicy.rule
      fi
      ;;
    magisk)
      ui_print "- 配置 Magisk 环境..."
      # Magisk 不需要 sepolicy.rule 文件
      # 但我们保留它以便兼容
      ;;
  esac
}

# ============================================
# Main Installation Flow
# ============================================

VERSION=$(grep_prop version "${TMPDIR}/module.prop")

ui_print "*************************************"
ui_print "  $MODULE_NAME 安装程序"
ui_print "  版本: $VERSION"
ui_print "*************************************"
ui_print ""

ui_print "- 检测安装环境..."
detect_environment

ui_print "- 验证系统兼容性..."
validate_system

extract_files
setup_config
set_permissions
setup_environment_specific

ui_print ""
ui_print "*************************************"
ui_print "  安装完成!"
ui_print "  环境: $ENV_TYPE"
ui_print "  架构: $ARCH"
ui_print "  配置: $CONFIG_DIR"
ui_print "  WebUI: http://localhost:8964"
ui_print "  请重启设备以激活模块"
ui_print "*************************************"
ui_print ""
