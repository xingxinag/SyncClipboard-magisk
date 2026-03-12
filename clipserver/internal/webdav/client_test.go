package webdav

import (
	"errors"
	"fmt"
	"net"
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

func TestClassifyNetworkError_DNSLoopbackRefused(t *testing.T) {
	err := fmt.Errorf("dial tcp: lookup quwenjian.com on [::1]:53: read udp [::1]:37941->[::1]:53: read: connection refused")
	if !isDNSLoopbackRefused(err) {
		t.Fatalf("expected dns loopback refused to be detected")
	}
}

func TestClassifyNetworkError_DNSError(t *testing.T) {
	err := &net.DNSError{Err: "server misbehaving", Name: "quwenjian.com", IsNotFound: false}
	if !isNetworkError(err) {
		t.Fatalf("expected DNSError to be recognized as network error")
	}
}

func TestClassifyNetworkError_NonNetwork(t *testing.T) {
	err := errors.New("http status 401 unauthorized")
	if isNetworkError(err) {
		t.Fatalf("did not expect non-network error to be recognized as network error")
	}
}
