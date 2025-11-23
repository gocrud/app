package logging

import (
	"time"
)

// Formatter 日志格式化接口
type Formatter interface {
	// Format 格式化日志条目
	Format(entry *LogEntry) ([]byte, error)
}

// LogEntry 日志条目
type LogEntry struct {
	Time     time.Time
	Level    LogLevel
	Category string
	Message  string
	Fields   []Field
}

