package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/handlers"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/sync"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/webdav"
)

const (
	defaultPort       = "8964"
	defaultConfigPath = "/data/adb/syncclipboard/config.json"
)

func main() {
	// 命令行参数
	port := flag.String("port", defaultPort, "HTTP server port")
	configPath := flag.String("config", defaultConfigPath, "Configuration file path")
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

	// 初始化 WebDAV 客户端和同步管理器
	var webdavClient *webdav.Client
	if cfg.WebDAVURL != "" {
		webdavClient, err = webdav.NewClient(cfg.WebDAVURL, cfg.WebDAVUsername, cfg.WebDAVPassword)
		if err != nil {
			log.Printf("Failed to initialize WebDAV client: %v", err)
		} else {
			log.Println("WebDAV client initialized")
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
	http.HandleFunc("/health", handlers.HealthHandler)
	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetConfigHandler(w, r)
		} else if r.Method == http.MethodPost {
			h.UpdateConfigHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/clipboard", h.GetClipboardHandler)
	http.HandleFunc("/api/sync/now", h.SyncNowHandler)
	http.HandleFunc("/api/sync/status", h.GetSyncStatusHandler)

	// 静态文件服务（WebUI）
	fs := http.FileServer(http.Dir("/data/adb/syncclipboard/webui"))
	http.Handle("/", fs)

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
	log.Printf("Auto-sync: %v", cfg.Enabled)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
