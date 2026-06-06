package serverclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/syncdata"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/webdav"
)

const (
	SyncClipboardFile = "SyncClipboard.json"
	DataFileDir       = "file"
)

type Client struct {
	baseURL    string
	username   string
	password   string
	serverType config.ServerType
	httpClient *http.Client
}

func NewClient(account config.ServerAccount) (*Client, error) {
	if account.URL == "" {
		return nil, fmt.Errorf("server URL cannot be empty")
	}
	baseURL := strings.TrimRight(account.URL, "/") + "/"
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	return &Client{
		baseURL:    baseURL,
		username:   account.Username,
		password:   account.Password,
		serverType: account.EffectiveType(),
		httpClient: &http.Client{Timeout: 15 * time.Second, Transport: newTransport()},
	}, nil
}

func (c *Client) UploadClipboard(data *syncdata.ClipboardData) error {
	jsonStr, err := data.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to convert to JSON: %w", err)
	}
	return c.put(SyncClipboardFile, []byte(jsonStr), "application/json", []int{http.StatusOK, http.StatusCreated, http.StatusNoContent})
}

func (c *Client) DownloadClipboard() (*syncdata.ClipboardData, error) {
	data, err := c.get(SyncClipboardFile, []int{http.StatusOK})
	if err != nil {
		return nil, err
	}
	clipData, err := syncdata.FromJSON(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return clipData, nil
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
	return c.put(remotePath, data, "application/octet-stream", []int{http.StatusOK, http.StatusCreated, http.StatusNoContent})
}

func (c *Client) DownloadFile(filename string) ([]byte, error) {
	remotePath, err := dataFilePath(filename)
	if err != nil {
		return nil, err
	}
	return c.get(remotePath, []int{http.StatusOK})
}

func (c *Client) TestConnection() error {
	_, err := c.get(SyncClipboardFile, []int{http.StatusOK, http.StatusNotFound})
	return err
}

func (c *Client) put(remotePath string, data []byte, contentType string, expected []int) error {
	req, err := http.NewRequest(http.MethodPut, c.urlFor(remotePath), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	c.authorize(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return webdav.ClassifyNetworkErr(err)
	}
	defer resp.Body.Close()
	if !statusAllowed(resp.StatusCode, expected) {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) get(remotePath string, expected []int) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.urlFor(remotePath), nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, webdav.ClassifyNetworkErr(err)
	}
	defer resp.Body.Close()
	if !statusAllowed(resp.StatusCode, expected) {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) authorize(req *http.Request) {
	if c.username == "" && c.password == "" {
		return
	}
	token := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
	req.Header.Set("Authorization", "Basic "+token)
}

func (c *Client) urlFor(remotePath string) string {
	return c.baseURL + strings.TrimLeft(remotePath, "/")
}

func dataFilePath(filename string) (string, error) {
	cleanName := path.Clean(strings.ReplaceAll(filename, "\\", "/"))
	if cleanName == "." || cleanName == "" || strings.HasPrefix(cleanName, "../") || cleanName == ".." || strings.HasPrefix(cleanName, "/") {
		return "", fmt.Errorf("invalid data filename: %s", filename)
	}
	return path.Join(DataFileDir, cleanName), nil
}

func statusAllowed(status int, expected []int) bool {
	for _, code := range expected {
		if status == code {
			return true
		}
	}
	return false
}

func newTransport() *http.Transport {
	resolver := &net.Resolver{
		PreferGo: false,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second, Resolver: resolver}
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if network == "tcp" {
				network = "tcp4"
			}
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ForceAttemptHTTP2:     false,
	}
}
