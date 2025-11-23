package logging

import (
	"encoding/json"
)

// JsonFormatter JSON 格式化器
type JsonFormatter struct {
	TimestampFormat string
}

// NewJsonFormatter 创建 JSON 格式化器
func NewJsonFormatter() *JsonFormatter {
	return &JsonFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	}
}

// Format 格式化日志
func (f *JsonFormatter) Format(entry *LogEntry) ([]byte, error) {
	data := make(map[string]interface{})

	data["time"] = entry.Time.Format(f.TimestampFormat)
	data["level"] = entry.Level.String()
	if entry.Category != "" {
		data["category"] = entry.Category
	}
	data["msg"] = entry.Message

	if len(entry.Fields) > 0 {
		fields := make(map[string]interface{})
		for _, field := range entry.Fields {
			fields[field.Key] = field.Value
		}
		data["fields"] = fields
	}

	return json.Marshal(data)
}
