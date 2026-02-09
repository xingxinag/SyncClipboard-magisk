# 安装和使用指南

## 📦 文件说明

生成的文件：
- `clipboard_whitelist_magisk.zip` - Magisk 模块安装包
- `clipboard_whitelist_kernelsu.zip` - KernelSU 模块安装包

## 🚀 快速开始

### 步骤 1: 选择正确的版本

- 如果你使用 **Magisk**，下载 `clipboard_whitelist_magisk.zip`
- 如果你使用 **KernelSU**，下载 `clipboard_whitelist_kernelsu.zip`

### 步骤 2: 安装模块

**Magisk 用户：**
1. 打开 Magisk Manager
2. 点击底部的「模块」标签
3. 点击「从本地安装」
4. 选择 `clipboard_whitelist_magisk.zip`
5. 等待安装完成
6. 重启设备

**KernelSU 用户：**
1. 打开 KernelSU Manager
2. 点击「模块」
3. 点击右上角的「+」或「安装」按钮
4. 选择 `clipboard_whitelist_kernelsu.zip`
5. 等待安装完成
6. 重启设备

### 步骤 3: 配置白名单（可选）

模块安装后，你可以自定义需要授权的应用列表：

1. 使用支持 Root 的文件管理器（如 MT Manager、Root Explorer 等）
2. 导航到 `/data/adb/clipboard_whitelist.txt`
3. 添加应用包名，每行一个
4. 保存文件
5. 重启设备或手动执行脚本

**示例配置：**
```
# SyncClipboard 相关应用
com.github.jericx.syncclipboard

# AutoX.js
com.autoxjs.autoxjs

# HTTP Request Shortcuts
ch.rmy.android.http_shortcuts

# 你的其他应用
# com.your.app.packagename
```

### 步骤 4: 验证安装

检查模块是否正常工作：

```bash
# 使用终端模拟器或 ADB

# 1. 检查模块日志
logcat | grep ClipboardWhitelist

# 2. 检查应用权限状态
appops get com.your.app.packagename READ_CLIPBOARD
# 应该显示: allow
```

## 🔧 高级配置

### 修改默认白名单

编辑模块中的 `clipboard_whitelist.sh` 文件，修改 `WHITELIST_APPS` 数组：

```bash
WHITELIST_APPS=(
    "com.example.app1"
    "com.example.app2"
    # 添加更多应用
)
```

### 手动执行授权脚本

```bash
su -c "/data/adb/modules/clipboard_whitelist_*/clipboard_whitelist.sh"
```

## 📱 查找应用包名的方法

### 方法 1: 使用 ADB
```bash
adb shell pm list packages | grep 应用名关键词
```

### 方法 2: 使用应用信息查看器
- Package Manager
- AppWererabbit
- 应用管家

### 方法 3: 使用终端模拟器
```bash
pm list packages | grep 应用名关键词
```

### 方法 4: 查看应用详情
在系统设置 → 应用管理中，长按应用信息通常会显示包名

## ⚠️ 常见问题

### 问题 1: 模块安装后不生效

**解决方案：**
1. 确认已重启设备
2. 检查模块是否已启用
3. 检查应用包名是否正确
4. 查看系统日志：`logcat -s ClipboardWhitelist`

### 问题 2: 某些应用仍然无法后台读取剪贴板

**可能原因：**
1. 应用包名不正确
2. 应用需要额外的权限
3. Android 版本过高，有额外限制

**解决方案：**
1. 使用 `pm list packages` 确认正确的包名
2. 检查应用是否需要其他权限
3. 查看模块日志了解详细信息

### 问题 3: 如何撤销某个应用的权限

```bash
appops set com.app.packagename READ_CLIPBOARD default
```

## 🛠️ 开发者信息

### 项目结构

```
clipboard-whitelist-module/
├── README.md                    # 项目说明
├── INSTALL_GUIDE.md            # 本安装指南
├── build.sh                     # Linux/Mac 构建脚本
├── build.bat                    # Windows 构建脚本
├── magisk/                      # Magisk 模块目录
│   ├── module.prop             # 模块配置
│   ├── customize.sh            # 安装脚本
│   ├── service.sh              # 开机服务
│   ├── clipboard_whitelist.sh  # 核心功能
│   └── uninstall.sh            # 卸载脚本
├── kernelsu/                    # KernelSU 模块目录
│   └── [相同的文件结构]
└── common/                      # 共享文件
    └── [脚本源文件]
```

### 重新构建模块

**Linux/Mac:**
```bash
cd clipboard-whitelist-module
bash build.sh
```

**Windows:**
```cmd
cd clipboard-whitelist-module
build.bat
```

## 📋 技术细节

### 工作原理

模块通过修改 Android 的 AppOps 权限来实现：

1. 在系统启动时执行 `service.sh`
2. `service.sh` 调用 `clipboard_whitelist.sh`
3. 脚本为白名单中的应用执行：
   ```bash
   appops set <package_name> READ_CLIPBOARD allow
   ```

### 兼容性

- ✅ Android 10 (API 29)
- ✅ Android 11 (API 30)
- ✅ Android 12 (API 31)
- ✅ Android 13 (API 33)
- ✅ Android 14 (API 34)

### 权限说明

`READ_CLIPBOARD` 权限是 Android 10 引入的 AppOps 权限，用于控制应用是否可以在后台读取剪贴板内容。

## 🔐 安全建议

1. **只授权可信应用**：仅为你信任的应用授予剪贴板读取权限
2. **定期检查白名单**：定期审查 `/data/adb/clipboard_whitelist.txt`
3. **监控日志**：注意应用的异常行为
4. **及时更新**：保持模块和应用的最新版本

## 📞 获取帮助

如果遇到问题：

1. 查看 README.md 中的常见问题部分
2. 检查系统日志：`logcat | grep ClipboardWhitelist`
3. 在项目仓库提交 Issue
4. 提供以下信息：
   - Android 版本
   - Magisk/KernelSU 版本
   - 相关日志
   - 问题描述

## 📄 许可证

MIT License - 详见 LICENSE 文件

---

**祝你使用愉快！**
