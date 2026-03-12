package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/clipboard"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/opslog"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/sync"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/webdav"
)

// Handler 封装所有HTTP处理器
type Handler struct {
	configPath  string
	syncManager *sync.Manager
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *Handler) writeError(w http.ResponseWriter, action string, status int, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["http_status"] = status
	fields["error"] = err.Error()
	fields["result"] = "error"
	fields["code"] = fmt.Sprintf("E_HTTP_%d", status)
	opslog.Error("api", action, err.Error(), fields)
	writeJSON(w, status, map[string]interface{}{
		"status":  "error",
		"action":  action,
		"message": err.Error(),
		"details": fields,
	})
}

func (h *Handler) writeOK(w http.ResponseWriter, action, message string, payload map[string]interface{}) {
	if payload == nil {
		payload = map[string]interface{}{}
	}
	payload["status"] = "ok"
	payload["action"] = action
	payload["message"] = message
	payload["result"] = "ok"
	payload["code"] = "OK"
	opslog.Info("api", action, message, payload)
	writeJSON(w, http.StatusOK, payload)
}

// StatusHandler 获取服务实时状态
func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	activeAccount := cfg.GetActiveAccount()
	accountCount := len(cfg.Accounts)
	webdavConfigured := activeAccount != nil && activeAccount.URL != ""

	var syncRunning bool
	var syncCount int64
	var lastSyncUnix int64
	if h.syncManager != nil {
		syncRunning = h.syncManager.IsRunning()
		syncCount, lastSyncUnix = h.syncManager.GetStats()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_status":    "running",
		"auto_sync_enabled": cfg.Enabled,
		"sync_running":      syncRunning,
		"sync_count":        syncCount,
		"last_sync_unix":    lastSyncUnix,
		"account_count":     accountCount,
		"webdav_configured": webdavConfigured,
		"active_account_name": func() string {
			if activeAccount != nil {
				return activeAccount.Name
			}
			return ""
		}(),
		"server_time_unix": time.Now().Unix(),
	})
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
		opslog.Warn("api", "get_config", "load config failed, fallback to defaults", map[string]interface{}{"error": err.Error()})
	}

	writeJSON(w, http.StatusOK, cfg)
}

// UpdateConfigHandler 更新配置
func (h *Handler) UpdateConfigHandler(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		h.writeError(w, "update_config", http.StatusBadRequest, err, nil)
		return
	}

	// 验证 SyncInterval 范围（1-3600 秒）
	if cfg.SyncInterval < 1 {
		cfg.SyncInterval = 1
	} else if cfg.SyncInterval > 3600 {
		cfg.SyncInterval = 3600
	}

	if err := config.SaveConfig(h.configPath, &cfg); err != nil {
		h.writeError(w, "update_config", http.StatusInternalServerError, err, nil)
		return
	}

	// 重新初始化 WebDAV 客户端和同步管理器
	var client *webdav.Client
	activeAccount := cfg.GetActiveAccount()
	if activeAccount != nil {
		var err error
		client, err = webdav.NewClient(activeAccount.URL, activeAccount.Username, activeAccount.Password)
		if err != nil {
			h.writeError(w, "update_config", http.StatusInternalServerError, err, map[string]interface{}{"stage": "init_webdav"})
			return
		}
	}

	// 更新同步管理器
	if h.syncManager != nil {
		h.syncManager.UpdateConfig(&cfg, client)
	}

	h.writeOK(w, "update_config", "配置已保存", map[string]interface{}{
		"sync_interval": cfg.SyncInterval,
		"enabled":       cfg.Enabled,
	})
}

