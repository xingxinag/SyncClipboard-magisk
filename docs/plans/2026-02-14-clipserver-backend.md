# SyncClipboard Android Backend 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 为SyncClipboard-Android Magisk模块创建完整的Go后端服务器，实现剪贴板同步、WebDAV集成和配置管理。

**架构：** 采用轻量级HTTP服务器架构，使用标准库实现RESTful API，通过WebDAV协议同步剪贴板内容。所有配置存储在JSON文件中，支持热重载。

**技术栈：** 
- Go 1.21+
- 标准库 net/http
- WebDAV客户端库 github.com/studio-b12/gowebdav
- JSON配置管理

---

## 任务1: 初始化Go模块和项目结构

**文件：**
- 创建: `clipserver/go.mod`
- 创建: `clipserver/go.sum`
- 创建: `clipserver/.gitignore`

**步骤1: 创建clipserver目录并初始化Go模块**

```bash
mkdir -p clipserver
cd clipserver
go mod init github.com/yourusername/syncclipboard-android/clipserver
```

**步骤2: 添加必要的依赖**

```bash
go get github.com/studio-b12/gowebdav@latest
```

预期输出：依赖包成功下载，go.mod和go.sum文件更新

**步骤3: 创建.gitignore文件**

```gitignore
# Binaries
*.exe
*.dll
*.so
*.dylib
bin/

# Test binary
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db
```

**步骤4: 提交初始化**

```bash
git add clipserver/go.mod clipserver/go.sum clipserver/.gitignore
git commit -m "feat: 初始化Go模块和项目结构"
```

---

## 任务2: 实现配置管理模块

**文件：**
- 创建: `clipserver/internal/config/config.go`
- 创建: `clipserver/internal/config/config_test.go`

**步骤1: 编写配置结构体测试**

创建 `clipserver/internal/config/config_test.go`:

```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// 测试配置数据
	testConfig := &Config{
		WebDAVURL:      "https://example.com/dav",
		WebDAVUsername: "testuser",
		WebDAVPassword: "testpass",
		SyncInterval:   60,
		Enabled:        true,
	}

	// 写入测试配置
	data, _ := json.MarshalIndent(testConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)

	// 测试加载
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.WebDAVURL != testConfig.WebDAVURL {
		t.Errorf("Expected URL %s, got %s", testConfig.WebDAVURL, config.WebDAVURL)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	config := &Config{
		WebDAVURL:      "https://example.com/dav",
		WebDAVUsername: "user",
		WebDAVPassword: "pass",
		SyncInterval:   30,
		Enabled:        false,
	}

	err := SaveConfig(configPath, config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}
```

**步骤2: 运行测试验证失败**

```bash
cd clipserver
go test ./internal/config -v
```

预期输出：FAIL - 函数未定义

**步骤3: 实现配置管理逻辑**

创建 `clipserver/internal/config/config.go`:

```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 代表应用配置结构
type Config struct {
	WebDAVURL      string `json:"webdav_url"`
	WebDAVUsername string `json:"webdav_username"`
	WebDAVPassword string `json:"webdav_password"`
	SyncInterval   int    `json:"sync_interval"` // 秒
	Enabled        bool   `json:"enabled"`
}

// LoadConfig 从指定路径加载配置文件
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig 保存配置到指定路径
func SaveConfig(path string, config *Config) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		WebDAVURL:      "",
		WebDAVUsername: "",
		WebDAVPassword: "",
		SyncInterval:   60,
		Enabled:        false,
	}
}
```

**步骤4: 运行测试验证通过**

```bash
go test ./internal/config -v
```

预期输出：PASS

**步骤5: 提交配置管理模块**

```bash
git add clipserver/internal/config/
git commit -m "feat: 实现配置管理模块"
```

---

## 任务3: 实现剪贴板处理逻辑

**文件：**
- 创建: `clipserver/internal/clipboard/clipboard.go`
- 创建: `clipserver/internal/clipboard/clipboard_test.go`

**步骤1: 编写剪贴板验证测试**

创建 `clipserver/internal/clipboard/clipboard_test.go`:

