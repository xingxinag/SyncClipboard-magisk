package clipboard

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const hookBaseDir = "/data/system/syncclipboard_hook"

var (
	hookStatePath    = filepath.Join(hookBaseDir, "clipboard_state.json")
	hookCommandPath  = filepath.Join(hookBaseDir, "clipboard_command.json")
	hookAckPath      = filepath.Join(hookBaseDir, "clipboard_ack.json")
	hookWaitTimeout  = 1200 * time.Millisecond
	hookPollInterval = 40 * time.Millisecond
)

type hookStatePayload struct {
	Content   string `json:"content"`
	Hash      string `json:"hash,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type hookCommandPayload struct {
	RequestID string `json:"request_id"`
	Action    string `json:"action"`
	Content   string `json:"content,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type hookAckPayload struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func getClipboardSystemHook() (string, error) {
	data, err := os.ReadFile(hookStatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("system hook state not available: %w (LSPosed module likely not enabled for android/system_server)", err)
		}
		return "", fmt.Errorf("system hook state not available: %w", err)
	}

	var state hookStatePayload
	if err := json.Unmarshal(data, &state); err != nil {
		return "", fmt.Errorf("invalid system hook state: %w", err)
	}

	content := strings.TrimSpace(state.Content)
	if content == "" {
		return "", errors.New("system hook clipboard is empty")
	}

	return content, nil
}

func setClipboardSystemHook(content string) error {
	if strings.TrimSpace(content) == "" {
		return errors.New("empty clipboard content")
	}

	if err := os.MkdirAll(filepath.Dir(hookCommandPath), 0755); err != nil {
		return fmt.Errorf("failed to create hook dir: %w", err)
	}

	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	cmd := hookCommandPayload{
		RequestID: requestID,
		Action:    "set",
		Content:   content,
		Timestamp: time.Now().Unix(),
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal hook command: %w", err)
	}

	if err := os.WriteFile(hookCommandPath, payload, 0644); err != nil {
		return fmt.Errorf("failed to write hook command: %w", err)
	}

	deadline := time.Now().Add(hookWaitTimeout)
	for time.Now().Before(deadline) {
		ackData, readErr := os.ReadFile(hookAckPath)
		if readErr == nil {
			var ack hookAckPayload
			if jsonErr := json.Unmarshal(ackData, &ack); jsonErr == nil && ack.RequestID == requestID {
				if ack.Status == "ok" {
					return nil
				}
				if ack.Error != "" {
					return fmt.Errorf("system hook set failed: %s", ack.Error)
				}
				return errors.New("system hook set failed")
			}
		}
		time.Sleep(hookPollInterval)
	}

	if !isSystemHookModuleInstalled() {
		return errors.New("system hook set timeout (system hook module package missing)")
	}

	return errors.New("system hook set timeout (LSPosed scope for android/system_server may be disabled)")
}

func isSystemHookModuleInstalled() bool {
	cmd := exec.Command("su", "-c", "pm path com.syncclipboard.systemhook")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}
