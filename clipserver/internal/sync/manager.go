package sync

import (
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/clipboard"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/filesaver"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/monitor"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/syncdata"
)

var ErrNotConfigured = webdavErrNotConfigured{}

var ErrRemoteClipboardEmpty = errors.New("remote clipboard content is empty")

type webdavErrNotConfigured struct{}

func (webdavErrNotConfigured) Error() string {
	return "sync client not configured"
}

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
	UploadClipboardText(content string) (*syncdata.ClipboardData, error)
	DownloadClipboardText() (*syncdata.ClipboardData, string, error)
	DownloadFile(filename string) ([]byte, error)
}

type DownloadResult struct {
	Type        string `json:"type"`
	AppliedText bool   `json:"applied_text"`
	SavedFile   bool   `json:"saved_file"`
	Skipped     bool   `json:"skipped"`
	Fallback    bool   `json:"fallback"`
	Message     string `json:"message"`
	Path        string `json:"path,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Size        int    `json:"size,omitempty"`
	Warning     string `json:"warning,omitempty"`
}

type RemoteSnapshot struct {
	Type     string `json:"type"`
	Hash     string `json:"hash"`
	Text     string `json:"text"`
	HasData  bool   `json:"has_data"`
	DataName string `json:"data_name,omitempty"`
	Size     int    `json:"size"`
	Filename string `json:"filename,omitempty"`
	Message  string `json:"message,omitempty"`
}

type DebugState struct {
	Running             bool   `json:"running"`
	ClientConfigured    bool   `json:"client_configured"`
	ConfigEnabled       bool   `json:"config_enabled"`
	AutoUploadEnabled   bool   `json:"auto_upload_enabled"`
	AutoDownloadEnabled bool   `json:"auto_download_enabled"`
	SyncInterval        int    `json:"sync_interval"`
	MonitorConfigured   bool   `json:"monitor_configured"`
	MonitorName         string `json:"monitor_name"`
	TickerConfigured    bool   `json:"ticker_configured"`
	LastHash            string `json:"last_hash"`
	SyncCount           int64  `json:"sync_count"`
	LastSyncUnix        int64  `json:"last_sync_unix"`
}

var (
	clipboardGetFn = clipboard.GetClipboard
	clipboardSetFn = clipboard.SetClipboard
)

// shortHash 安全返回 hash 的前 n 字符，防止空串 slice panic
func shortHash(hash string, n int) string {
	if len(hash) < n {
		return hash
	}
	return hash[:n]
}

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
	log.Printf("[auto-sync-debug] start requested enabled=%v upload=%v download=%v interval=%d client=%v running=%v", m.config.Enabled, m.config.AutoUploadEnabled, m.config.AutoDownloadEnabled, m.config.SyncInterval, m.webdavClient != nil, m.running)
	if m.running {
		log.Println("[auto-sync-debug] start skipped: already running")
		return
	}

	if !m.config.Enabled {
		log.Println("[auto-sync-debug] start skipped: config disabled")
		return
	}

	if !m.config.AutoUploadEnabled && !m.config.AutoDownloadEnabled {
		log.Println("[auto-sync-debug] start skipped: both directions disabled")
		return
	}

	if m.webdavClient == nil {
		log.Println("[auto-sync-debug] start skipped: sync client not configured")
		return
	}

	m.running = true
	var tickerC <-chan time.Time
	if m.config.AutoDownloadEnabled {
		interval := time.Duration(m.config.SyncInterval) * time.Second
		m.ticker = time.NewTicker(interval)
		tickerC = m.ticker.C
	}

	log.Printf("Starting auto-sync (upload=%v, download=%v, interval=%ds)",
		m.config.AutoUploadEnabled, m.config.AutoDownloadEnabled, m.config.SyncInterval)

	// 只有自动上传开启时才需要实时监听剪贴板变化
	if m.config.AutoUploadEnabled {
		interval := time.Duration(m.config.SyncInterval) * time.Second
		m.monitor = monitor.NewHybridMonitor(interval)
		if err := m.monitor.Start(func(content string) {
			log.Printf("[auto-sync-debug] monitor callback content_size=%d", len([]rune(content)))
			m.syncUploadWithContent(content)
		}); err != nil {
			log.Printf("[auto-sync-debug] clipboard monitor start error: %v", err)
		}
		log.Printf("[auto-sync-debug] clipboard monitor state name=%s running=%v", m.monitor.Name(), m.monitor.IsRunning())
	}

	go func() {
		// 启动时先执行一次下载同步
		if m.config.AutoDownloadEnabled {
			log.Println("[auto-sync-debug] initial auto download tick")
			m.syncDownload()
		}

		for {
			select {
			case <-tickerC:
				log.Println("[auto-sync-debug] scheduled auto download tick")
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
		log.Println("[auto-sync-debug] upload skipped: sync client not configured")
		return
	}

	clipData := syncdata.NewTextClipboard(content)
	log.Printf("[auto-sync-debug] upload candidate hash=%s size=%d last_hash=%s", shortHash(clipData.GetHash(), 8), clipData.Size, shortHash(m.lastHash, 8))

	// 如果 hash 没有变化，跳过上传
	if clipData.GetHash() == m.lastHash {
		log.Println("[auto-sync-debug] upload skipped: same hash")
		return
	}

	clipData, err := m.webdavClient.UploadClipboardText(content)
	if err != nil {
		log.Printf("Failed to upload to server: %v", err)
		return
	}

	m.recordSync(clipData.GetHash())
	log.Printf("Uploaded clipboard to server (hash: %s, size: %d bytes)", shortHash(clipData.GetHash(), 8), clipData.Size)
}

// syncDownload 从 WebDAV 下载并更新本地剪贴板
func (m *Manager) syncDownload() {
	// 从 WebDAV 下载
	clipData, content, err := m.webdavClient.DownloadClipboardText()
	if err != nil {
		// 文件可能不存在，这是正常的
		log.Printf("[auto-sync-debug] download failed: %v", err)
		return
	}
	if clipData == nil {
		log.Println("[auto-sync-debug] download skipped: remote clipboard is nil")
		return
	}
	log.Printf("[auto-sync-debug] download candidate type=%s hash=%s size=%d content_size=%d last_hash=%s", clipData.Type, shortHash(clipData.GetHash(), 8), clipData.Size, len([]rune(content)), shortHash(m.lastHash, 8))

	// 只处理文本类型
	if !clipData.IsText() {
		log.Printf("[auto-sync-debug] download skipped: non-text type=%s", clipData.Type)
		return
	}

	// 如果 hash 相同，说明内容没变化
	if clipData.GetHash() == m.lastHash {
		log.Println("[auto-sync-debug] download skipped: same hash")
		return
	}

	if content == "" {
		log.Println("[auto-sync-debug] download skipped: empty remote text")
		return
	}

	// 更新本地剪贴板
	err = clipboardSetFn(content)
	if err != nil {
		log.Printf("[auto-sync-debug] download failed: set clipboard error=%v", err)
		return
	}

	m.recordSync(clipData.GetHash())
	log.Printf("Downloaded clipboard from server (hash: %s, size: %d bytes)", shortHash(clipData.GetHash(), 8), clipData.Size)
}

// UploadNow 手动上传本地剪贴板到 WebDAV
func (m *Manager) UploadNow() error {
	if m.webdavClient == nil {
		return ErrNotConfigured
	}

	content, err := clipboardGetFn()
	if err != nil {
		return err
	}

	localData, err := m.webdavClient.UploadClipboardText(content)
	if err != nil {
		return err
	}

	m.recordSync(localData.GetHash())
	log.Printf("Manual upload pushed to server (hash: %s, size: %d bytes)", shortHash(localData.GetHash(), 8), localData.Size)
	return nil
}

// DownloadNow 手动从 WebDAV 拉取到本地剪贴板
func (m *Manager) DownloadNow() error {
	_, err := m.DownloadRemoteNow()
	return err
}

func (m *Manager) DownloadRemoteNow() (*DownloadResult, error) {
	if m.webdavClient == nil {
		return nil, ErrNotConfigured
	}

	remoteData, err := m.webdavClient.DownloadClipboard()
	if err != nil {
		return nil, err
	}

	if remoteData == nil {
		return &DownloadResult{Skipped: true, Message: "远端剪贴板为空"}, nil
	}

	if remoteData.IsText() {
		content := remoteData.Text
		if remoteData.NeedsDataFile() {
			dataName, err := remoteData.RemoteDataName()
			if err != nil {
				return nil, err
			}
			data, err := m.webdavClient.DownloadFile(dataName)
			if err != nil {
				return nil, err
			}
			content = string(data)
		}

		if content == "" {
			return &DownloadResult{Type: remoteData.Type, Skipped: true, Message: "远端剪贴板为空，已跳过"}, ErrRemoteClipboardEmpty
		}

		if err := clipboardSetFn(content); err != nil {
			if errors.Is(err, clipboard.ErrClipboardAccess) || errors.Is(err, clipboard.ErrClipboardWrite) {
				saved, saveErr := filesaver.SaveBytes("remote_text.txt", []byte(content))
				if saveErr != nil {
					return nil, err
				}
				m.recordSync(remoteData.GetHash())
				log.Printf("Manual download could not write clipboard, saved text fallback to %s (hash: %s, size: %d bytes)", saved.Path, shortHash(remoteData.GetHash(), 8), saved.Size)
				return &DownloadResult{Type: remoteData.Type, SavedFile: true, Fallback: true, Message: "系统剪贴板写入失败，远端文本已保存为文件", Path: saved.Path, Filename: saved.Filename, Size: saved.Size, Warning: err.Error()}, nil
			}
			return nil, err
		}

		m.recordSync(remoteData.GetHash())
		log.Printf("Manual download pulled text from server (hash: %s, size: %d bytes)", shortHash(remoteData.GetHash(), 8), remoteData.Size)
		return &DownloadResult{Type: remoteData.Type, AppliedText: true, Message: "文本已写入本机剪贴板", Size: len(content)}, nil
	}

	if remoteData.IsDownloadableFile() {
		return &DownloadResult{Type: remoteData.Type, Skipped: true, Message: "远端图片/文件需要手动处理，已跳过自动下载", Filename: remoteData.DisplayName(), Size: remoteData.Size}, nil
	}

	return &DownloadResult{Type: remoteData.Type, Skipped: true, Message: "暂不支持的远端剪贴板类型，已跳过"}, nil
}

func (m *Manager) GetRemoteSnapshot() (*RemoteSnapshot, error) {
	if m.webdavClient == nil {
		return nil, ErrNotConfigured
	}
	remoteData, err := m.webdavClient.DownloadClipboard()
	if err != nil {
		return nil, err
	}
	if remoteData == nil {
		return &RemoteSnapshot{Message: "远端剪贴板为空"}, nil
	}
	item := &RemoteSnapshot{
		Type:     remoteData.Type,
		Hash:     remoteData.GetHash(),
		Text:     remoteData.Text,
		HasData:  remoteData.HasData,
		Size:     remoteData.Size,
		Filename: remoteData.DisplayName(),
	}
	if remoteData.DataName != nil {
		item.DataName = *remoteData.DataName
	}
	return item, nil
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

func (m *Manager) GetDebugState() DebugState {
	state := DebugState{
		Running:             m.running,
		ClientConfigured:    m.webdavClient != nil,
		ConfigEnabled:       m.config != nil && m.config.Enabled,
		AutoUploadEnabled:   m.config != nil && m.config.AutoUploadEnabled,
		AutoDownloadEnabled: m.config != nil && m.config.AutoDownloadEnabled,
		MonitorConfigured:   m.monitor != nil,
		TickerConfigured:    m.ticker != nil,
		LastHash:            shortHash(m.lastHash, 8),
		SyncCount:           atomic.LoadInt64(&m.syncCount),
		LastSyncUnix:        atomic.LoadInt64(&m.lastSyncUnix),
	}
	if m.config != nil {
		state.SyncInterval = m.config.SyncInterval
	}
	if m.monitor != nil {
		state.MonitorName = m.monitor.Name()
	}
	return state
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
