package clipboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func withHookTestPaths(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	oldState := hookStatePath
	oldCmd := hookCommandPath
	oldAck := hookAckPath
	oldTimeout := hookWaitTimeout
	oldPoll := hookPollInterval

	hookStatePath = filepath.Join(dir, "state.json")
	hookCommandPath = filepath.Join(dir, "command.json")
	hookAckPath = filepath.Join(dir, "ack.json")
	hookWaitTimeout = 200 * time.Millisecond
	hookPollInterval = 10 * time.Millisecond

	t.Cleanup(func() {
		hookStatePath = oldState
		hookCommandPath = oldCmd
		hookAckPath = oldAck
		hookWaitTimeout = oldTimeout
		hookPollInterval = oldPoll
	})

	return dir
}

func TestGetClipboardSystemHook_OK(t *testing.T) {
	withHookTestPaths(t)

	state := hookStatePayload{Content: "hello-hook", Timestamp: time.Now().Unix()}
	raw, _ := json.Marshal(state)
	if err := os.WriteFile(hookStatePath, raw, 0644); err != nil {
		t.Fatalf("write state failed: %v", err)
	}

	got, err := getClipboardSystemHook()
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if got != "hello-hook" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestGetClipboardSystemHook_Empty(t *testing.T) {
	withHookTestPaths(t)
	state := hookStatePayload{Content: "   ", Timestamp: time.Now().Unix()}
	raw, _ := json.Marshal(state)
	_ = os.WriteFile(hookStatePath, raw, 0644)

	_, err := getClipboardSystemHook()
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty error, got: %v", err)
	}
}

func TestSetClipboardSystemHook_Timeout(t *testing.T) {
	withHookTestPaths(t)

	err := setClipboardSystemHook("payload")
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout error, got: %v", err)
	}

	if _, statErr := os.Stat(hookCommandPath); statErr != nil {
		t.Fatalf("expected command file to be written, got: %v", statErr)
	}
}

func TestSetClipboardSystemHook_OK(t *testing.T) {
	withHookTestPaths(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			data, err := os.ReadFile(hookCommandPath)
			if err == nil {
				var cmd hookCommandPayload
				if json.Unmarshal(data, &cmd) == nil && cmd.RequestID != "" {
					ack := hookAckPayload{RequestID: cmd.RequestID, Status: "ok", Timestamp: time.Now().Unix()}
					raw, _ := json.Marshal(ack)
					_ = os.WriteFile(hookAckPath, raw, 0644)
					return
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	err := setClipboardSystemHook("payload")
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	<-done
}
