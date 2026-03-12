package clipboard

import (
	"bytes"
	"errors"
	"fmt"
	"log"
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

func isInvalidClipboardOutput(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return true
	}

	invalidMarks := []string{
		"no shell command implementation",
		"unknown command",
		"not implemented",
		"permission denied",
		"clipboard service not found",
		"cmd: can't find service",
		"error while accessing provider",
		"could not find provider",
		"java.lang.illegalstateexception",
		"null",
	}

	for _, mark := range invalidMarks {
		if strings.Contains(lower, mark) {
			return true
		}
	}

	return false
}

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
// 使用 7 种方法尝试，确保在各种深度定制系统上都能工作
func GetClipboard() (string, error) {
	strat := detectClipboardStrategy()
	triedCmd := false
	skippedCmd := false

	for _, method := range strat.readOrder {
		if method.name == "cmd_clipboard" {
			triedCmd = true
		}
	}
	for _, method := range strat.readOrder {
		log.Printf("[clipboard/get] start method=%s", method.name)
		content, err := method.fn()
		if err == nil && content != "" {
			log.Printf("[clipboard/get] ok method=%s size=%d", method.name, len(content))
			return content, nil
		}
		if err != nil {
			log.Printf("[clipboard/get] fail method=%s err=%v", method.name, err)
			if method.name == "cmd_clipboard" && strings.Contains(strings.ToLower(err.Error()), "invalid output") {
				skippedCmd = true
			}
		}
	}

	if triedCmd && skippedCmd {
		log.Printf("[clipboard/get] strategy_hint: cmd_clipboard unstable on this ROM, prioritize service_call/dumpsys")
	}

	log.Printf("[clipboard/get] failed all methods")
	return "", fmt.Errorf("%w: all 7 methods failed", ErrClipboardAccess)
}

// getClipboardCmd 使用 cmd clipboard 命令（Android 10+）
func getClipboardCmd() (string, error) {
	cmd := exec.Command("su", "-c", "cmd clipboard get-text")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(output))
	if isInvalidClipboardOutput(content) || strings.Contains(content, "Error") || strings.Contains(content, "Exception") {
		return "", errors.New("cmd clipboard returned invalid output")
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
	if isInvalidClipboardOutput(content) {
		return "", errors.New("service call returned invalid output")
	}
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
	if isInvalidClipboardOutput(content) {
		return "", errors.New("content provider returned invalid output")
	}
	return content, nil
}

// parseServiceCallOutput 解析 service call 的输出
func parseServiceCallOutput(output string) string {
	// fffffffd 通常表示 Java 异常（例如 No shell command implementation）
	if strings.Contains(strings.ToLower(output), "fffffffd") {
		return ""
	}

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
	content := strings.TrimSpace(output)
	if looksLikeDottedGarbage(content) {
		return ""
	}
	return content
}

func looksLikeDottedGarbage(s string) bool {
	if s == "" {
		return true
	}
	runes := []rune(s)
	if len(runes) < 6 {
		return false
	}
	dotCount := 0
	alphaCount := 0
	for _, r := range runes {
		if r == '.' {
			dotCount++
		}
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			alphaCount++
		}
	}

	// 类似 "........N.o. .i." 这种噪声：点号明显过多
	if dotCount >= len(runes)/3 && alphaCount <= len(runes)/2 {
		return true
	}
	return false
}

// SetClipboard 设置系统剪贴板内容（需要Root权限）
// 使用 6 种方法尝试，确保在各种深度定制系统上都能工作
func SetClipboard(content string) error {
	if err := ValidateContent(content); err != nil {
		return err
	}

	strat := detectClipboardStrategy()
	for _, method := range strat.writeOrder {
		log.Printf("[clipboard/set] start method=%s size=%d", method.name, len(content))
		err := method.fn(content)
		if err == nil {
			log.Printf("[clipboard/set] ok method=%s", method.name)
			return nil
		}
		log.Printf("[clipboard/set] fail method=%s err=%v", method.name, err)
	}

	log.Printf("[clipboard/set] failed all methods")
	return fmt.Errorf("%w: all 6 methods failed", ErrClipboardAccess)
}

// setClipboardCmd 使用 cmd clipboard 命令（Android 10+）
func setClipboardCmd(content string) error {
	// 使用 stdin 传递内容，避免命令行长度限制
	cmd := exec.Command("su", "-c", "cmd clipboard set-text")
	cmd.Stdin = bytes.NewBufferString(content)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	if isInvalidClipboardOutput(string(output)) {
		return errors.New("cmd clipboard set-text returned invalid output")
	}
	return nil
}

// setClipboardServiceCall 使用 service call 方法
func setClipboardServiceCall(content string) error {
	// 转义特殊字符
	escapedContent := strings.ReplaceAll(content, "'", "\\'")
	escapedContent = strings.ReplaceAll(escapedContent, "\"", "\\\"")

	cmdStr := fmt.Sprintf("service call clipboard 1 i32 1 s16 com.android.shell s16 '%s'", escapedContent)
	cmd := exec.Command("su", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	if isInvalidClipboardOutput(string(output)) {
		return errors.New("service call set clipboard returned invalid output")
	}
	return nil
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