```go
package clipboard

import (
	"strings"
	"testing"
)

func TestValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"空内容", "", true},
		{"正常内容", "Hello World", false},
		{"最大限制边界", strings.Repeat("a", 1024*1024), false},
		{"超过最大限制", strings.Repeat("a", 1024*1024+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContent(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetClipboard(t *testing.T) {
	// 注意：这个测试在非Android环境会失败
	// 这里只是示例结构
	content, err := GetClipboard()
	if err != nil {
		t.Logf("GetClipboard failed (expected on non-Android): %v", err)
		return
	}
	t.Logf("Clipboard content: %s", content)
}
```

**步骤2: 运行测试验证失败**

```bash
go test ./internal/clipboard -v
```

预期输出：FAIL - 函数未定义

**步骤3: 实现剪贴板处理逻辑**

创建 `clipserver/internal/clipboard/clipboard.go`:

```go
package clipboard

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const (
	// MaxClipboardSize 剪贴板内容最大1MB
	MaxClipboardSize = 1024 * 1024
)

var (
	ErrEmptyContent     = errors.New("clipboard content is empty")
	ErrContentTooLarge  = errors.New("clipboard content exceeds maximum size")
	ErrClipboardAccess  = errors.New("failed to access system clipboard")
)

// ValidateContent 验证剪贴板内容是否符合要求
func ValidateContent(content string) error {
	if content == "" {
		return ErrEmptyContent
	}
	if len(content) > MaxClipboardSize {
		return ErrContentTooLarge
	}
	return nil
}

// GetClipboard 从系统剪贴板获取内容（需要Root权限）
func GetClipboard() (string, error) {
	// 使用 am broadcast 获取剪贴板内容
	cmd := exec.Command("su", "-c", "service call clipboard 2 s16 com.android.shell")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrClipboardAccess, err)
	}

	content := strings.TrimSpace(string(output))
	if err := ValidateContent(content); err != nil {
		return "", err
	}

	return content, nil
}

// SetClipboard 设置系统剪贴板内容（需要Root权限）
func SetClipboard(content string) error {
	if err := ValidateContent(content); err != nil {
		return err
	}

	// 使用 am broadcast 设置剪贴板
	cmd := exec.Command("su", "-c", fmt.Sprintf("service call clipboard 1 i32 1 s16 com.android.shell s16 %s", content))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %v", ErrClipboardAccess, err)
	}

	return nil
}
```

**步骤4: 运行测试验证通过**

```bash
go test ./internal/clipboard -v
```

预期输出：PASS（ValidateContent测试通过，GetClipboard在非Android环境会记录日志）

**步骤5: 提交剪贴板模块**

```bash
git add clipserver/internal/clipboard/
git commit -m "feat: 实现剪贴板处理和验证逻辑"
```

---

## 任务4: 实现WebDAV同步客户端

**文件：**
- 创建: `clipserver/internal/webdav/client.go`
- 创建: `clipserver/internal/webdav/client_test.go`

**步骤1: 编写WebDAV客户端测试**

创建 `clipserver/internal/webdav/client_test.go`:

```go
package webdav

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient("https://example.com/dav", "user", "pass")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client == nil {
		t.Error("Expected non-nil client")
	}
}

func TestUploadContent(t *testing.T) {
	// 这是集成测试的示例，实际需要mock WebDAV服务器
	t.Skip("Requires WebDAV server")

	client, _ := NewClient("https://example.com/dav", "user", "pass")
	err := client.UploadContent("test.txt", "Hello World")
	if err != nil {
		t.Errorf("UploadContent failed: %v", err)
	}
}
```

**步骤2: 运行测试验证失败**

```bash
go test ./internal/webdav -v
```

预期输出：FAIL - 函数未定义

**步骤3: 实现WebDAV客户端**

创建 `clipserver/internal/webdav/client.go`:

```go
package webdav

import (
	"bytes"
	"fmt"

	"github.com/studio-b12/gowebdav"
)

// Client 封装WebDAV客户端
type Client struct {
	client *gowebdav.Client
}

// NewClient 创建新的WebDAV客户端
func NewClient(url, username, password string) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("WebDAV URL cannot be empty")
	}

	client := gowebdav.NewClient(url, username, password)
	return &Client{client: client}, nil
}

// UploadContent 上传内容到WebDAV服务器
func (c *Client) UploadContent(remotePath, content string) error {
	reader := bytes.NewReader([]byte(content))
	return c.client.WriteStream(remotePath, reader, 0644)
}

// DownloadContent 从WebDAV服务器下载内容
func (c *Client) DownloadContent(remotePath string) (string, error) {
	data, err := c.client.Read(remotePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TestConnection 测试WebDAV连接
func (c *Client) TestConnection() error {
	return c.client.Connect()
}
```

