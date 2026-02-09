# Clipboard Whitelist Module
# 剪贴板白名单模块

## 简介

这是一个同时支持 **Magisk** 和 **KernelSU** 的 Android Root 模块，用于解除 Android 10+ 系统对应用后台读取剪贴板的限制。

## 功能特性

- ✅ 同时支持 Magisk 和 KernelSU
- ✅ 自动检测运行环境
- ✅ 允许指定应用在后台读取剪贴板
- ✅ 支持自定义白名单配置
- ✅ 适用于 Android 10 及以上版本
- ✅ 开机自动激活
- ✅ 无需手动重启应用

## 背景

Android 10 (API 29) 及以上版本引入了新的隐私保护机制，限制应用在后台读取剪贴板内容。这对于一些需要后台同步剪贴板的应用（如 SyncClipboard、AutoX.js 等）造成了困扰。

本模块通过修改系统的 AppOps 权限设置，为指定的应用授予后台读取剪贴板的权限，从而解决这一问题。

## 系统要求

- Android 10 (API 29) 或更高版本
- 已安装 Magisk (v20.4+) 或 KernelSU
- Root 权限

## 安装方法

### Magisk 用户

1. 下载 `clipboard_whitelist_magisk.zip`
2. 打开 Magisk Manager
3. 点击「模块」
4. 点击「从本地安装」
5. 选择下载的 zip 文件
6. 等待安装完成
7. 重启设备

### KernelSU 用户

1. 下载 `clipboard_whitelist_kernelsu.zip`
2. 打开 KernelSU Manager
3. 点击「模块」
4. 点击右上角「+」或「安装」
5. 选择下载的 zip 文件
6. 等待安装完成
7. 重启设备

## 使用方法

### 方法一：使用预设白名单

模块安装后会自动为预设的应用授予权限。默认包含：
- `com.example.syncclipboard`
- `com.github.jericx.syncclipboard`

### 方法二：自定义白名单

1. 使用文件管理器（需要 Root 权限）打开：
   ```
   /data/adb/clipboard_whitelist.txt
   ```

2. 添加需要授权的应用包名，每行一个：
   ```
   # 示例：
   com.example.myapp
   com.github.yourapp
   com.autoxjs.autoxjs
   ```

3. 保存文件后重启设备，或手动执行：
   ```bash
   su -c "/data/adb/modules/clipboard_whitelist_*/clipboard_whitelist.sh"
   ```

### 如何查找应用包名

1. 使用 ADB:
   ```bash
   adb shell pm list packages | grep 应用名称关键词
   ```

2. 使用应用信息查看器（如 Package Manager、应用管家等）

3. 使用终端模拟器（需要 Root）：
   ```bash
   pm list packages | grep 应用名称关键词
   ```

## 配置说明

### 模块目录结构

```
clipboard-whitelist-module/
├── magisk/
│   └── module.prop           # Magisk 模块配置
├── kernelsu/
│   └── module.prop           # KernelSU 模块配置
├── common/
│   ├── customize.sh          # 安装脚本
│   ├── service.sh            # 开机服务脚本
│   └── clipboard_whitelist.sh # 核心功能脚本
└── README.md                 # 本文档
```

### 白名单配置文件

位置：`/data/adb/clipboard_whitelist.txt`

格式：
```
# 这是注释
com.package.name1
com.package.name2
# 可以添加更多应用包名
```

## 常见问题

### Q1: 安装后不生效？

1. 确认已重启设备
2. 检查应用包名是否正确
3. 查看日志：
   ```bash
   logcat -s ClipboardWhitelist
   ```

### Q2: 如何验证模块是否工作？

1. 检查模块状态：
   - Magisk: 在 Magisk Manager 中查看模块是否已启用
   - KernelSU: 在 KernelSU Manager 中查看模块是否已启用

2. 查看应用权限：
   ```bash
   appops get 应用包名 READ_CLIPBOARD
   ```
   应该显示 `allow`

3. 查看日志：
   ```bash
   logcat | grep ClipboardWhitelist
   ```

### Q3: Android 版本低于 10 可以使用吗？

Android 10 以下版本没有后台剪贴板读取限制，不需要使用本模块。模块会自动检测系统版本，在低版本上不会执行任何操作。

### Q4: Magisk 和 KernelSU 版本有什么区别？

功能完全相同，只是模块 ID 和配置文件不同，以适配不同的 Root 管理器。

### Q5: 可以同时安装两个版本吗？

不建议。选择与你的 Root 管理器匹配的版本即可。

## 卸载方法

### Magisk
1. 打开 Magisk Manager
2. 点击「模块」
3. 找到本模块，点击删除图标
4. 重启设备

### KernelSU
1. 打开 KernelSU Manager
2. 点击「模块」
3. 找到本模块，点击卸载
4. 重启设备

## 技术原理

本模块通过 `appops` 命令修改应用的 `READ_CLIPBOARD` 权限：

```bash
appops set <package_name> READ_CLIPBOARD allow
```

这是 Android 系统提供的合法权限管理方式，不会破坏系统完整性。

## 兼容性

已测试的环境：
- ✅ Android 10 - 14
- ✅ Magisk 20.4+
- ✅ KernelSU 0.6.0+

## 适用场景

本模块适用于以下应用和场景：
- SyncClipboard（剪贴板同步）
- AutoX.js（自动化脚本）
- Tasker（任务自动化）
- 其他需要后台读取剪贴板的应用

## 隐私说明

⚠️ **重要提示**：
- 本模块仅授予指定应用读取剪贴板的权限
- 请只为可信任的应用授权
- 授权后，应用可以在后台读取您复制的所有内容
- 建议定期检查白名单，移除不再需要的应用

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

## 相关项目

- [SyncClipboard](https://github.com/Jeric-X/SyncClipboard) - 跨平台剪贴板同步工具
- [Riru-ClipboardWhitelist](https://github.com/Kr328/Riru-ClipboardWhitelist) - 基于 Riru 的剪贴板白名单

## 更新日志

### v1.0.0 (2024-02-09)
- 🎉 首次发布
- ✅ 支持 Magisk 和 KernelSU
- ✅ 支持 Android 10-14
- ✅ 支持自定义白名单

---

**注意**: 本模块仅用于学习和研究目的，请勿用于非法用途。
