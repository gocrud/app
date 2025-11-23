package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTextFormatter(t *testing.T) {
	f := NewTextFormatter()
	f.ColorOutput = false
	entry := &LogEntry{
		Time:     time.Now(),
		Level:    LogLevelInfo,
		Category: "Test",
		Message:  "Hello",
		Fields:   []Field{{Key: "key", Value: "val"}},
	}

	out, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	str := string(out)
	if !strings.Contains(str, "INFO") {
		t.Error("Expected level INFO")
	}
	if !strings.Contains(str, "[Test]") {
		t.Error("Expected category [Test]")
	}
	if !strings.Contains(str, "Hello") {
		t.Error("Expected message Hello")
	}
	if !strings.Contains(str, "key=val") {
		t.Error("Expected field key=val")
	}
}

func TestJsonFormatter(t *testing.T) {
	f := NewJsonFormatter()
	entry := &LogEntry{
		Time:     time.Now(),
		Level:    LogLevelInfo,
		Category: "Test",
		Message:  "Hello",
		Fields:   []Field{{Key: "key", Value: "val"}},
	}

	out, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(out, &data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if data["level"] != "INFO" {
		t.Error("Expected level INFO")
	}
	if data["category"] != "Test" {
		t.Error("Expected category Test")
	}
	fields, ok := data["fields"].(map[string]interface{})
	if !ok {
		t.Error("Expected fields map")
	} else if fields["key"] != "val" {
		t.Error("Expected key=val")
	}
}

func TestAsyncWriter(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex

	// 简单的线程安全 Writer wrapper
	writer := &syncWriter{buf: &buf, mu: &mu}

	formatter := NewTextFormatter()
	asyncWriter := NewAsyncWriter(writer, formatter, 10)

	entry := &LogEntry{
		Time:    time.Now(),
		Level:   LogLevelInfo,
		Message: "Async",
	}

	// 写入多条日志
	for i := 0; i < 5; i++ {
		asyncWriter.WriteLog(entry)
	}

	// 关闭以刷新
	asyncWriter.Close()

	// 检查输出行数
	output := writer.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}
}

type syncWriter struct {
	buf *bytes.Buffer
	mu  *sync.Mutex
}

func (w *syncWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *syncWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func BenchmarkAsyncLogging(b *testing.B) {
	formatter := NewTextFormatter()
	// 使用 io.Discard 避免 I/O 瓶颈，测试 AsyncWriter 自身的开销
	asyncWriter := NewAsyncWriter(io.Discard, formatter, 10000)
	defer asyncWriter.Close()

	entry := &LogEntry{
		Time:    time.Now(),
		Level:   LogLevelInfo,
		Message: "Benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		asyncWriter.WriteLog(entry)
	}
}