**步骤4: 运行测试验证通过**

```bash
go test ./internal/webdav -v
```

预期输出：PASS（NewClient通过，UploadContent跳过）

**步骤5: 提交WebDAV客户端**

```bash
git add clipserver/internal/webdav/
git commit -m "feat: 实现WebDAV同步客户端"
```

---

## 任务5: 实现HTTP API处理器

**文件：**
- 创建: `clipserver/internal/handlers/handlers.go`
- 创建: `clipserver/internal/handlers/handlers_test.go`

**步骤1: 编写API处理器测试**

创建 `clipserver/internal/handlers/handlers_test.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetConfigHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()

	h := NewHandler("/tmp/test-config.json")
	h.GetConfigHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
```

**步骤2: 运行测试验证失败**

```bash
go test ./internal/handlers -v
```

预期输出：FAIL - 函数未定义

**步骤3: 实现HTTP处理器**

创建 `clipserver/internal/handlers/handlers.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/clipboard"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/webdav"
)

// Handler 封装所有HTTP处理器
type Handler struct {
	configPath string
	webdavClient *webdav.Client
}

// NewHandler 创建新的处理器实例
func NewHandler(configPath string) *Handler {
	return &Handler{
		configPath: configPath,
	}
}

// HealthHandler 健康检查端点
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetConfigHandler 获取当前配置
func (h *Handler) GetConfigHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		// 返回默认配置
		cfg = config.DefaultConfig()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// UpdateConfigHandler 更新配置
func (h *Handler) UpdateConfigHandler(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := config.SaveConfig(h.configPath, &cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 重新初始化WebDAV客户端
	if cfg.WebDAVURL != "" {
		client, err := webdav.NewClient(cfg.WebDAVURL, cfg.WebDAVUsername, cfg.WebDAVPassword)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.webdavClient = client
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetClipboardHandler 获取当前剪贴板内容
func (h *Handler) GetClipboardHandler(w http.ResponseWriter, r *http.Request) {
	content, err := clipboard.GetClipboard()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"content": content})
}

// SyncNowHandler 立即触发同步
func (h *Handler) SyncNowHandler(w http.ResponseWriter, r *http.Request) {
	if h.webdavClient == nil {
		http.Error(w, "WebDAV not configured", http.StatusBadRequest)
		return
	}

	content, err := clipboard.GetClipboard()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.webdavClient.UploadContent("clipboard.txt", content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
}
```

**步骤4: 运行测试验证通过**

```bash
go test ./internal/handlers -v
```

预期输出：PASS

**步骤5: 提交HTTP处理器**

```bash
git add clipserver/internal/handlers/
git commit -m "feat: 实现HTTP API处理器"
```

---

## 任务6: 实现HTTP服务器主入口

**文件：**
- 创建: `clipserver/cmd/clipserver/main.go`

**步骤1: 创建主入口文件**

创建 `clipserver/cmd/clipserver/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/handlers"
)

const (
	defaultPort       = "9527"
	defaultConfigPath = "/data/adb/syncclipboard/config.json"
)

func main() {
	// 命令行参数
	port := flag.String("port", defaultPort, "HTTP server port")
	configPath := flag.String("config", defaultConfigPath, "Configuration file path")
	flag.Parse()

	// 确保配置目录存在
	configDir := filepath.Dir(*configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	// 创建处理器
	h := handlers.NewHandler(*configPath)

	// 注册路由
	http.HandleFunc("/health", handlers.HealthHandler)
	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetConfigHandler(w, r)
		} else if r.Method == http.MethodPost {
			h.UpdateConfigHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/clipboard", h.GetClipboardHandler)
	http.HandleFunc("/api/sync", h.SyncNowHandler)

	// 静态文件服务（WebUI）
	fs := http.FileServer(http.Dir("/data/adb/syncclipboard/webui"))
	http.Handle("/", fs)

	// 启动服务器
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Starting SyncClipboard server on %s", addr)
	log.Printf("WebUI: http://localhost%s", addr)
	log.Printf("Config: %s", *configPath)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

**步骤2: 测试编译主程序**

```bash
cd clipserver
go build -o bin/clipserver ./cmd/clipserver
```

预期输出：成功编译，生成 bin/clipserver 可执行文件

**步骤3: 提交主入口程序**

```bash
git add clipserver/cmd/clipserver/main.go
git commit -m "feat: 实现HTTP服务器主入口点"
```

---

## 任务7: 更新构建脚本

**文件：**
- 修改: `build.sh`

**步骤1: 更新build.sh以构建Go二进制文件**

修改 `build.sh`，添加Go交叉编译逻辑：

```bash
#!/bin/bash

