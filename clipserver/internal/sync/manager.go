package sync

import (
	"log"
	"time"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/clipboard"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/webdav"
)

// Manager 管理自动同步
type Manager struct {
	config       *config.Config
	webdavClient *webdav.Client
	ticker       *time.Ticker
	stopChan     chan bool
	lastContent  string
	running      bool
}

// NewManager 创建同步管理器
func NewManager(cfg *config.Config, client *webdav.Client) *Manager {
	return &Manager{
		config:       cfg,
		webdavClient: client,
		stopChan:     make(chan bool),
		running:      false,
	}
}

// Start 启动自动同步
func (m *Manager) Start() {
	if m.running {
		log.Println("Sync manager already running")
		return
	}

	if !m.config.Enabled {
		log.Println("Sync is disabled in config")
		return
	}

	if m.webdavClient == nil {
		log.Println("WebDAV client not configured")
		return
	}

	m.running = true
	interval := time.Duration(m.config.SyncInterval) * time.Second
	m.ticker = time.NewTicker(interval)

	log.Printf("Starting auto-sync with interval: %d seconds", m.config.SyncInterval)

	go func() {
		for {
			select {
			case <-m.ticker.C:
				m.syncOnce()
			case <-m.stopChan:
				log.Println("Stopping auto-sync")
				return
			}
		}
	}()
}

// Stop 停止自动同步
func (m *Manager) Stop() {
	if !m.running {
		return
	}

	m.running = false
	if m.ticker != nil {
		m.ticker.Stop()
	}
	m.stopChan <- true
}

// syncOnce 执行一次同步
func (m *Manager) syncOnce() {
	// 获取当前剪贴板内容
	content, err := clipboard.GetClipboard()
	if err != nil {
		log.Printf("Failed to get clipboard: %v", err)
		return
	}

	// 如果内容没有变化，跳过
	if content == m.lastContent {
		return
	}

	// 上传到 WebDAV
	err = m.webdavClient.UploadContent("clipboard.txt", content)
	if err != nil {
		log.Printf("Failed to upload to WebDAV: %v", err)
		return
	}

	m.lastContent = content
	log.Printf("Synced clipboard content (%d bytes)", len(content))
}

// SyncNow 立即执行同步
func (m *Manager) SyncNow() error {
	if m.webdavClient == nil {
		return webdav.ErrNotConfigured
	}

	content, err := clipboard.GetClipboard()
	if err != nil {
		return err
	}

	err = m.webdavClient.UploadContent("clipboard.txt", content)
	if err != nil {
		return err
	}

	m.lastContent = content
	log.Printf("Manual sync completed (%d bytes)", len(content))
	return nil
}

// IsRunning 返回同步状态
func (m *Manager) IsRunning() bool {
	return m.running
}

// UpdateConfig 更新配置并重启同步
func (m *Manager) UpdateConfig(cfg *config.Config, client *webdav.Client) {
	wasRunning := m.running

	if m.running {
		m.Stop()
	}

	m.config = cfg
	m.webdavClient = client

	if wasRunning && cfg.Enabled {
		m.Start()
	}
}
