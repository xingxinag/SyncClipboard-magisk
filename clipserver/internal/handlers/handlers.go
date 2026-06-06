package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/clipboard"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/opslog"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/sync"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/syncclient"
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
	message := friendlyErrorMessage(err)
	fields["http_status"] = status
	fields["error"] = err.Error()
	fields["result"] = "error"
	fields["code"] = fmt.Sprintf("E_HTTP_%d", status)
	opslog.Error("api", action, message, fields)
	writeJSON(w, status, map[string]interface{}{
		"status":  "error",
		"action":  action,
		"message": message,
		"details": fields,
	})
}

func friendlyErrorMessage(err error) string {
	if errors.Is(err, clipboard.ErrClipboardRead) {
		return "读取系统剪贴板失败：请检查 LSPosed 系统 Hook 是否启用，或尝试复制一段文本后重试"
	}
	if errors.Is(err, clipboard.ErrClipboardWrite) {
		return "写入系统剪贴板失败：请检查 LSPosed 系统 Hook 是否启用；远端文本可改为保存文件兜底"
	}
	if errors.Is(err, clipboard.ErrClipboardAccess) {
		return "访问系统剪贴板失败：请检查 Root/LSPosed Hook 权限"
	}
	return err.Error()
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
	serverConfigured := activeAccount != nil && activeAccount.URL != ""

	var syncRunning bool
	var syncCount int64
	var lastSyncUnix int64
	var syncDebug interface{}
	if h.syncManager != nil {
		syncRunning = h.syncManager.IsRunning()
		syncCount, lastSyncUnix = h.syncManager.GetStats()
		syncDebug = h.syncManager.GetDebugState()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_status":        "running",
		"auto_sync_enabled":     cfg.Enabled,
		"auto_upload_enabled":   cfg.AutoUploadEnabled,
		"auto_download_enabled": cfg.AutoDownloadEnabled,
		"sync_running":          syncRunning,
		"sync_count":            syncCount,
		"last_sync_unix":        lastSyncUnix,
		"account_count":         accountCount,
		"webdav_configured":     serverConfigured,
		"server_configured":     serverConfigured,
		"sync_debug":            syncDebug,
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

	cfg.Enabled = cfg.AutoUploadEnabled || cfg.AutoDownloadEnabled
	opslog.Info("debug", "auto_sync_config_update", "received config update", map[string]interface{}{"enabled": cfg.Enabled, "auto_upload_enabled": cfg.AutoUploadEnabled, "auto_download_enabled": cfg.AutoDownloadEnabled, "sync_interval": cfg.SyncInterval, "account_count": len(cfg.Accounts), "active_account_id": cfg.ActiveAccountID})

	if err := config.SaveConfig(h.configPath, &cfg); err != nil {
		h.writeError(w, "update_config", http.StatusInternalServerError, err, nil)
		return
	}

	var client sync.SyncClient
	activeAccount := cfg.GetActiveAccount()
	if activeAccount != nil {
		var err error
		client, err = syncclient.New(*activeAccount)
		if err != nil {
			opslog.Error("debug", "auto_sync_client_init", "sync client init failed after config update", map[string]interface{}{"error": err.Error(), "server_type": activeAccount.EffectiveType(), "account_name": activeAccount.Name})
			h.writeError(w, "update_config", http.StatusInternalServerError, err, map[string]interface{}{"stage": "init_sync_client", "server_type": activeAccount.EffectiveType()})
			return
		}
		opslog.Info("debug", "auto_sync_client_init", "sync client init ok after config update", map[string]interface{}{"server_type": activeAccount.EffectiveType(), "account_name": activeAccount.Name})
	} else {
		opslog.Warn("debug", "auto_sync_client_init", "no active account after config update", nil)
	}

	// 更新同步管理器
	if h.syncManager != nil {
		h.syncManager.UpdateConfig(&cfg, client)
	}

	h.writeOK(w, "update_config", "配置已保存", map[string]interface{}{
		"sync_interval":         cfg.SyncInterval,
		"enabled":               cfg.Enabled,
		"auto_upload_enabled":   cfg.AutoUploadEnabled,
		"auto_download_enabled": cfg.AutoDownloadEnabled,
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

func (h *Handler) GetClipboardListHandler(w http.ResponseWriter, r *http.Request) {
	localContent, localErr := clipboard.GetClipboard()
	localItem := map[string]interface{}{
		"source":     "local",
		"type":       "Text",
		"text":       localContent,
		"size":       len([]rune(localContent)),
		"available":  localErr == nil,
		"updated_at": time.Now().Unix(),
	}
	if localErr != nil {
		localItem["error"] = friendlyErrorMessage(localErr)
	}

	remoteItem := map[string]interface{}{
		"source":    "remote",
		"available": false,
	}
	if h.syncManager == nil {
		remoteItem["error"] = "同步管理器未初始化"
	} else {
		remoteSnapshot, err := h.syncManager.GetRemoteSnapshot()
		if err != nil {
			remoteItem["error"] = friendlyErrorMessage(err)
		} else if remoteSnapshot != nil {
			remoteItem["available"] = true
			remoteItem["type"] = remoteSnapshot.Type
			remoteItem["hash"] = remoteSnapshot.Hash
			remoteItem["text"] = remoteSnapshot.Text
			remoteItem["has_data"] = remoteSnapshot.HasData
			remoteItem["data_name"] = remoteSnapshot.DataName
			remoteItem["size"] = remoteSnapshot.Size
			remoteItem["filename"] = remoteSnapshot.Filename
			remoteItem["message"] = remoteSnapshot.Message
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"items":  []map[string]interface{}{localItem, remoteItem},
	})
}

// SyncNowHandler 立即触发同步
func (h *Handler) SyncNowHandler(w http.ResponseWriter, r *http.Request) {
	if h.syncManager == nil {
		h.writeError(w, "sync_now", http.StatusInternalServerError, fmt.Errorf("sync manager not initialized"), nil)
		return
	}

	if err := h.syncManager.UploadNow(); err != nil {
		h.writeError(w, "sync_now", http.StatusInternalServerError, err, nil)
		return
	}
	h.writeOK(w, "sync_now", "上传完成", nil)
}

// SyncPullHandler 手动从 WebDAV 拉取
func (h *Handler) SyncPullHandler(w http.ResponseWriter, r *http.Request) {
	if h.syncManager == nil {
		h.writeError(w, "sync_pull", http.StatusInternalServerError, fmt.Errorf("sync manager not initialized"), nil)
		return
	}

	result, err := h.syncManager.DownloadRemoteNow()
	if err != nil {
		if errors.Is(err, sync.ErrRemoteClipboardEmpty) {
			h.writeOK(w, "sync_pull", "远端剪贴板为空，已跳过", map[string]interface{}{"result_data": result, "skipped": true})
			return
		}
		h.writeError(w, "sync_pull", http.StatusInternalServerError, err, nil)
		return
	}
	message := "下载完成"
	if result != nil && result.Message != "" {
		message = result.Message
	}
	h.writeOK(w, "sync_pull", message, map[string]interface{}{"result_data": result})
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
		Type     config.ServerType `json:"type"`
		Name     string            `json:"name"`
		URL      string            `json:"url"`
		Username string            `json:"username"`
		Password string            `json:"password"`
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

	account := cfg.AddAccount(req.Name, req.Type, req.URL, req.Username, req.Password)
	opslog.Info("debug", "auto_sync_account_add", "account added", map[string]interface{}{"enabled": cfg.Enabled, "auto_upload_enabled": cfg.AutoUploadEnabled, "auto_download_enabled": cfg.AutoDownloadEnabled, "active_account_id": cfg.ActiveAccountID, "account_id": account.ID, "account_count": len(cfg.Accounts)})

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
		var client sync.SyncClient
		activeAccount := cfg.GetActiveAccount()
		if activeAccount != nil {
			client, _ = syncclient.New(*activeAccount)
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
		var client sync.SyncClient
		activeAccount := cfg.GetActiveAccount()
		if activeAccount != nil {
			client, _ = syncclient.New(*activeAccount)
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
		Type     config.ServerType `json:"type"`
		URL      string            `json:"url"`
		Username string            `json:"username"`
		Password string            `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "test_account", http.StatusBadRequest, err, nil)
		return
	}

	if req.URL == "" {
		h.writeError(w, "test_account", http.StatusBadRequest, fmt.Errorf("url is required"), nil)
		return
	}

	client, err := syncclient.NewTestable(config.ServerAccount{Type: req.Type, URL: req.URL, Username: req.Username, Password: req.Password})
	if err != nil {
		h.writeError(w, "test_account", http.StatusOK, err, map[string]interface{}{"url": req.URL})
		return
	}

	if err := client.TestConnection(); err != nil {
		h.writeError(w, "test_account", http.StatusOK, err, map[string]interface{}{"url": req.URL})
		return
	}

	h.writeOK(w, "test_account", "连接成功", map[string]interface{}{
		"url": req.URL,
	})
}
