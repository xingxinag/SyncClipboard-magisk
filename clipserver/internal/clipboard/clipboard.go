package clipboard

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const (
	// MaxClipboardSize 剪贴板内容最大1MB
	MaxClipboardSize = 1024 * 1024
)

var (
	ErrEmptyContent    = errors.New("clipboard content is empty")
	ErrContentTooLarge = errors.New("clipboard content exceeds maximum size")
	ErrClipboardAccess = errors.New("failed to access system clipboard")
)

// ValidateContent 验证剪贴板内容是否符合要求
func ValidateContent(content string) error {
	if content == "" {
		return ErrEmptyContent
	}
	if len(content) > MaxClipboardSize {
		return ErrContentTooLarge
	}
	return nil
}

// GetClipboard 从系统剪贴板获取内容（需要Root权限）
// 使用多种方法尝试，确保在各种深度定制系统上都能工作
func GetClipboard() (string, error) {
	// 方法1: 使用 cmd clipboard (Android 10+)
	content, err := getClipboardCmd()
	if err == nil && content != "" {
		return content, nil
	}

	// 方法2: 使用 service call (通用方法)
	content, err = getClipboardServiceCall()
	if err == nil && content != "" {
		return content, nil
	}

	// 方法3: 使用 am broadcast (备用方法)
	content, err = getClipboardAmBroadcast()
	if err == nil && content != "" {
		return content, nil
	}

	// 方法4: 使用 content provider (最后的尝试)
	content, err = getClipboardContentProvider()
	if err == nil && content != "" {
		return content, nil
	}

	return "", fmt.Errorf("%w: all methods failed", ErrClipboardAccess)
}

// getClipboardCmd 使用 cmd clipboard 命令（Android 10+）
func getClipboardCmd() (string, error) {
	cmd := exec.Command("su", "-c", "cmd clipboard get-text")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(output))
	// 移除可能的错误信息
	if strings.Contains(content, "Error") || strings.Contains(content, "Exception") {
		return "", errors.New("cmd clipboard failed")
	}

	return content, nil
}

// getClipboardServiceCall 使用 service call 方法
func getClipboardServiceCall() (string, error) {
	// 获取剪贴板内容
	cmd := exec.Command("su", "-c", "service call clipboard 2 s16 com.android.shell")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// 解析输出 (service call 返回的是十六进制格式)
	content := parseServiceCallOutput(string(output))
	return content, nil
}

// getClipboardAmBroadcast 使用 am broadcast 方法
func getClipboardAmBroadcast() (string, error) {
	// 这个方法需要一个接收器，暂时返回错误
	return "", errors.New("am broadcast method not implemented")
}

// getClipboardContentProvider 使用 content provider 方法
func getClipboardContentProvider() (string, error) {
	// 尝试通过 content provider 读取
	cmd := exec.Command("su", "-c", "content query --uri content://clipboard/text")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(output))
	return content, nil
}

// parseServiceCallOutput 解析 service call 的输出
func parseServiceCallOutput(output string) string {
	// service call 返回格式类似: Result: Parcel(00000000 00000014 'text content'  00000000)
	// 需要提取引号中的内容
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "'") {
			start := strings.Index(line, "'")
			end := strings.LastIndex(line, "'")
			if start != -1 && end != -1 && start < end {
				return line[start+1 : end]
			}
		}
	}
	return strings.TrimSpace(output)
}

// SetClipboard 设置系统剪贴板内容（需要Root权限）
// 使用多种方法尝试，确保在各种深度定制系统上都能工作
func SetClipboard(content string) error {
	if err := ValidateContent(content); err != nil {
		return err
	}

	// 方法1: 使用 cmd clipboard (Android 10+)
	err := setClipboardCmd(content)
	if err == nil {
		return nil
	}

	// 方法2: 使用 service call (通用方法)
	err = setClipboardServiceCall(content)
	if err == nil {
		return nil
	}

	// 方法3: 使用 am broadcast (备用方法)
	err = setClipboardAmBroadcast(content)
	if err == nil {
		return nil
	}

	// 方法4: 使用 input text (最后的尝试，但只适用于简单文本)
	err = setClipboardInputText(content)
	if err == nil {
		return nil
	}

	return fmt.Errorf("%w: all methods failed", ErrClipboardAccess)
}

// setClipboardCmd 使用 cmd clipboard 命令（Android 10+）
func setClipboardCmd(content string) error {
	// 使用 stdin 传递内容，避免命令行长度限制
	cmd := exec.Command("su", "-c", "cmd clipboard set-text")
	cmd.Stdin = bytes.NewBufferString(content)
	return cmd.Run()
}

// setClipboardServiceCall 使用 service call 方法
func setClipboardServiceCall(content string) error {
	// 转义特殊字符
	escapedContent := strings.ReplaceAll(content, "'", "\\'")
	escapedContent = strings.ReplaceAll(escapedContent, "\"", "\\\"")

	cmdStr := fmt.Sprintf("service call clipboard 1 i32 1 s16 com.android.shell s16 '%s'", escapedContent)
	cmd := exec.Command("su", "-c", cmdStr)
	return cmd.Run()
}

// setClipboardAmBroadcast 使用 am broadcast 方法
func setClipboardAmBroadcast(content string) error {
	// 使用 am broadcast 发送剪贴板内容
	escapedContent := strings.ReplaceAll(content, "'", "\\'")
	cmdStr := fmt.Sprintf("am broadcast -a clipper.set -e text '%s'", escapedContent)
	cmd := exec.Command("su", "-c", cmdStr)
	return cmd.Run()
}

// setClipboardInputText 使用 input text 方法（仅适用于简单文本）
func setClipboardInputText(content string) error {
	// 这个方法有很多限制，只作为最后的备用方案
	if strings.ContainsAny(content, "\n\r\t") {
		return errors.New("input text method does not support multiline")
	}

	cmdStr := fmt.Sprintf("input text '%s'", content)
	cmd := exec.Command("su", "-c", cmdStr)
	return cmd.Run()
}
