package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// 测试配置数据
	testConfig := &Config{
		WebDAVURL:      "https://example.com/dav",
		WebDAVUsername: "testuser",
		WebDAVPassword: "testpass",
		SyncInterval:   60,
		Enabled:        true,
	}

	// 写入测试配置
	data, _ := json.MarshalIndent(testConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)

	// 测试加载
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.WebDAVURL != testConfig.WebDAVURL {
		t.Errorf("Expected URL %s, got %s", testConfig.WebDAVURL, config.WebDAVURL)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	config := &Config{
		WebDAVURL:      "https://example.com/dav",
		WebDAVUsername: "user",
		WebDAVPassword: "pass",
		SyncInterval:   30,
		Enabled:        false,
	}

	err := SaveConfig(configPath, config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}
