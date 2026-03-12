package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/handlers"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/opslog"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/sync"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/webdav"
)

var reqSeq int64

func nextRequestID() string {
	seq := atomic.AddInt64(&reqSeq, 1)
	return fmt.Sprintf("r-%d-%d", time.Now().UnixNano(), seq)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func withRequestLog(action string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		reqID := nextRequestID()

		r.Header.Set("X-Request-ID", reqID)
		w.Header().Set("X-Request-ID", reqID)

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next(sw, r)

		duration := time.Since(started).Milliseconds()
		level := opslog.Info
		result := "ok"
		if sw.status >= 400 {
			level = opslog.Error
			result = "error"
		}

		level("http", action, fmt.Sprintf("%s %s", r.Method, r.URL.Path), map[string]interface{}{
			"request_id":  reqID,
			"duration_ms": duration,
			"result":      result,
			"code":        strconv.Itoa(sw.status),
			"method":      r.Method,
			"path":        r.URL.Path,
			"remote_addr": r.RemoteAddr,
		})
	}
}

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

const (
	defaultPort        = "8964"
	defaultConfigPath  = "/data/adb/syncclipboard/config.json"
	defaultWebrootPath = "/data/adb/modules/syncclipboard/webroot"
)

func main() {
	// 命令行参数
	port := flag.String("port", defaultPort, "HTTP server port")
	configPath := flag.String("config", defaultConfigPath, "Configuration file path")
	webrootPath := flag.String("webroot", defaultWebrootPath, "WebUI root directory path")
	flag.Parse()

	// 确保配置目录存在
	configDir := filepath.Dir(*configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Printf("Failed to load config, using defaults: %v", err)
		cfg = config.DefaultConfig()
		config.SaveConfig(*configPath, cfg)
	}

	// 创建处理器
	h := handlers.NewHandler(*configPath)
	opslog.SetLogFile(filepath.Join(configDir, "clipserver.log"))
	opslog.Info("server", "startup", "clipserver starting", map[string]interface{}{
		"port":        *port,
		"config_path": *configPath,
		"webroot":     *webrootPath,
	})

	// 初始化 WebDAV 客户端和同步管理器
	var webdavClient *webdav.Client
	activeAccount := cfg.GetActiveAccount()
	if activeAccount != nil {
		webdavClient, err = webdav.NewClient(activeAccount.URL, activeAccount.Username, activeAccount.Password)
		if err != nil {
			log.Printf("Failed to initialize WebDAV client: %v", err)
		} else {
			log.Printf("WebDAV client initialized (account: %s)", activeAccount.Name)
		}
	}

	// 创建同步管理器
	syncManager := sync.NewManager(cfg, webdavClient)
	h.SetSyncManager(syncManager)

	// 如果配置启用，启动自动同步
	if cfg.Enabled && webdavClient != nil {
		syncManager.Start()
	}

	// 注册路由
	http.HandleFunc("/health", withRequestLog("health", withCORS(handlers.HealthHandler)))
	http.HandleFunc("/api/config", withRequestLog("config", withCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetConfigHandler(w, r)
		} else if r.Method == http.MethodPost {
			h.UpdateConfigHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	http.HandleFunc("/api/clipboard", withRequestLog("clipboard_get", withCORS(h.GetClipboardHandler)))
	http.HandleFunc("/api/sync/now", withRequestLog("sync_now", withCORS(h.SyncNowHandler)))
	http.HandleFunc("/api/sync/status", withRequestLog("sync_status", withCORS(h.GetSyncStatusHandler)))
	http.HandleFunc("/api/status", withRequestLog("status", withCORS(h.StatusHandler)))

	// 账号管理 API
	http.HandleFunc("/api/accounts/add", withRequestLog("account_add", withCORS(h.AddAccountHandler)))
	http.HandleFunc("/api/accounts/remove", withRequestLog("account_remove", withCORS(h.RemoveAccountHandler)))
	http.HandleFunc("/api/accounts/set-active", withRequestLog("account_set_active", withCORS(h.SetActiveAccountHandler)))
	http.HandleFunc("/api/accounts/test", withRequestLog("account_test", withCORS(h.TestAccountHandler)))

	// 静态文件服务（WebUI）
	// 检查 webroot 路径是否存在
	if _, err := os.Stat(*webrootPath); os.IsNotExist(err) {
		log.Printf("WARNING: WebUI directory not found: %s", *webrootPath)
		log.Println("WebUI will not be available. Please check the installation.")
	} else {
		log.Printf("WebUI path: %s", *webrootPath)
		fs := http.FileServer(http.Dir(*webrootPath))
		http.Handle("/", fs)
	}

	// 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		syncManager.Stop()
		os.Exit(0)
	}()

	// 启动服务器
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Starting SyncClipboard server on %s", addr)
	log.Printf("WebUI: http://localhost%s", addr)
	log.Printf("Config: %s", *configPath)
	log.Printf("Webroot: %s", *webrootPath)
	log.Printf("Auto-sync: %v", cfg.Enabled)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