# 设置错误时退出
set -e

echo "=== Building SyncClipboard Android Module ==="

# 清理旧的构建文件
rm -rf bin/*

# 进入clipserver目录
cd clipserver

echo "Building Go binaries..."

# 构建arm64版本
echo "  - arm64-v8a"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../bin/arm64-v8a/clipserver ./cmd/clipserver

# 构建armv7版本
echo "  - armeabi-v7a"
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o ../bin/armeabi-v7a/clipserver ./cmd/clipserver

# 构建x86版本
echo "  - x86"
CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o ../bin/x86/clipserver ./cmd/clipserver

# 构建x86_64版本
echo "  - x86_64"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/x86_64/clipserver ./cmd/clipserver

cd ..

echo "Binary sizes:"
du -h bin/*/clipserver

echo "=== Build complete ==="
```

**步骤2: 使构建脚本可执行**

```bash
chmod +x build.sh
```

**步骤3: 测试构建脚本**

```bash
./build.sh
```

预期输出：成功为所有架构编译二进制文件

**步骤4: 提交构建脚本更新**

```bash
git add build.sh
git commit -m "feat: 更新构建脚本支持Go交叉编译"
```

---

## 任务8: 创建默认配置文件模板

**文件：**
- 创建: `config/config.json.template`

**步骤1: 创建配置模板**

创建 `config/config.json.template`:

```json
{
  "webdav_url": "",
  "webdav_username": "",
  "webdav_password": "",
  "sync_interval": 60,
  "enabled": false
}
```

**步骤2: 更新customize.sh以复制配置模板**

在 `customize.sh` 中添加配置初始化逻辑：

```bash
# 在安装过程中添加
if [ ! -f "$MODPATH/config/config.json" ]; then
    cp "$MODPATH/config/config.json.template" "$MODPATH/config/config.json"
    ui_print "  - Created default configuration"
fi
```

**步骤3: 提交配置模板**

```bash
git add config/config.json.template
git add customize.sh
git commit -m "feat: 添加默认配置文件模板"
```

---

## 任务9: 集成测试和验证

**步骤1: 运行所有单元测试**

```bash
cd clipserver
go test ./... -v
```

预期输出：所有测试通过

**步骤2: 构建所有架构的二进制文件**

```bash
cd ..
./build.sh
```

预期输出：成功构建所有架构的二进制文件

**步骤3: 验证二进制文件大小**

```bash
ls -lh bin/*/clipserver
```

预期：每个二进制文件应该在5-10MB左右（经过strip优化）

**步骤4: 创建测试README**

创建 `clipserver/README.md` 说明如何测试：

```markdown
# SyncClipboard Android Backend

## 本地测试

1. 编译：`go build -o clipserver ./cmd/clipserver`
2. 运行：`./clipserver -port 9527 -config ./test-config.json`
3. 访问：`http://localhost:9527`

## API端点

- `GET /health` - 健康检查
- `GET /api/config` - 获取配置
- `POST /api/config` - 更新配置
- `GET /api/clipboard` - 获取剪贴板
- `POST /api/sync` - 立即同步

## 交叉编译

运行项目根目录的 `build.sh` 脚本。
```

**步骤5: 最终提交**

```bash
git add clipserver/README.md
git commit -m "docs: 添加后端测试和使用说明"
```

---

## 验收标准

✅ 所有Go测试通过
✅ 成功编译所有架构的二进制文件
✅ HTTP服务器可以启动并响应健康检查
✅ WebUI可以访问配置API
✅ 剪贴板验证逻辑正常工作
✅ WebDAV客户端可以初始化
✅ 所有代码已提交到git

## 后续步骤

1. 在Android设备上测试安装和运行
2. 实现自动同步定时器
3. 添加更完善的错误处理和日志
4. 实现剪贴板变化监听
5. 添加WebUI的实时状态更新
