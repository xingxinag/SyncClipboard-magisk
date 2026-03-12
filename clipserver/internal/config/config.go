package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// WebDAVAccount 代表一个 WebDAV 账号
type WebDAVAccount struct {
	ID       string `json:"id"`       // 账号唯一标识
	Name     string `json:"name"`     // 账号名称（用户自定义）
	URL      string `json:"url"`      // WebDAV 服务器地址
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码
	Created  int64  `json:"created"`  // 创建时间戳
}

// Config 代表应用配置结构
type Config struct {
	// 多账号管理（新版本）
	Accounts        []WebDAVAccount `json:"accounts"`          // 账号列表
	ActiveAccountID string          `json:"active_account_id"` // 当前激活的账号 ID

	// 通用配置
	SyncInterval int  `json:"sync_interval"` // 秒（1-3600）
	Enabled      bool `json:"enabled"`       // 是否启用自动同步

	// 剪贴板策略（自动探测并持久化）
	ClipboardStrategy ClipboardStrategyConfig `json:"clipboard_strategy"`
}

type ClipboardStrategyConfig struct {
	Enabled           bool     `json:"enabled"`
	ReadOrder         []string `json:"read_order"`
	WriteOrder        []string `json:"write_order"`
	DeviceFingerprint string   `json:"device_fingerprint"`
	LastProbeUnix     int64    `json:"last_probe_unix"`
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
		Accounts:        []WebDAVAccount{},
		ActiveAccountID: "",
		SyncInterval:    60,
		Enabled:         false,
		ClipboardStrategy: ClipboardStrategyConfig{
			Enabled: true,
		},
	}
}

// GetActiveAccount 获取当前激活的账号
func (c *Config) GetActiveAccount() *WebDAVAccount {
	if c.ActiveAccountID == "" {
		return nil
	}

	for i := range c.Accounts {
		if c.Accounts[i].ID == c.ActiveAccountID {
			return &c.Accounts[i]
		}
	}

	return nil
}

// AddAccount 添加新账号
func (c *Config) AddAccount(name, url, username, password string) *WebDAVAccount {
	account := WebDAVAccount{
		ID:       generateAccountID(),
		Name:     name,
		URL:      url,
		Username: username,
		Password: password,
		Created:  time.Now().Unix(),
	}

	c.Accounts = append(c.Accounts, account)

	// 如果是第一个账号，自动设为激活
	if len(c.Accounts) == 1 {
		c.ActiveAccountID = account.ID
	}

	return &account
}

// RemoveAccount 删除账号
func (c *Config) RemoveAccount(id string) bool {
	for i, account := range c.Accounts {
		if account.ID == id {
			c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)

			// 如果删除的是激活账号，清空激活状态
			if c.ActiveAccountID == id {
				c.ActiveAccountID = ""
				// 如果还有其他账号，激活第一个
				if len(c.Accounts) > 0 {
					c.ActiveAccountID = c.Accounts[0].ID
				}
			}

			return true
		}
	}
	return false
}

// SetActiveAccount 设置激活账号
func (c *Config) SetActiveAccount(id string) bool {
	for _, account := range c.Accounts {
		if account.ID == id {
			c.ActiveAccountID = id
			return true
		}
	}
	return false
}

// generateAccountID 生成账号 ID
func generateAccountID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond) // 确保每次生成不同
	}
	return string(result)
}
