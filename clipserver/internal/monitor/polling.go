package monitor

import (
	"log"
	"sync"
	"time"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/clipboard"
)

// PollingMonitor 轮询监听器（兜底方案，100% 可靠）
type PollingMonitor struct {
	ticker   *time.Ticker
	stopChan chan bool
	running  bool
	callback func(string)
	lastHash string
	mu       sync.Mutex
	interval time.Duration
}

// NewPollingMonitor 创建轮询监听器
// interval: 轮询间隔，建议 1-5 秒
func NewPollingMonitor(interval time.Duration) *PollingMonitor {
	if interval < time.Second {
		interval = time.Second
	}
	return &PollingMonitor{
		stopChan: make(chan bool, 1),
		interval: interval,
	}
}

// Start 启动轮询监听
func (m *PollingMonitor) Start(callback func(string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.callback = callback
	m.ticker = time.NewTicker(m.interval)
	m.running = true

	log.Printf("[PollingMonitor] Started with interval: %v", m.interval)

	go m.poll()

	return nil
}

// Stop 停止轮询监听
func (m *PollingMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	if m.ticker != nil {
		m.ticker.Stop()
	}
	m.stopChan <- true

	log.Printf("[PollingMonitor] Stopped")
}

// IsRunning 返回是否正在运行
func (m *PollingMonitor) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// Name 返回监听器名称
func (m *PollingMonitor) Name() string {
	return "PollingMonitor"
}

// poll 轮询剪贴板内容
func (m *PollingMonitor) poll() {
	for {
		select {
		case <-m.ticker.C:
			m.checkClipboard()
		case <-m.stopChan:
			return
		}
	}
}

// checkClipboard 检查剪贴板是否变化
func (m *PollingMonitor) checkClipboard() {
	content, err := clipboard.GetClipboard()
	if err != nil {
		// 读取失败，跳过本次检查
		return
	}

	hash := calculateHash(content)
	if hash != m.lastHash {
		m.lastHash = hash
		log.Printf("[PollingMonitor] Clipboard changed (hash: %s...)", hash[:8])
		if m.callback != nil {
			m.callback(content)
		}
	}
}
