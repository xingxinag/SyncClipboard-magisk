package webdav

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/studio-b12/gowebdav"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/syncdata"
)

var (
	ErrNotConfigured = errors.New("WebDAV client not configured")
	ErrNetworkStack  = errors.New("network stack unavailable")
)

const (
	SyncClipboardFile = "SyncClipboard.json"
	DataFileDir       = "file"
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
	client.SetTimeout(15 * time.Second)

	// 使用自定义 Resolver 避免 Android 的 loopback DNS 问题
	resolver := &net.Resolver{
		PreferGo: false,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// 强制使用 Google DNS 8.8.8.8:53
			d := net.Dialer{
				Timeout: 5 * time.Second,
			}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  resolver,
	}

	client.SetTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 强制使用 IPv4
			if network == "tcp" {
				network = "tcp4"
			}
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{
			// 跳过证书验证（Android CA 证书路径不同）
			// TODO: 加载 /system/etc/security/cacerts 中的证书
			InsecureSkipVerify: true,
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ForceAttemptHTTP2:     false,
	})
	return &Client{client: client}, nil
}

// UploadContent 上传内容到WebDAV服务器（兼容接口）
func (c *Client) UploadContent(remotePath, content string) error {
	reader := bytes.NewReader([]byte(content))
	err := c.client.WriteStream(remotePath, reader, 0644)
	if err != nil {
		return classifyNetworkErr(err)
	}
	return nil
}

// DownloadContent 从WebDAV服务器下载内容（兼容接口）
func (c *Client) DownloadContent(remotePath string) (string, error) {
	data, err := c.client.Read(remotePath)
	if err != nil {
		return "", classifyNetworkErr(err)
	}
	return string(data), nil
}

// UploadClipboard 上传剪贴板数据到 SyncClipboard.json
func (c *Client) UploadClipboard(data *syncdata.ClipboardData) error {
	jsonStr, err := data.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to convert to JSON: %w", err)
	}

	reader := bytes.NewReader([]byte(jsonStr))
	err = c.client.WriteStream(SyncClipboardFile, reader, 0644)
	if err != nil {
		return classifyNetworkErr(err)
	}
	return nil
}

func (c *Client) UploadClipboardText(content string) (*syncdata.ClipboardData, error) {
	data := syncdata.NewTextClipboard(content)
	if data.NeedsDataFile() {
		if err := c.UploadFile(*data.DataName, []byte(content)); err != nil {
			return nil, err
		}
	}
	if err := c.UploadClipboard(data); err != nil {
		return nil, err
	}
	return data, nil
}

// DownloadClipboard 从 SyncClipboard.json 下载剪贴板数据
func (c *Client) DownloadClipboard() (*syncdata.ClipboardData, error) {
	data, err := c.client.Read(SyncClipboardFile)
	if err != nil {
		return nil, classifyNetworkErr(fmt.Errorf("failed to read file: %w", err))
	}

	clipData, err := syncdata.FromJSON(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return clipData, nil
}

func (c *Client) DownloadClipboardText() (*syncdata.ClipboardData, string, error) {
	clipData, err := c.DownloadClipboard()
	if err != nil {
		return nil, "", err
	}
	if clipData == nil || !clipData.IsText() {
		return clipData, "", nil
	}
	if !clipData.NeedsDataFile() {
		return clipData, clipData.Text, nil
	}
	data, err := c.DownloadFile(*clipData.DataName)
	if err != nil {
		return nil, "", err
	}
	return clipData, string(data), nil
}

func (c *Client) UploadFile(filename string, data []byte) error {
	remotePath, err := dataFilePath(filename)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(data)
	err = c.client.WriteStream(remotePath, reader, 0644)
	if err != nil && isNotFoundError(err) {
		if mkdirErr := c.client.MkdirAll(DataFileDir, 0755); mkdirErr != nil {
			return classifyNetworkErr(mkdirErr)
		}
		err = c.client.WriteStream(remotePath, bytes.NewReader(data), 0644)
	}
	if err != nil {
		return classifyNetworkErr(err)
	}
	return nil
}

func (c *Client) DownloadFile(filename string) ([]byte, error) {
	remotePath, err := dataFilePath(filename)
	if err != nil {
		return nil, err
	}
	data, err := c.client.Read(remotePath)
	if err != nil {
		return nil, classifyNetworkErr(err)
	}
	return data, nil
}

func dataFilePath(filename string) (string, error) {
	cleanName := path.Clean(strings.ReplaceAll(filename, "\\", "/"))
	if cleanName == "." || cleanName == "" || strings.HasPrefix(cleanName, "../") || cleanName == ".." || strings.HasPrefix(cleanName, "/") {
		return "", fmt.Errorf("invalid data filename: %s", filename)
	}
	return path.Join(DataFileDir, cleanName), nil
}

// TestConnection 测试WebDAV连接
func (c *Client) TestConnection() error {
	err := c.client.Connect()
	if err != nil {
		return classifyNetworkErr(err)
	}
	return nil
}

func ensureRootNetwork() error {
	// 直接测试网络连通性，不依赖 getprop（某些 ROM 的 DNS 配置不在标准位置）
	cmd := exec.Command("su", "-c", "ping -c 1 -W 2 8.8.8.8")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%w: network unreachable (ping 8.8.8.8 failed)", ErrNetworkStack)
	}
	return nil
}

func isDNSLoopbackRefused(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "lookup") &&
		(strings.Contains(lower, "on [::1]:53") || strings.Contains(lower, "on 127.0.0.1:53")) &&
		strings.Contains(lower, "connection refused")
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "lookup") || strings.Contains(lower, "i/o timeout") || strings.Contains(lower, "no such host") {
		return true
	}
	return false
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "404") || strings.Contains(lower, "not found") || strings.Contains(lower, "no such file")
}

func ClassifyNetworkErr(err error) error {
	if err == nil {
		return nil
	}
	if isDNSLoopbackRefused(err) {
		return fmt.Errorf("%w: local loopback dns refused ([::1]:53)", ErrNetworkStack)
	}
	if isNetworkError(err) {
		return fmt.Errorf("%w: %v", ErrNetworkStack, err)
	}
	return err
}

func classifyNetworkErr(err error) error {
	return ClassifyNetworkErr(err)
}
