package clipboard

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// getClipboardDatabase 方法5: 直接读取剪贴板数据库（Root 特权）
func getClipboardDatabase() (string, error) {
	// 可能的数据库路径
	dbPaths := []string{
		"/data/data/com.android.providers.settings/databases/clipboard.db",
		"/data/system/users/0/clipboard.db",
		"/data/clipboard/clipboard.db",
	}

	for _, dbPath := range dbPaths {
		content, err := readClipboardFromDB(dbPath)
		if err == nil && content != "" {
			return content, nil
		}
	}

	return "", errors.New("database method failed")
}

// readClipboardFromDB 从数据库读取剪贴板
func readClipboardFromDB(dbPath string) (string, error) {
	// 使用 sqlite3 命令读取
	query := "SELECT text FROM clipboard ORDER BY _id DESC LIMIT 1"
	cmdStr := fmt.Sprintf("sqlite3 %s \"%s\"", dbPath, query)
	cmd := exec.Command("su", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(output))
	if content == "" || strings.Contains(content, "Error") {
		return "", errors.New("no data in database")
	}

	return content, nil
}

// getClipboardDumpsys 方法6: 通过 dumpsys 获取（调试用）
func getClipboardDumpsys() (string, error) {
	cmd := exec.Command("su", "-c", "dumpsys clipboard")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// 解析 dumpsys 输出
	return parseClipboardDumpsys(string(output)), nil
}

// parseClipboardDumpsys 解析 dumpsys clipboard 输出
func parseClipboardDumpsys(output string) string {
	// dumpsys clipboard 输出格式示例:
	// Current clipboard: ClipData { text/plain "content" }
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Current clipboard") || strings.Contains(line, "text/plain") {
			// 提取引号中的内容
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start != -1 && end != -1 && start < end {
				return line[start+1 : end]
			}
		}
	}

	// ColorOS / 深度定制系统常见格式：service call 异常栈 + Parcel 十六进制字符块
	decoded := decodeParcelWideString(output)
	if decoded != "" && !isInvalidClipboardOutput(decoded) {
		return decoded
	}
	return ""
}

func decodeParcelWideString(output string) string {
	lineRe := regexp.MustCompile(`0x[0-9a-fA-F]+:\s*(.*)$`)
	tokenRe := regexp.MustCompile(`[0-9a-fA-F]{8}`)

	var bytesBuf []byte
	for _, line := range strings.Split(output, "\n") {
		m := lineRe.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		tokens := tokenRe.FindAllString(m[1], -1)
		for _, tok := range tokens {
			b, err := hex.DecodeString(tok)
			if err != nil || len(b) != 4 {
				continue
			}
			bytesBuf = append(bytesBuf, b...)
		}
	}

	if len(bytesBuf) < 4 {
		return ""
	}

	// 从 LE UTF-16 片段恢复字符串
	var r []rune
	for i := 0; i+1 < len(bytesBuf); i += 2 {
		code := uint16(bytesBuf[i]) | uint16(bytesBuf[i+1])<<8
		if code == 0 {
			continue
		}
		if code >= 0x20 && code <= 0x7e {
			r = append(r, rune(code))
			continue
		}
		if code >= 0x4e00 && code <= 0x9fff {
			r = append(r, rune(code))
		}
	}

	if len(r) == 0 {
		return ""
	}

	decoded := strings.TrimSpace(string(r))
	if strings.Contains(decoded, "android") && strings.Contains(decoded, "clipboard") {
		return ""
	}
	return decoded
}

// getClipboardSharedMemory 方法7: 直接读取共享内存（某些系统）
func getClipboardSharedMemory() (string, error) {
	// 尝试读取可能的共享内存路径
	paths := []string{
		"/dev/clipboard",
		"/dev/shm/clipboard",
		"/tmp/clipboard",
	}

	for _, path := range paths {
		cmd := exec.Command("su", "-c", "cat "+path)
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			return string(output), nil
		}
	}

	return "", errors.New("shared memory method failed")
}

// setClipboardDatabase 方法5: 直接写入数据库
func setClipboardDatabase(content string) error {
	dbPaths := []string{
		"/data/data/com.android.providers.settings/databases/clipboard.db",
		"/data/system/users/0/clipboard.db",
	}

	for _, dbPath := range dbPaths {
		// 转义 SQL 特殊字符
		escapedContent := strings.ReplaceAll(content, "'", "''")
		query := fmt.Sprintf("INSERT INTO clipboard (text, timestamp) VALUES ('%s', %d)",
			escapedContent, time.Now().Unix())

		cmdStr := fmt.Sprintf("sqlite3 %s \"%s\"", dbPath, query)
		cmd := exec.Command("su", "-c", cmdStr)
		err := cmd.Run()
		if err == nil {
			return nil
		}
	}

	return errors.New("database write failed")
}

// setClipboardSharedMemory 方法6: 直接写入共享内存
func setClipboardSharedMemory(content string) error {
	paths := []string{
		"/dev/clipboard",
		"/dev/shm/clipboard",
		"/tmp/clipboard",
	}

	for _, path := range paths {
		cmdStr := fmt.Sprintf("echo '%s' > %s", content, path)
		cmd := exec.Command("su", "-c", cmdStr)
		err := cmd.Run()
		if err == nil {
			return nil
		}
	}

	return errors.New("shared memory write failed")
}
