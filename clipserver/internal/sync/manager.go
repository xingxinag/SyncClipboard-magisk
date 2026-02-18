package sync

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/clipboard"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/monitor"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/syncdata"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/webdav"
)

// Manager 管理自动同步
type Manager struct {
	config       *config.Config
	webdavClient SyncClient
	monitor      monitor.ClipboardMonitor
	ticker       *time.Ticker
	stopChan     chan bool
	lastHash     string // 使用 hash 而不是内容来判断是否需要同步
	running      bool
	syncCount    int64
	lastSyncUnix int64
}

type SyncClient interface {
	UploadClipboard(data *syncdata.ClipboardData) error
	DownloadClipboard() (*syncdata.ClipboardData, error)
}

var (
	clipboardGetFn = clipboard.GetClipboard
	clipboardSetFn = clipboard.SetClipboard
)

// NewManager 创建同步管理器
func NewManager(cfg *config.Config, client SyncClient) *Manager {
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

	// 创建混合监听器（自动降级，确保 100% 可靠）
	m.monitor = monitor.NewHybridMonitor()

	log.Printf("Starting auto-sync with interval: %d seconds", m.config.SyncInterval)
	log.Println("Starting clipboard monitor for real-time sync")

	// 启动监听器，传入回调函数
	if err := m.monitor.Start(func(content string) {
		// 剪贴板变化时立即触发上传
		m.syncUploadWithContent(content)
	}); err != nil {
		log.Printf("Failed to start clipboard monitor: %v", err)
		// 监听器启动失败不影响定时同步
	}

	go func() {
		// 启动时先执行一次下载同步
		m.syncDownload()

		for {
			select {
			case <-m.ticker.C:
				// 定时下载检查（上传由 monitor 触发）
				m.syncDownload()
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

	// 停止监听器
	if m.monitor != nil {
		m.monitor.Stop()
		log.Println("Clipboard monitor stopped")
	}

	if m.ticker != nil {
		m.ticker.Stop()
	}
	m.stopChan <- true
}

// syncUpload 上传本地剪贴板到 WebDAV
func (m *Manager) syncUpload() {
	// 获取当前剪贴板内容
	content, err := clipboardGetFn()
	if err != nil {
		log.Printf("Failed to get clipboard: %v", err)
		return
	}

	m.syncUploadWithContent(content)
}

// syncUploadWithContent 使用指定内容上传到 WebDAV
func (m *Manager) syncUploadWithContent(content string) {
	if m.webdavClient == nil {
		return
	}

	// 创建 SyncClipboard 数据结构
	clipData := syncdata.NewTextClipboard(content)

	// 如果 hash 没有变化，跳过上传
	if clipData.GetHash() == m.lastHash {
		return
	}

	// 上传到 WebDAV
	err := m.webdavClient.UploadClipboard(clipData)
	if err != nil {
		log.Printf("Failed to upload to WebDAV: %v", err)
		return
	}

	m.recordSync(clipData.GetHash())
	log.Printf("Uploaded clipboard to WebDAV (hash: %s, size: %d bytes)", clipData.GetHash()[:8], clipData.Size)
}

// syncDownload 从 WebDAV 下载并更新本地剪贴板
func (m *Manager) syncDownload() {
	// 从 WebDAV 下载
	clipData, err := m.webdavClient.DownloadClipboard()
	if err != nil {
		// 文件可能不存在，这是正常的
		log.Printf("Failed to download from WebDAV: %v", err)
		return
	}

	// 只处理文本类型
	if !clipData.IsText() {
		log.Printf("Skipping non-text clipboard type: %s", clipData.Type)
		return
	}

	// 如果 hash 相同，说明内容没变化
	if clipData.GetHash() == m.lastHash {
		return
	}

	// 更新本地剪贴板
	err = clipboardSetFn(clipData.GetText())
	if err != nil {
		log.Printf("Failed to set clipboard: %v", err)
		return
	}

	m.recordSync(clipData.GetHash())
	log.Printf("Downloaded clipboard from WebDAV (hash: %s, size: %d bytes)", clipData.GetHash()[:8], clipData.Size)
}

// UploadNow 手动上传本地剪贴板到 WebDAV
func (m *Manager) UploadNow() error {
	if m.webdavClient == nil {
		return webdav.ErrNotConfigured
	}

	content, err := clipboardGetFn()
	if err != nil {
		return err
	}

	localData := syncdata.NewTextClipboard(content)
	err = m.webdavClient.UploadClipboard(localData)
	if err != nil {
		return err
	}

	m.recordSync(localData.GetHash())
	log.Printf("Manual upload pushed to WebDAV (hash: %s, size: %d bytes)", localData.GetHash()[:8], localData.Size)
	return nil
}

// DownloadNow 手动从 WebDAV 拉取到本地剪贴板
func (m *Manager) DownloadNow() error {
	if m.webdavClient == nil {
		return webdav.ErrNotConfigured
	}

	remoteData, err := m.webdavClient.DownloadClipboard()
	if err != nil {
		return err
	}

	if remoteData == nil || !remoteData.IsText() {
		return nil
	}

	if err := clipboardSetFn(remoteData.GetText()); err != nil {
		return err
	}

	m.recordSync(remoteData.GetHash())
	log.Printf("Manual download pulled from WebDAV (hash: %s, size: %d bytes)", remoteData.GetHash()[:8], remoteData.Size)
	return nil
}

// SyncNow 兼容旧接口，默认执行手动上传
func (m *Manager) SyncNow() error {
	return m.UploadNow()
}

func (m *Manager) recordSync(hash string) {
	m.lastHash = hash
	atomic.AddInt64(&m.syncCount, 1)
	atomic.StoreInt64(&m.lastSyncUnix, time.Now().Unix())
}

// IsRunning 返回同步状态
func (m *Manager) IsRunning() bool {
	return m.running
}

// GetStats 返回同步统计信息
func (m *Manager) GetStats() (syncCount int64, lastSyncUnix int64) {
	return atomic.LoadInt64(&m.syncCount), atomic.LoadInt64(&m.lastSyncUnix)
}

// UpdateConfig 更新配置并重启同步
func (m *Manager) UpdateConfig(cfg *config.Config, client SyncClient) {
	if m.running {
		m.Stop()
	}

	m.config = cfg
	m.webdavClient = client

	if cfg.Enabled && client != nil {
		m.Start()
	}
}
