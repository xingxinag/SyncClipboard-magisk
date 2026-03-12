package monitor

import (
	"bufio"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/clipboard"
)

// LogcatMonitor logcat 日志监听器（实时监听，延迟 < 100ms）
type LogcatMonitor struct {
	cmd      *exec.Cmd
	stopChan chan bool
	running  bool
	callback func(string)
	lastHash string
	mu       sync.Mutex
}

// NewLogcatMonitor 创建 logcat 监听器
func NewLogcatMonitor() *LogcatMonitor {
	return &LogcatMonitor{
		stopChan: make(chan bool, 1),
	}
}

// Start 启动 logcat 监听
func (m *LogcatMonitor) Start(callback func(string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.callback = callback

	// 启动 logcat 监听 ClipboardService
	m.cmd = exec.Command("su", "-c", "logcat", "-v", "time", "ClipboardService:*", "*:S")

	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		log.Printf("[LogcatMonitor] Failed to create stdout pipe: %v", err)
		return err
	}

	if err := m.cmd.Start(); err != nil {
		log.Printf("[LogcatMonitor] Failed to start logcat: %v", err)
		return err
	}

	m.running = true
	log.Printf("[LogcatMonitor] Started")

	// 启动日志读取协程
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-m.stopChan:
				return
			default:
				line := scanner.Text()
				m.processLogLine(line)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("[LogcatMonitor] Scanner error: %v", err)
		}
	}()

	return nil
}

// Stop 停止 logcat 监听
func (m *LogcatMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	m.stopChan <- true

	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
	}

	log.Printf("[LogcatMonitor] Stopped")
}

// IsRunning 返回是否正在运行
func (m *LogcatMonitor) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// Name 返回监听器名称
func (m *LogcatMonitor) Name() string {
	return "LogcatMonitor"
}

// processLogLine 处理日志行
func (m *LogcatMonitor) processLogLine(line string) {
	// 检测剪贴板变化的关键字
	keywords := []string{
		"setPrimaryClip",
		"Setting clip",
		"clipboard changed",
	}

	lineLower := strings.ToLower(line)
	for _, keyword := range keywords {
		if strings.Contains(lineLower, strings.ToLower(keyword)) {
			log.Printf("[LogcatMonitor] Detected clipboard change: %s", line)
			m.onClipboardChanged()
			return
		}
	}
}

// onClipboardChanged 剪贴板变化时调用
func (m *LogcatMonitor) onClipboardChanged() {
	content, err := clipboard.GetClipboard()
	if err != nil {
		log.Printf("[LogcatMonitor] Failed to get clipboard: %v", err)
		return
	}

	hash := calculateHash(content)
	if hash != m.lastHash {
		m.lastHash = hash
		log.Printf("[LogcatMonitor] Clipboard changed (hash: %s...)", hash[:8])
		if m.callback != nil {
			m.callback(content)
		}
	}
}
