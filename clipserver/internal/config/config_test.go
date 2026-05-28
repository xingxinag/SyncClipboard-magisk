package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	config := &Config{
		Accounts: []ServerAccount{{
			ID:       "acc1",
			Name:     "test",
			Type:     ServerTypeWebDAV,
			URL:      "https://example.com/dav",
			Username: "user",
			Password: "pass",
			Created:  1700000000,
		}},
		ActiveAccountID: "acc1",
		SyncInterval:    30,
		Enabled:         false,
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
