package webdav

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/studio-b12/gowebdav"
)

var (
	ErrNotConfigured = errors.New("WebDAV client not configured")
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
