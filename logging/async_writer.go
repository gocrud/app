package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// AsyncWriter 异步日志写入器
type AsyncWriter struct {
	writer     io.Writer
	formatter  Formatter
	entryCh    chan *LogEntry
	wg         sync.WaitGroup
	closeOnce  sync.Once
	errHandler func(error)
}

// NewAsyncWriter 创建新的异步写入器
func NewAsyncWriter(writer io.Writer, formatter Formatter, bufferSize int) *AsyncWriter {
	w := &AsyncWriter{
		writer:    writer,
		formatter: formatter,
		entryCh:   make(chan *LogEntry, bufferSize),
	}

	// 启动后台写入协程
	w.wg.Add(1)
	go w.process()

	return w
}

// WriteLog 写入日志条目（非阻塞，除非 buffer 满）
func (w *AsyncWriter) WriteLog(entry *LogEntry) {
	select {
	case w.entryCh <- entry:
		// 成功入队
	default:
		// 队列满，阻塞等待直到有空间，保证不丢日志
		w.entryCh <- entry
	}
}

// Close 关闭写入器
func (w *AsyncWriter) Close() error {
	w.closeOnce.Do(func() {
		close(w.entryCh)
	})
	w.wg.Wait()
	return nil
}

func (w *AsyncWriter) process() {
	defer w.wg.Done()

	for entry := range w.entryCh {
		data, err := w.formatter.Format(entry)
		if err != nil {
			if w.errHandler != nil {
				w.errHandler(err)
			} else {
				fmt.Fprintf(os.Stderr, "AsyncWriter format error: %v\n", err)
			}
			continue
		}

		_, err = w.writer.Write(data)
		if err != nil {
			if w.errHandler != nil {
				w.errHandler(err)
			} else {
				fmt.Fprintf(os.Stderr, "AsyncWriter write error: %v\n", err)
			}
		}

		// 如果是 JSON formatter，Format 可能已经包含了 newline？
		// TextFormatter 是包含的。Json Marshal 通常不包含换行。
		// 这里我们简单判断一下，如果结尾没有换行就补一个。
		if len(data) > 0 && data[len(data)-1] != '\n' {
			w.writer.Write([]byte{'\n'})
		}
	}
}

// SetErrorHandler 设置错误处理函数
func (w *AsyncWriter) SetErrorHandler(handler func(error)) {
	w.errHandler = handler
}
