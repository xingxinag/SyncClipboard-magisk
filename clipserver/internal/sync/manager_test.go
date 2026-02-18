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
