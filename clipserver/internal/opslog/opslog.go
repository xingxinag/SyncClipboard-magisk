package opslog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	ID        int64                  `json:"id"`
	Timestamp int64                  `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
	Level     string                 `json:"level"`
	Source    string                 `json:"source"`
	Action    string                 `json:"action"`
	Duration  int64                  `json:"duration_ms,omitempty"`
	Result    string                 `json:"result,omitempty"`
	Code      string                 `json:"code,omitempty"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

var (
	mu        sync.Mutex
	nextID    int64 = 1
	maxKeep   int   = 500
	entries   []Entry
	logPath   string
	timeNowFn = func() int64 { return time.Now().Unix() }
)

func SetLogFile(path string) {
	mu.Lock()
	defer mu.Unlock()
	logPath = path
}

func Info(source, action, message string, fields map[string]interface{}) {
	add("info", source, action, message, fields)
}

func Error(source, action, message string, fields map[string]interface{}) {
	add("error", source, action, message, fields)
}

func Warn(source, action, message string, fields map[string]interface{}) {
	add("warn", source, action, message, fields)
}

func add(level, source, action, message string, fields map[string]interface{}) {
	entryFields := map[string]interface{}{}
	for k, v := range fields {
		entryFields[k] = v
	}

	entry := Entry{
		ID:        0,
		Timestamp: 0,
		Level:     level,
		Source:    source,
		Action:    action,
		Message:   message,
		Fields:    entryFields,
	}

	if rid, ok := entryFields["request_id"].(string); ok {
		entry.RequestID = rid
		delete(entryFields, "request_id")
	}
	if ms, ok := entryFields["duration_ms"].(int64); ok {
		entry.Duration = ms
		delete(entryFields, "duration_ms")
	}
	if msf, ok := entryFields["duration_ms"].(float64); ok {
		entry.Duration = int64(msf)
		delete(entryFields, "duration_ms")
	}
	if result, ok := entryFields["result"].(string); ok {
		entry.Result = result
		delete(entryFields, "result")
	}
	if code, ok := entryFields["code"].(string); ok {
		entry.Code = code
		delete(entryFields, "code")
	}

	mu.Lock()
	entry.ID = nextID
	entry.Timestamp = timeNowFn()
	nextID++
	entries = append(entries, entry)
	if len(entries) > maxKeep {
		entries = entries[len(entries)-maxKeep:]
	}
	path := logPath
	mu.Unlock()

	if path != "" {
		appendToFile(path, entry)
	}
}

func appendToFile(path string, entry Entry) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}

	b, err := json.Marshal(entry)
	if err != nil {
		return
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	_, _ = f.WriteString(string(b) + "\n")
}

func List(limit int) []Entry {
	mu.Lock()
	defer mu.Unlock()

	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}

	out := make([]Entry, 0, limit)
	start := len(entries) - limit
	if start < 0 {
		start = 0
	}
	for i := len(entries) - 1; i >= start; i-- {
		out = append(out, entries[i])
	}

	return out
}

func Clear() error {
	mu.Lock()
	entries = nil
	path := logPath
	mu.Unlock()

	if path == "" {
		return nil
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func Text(limit int) string {
	items := List(limit)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp < items[j].Timestamp
	})

	var b strings.Builder
	for _, it := range items {
		ts := time.Unix(it.Timestamp, 0).Format("2006-01-02 15:04:05")
		rid := "-"
		if it.RequestID != "" {
			rid = it.RequestID
		}
		dur := "-"
		if it.Duration > 0 {
			dur = fmt.Sprintf("%dms", it.Duration)
		}
		result := it.Result
		if result == "" {
			result = "-"
		}
		code := it.Code
		if code == "" {
			code = "-"
		}

		_, _ = b.WriteString(fmt.Sprintf("[%s] [%s] [%s] [%s/%s] [%s] [%s] [%s] %s", ts, strings.ToUpper(it.Level), rid, it.Source, it.Action, dur, result, code, it.Message))
		if len(it.Fields) > 0 {
			fb, _ := json.Marshal(it.Fields)
			_, _ = b.WriteString(" ")
			_, _ = b.Write(fb)
		}
		_, _ = b.WriteString("\n")
	}

	return b.String()
}
