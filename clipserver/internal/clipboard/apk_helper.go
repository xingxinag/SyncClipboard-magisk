package clipboard

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	// APK 结果文件（由 helper 写入）
	DataFile   = "/data/user/0/com.syncclipboard.helper/files/clipboard_data.txt"
	StatusFile = "/data/user/0/com.syncclipboard.helper/files/clipboard_status.txt"

	// APK 包名和服务
	APKPackage = "com.syncclipboard.helper"
	APKService = "com.syncclipboard.helper/.ClipboardService"
)

func repairHelperFilesOwnership() {
	repairCmd := `uid=$(dumpsys package com.syncclipboard.helper | grep userId= | head -1 | cut -d= -f2); if [ -n "$uid" ]; then mkdir -p /data/user/0/com.syncclipboard.helper/files; chown $uid:$uid /data/user/0/com.syncclipboard.helper/files; chmod 771 /data/user/0/com.syncclipboard.helper/files; fi`
	_ = exec.Command("su", "-c", repairCmd).Run()
}

func startHelper(op, content string) error {
	cmdStr := fmt.Sprintf("am start-foreground-service --user 0 -n %s --es op %s", APKService, op)
	if content != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		cmdStr += fmt.Sprintf(" --es text_b64 '%s'", encoded)
	}
	cmd := exec.Command("su", "-c", cmdStr)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// getClipboardAPK 通过辅助 APK 读取系统剪贴板
func getClipboardAPK() (string, error) {
	repairHelperFilesOwnership()

	// 清空上次结果
	os.Remove(StatusFile)
	os.Remove(DataFile)

	if err := startHelper("get", ""); err != nil {
		return "", fmt.Errorf("failed to start APK service: %w", err)
	}

	// 等待处理完成（最多 3 秒）
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if _, err := os.Stat(StatusFile); err == nil {
			break
		}
	}

	// 读取状态
	statusData, err := os.ReadFile(StatusFile)
	if err != nil {
		return "", fmt.Errorf("failed to read status: %w", err)
	}

	status := strings.TrimSpace(string(statusData))
	if status != "ok" {
		return "", fmt.Errorf("APK returned error: %s", status)
	}

	// 读取数据
	data, err := os.ReadFile(DataFile)
	if err != nil {
		return "", fmt.Errorf("failed to read data: %w", err)
	}

	content := string(data)
	if content == "" {
		return "", fmt.Errorf("clipboard is empty")
	}

	return content, nil
}

// setClipboardAPK 通过辅助 APK 写入系统剪贴板
func setClipboardAPK(content string) error {
	repairHelperFilesOwnership()

	// 清空上次状态
	os.Remove(StatusFile)

	if err := startHelper("set", content); err != nil {
		return fmt.Errorf("failed to start APK service: %w", err)
	}

	// 等待处理完成（最多 3 秒）
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if _, err := os.Stat(StatusFile); err == nil {
			break
		}
	}

	// 读取状态
	statusData, err := os.ReadFile(StatusFile)
	if err != nil {
		return fmt.Errorf("failed to read status: %w", err)
	}

	status := strings.TrimSpace(string(statusData))
	if status != "ok" {
		return fmt.Errorf("APK returned error: %s", status)
	}

	return nil
}

// checkAPKInstalled 检查辅助 APK 是否已安装
func checkAPKInstalled() bool {
	cmd := exec.Command("su", "-c", fmt.Sprintf("pm list packages | grep %s", APKPackage))
	err := cmd.Run()
	return err == nil
}