// GetClipboardHandler 获取当前剪贴板内容
func (h *Handler) GetClipboardHandler(w http.ResponseWriter, r *http.Request) {
	content, err := clipboard.GetClipboard()
	if err != nil {
		h.writeError(w, "get_clipboard", http.StatusInternalServerError, err, nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}

// SyncNowHandler 立即触发同步
func (h *Handler) SyncNowHandler(w http.ResponseWriter, r *http.Request) {
	if h.syncManager == nil {
		h.writeError(w, "sync_now", http.StatusInternalServerError, fmt.Errorf("sync manager not initialized"), nil)
		return
	}

	if err := h.syncManager.SyncNow(); err != nil {
		h.writeError(w, "sync_now", http.StatusInternalServerError, err, nil)
		return
	}
	h.writeOK(w, "sync_now", "同步完成", nil)
}

// GetSyncStatusHandler 获取同步状态
func (h *Handler) GetSyncStatusHandler(w http.ResponseWriter, r *http.Request) {
	if h.syncManager == nil {
		h.writeError(w, "sync_status", http.StatusInternalServerError, fmt.Errorf("sync manager not initialized"), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"running": h.syncManager.IsRunning(),
	})
}

// AddAccountHandler 添加新账号
func (h *Handler) AddAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "add_account", http.StatusBadRequest, err, nil)
		return
	}

	// 验证必填字段
	if req.Name == "" || req.URL == "" {
		h.writeError(w, "add_account", http.StatusBadRequest, fmt.Errorf("name and url are required"), nil)
		return
	}

	// 加载配置
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// 添加账号
	account := cfg.AddAccount(req.Name, req.URL, req.Username, req.Password)

	// 保存配置
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		h.writeError(w, "add_account", http.StatusInternalServerError, err, nil)
		return
	}

	h.writeOK(w, "add_account", "账号添加成功", map[string]interface{}{
		"account": account,
	})
}

// RemoveAccountHandler 删除账号
func (h *Handler) RemoveAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "remove_account", http.StatusBadRequest, err, nil)
		return
	}

	if req.ID == "" {
		h.writeError(w, "remove_account", http.StatusBadRequest, fmt.Errorf("id is required"), nil)
		return
	}

	// 加载配置
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		h.writeError(w, "remove_account", http.StatusInternalServerError, fmt.Errorf("failed to load config"), nil)
		return
	}

	// 删除账号
	if !cfg.RemoveAccount(req.ID) {
		h.writeError(w, "remove_account", http.StatusNotFound, fmt.Errorf("account not found"), map[string]interface{}{"account_id": req.ID})
		return
	}

	// 保存配置
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		h.writeError(w, "remove_account", http.StatusInternalServerError, err, nil)
		return
	}

	// 如果删除的是激活账号，需要更新同步管理器
	if h.syncManager != nil {
		var client *webdav.Client
		activeAccount := cfg.GetActiveAccount()
		if activeAccount != nil {
			client, _ = webdav.NewClient(activeAccount.URL, activeAccount.Username, activeAccount.Password)
		}
		h.syncManager.UpdateConfig(cfg, client)
	}

	h.writeOK(w, "remove_account", "账号删除成功", map[string]interface{}{
		"account_id": req.ID,
	})
}

// SetActiveAccountHandler 设置激活账号
func (h *Handler) SetActiveAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "set_active_account", http.StatusBadRequest, err, nil)
		return
	}

	if req.ID == "" {
		h.writeError(w, "set_active_account", http.StatusBadRequest, fmt.Errorf("id is required"), nil)
		return
	}

	// 加载配置
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		h.writeError(w, "set_active_account", http.StatusInternalServerError, fmt.Errorf("failed to load config"), nil)
		return
	}

	// 设置激活账号
	if !cfg.SetActiveAccount(req.ID) {
		h.writeError(w, "set_active_account", http.StatusNotFound, fmt.Errorf("account not found"), map[string]interface{}{"account_id": req.ID})
		return
	}

	// 保存配置
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		h.writeError(w, "set_active_account", http.StatusInternalServerError, err, nil)
		return
	}

	// 更新同步管理器
	if h.syncManager != nil {
		var client *webdav.Client
		activeAccount := cfg.GetActiveAccount()
		if activeAccount != nil {
			client, _ = webdav.NewClient(activeAccount.URL, activeAccount.Username, activeAccount.Password)
		}
		h.syncManager.UpdateConfig(cfg, client)
	}

	h.writeOK(w, "set_active_account", "账号切换成功", map[string]interface{}{
		"account_id": req.ID,
	})
}

// TestAccountHandler 测试账号连接
func (h *Handler) TestAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "test_account", http.StatusBadRequest, err, nil)
		return
	}

	if req.URL == "" {
		h.writeError(w, "test_account", http.StatusBadRequest, fmt.Errorf("url is required"), nil)
		return
	}

	// 尝试连接
	client, err := webdav.NewClient(req.URL, req.Username, req.Password)
	if err != nil {
		h.writeError(w, "test_account", http.StatusOK, err, map[string]interface{}{"url": req.URL})
		return
	}

	// 尝试列出目录（测试连接）
	if err := client.TestConnection(); err != nil {
		h.writeError(w, "test_account", http.StatusOK, err, map[string]interface{}{"url": req.URL})
		return
	}

	h.writeOK(w, "test_account", "连接成功", map[string]interface{}{
		"url": req.URL,
	})
}
