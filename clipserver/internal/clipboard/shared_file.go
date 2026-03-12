package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// ClipboardFile root 用户共享的剪贴板文件
	ClipboardFile = "/data/local/tmp/syncclipboard_shared.txt"
	// ClipboardLock 锁文件
	ClipboardLock = "/data/local/tmp/syncclipboard_shared.lock"
)

// getClipboardSharedFile 通过共享文件读取剪贴板（root 用户之间共享）
func getClipboardSharedFile() (string, error) {
	// 尝试从共享文件读取
	data, err := os.ReadFile(ClipboardFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("shared clipboard file not found (no content yet)")
		}
		return "", fmt.Errorf("failed to read shared clipboard: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("shared clipboard is empty")
	}

	return content, nil
}

// setClipboardSharedFile 通过共享文件写入剪贴板（root 用户之间共享）
func setClipboardSharedFile(content string) error {
	// 确保目录存在
	dir := filepath.Dir(ClipboardFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 写入临时文件
	tmpFile := ClipboardFile + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// 原子性重命名
	if err := os.Rename(tmpFile, ClipboardFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// 确保权限正确
	os.Chmod(ClipboardFile, 0644)

	return nil
}

// getClipboardTermux 尝试通过 Termux API 读取（如果安装了 Termux）
func getClipboardTermux() (string, error) {
	cmd := exec.Command("su", "-c", "termux-clipboard-get")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("termux-clipboard-get failed: %w", err)
	}

	content := strings.TrimSpace(string(output))
	if content == "" {
		return "", fmt.Errorf("termux clipboard is empty")
	}

	return content, nil
}

// setClipboardTermux 尝试通过 Termux API 写入（如果安装了 Termux）
func setClipboardTermux(content string) error {
	cmd := exec.Command("su", "-c", "termux-clipboard-set")
	cmd.Stdin = strings.NewReader(content)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("termux-clipboard-set failed: %w", err)
	}
	return nil
}
