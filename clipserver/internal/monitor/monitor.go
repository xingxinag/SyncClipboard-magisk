package monitor

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ClipboardMonitor 剪贴板监听器接口
type ClipboardMonitor interface {
	// Start 启动监听器
	// callback: 当剪贴板变化时调用，传入新内容
	Start(callback func(content string)) error

	// Stop 停止监听器
	Stop()

	// IsRunning 返回监听器是否正在运行
	IsRunning() bool

	// Name 返回监听器名称（用于日志）
	Name() string
}

// calculateHash 计算内容的 SHA256 哈希值
func calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}
