package monitor

import (
	"log"
	"sync"
	"time"
)

// HybridMonitor 混合监听器（自动降级策略）
type HybridMonitor struct {
	monitors []ClipboardMonitor
	active   ClipboardMonitor
	callback func(string)
	running  bool
	mu       sync.Mutex
}

// NewHybridMonitor 创建混合监听器
func NewHybridMonitor(interval time.Duration) *HybridMonitor {
	return &HybridMonitor{
		monitors: []ClipboardMonitor{
			NewPollingMonitor(interval),
			NewLogcatMonitor(),
			NewInotifyMonitor(),
		},
	}
}

// Start 启动混合监听（自动选择最佳监听器）
func (h *HybridMonitor) Start(callback func(string)) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return nil
	}

	h.callback = callback

	// 依次尝试每个监听器
	for _, monitor := range h.monitors {
		log.Printf("[HybridMonitor] Trying %s...", monitor.Name())
		err := monitor.Start(callback)
		if err == nil {
			h.active = monitor
			h.running = true
			log.Printf("[HybridMonitor] Successfully started with %s", monitor.Name())
			return nil
		}
		log.Printf("[HybridMonitor] %s failed: %v", monitor.Name(), err)
	}

	return nil
}

// Stop 停止混合监听
func (h *HybridMonitor) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	if h.active != nil {
		h.active.Stop()
		log.Printf("[HybridMonitor] Stopped %s", h.active.Name())
	}

	h.running = false
}

// IsRunning 返回是否正在运行
func (h *HybridMonitor) IsRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}

// Name 返回监听器名称
func (h *HybridMonitor) Name() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.active != nil {
		return "HybridMonitor(" + h.active.Name() + ")"
	}
	return "HybridMonitor(inactive)"
}

// GetActiveMonitor 返回当前激活的监听器
func (h *HybridMonitor) GetActiveMonitor() ClipboardMonitor {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.active
}
