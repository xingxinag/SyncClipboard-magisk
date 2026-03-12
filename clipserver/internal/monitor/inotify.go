package monitor

import (
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/clipboard"
)

// InotifyMonitor 文件监听器（实时监听，延迟 < 500ms）
type InotifyMonitor struct {
	watcher  *fsnotify.Watcher
	stopChan chan bool
	running  bool
	callback func(string)
	lastHash string
	mu       sync.Mutex
}

// NewInotifyMonitor 创建文件监听器
func NewInotifyMonitor() *InotifyMonitor {
	return &InotifyMonitor{
		stopChan: make(chan bool, 1),
	}
}

// Start 启动文件监听
func (m *InotifyMonitor) Start(callback func(string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.callback = callback

	// 创建文件监听器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("[InotifyMonitor] Failed to create watcher: %v", err)
		return err
	}
	m.watcher = watcher

	// 尝试监听可能的剪贴板文件路径
	paths := []string{
		"/data/system/users/0/clipboard",
		"/data/clipboard",
		"/data/data/com.android.providers.settings/databases",
	}

	watchedCount := 0
	for _, path := range paths {
		if err := watcher.Add(path); err == nil {
			log.Printf("[InotifyMonitor] Watching: %s", path)
			watchedCount++
		}
	}

	if watchedCount == 0 {
		watcher.Close()
		log.Printf("[InotifyMonitor] No paths available to watch")
		return err
	}

	m.running = true
	log.Printf("[InotifyMonitor] Started (watching %d paths)", watchedCount)

	// 启动事件处理协程
	go m.watchEvents()

	return nil
}

// Stop 停止文件监听
func (m *InotifyMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	m.stopChan <- true

	if m.watcher != nil {
		m.watcher.Close()
	}

	log.Printf("[InotifyMonitor] Stopped")
}

// IsRunning 返回是否正在运行
func (m *InotifyMonitor) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// Name 返回监听器名称
func (m *InotifyMonitor) Name() string {
	return "InotifyMonitor"
}

// watchEvents 监听文件事件
func (m *InotifyMonitor) watchEvents() {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("[InotifyMonitor] File event: %s", event.Name)
				m.onFileChanged()
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[InotifyMonitor] Watcher error: %v", err)
		case <-m.stopChan:
			return
		}
	}
}

// onFileChanged 文件变化时调用
func (m *InotifyMonitor) onFileChanged() {
	content, err := clipboard.GetClipboard()
	if err != nil {
		return
	}

	hash := calculateHash(content)
	if hash != m.lastHash {
		m.lastHash = hash
		log.Printf("[InotifyMonitor] Clipboard changed (hash: %s...)", hash[:8])
		if m.callback != nil {
			m.callback(content)
		}
	}
}
