# SyncClipboard Helper APK

这是 SyncClipboard 的辅助 APK，用于访问 Android 系统剪贴板。

## 为什么需要这个 APK？

在深度定制的 Android ROM（如 ColorOS）上，标准的 shell 命令（`cmd clipboard`、`service call clipboard`）无法正常工作。这个 APK 通过 Android API 直接访问系统剪贴板，确保在所有 ROM 上都能正常工作。

## 功能

- 读取系统剪贴板内容
- 写入内容到系统剪贴板
- 通过文件协议与 Go 服务通信
- 无需用户交互，后台运行
- 自动授权（通过 Magisk 模块）

## 通信协议

APK 与 Go 服务通过以下文件通信：

- `/data/local/tmp/clipboard_cmd.txt` - 命令文件（"get" 或 "set"）
- `/data/local/tmp/clipboard_data.txt` - 数据文件
- `/data/local/tmp/clipboard_status.txt` - 状态文件（"ok" 或 "error: ..."）

## 构建

需要 Android SDK 和 Java 8+：

```bash
cd cliphelper
./build.sh
```

或使用 Android Studio 打开项目并构建。

GitHub Actions 会自动构建 APK。

## 安装

APK 会通过 Magisk 模块自动安装到 `/system/priv-app/SyncClipboardHelper/`。

## 权限

- `READ_CLIPBOARD_IN_BACKGROUND` - 后台读取剪贴板（Android 10+）
- `com.oppo.permission.safe.CLIPBOARD` - ColorOS 特殊权限
- `com.oplus.permission.safe.CLIPBOARD` - OPPO 特殊权限

权限会在模块安装后自动授予（通过 `post-fs-data.sh`）。

## 兼容性

- 最低 Android 5.0 (API 21)
- 目标 Android 12 (API 31)
- 支持所有 ROM（原生、MIUI、ColorOS、EMUI 等）
# Build
# Trigger rebuild
