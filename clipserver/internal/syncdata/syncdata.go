package syncdata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"
)

const TextTransferDataThreshold = 10240

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
	textSize := utf8.RuneCountInString(text)
	if textSize > TextTransferDataThreshold {
		dataName := fmt.Sprintf("text_%s.txt", hash)
		preview := string([]rune(text)[:TextTransferDataThreshold])
		return &ClipboardData{
			Type:     "Text",
			Hash:     hash,
			Text:     preview,
			HasData:  true,
			DataName: &dataName,
			Size:     textSize,
		}
	}

	return &ClipboardData{
		Type:     "Text",
		Hash:     hash,
		Text:     text,
		HasData:  false,
		DataName: nil,
		Size:     textSize,
	}
}

func (c *ClipboardData) NeedsDataFile() bool {
	return c.IsText() && c.HasData && c.DataName != nil && *c.DataName != ""
}

func (c *ClipboardData) HasRemoteDataFile() bool {
	return c.HasData && c.DataName != nil && strings.TrimSpace(*c.DataName) != ""
}

func (c *ClipboardData) RemoteDataName() (string, error) {
	if !c.HasRemoteDataFile() {
		return "", fmt.Errorf("remote clipboard missing dataName")
	}
	name := strings.TrimSpace(*c.DataName)
	cleanName := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	if cleanName == "." || cleanName == "" || strings.HasPrefix(cleanName, "../") || cleanName == ".." || strings.HasPrefix(cleanName, "/") {
		return "", fmt.Errorf("invalid remote dataName: %s", name)
	}
	return cleanName, nil
}

func (c *ClipboardData) DisplayName() string {
	if c.Text != "" {
		return path.Base(strings.ReplaceAll(c.Text, "\\", "/"))
	}
	if c.DataName != nil && *c.DataName != "" {
		return path.Base(strings.ReplaceAll(*c.DataName, "\\", "/"))
	}
	return "SyncClipboard.data"
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

// GetHash 返回当前数据的哈希值。
// 若 Hash 字段为空（如旧版 WebDAV 服务端未写入），则按 Text 内容懒计算并缓存，
// 防止调用方对空串做切片操作导致 panic。
func (c *ClipboardData) GetHash() string {
	if c.Hash == "" && c.Text != "" {
		c.Hash = calculateHash(c.Text)
	}
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

func (c *ClipboardData) IsImage() bool {
	return c.Type == "Image"
}

func (c *ClipboardData) IsFile() bool {
	return c.Type == "File"
}

func (c *ClipboardData) IsGroup() bool {
	return c.Type == "Group"
}

func (c *ClipboardData) IsDownloadableFile() bool {
	return c.IsImage() || c.IsFile() || c.IsGroup()
}
