package sync

import (
	"errors"
	"testing"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/syncdata"
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

func TestSyncNowPullsRemoteWhenDifferent(t *testing.T) {
	client := &fakeSyncClient{downloadData: syncdata.NewTextClipboard("REMOTE_TEXT")}
	m := NewManager(&config.Config{}, client)

	var setValue string
	withClipboardStubs(t,
		func() (string, error) { return "LOCAL_TEXT", nil },
		func(content string) error {
			setValue = content
			return nil
		},
	)

	if err := m.SyncNow(); err != nil {
		t.Fatalf("SyncNow returned error: %v", err)
	}

	if setValue != "REMOTE_TEXT" {
		t.Fatalf("expected clipboard set to REMOTE_TEXT, got %q", setValue)
	}
	if client.uploadCalled {
		t.Fatalf("expected no upload when remote differs")
	}
	if m.lastHash != syncdata.NewTextClipboard("REMOTE_TEXT").GetHash() {
		t.Fatalf("expected lastHash to track remote hash")
	}
}

func TestSyncNowPushesWhenRemoteSame(t *testing.T) {
	client := &fakeSyncClient{downloadData: syncdata.NewTextClipboard("LOCAL_TEXT")}
	m := NewManager(&config.Config{}, client)

	setCalled := false
	withClipboardStubs(t,
		func() (string, error) { return "LOCAL_TEXT", nil },
		func(content string) error {
			setCalled = true
			return nil
		},
	)

	if err := m.SyncNow(); err != nil {
		t.Fatalf("SyncNow returned error: %v", err)
	}

	if setCalled {
		t.Fatalf("expected clipboard set not called when remote same")
	}
	if !client.uploadCalled {
		t.Fatalf("expected upload when remote same")
	}
	if client.uploadPayload == nil || client.uploadPayload.GetText() != "LOCAL_TEXT" {
		t.Fatalf("expected LOCAL_TEXT upload payload")
	}
}

func TestSyncNowPushesWhenDownloadFails(t *testing.T) {
	client := &fakeSyncClient{downloadErr: errors.New("download failed")}
	m := NewManager(&config.Config{}, client)

	withClipboardStubs(t,
		func() (string, error) { return "LOCAL_TEXT", nil },
		func(content string) error { return nil },
	)

	if err := m.SyncNow(); err != nil {
		t.Fatalf("SyncNow returned error: %v", err)
	}
	if !client.uploadCalled {
		t.Fatalf("expected upload when download fails")
	}
}
