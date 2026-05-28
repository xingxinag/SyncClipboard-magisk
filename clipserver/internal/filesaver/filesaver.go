package filesaver

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const DefaultDownloadDir = "/sdcard/Download/SyncClipboard"

type SaveResult struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
	Size     int    `json:"size"`
}

func SaveBytes(filename string, data []byte) (*SaveResult, error) {
	safeName, err := safeFilename(filename)
	if err != nil {
		return nil, err
	}
	dir := downloadDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}
	target := uniquePath(filepath.Join(dir, safeName))
	if err := os.WriteFile(target, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}
	return &SaveResult{Path: target, Filename: filepath.Base(target), Size: len(data)}, nil
}

func safeFilename(filename string) (string, error) {
	name := strings.TrimSpace(filename)
	if name == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "." || name == ".." || name == "" {
		return "", fmt.Errorf("invalid filename: %s", filename)
	}
	return name, nil
}

func downloadDir() string {
	if dir := strings.TrimSpace(os.Getenv("SYNCCLIPBOARD_DOWNLOAD_DIR")); dir != "" {
		return dir
	}
	return DefaultDownloadDir
}

func uniquePath(target string) string {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target
	}
	ext := filepath.Ext(target)
	base := strings.TrimSuffix(target, ext)
	suffix := time.Now().Format("20060102_150405")
	candidate := fmt.Sprintf("%s_%s%s", base, suffix, ext)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	for i := 1; ; i++ {
		candidate = fmt.Sprintf("%s_%s_%d%s", base, suffix, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
