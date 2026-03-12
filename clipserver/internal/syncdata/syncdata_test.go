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
