package syncdata

import (
	"strings"
	"testing"
)

func TestNewTextClipboard(t *testing.T) {
	text := "Hello, World!"
	data := NewTextClipboard(text)

	if data.Type != "Text" {
		t.Errorf("Expected Type to be 'Text', got '%s'", data.Type)
	}

	if data.Text != text {
		t.Errorf("Expected Text to be '%s', got '%s'", text, data.Text)
	}

	if data.Size != len(text) {
		t.Errorf("Expected Size to be %d, got %d", len(text), data.Size)
	}

	if data.HasData {
		t.Error("Expected HasData to be false")
	}

	if data.DataName != nil {
		t.Error("Expected DataName to be nil")
	}

	if data.Hash == "" {
		t.Error("Expected Hash to be non-empty")
	}

	// Hash should be uppercase
	if data.Hash != strings.ToUpper(data.Hash) {
		t.Error("Expected Hash to be uppercase")
	}
}

func TestToJSON(t *testing.T) {
	text := "Test content"
	data := NewTextClipboard(text)

	jsonStr, err := data.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if !strings.Contains(jsonStr, `"type":"Text"`) {
		t.Error("JSON should contain type field")
	}

	if !strings.Contains(jsonStr, `"text":"Test content"`) {
		t.Error("JSON should contain text field")
	}
}

func TestFromJSON(t *testing.T) {
	jsonStr := `{
		"type": "Text",
		"hash": "ABC123",
		"text": "Hello",
		"hasData": false,
		"dataName": null,
		"size": 5
	}`

	data, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if data.Type != "Text" {
		t.Errorf("Expected Type to be 'Text', got '%s'", data.Type)
	}

	if data.Text != "Hello" {
		t.Errorf("Expected Text to be 'Hello', got '%s'", data.Text)
	}

	if data.Hash != "ABC123" {
		t.Errorf("Expected Hash to be 'ABC123', got '%s'", data.Hash)
	}

	if data.Size != 5 {
		t.Errorf("Expected Size to be 5, got %d", data.Size)
	}
}

func TestRoundTrip(t *testing.T) {
	text := "Round trip test"
	original := NewTextClipboard(text)

	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	parsed, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if parsed.Text != original.Text {
		t.Errorf("Text mismatch: expected '%s', got '%s'", original.Text, parsed.Text)
	}

	if parsed.Hash != original.Hash {
		t.Errorf("Hash mismatch: expected '%s', got '%s'", original.Hash, parsed.Hash)
	}
}

// TestGetHashEmptyFieldLazyCompute 验证旧版服务端未写 hash 时不会 panic，且能正确懒计算
func TestGetHashEmptyFieldLazyCompute(t *testing.T) {
	// 模拟从旧版 WebDAV 服务端反序列化的数据（hash 字段缺失）
	jsonStr := `{"type":"Text","hash":"","text":"hello world","hasData":false,"dataName":null,"size":11}`
	data, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	// 必须不 panic，且返回非空 hash
	hash := data.GetHash()
	if hash == "" {
		t.Error("GetHash() should lazy-compute hash when field is empty")
	}
	if len(hash) < 8 {
		t.Errorf("GetHash() returned suspiciously short hash: %q", hash)
	}
	// 第二次调用应返回相同值（缓存）
	if data.GetHash() != hash {
		t.Error("GetHash() should return cached value on second call")
	}
}

// TestGetHashBothEmptyReturnsEmpty 验证 hash 和 text 都为空时返回空串而非 panic
func TestGetHashBothEmptyReturnsEmpty(t *testing.T) {
	data := &ClipboardData{Type: "Text", Hash: "", Text: ""}
	if got := data.GetHash(); got != "" {
		t.Errorf("expected empty hash, got %q", got)
	}
}

func TestCalculateHash(t *testing.T) {
	// Test that same content produces same hash
	text := "Test"
	hash1 := calculateHash(text)
	hash2 := calculateHash(text)

	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}

	// Test that different content produces different hash
	hash3 := calculateHash("Different")
	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}
}
