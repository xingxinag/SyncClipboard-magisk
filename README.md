# SyncClipboard

> **v2.0.0 重大更新**：100% 可靠的剪贴板监听 + 多账号管理 + 实时同步

[中文](#中文) | [English](#english)

---

## 中文

基于 Root 权限的跨设备剪贴板同步 Magisk/KernelSU/APatch 模块。

### ✨ 特性

#### 🆕 v2.0.0 新特性

- ⚡ **100% 可靠监听** - 三层监听保障（Logcat → Inotify → Polling），确保剪贴板变化不遗漏
- 🔄 **实时同步** - 剪贴板变化时立即上传（延迟 < 1 秒）
- 👥 **多账号管理** - 支持添加多个 WebDAV 账号，一键切换
- 🧪 **连接测试** - 添加账号前可测试连接是否正常
- 📊 **监听状态显示** - 实时显示监听器工作状态
- 🛡️ **深度定制系统兼容** - 7 种读取方法 + 6 种写入方法，自动降级

#### 核心特性

- 🔄 **自动同步** - 后台自动同步剪贴板内容到 WebDAV
- ☁️ **WebDAV 支持** - 兼容坚果云、Nextcloud 等 WebDAV 服务
- 🔗 **SyncClipboard 兼容** - 使用 SyncClipboard.json 格式，与官方客户端完全兼容
- 🌐 **Web UI** - 现代化的 Web 配置界面（支持中英文）
- 🔧 **灵活配置** - 可配置同步间隔（1-3600 秒）、启用/禁用自动同步
- 📱 **通用兼容** - 一次安装，支持 Magisk/KernelSU/APatch
- 🏗️ **多架构** - 支持 ARM64/ARMv7/x86/x86_64
- 🔐 **智能去重** - 使用 SHA256 哈希避免重复同步

### 📋 系统要求

- **Android**: 5.0+ (API 21+)
- **Root 环境**:
  - Magisk 26.4+ 或
  - KernelSU 0.6.6+ 或
  - APatch 0.10.7+

### 🚀 安装

1. 下载最新的 `SyncClipboard-v2.0.0.zip`
2. 在 Magisk/KernelSU/APatch 管理器中安装模块
3. 重启设备
4. 访问 `http://localhost:8964` 配置 WebDAV

### ⚙️ 配置

#### Web UI 配置

访问 `http://localhost:8964` 进行配置：

1. **账号管理**（v2.0 新增）
   - 添加多个 WebDAV 账号
   - 测试连接是否正常
   - 一键切换激活账号
   - 删除不需要的账号

2. **同步设置**
   - 同步间隔: 自动同步的时间间隔（1-3600 秒）
   - 启用自动同步: 开启/关闭自动同步功能
   - 监听状态: 实时显示剪贴板监听状态

#### 命令行配置

配置文件位于: `/data/adb/syncclipboard/config.json`

```json
{
  "accounts": [
    {
      "id": "acc-1",
      "name": "我的坚果云",
      "url": "https://dav.jianguoyun.com/dav/",
      "username": "your_username",
      "password": "your_password",
      "created": 1700000000
    }
  ],
  "active_account_id": "acc-1",
  "sync_interval": 60,
  "enabled": true
}
```

### 📖 使用说明

#### 自动同步模式

启用自动同步后，模块会：
1. 每隔指定时间检查剪贴板内容
2. 如果内容有变化（通过 SHA256 哈希判断），自动上传到 WebDAV
3. 同时从 WebDAV 下载最新内容并更新本地剪贴板
4. 在后台持续运行，无需手动操作

#### 数据格式

本模块使用 SyncClipboard 官方的 `SyncClipboard.json` 格式，与 Windows/Linux/macOS 客户端完全兼容：

```json
{
  "type": "Text",
  "hash": "A00A60CB9A0F34C816E90D3D1058881EAB9DACADAE6E7753334A1F939490FBD5",
  "text": "剪贴板内容",
  "hasData": false,
  "dataName": null,
  "size": 151
}
```

#### 手动同步

在 Web UI 中点击"立即同步"按钮，手动触发一次同步。

### 🔧 API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/api/config` | GET | 获取配置 |
| `/api/config` | POST | 更新配置 |
| `/api/clipboard` | GET | 获取剪贴板内容 |
| `/api/sync/now` | POST | 立即同步 |
| `/api/sync/status` | GET | 同步状态 |

### 📁 项目结构

```
SyncClipboard-magisk/
├── bin/                    # 编译的二进制文件
│   ├── arm64-v8a/
│   ├── armeabi-v7a/
│   ├── x86_64/
│   └── x86/
├── clipserver/             # Go 后端源码
│   ├── cmd/clipserver/     # 主程序
│   └── internal/           # 内部模块
│       ├── clipboard/      # 剪贴板处理
│       ├── config/         # 配置管理
│       ├── handlers/       # HTTP 处理器
│       ├── sync/           # 同步管理器
│       └── webdav/         # WebDAV 客户端
├── webroot/                # Web UI (Module Manager Integration)
├── customize.sh            # 安装脚本
├── service.sh              # 服务脚本
└── module.prop             # 模块信息
```

### 🛠️ 开发

#### 构建模块

```bash
# Linux/macOS
bash build.sh

# Windows (需要 WSL 或 Git Bash)
bash build.sh
```

#### 编译 Go 后端

```bash
cd clipserver

# 本地测试
go build -o clipserver ./cmd/clipserver

# 交叉编译 (ARM64)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../bin/arm64-v8a/clipserver ./cmd/clipserver
```

#### 运行测试

```bash
cd clipserver
go test ./... -v
```

### 🐛 故障排除

#### 服务未启动

```bash
# 检查服务状态
ps | grep clipserver

# 查看日志
cat /data/adb/syncclipboard/clipserver.log

# 手动启动
/data/adb/modules/syncclipboard/bin/arm64-v8a/clipserver -port 8964 -config /data/adb/syncclipboard/config.json
```

#### 无法访问 Web UI

1. 确认服务已启动
2. 检查端口是否被占用: `netstat -tuln | grep 8964`
3. 尝试使用 `http://127.0.0.1:8964` 访问

#### 剪贴板读取失败

确保模块已正确安装并重启设备。剪贴板操作需要 Root 权限。

### 📝 更新日志

#### v1.0.0 (2026-02-15)

- ✅ 初始版本发布
- ✅ 支持 WebDAV 同步
- ✅ 自动/手动同步模式
- ✅ Web UI 配置界面
- ✅ 多语言支持（中文/英文）
- ✅ 通用环境支持（Magisk/KernelSU/APatch）

### 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 📄 许可证

MIT License

---

## English

Root-based cross-device clipboard synchronization module for Magisk/KernelSU/APatch.

### ✨ Features

- 🔄 **Auto Sync** - Automatically sync clipboard content to WebDAV in background
- ☁️ **WebDAV Support** - Compatible with Jianguoyun, Nextcloud, and other WebDAV services
- 🌐 **Web UI** - Modern web configuration interface (Chinese/English)
- 🔧 **Flexible Config** - Configurable sync interval, enable/disable auto sync
- 📱 **Universal** - One installation for Magisk/KernelSU/APatch
- 🏗️ **Multi-arch** - Supports ARM64/ARMv7/x86/x86_64

### 📋 Requirements

- **Android**: 8.0+ (API 26+)
- **Root Environment**:
  - Magisk 26.4+ or
  - KernelSU 0.6.6+ or
  - APatch 0.10.7+

### 🚀 Installation

1. Download the latest `SyncClipboard_v1.0.0.zip`
2. Install the module in Magisk/KernelSU/APatch Manager
3. Reboot device
4. Visit `http://localhost:8964` to configure WebDAV

### ⚙️ Configuration

#### Web UI Configuration

Visit `http://localhost:8964` to configure:

1. **Account Management**
   - Add multiple WebDAV accounts
   - Test account connectivity
   - Switch active account instantly
   - Remove unused accounts

2. **Sync Settings**
   - Sync Interval: Time interval for auto sync (seconds)
   - Enable Auto Sync: Turn on/off auto sync feature

#### Command Line Configuration

Config file location: `/data/adb/syncclipboard/config.json`

```json
{
  "accounts": [
    {
      "id": "acc-1",
      "name": "Primary Account",
      "url": "https://dav.jianguoyun.com/dav/",
      "username": "your_username",
      "password": "your_password",
      "created": 1700000000
    }
  ],
  "active_account_id": "acc-1",
  "sync_interval": 60,
  "enabled": true
}
```

### 📖 Usage

#### Auto Sync Mode

When auto sync is enabled, the module will:
1. Check clipboard content at specified intervals
2. Automatically upload to WebDAV if content changes
3. Run continuously in background, no manual operation needed

#### Manual Sync

Click "Sync Now" button in Web UI to manually trigger a sync.

### 🔧 API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/config` | GET | Get configuration |
| `/api/config` | POST | Update configuration |
| `/api/clipboard` | GET | Get clipboard content |
| `/api/sync/now` | POST | Sync now |
| `/api/sync/status` | GET | Sync status |

### 🛠️ Development

#### Build Module

```bash
# Linux/macOS
bash build.sh

# Windows (requires WSL or Git Bash)
bash build.sh
```

#### Compile Go Backend

```bash
cd clipserver

# Local testing
go build -o clipserver ./cmd/clipserver

# Cross compile (ARM64)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../bin/arm64-v8a/clipserver ./cmd/clipserver
```

#### Run Tests

```bash
cd clipserver
go test ./... -v
```

### 🐛 Troubleshooting

#### Service Not Started

```bash
# Check service status
ps | grep clipserver

# View logs
cat /data/adb/syncclipboard/clipserver.log

# Start manually
/data/adb/modules/syncclipboard/bin/arm64-v8a/clipserver -port 8964 -config /data/adb/syncclipboard/config.json
```

#### Cannot Access Web UI

1. Confirm service is running
2. Check if port is occupied: `netstat -tuln | grep 8964`
3. Try accessing `http://127.0.0.1:8964`

#### Clipboard Read Failed

Ensure the module is properly installed and device is rebooted. Clipboard operations require Root privileges.

### 📝 Changelog

#### v1.0.0 (2026-02-15)

- ✅ Initial release
- ✅ WebDAV sync support
- ✅ Auto/manual sync modes
- ✅ Web UI configuration interface
- ✅ Multi-language support (Chinese/English)
- ✅ Universal environment support (Magisk/KernelSU/APatch)

### 🤝 Contributing

Issues and Pull Requests are welcome!

### 📄 License

MIT License
