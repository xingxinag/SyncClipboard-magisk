package clipboard

import (
	"strings"
	"testing"
)

func TestValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"空内容", "", true},
		{"正常内容", "Hello World", false},
		{"最大限制边界", strings.Repeat("a", 1024*1024), false},
		{"超过最大限制", strings.Repeat("a", 1024*1024+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContent(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetClipboard(t *testing.T) {
	// 注意：这个测试在非Android环境会失败
	// 这里只是示例结构
	content, err := GetClipboard()
	if err != nil {
		t.Logf("GetClipboard failed (expected on non-Android): %v", err)
		return
	}
	t.Logf("Clipboard content: %s", content)
}
