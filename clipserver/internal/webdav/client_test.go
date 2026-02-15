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
