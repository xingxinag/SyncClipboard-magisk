package syncdata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// ClipboardData 表示 SyncClipboard.json 的数据结构
type ClipboardData struct {
	Type     string  `json:"type"`     // "Text", "Image", "File" 等
	Hash     string  `json:"hash"`     // SHA256 哈希值
	Text     string  `json:"text"`     // 文本内容
	HasData  bool    `json:"hasData"`  // 是否有额外数据
	DataName *string `json:"dataName"` // 数据文件名（可为 null）
	Size     int     `json:"size"`     // 内容大小（字节）
}

// NewTextClipboard 创建文本类型的剪贴板数据
func NewTextClipboard(text string) *ClipboardData {
	hash := calculateHash(text)
	return &ClipboardData{
		Type:     "Text",
		Hash:     hash,
		Text:     text,
		HasData:  false,
		DataName: nil,
		Size:     len(text),
	}
}

// ToJSON 将数据转换为 JSON 字符串
func (c *ClipboardData) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从 JSON 字符串解析数据
func FromJSON(jsonStr string) (*ClipboardData, error) {
	var data ClipboardData
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// calculateHash 计算文本的 SHA256 哈希值（大写）
func calculateHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

// GetHash 返回当前数据的哈希值
func (c *ClipboardData) GetHash() string {
	return c.Hash
}

// GetText 返回文本内容
func (c *ClipboardData) GetText() string {
	return c.Text
}

// IsText 判断是否为文本类型
func (c *ClipboardData) IsText() bool {
	return c.Type == "Text"
}
