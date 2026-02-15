# SyncClipboard

[中文](#中文) | [English](#english)

---

## 中文

基于 Root 权限的跨设备剪贴板同步 Magisk/KernelSU/APatch 模块。

### ✨ 特性

- 🔄 **自动同步** - 后台自动同步剪贴板内容到 WebDAV
- ☁️ **WebDAV 支持** - 兼容坚果云、Nextcloud 等 WebDAV 服务
- 🌐 **Web UI** - 现代化的 Web 配置界面（支持中英文）
- 🔧 **灵活配置** - 可配置同步间隔、启用/禁用自动同步
- 📱 **通用兼容** - 一次安装，支持 Magisk/KernelSU/APatch
- 🏗️ **多架构** - 支持 ARM64/ARMv7/x86/x86_64

### 📋 系统要求

- **Android**: 8.0+ (API 26+)
- **Root 环境**:
  - Magisk 26.4+ 或
  - KernelSU 0.6.6+ 或
  - APatch 0.10.7+

### 🚀 安装

1. 下载最新的 `SyncClipboard_v1.0.0.zip`
2. 在 Magisk/KernelSU/APatch 管理器中安装模块
3. 重启设备
4. 访问 `http://localhost:8964` 配置 WebDAV

### ⚙️ 配置

#### Web UI 配置

访问 `http://localhost:8964` 进行配置：

1. **WebDAV 配置**
   - WebDAV URL: 你的 WebDAV 服务器地址
   - 用户名: WebDAV 账户用户名
   - 密码: WebDAV 账户密码

2. **同步设置**
   - 同步间隔: 自动同步的时间间隔（秒）
   - 启用自动同步: 开启/关闭自动同步功能

#### 命令行配置

配置文件位于: `/data/adb/syncclipboard/config.json`

```json
{
  "webdav_url": "https://dav.jianguoyun.com/dav/",
  "webdav_username": "your_username",
  "webdav_password": "your_password",
  "sync_interval": 60,
  "enabled": true
}
```

### 📖 使用说明

#### 自动同步模式

启用自动同步后，模块会：
1. 每隔指定时间检查剪贴板内容
2. 如果内容有变化，自动上传到 WebDAV
3. 在后台持续运行，无需手动操作

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
├── webui/                  # Web UI
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

1. **WebDAV Config**
   - WebDAV URL: Your WebDAV server address
   - Username: WebDAV account username
   - Password: WebDAV account password

2. **Sync Settings**
   - Sync Interval: Time interval for auto sync (seconds)
   - Enable Auto Sync: Turn on/off auto sync feature

#### Command Line Configuration

Config file location: `/data/adb/syncclipboard/config.json`

```json
{
  "webdav_url": "https://dav.jianguoyun.com/dav/",
  "webdav_username": "your_username",
  "webdav_password": "your_password",
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
