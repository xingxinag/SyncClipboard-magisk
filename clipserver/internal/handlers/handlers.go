package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/clipboard"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/sync"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/webdav"
)

// Handler 封装所有HTTP处理器
type Handler struct {
	configPath  string
	syncManager *sync.Manager
}

// NewHandler 创建新的处理器实例
func NewHandler(configPath string) *Handler {
	return &Handler{
		configPath: configPath,
	}
}

// SetSyncManager 设置同步管理器
func (h *Handler) SetSyncManager(sm *sync.Manager) {
	h.syncManager = sm
}

// HealthHandler 健康检查端点
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetConfigHandler 获取当前配置
func (h *Handler) GetConfigHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		// 返回默认配置
		cfg = config.DefaultConfig()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// UpdateConfigHandler 更新配置
func (h *Handler) UpdateConfigHandler(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := config.SaveConfig(h.configPath, &cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 重新初始化WebDAV客户端和同步管理器
	var client *webdav.Client
	if cfg.WebDAVURL != "" {
		var err error
		client, err = webdav.NewClient(cfg.WebDAVURL, cfg.WebDAVUsername, cfg.WebDAVPassword)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 更新同步管理器
	if h.syncManager != nil {
		h.syncManager.UpdateConfig(&cfg, client)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetClipboardHandler 获取当前剪贴板内容
func (h *Handler) GetClipboardHandler(w http.ResponseWriter, r *http.Request) {
	content, err := clipboard.GetClipboard()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"content": content})
}

// SyncNowHandler 立即触发同步
func (h *Handler) SyncNowHandler(w http.ResponseWriter, r *http.Request) {
	if h.syncManager == nil {
		http.Error(w, "Sync manager not initialized", http.StatusInternalServerError)
		return
	}

	if err := h.syncManager.SyncNow(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
}

// GetSyncStatusHandler 获取同步状态
func (h *Handler) GetSyncStatusHandler(w http.ResponseWriter, r *http.Request) {
	if h.syncManager == nil {
		http.Error(w, "Sync manager not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"running": h.syncManager.IsRunning(),
	})
}
