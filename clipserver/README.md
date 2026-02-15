# SyncClipboard Android Backend

Go 后端服务器，用于 SyncClipboard Magisk/KernelSU/APatch 模块。

## 功能特性

- 剪贴板读写（需要 Root 权限）
- WebDAV 同步支持
- RESTful API
- 配置管理
- 跨平台编译（ARM64/ARM32/x86/x86_64）

## 本地测试

### 编译

```bash
go build -o clipserver ./cmd/clipserver
```

### 运行

```bash
./clipserver -port 8964 -config ./test-config.json
```

### 访问

打开浏览器访问：`http://localhost:8964`

## API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/api/config` | GET | 获取配置 |
| `/api/config` | POST | 更新配置 |
| `/api/clipboard` | GET | 获取剪贴板内容 |
| `/api/sync` | POST | 立即同步到 WebDAV |

## 配置文件格式

```json
{
  "webdav_url": "https://example.com/dav",
  "webdav_username": "user",
  "webdav_password": "pass",
  "sync_interval": 60,
  "enabled": false
}
```

## 交叉编译

### Linux/macOS

运行项目根目录的 `build.sh` 脚本：

```bash
cd ..
bash build.sh
```

### Windows

```powershell
# ARM64
$env:CGO_ENABLED=0; $env:GOOS="linux"; $env:GOARCH="arm64"
go build -ldflags="-s -w" -o ../bin/arm64-v8a/clipserver ./cmd/clipserver

# ARMv7
$env:CGO_ENABLED=0; $env:GOOS="linux"; $env:GOARCH="arm"; $env:GOARM="7"
go build -ldflags="-s -w" -o ../bin/armeabi-v7a/clipserver ./cmd/clipserver

# x86_64
$env:CGO_ENABLED=0; $env:GOOS="linux"; $env:GOARCH="amd64"
go build -ldflags="-s -w" -o ../bin/x86_64/clipserver ./cmd/clipserver

# x86
$env:CGO_ENABLED=0; $env:GOOS="linux"; $env:GOARCH="386"
go build -ldflags="-s -w" -o ../bin/x86/clipserver ./cmd/clipserver
```

## 测试

运行所有测试：

```bash
go test ./... -v
```

运行特定模块测试：

```bash
go test ./internal/config -v
go test ./internal/clipboard -v
go test ./internal/webdav -v
go test ./internal/handlers -v
```

## 项目结构

```
clipserver/
├── cmd/
│   └── clipserver/
│       └── main.go           # 主程序入口
├── internal/
│   ├── clipboard/            # 剪贴板处理
│   │   ├── clipboard.go
│   │   └── clipboard_test.go
│   ├── config/               # 配置管理
│   │   ├── config.go
│   │   └── config_test.go
│   ├── handlers/             # HTTP 处理器
│   │   ├── handlers.go
│   │   └── handlers_test.go
│   └── webdav/               # WebDAV 客户端
│       ├── client.go
│       └── client_test.go
├── go.mod
├── go.sum
└── README.md
```

## 依赖

- Go 1.21+
- github.com/studio-b12/gowebdav v0.9.0

## 注意事项

1. 剪贴板操作需要 Root 权限
2. 在非 Android 环境下，剪贴板测试会失败（这是预期行为）
3. WebDAV 测试需要实际的 WebDAV 服务器（默认跳过）
4. 编译时使用 `-ldflags="-s -w"` 减小二进制文件大小

## 许可证

与主项目相同
