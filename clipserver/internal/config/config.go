package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 代表应用配置结构
type Config struct {
	WebDAVURL      string `json:"webdav_url"`
	WebDAVUsername string `json:"webdav_username"`
	WebDAVPassword string `json:"webdav_password"`
	SyncInterval   int    `json:"sync_interval"` // 秒
	Enabled        bool   `json:"enabled"`
}

// LoadConfig 从指定路径加载配置文件
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig 保存配置到指定路径
func SaveConfig(path string, config *Config) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		WebDAVURL:      "",
		WebDAVUsername: "",
		WebDAVPassword: "",
		SyncInterval:   60,
		Enabled:        false,
	}
}
