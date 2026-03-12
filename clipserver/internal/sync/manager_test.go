package sync

import (
	"errors"
	"testing"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/syncdata"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/webdav"
)

type fakeSyncClient struct {
	downloadData  *syncdata.ClipboardData
	downloadErr   error
	uploadErr     error
	uploadCalled  bool
	uploadPayload *syncdata.ClipboardData
}

func (f *fakeSyncClient) UploadClipboard(data *syncdata.ClipboardData) error {
	f.uploadCalled = true
	f.uploadPayload = data
	return f.uploadErr
}

func (f *fakeSyncClient) DownloadClipboard() (*syncdata.ClipboardData, error) {
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}
	return f.downloadData, nil
}

func withClipboardStubs(t *testing.T, getFn func() (string, error), setFn func(string) error) {
	t.Helper()
	origGet := clipboardGetFn
	origSet := clipboardSetFn
	clipboardGetFn = getFn
	clipboardSetFn = setFn
	t.Cleanup(func() {
		clipboardGetFn = origGet
		clipboardSetFn = origSet
	})
}

func TestSyncNowUploadsLocalClipboard(t *testing.T) {
	client := &fakeSyncClient{}
	m := NewManager(&config.Config{}, client)

	withClipboardStubs(t,
		func() (string, error) { return "LOCAL_TEXT", nil },
		func(content string) error { return nil },
	)

	if err := m.SyncNow(); err != nil {
		t.Fatalf("SyncNow returned error: %v", err)
	}

	if !client.uploadCalled {
		t.Fatalf("expected upload in SyncNow")
	}
	if client.uploadPayload == nil || client.uploadPayload.GetText() != "LOCAL_TEXT" {
		t.Fatalf("expected LOCAL_TEXT upload payload")
	}
}

func TestUploadNowReturnsNotConfiguredWithoutClient(t *testing.T) {
	m := NewManager(&config.Config{}, nil)

	if err := m.UploadNow(); !errors.Is(err, webdav.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestDownloadNowWritesRemoteClipboard(t *testing.T) {
	client := &fakeSyncClient{downloadData: syncdata.NewTextClipboard("REMOTE_TEXT")}
	m := NewManager(&config.Config{}, client)
	var setValue string

	withClipboardStubs(t,
		func() (string, error) { return "IGNORED", nil },
		func(content string) error {
			setValue = content
			return nil
		},
	)

	if err := m.DownloadNow(); err != nil {
		t.Fatalf("DownloadNow returned error: %v", err)
	}

	if setValue != "REMOTE_TEXT" {
		t.Fatalf("expected clipboard set to REMOTE_TEXT, got %q", setValue)
	}
	if client.uploadCalled {
		t.Fatalf("download should not trigger upload")
	}
}

// TestDownloadNowEmptyHashNoPanic 回归测试：远端数据 hash 字段为空时不应 panic
func TestDownloadNowEmptyHashNoPanic(t *testing.T) {
	// 模拟旧版服务端返回的无 hash 数据
	remoteData := &syncdata.ClipboardData{
		Type: "Text",
		Hash: "", // 关键：hash 为空
		Text: "clipboard content from old server",
		Size: 34,
	}
	client := &fakeSyncClient{downloadData: remoteData}
	m := NewManager(&config.Config{}, client)

	withClipboardStubs(t,
		func() (string, error) { return "", nil },
		func(content string) error { return nil },
	)

	// 不应 panic
	if err := m.DownloadNow(); err != nil {
		t.Fatalf("DownloadNow with empty hash returned error: %v", err)
	}
}

// TestStartSkipsMonitorWhenUploadDisabled 验证仅开启自动拉取时不创建监听器（省电）
func TestStartSkipsMonitorWhenUploadDisabled(t *testing.T) {
	client := &fakeSyncClient{downloadData: syncdata.NewTextClipboard("x")}
	cfg := &config.Config{
		Enabled:             true,
		AutoUploadEnabled:   false, // 上传关闭
		AutoDownloadEnabled: true,  // 拉取开启
		SyncInterval:        3600,  // 极长间隔，避免测试期间真正触发下载
	}
	m := NewManager(cfg, client)
	withClipboardStubs(t,
		func() (string, error) { return "", nil },
		func(string) error { return nil },
	)
	m.Start()
	defer m.Stop()

	if m.monitor != nil {
		t.Error("monitor should be nil when AutoUploadEnabled=false")
	}
}

// TestStartSkipsMonitorWhenBothDisabled 验证两个自动开关都关闭时 manager 不启动
func TestStartSkipsMonitorWhenBothDisabled(t *testing.T) {
	client := &fakeSyncClient{}
	cfg := &config.Config{
		Enabled:             true,
		AutoUploadEnabled:   false,
		AutoDownloadEnabled: false,
	}
	m := NewManager(cfg, client)
	m.Start()

	if m.running {
		t.Error("manager should not be running when both auto switches are off")
	}
	if m.monitor != nil {
		t.Error("monitor should be nil when both auto switches are off")
	}
}

// TestStartCreatesMonitorWhenUploadEnabled 验证开启自动上传时监听器被创建
func TestStartCreatesMonitorWhenUploadEnabled(t *testing.T) {
	client := &fakeSyncClient{}
	cfg := &config.Config{
		Enabled:             true,
		AutoUploadEnabled:   true,
		AutoDownloadEnabled: false,
		SyncInterval:        3600,
	}
	m := NewManager(cfg, client)
	withClipboardStubs(t,
		func() (string, error) { return "", nil },
		func(string) error { return nil },
	)
	m.Start()
	defer m.Stop()

	if m.monitor == nil {
		t.Error("monitor should be created when AutoUploadEnabled=true")
	}
}

// TestShortHash 验证 shortHash 对空串和短串都安全
func TestShortHash(t *testing.T) {
	cases := []struct {
		input string
		n     int
		want  string
	}{
		{"", 8, ""},
		{"ABC", 8, "ABC"},
		{"ABCDEFGH", 8, "ABCDEFGH"},
		{"ABCDEFGHI", 8, "ABCDEFGH"},
	}
	for _, c := range cases {
		got := shortHash(c.input, c.n)
		if got != c.want {
			t.Errorf("shortHash(%q, %d) = %q, want %q", c.input, c.n, got, c.want)
		}
	}
}
