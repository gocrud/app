package logging

import (
	"fmt"
)

// TextFormatter 文本格式化器
type TextFormatter struct {
	IncludeTimestamp bool
	TimestampFormat  string
	ColorOutput      bool
}

// NewTextFormatter 创建文本格式化器
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		IncludeTimestamp: true,
		TimestampFormat:  "2006-01-02 15:04:05",
		ColorOutput:      false,
	}
}

// Format 格式化日志
func (f *TextFormatter) Format(entry *LogEntry) ([]byte, error) {
	buffer := GlobalBufferPool.Get()
	// 注意：调用者负责在写入完成后 Put 回 Pool，或者我们在这里返回 []byte 的副本
	// 为了性能，通常 Writer 会直接使用 Buffer，但这里接口定义返回 []byte
	// 我们先简单实现，返回 bytes，调用者如果不复用 buffer 也没关系，
	// 但为了配合 AsyncWriter，我们可能需要调整接口设计或者在 Writer 中使用 BufferPool。
	// 鉴于 Formatter 接口定义返回 []byte，这里我们返回 buffer.Bytes()
	// 但 buffer 需要归还。
	// 更好的方式是 Formatter 接受一个 Buffer 参数。
	// 不过为了遵循 current plan，我们先按 buffer pool 的用法。

	// 重新思考：如果 Format 返回 []byte，那么 Buffer 必须在 Format 内部释放？不行。
	// 如果返回的是 slice referencing buffer's array，那么 buffer 不能过早 Reset。
	// 所以 Formatter 最好接受 io.Writer 或者 *bytes.Buffer。
	// 但为了简单起见，我们让 Format 只是生成内容。
	// 实际上，由于 AsyncWriter 需要拷贝数据（因为是异步的），
	// 所以这里 format 出来的数据最好是独立的。

	// 让我们修改一下策略：Format 返回一个新的 []byte，或者我们就在这里拼接 string
	// 既然是为了优化，我们应该避免分配。
	// 但是 AsyncWriter 接收的是 []byte。

	// 让我们先实现基本的写入逻辑。

	// 时间戳
	if f.IncludeTimestamp {
		buffer.WriteString(entry.Time.Format(f.TimestampFormat))
		buffer.WriteByte(' ')
	}

	// 级别
	levelStr := entry.Level.String()
	if f.ColorOutput {
		buffer.WriteString(colorize(entry.Level, levelStr))
	} else {
		buffer.WriteString(levelStr)
	}

	// 类别
	if entry.Category != "" {
		buffer.WriteString(" [")
		buffer.WriteString(entry.Category)
		buffer.WriteString("]")
	}

	// 消息
	buffer.WriteByte(' ')
	buffer.WriteString(entry.Message)

	// 字段
	if len(entry.Fields) > 0 {
		buffer.WriteString(" {")
		for i, field := range entry.Fields {
			if i > 0 {
				buffer.WriteString(", ")
			}
			buffer.WriteString(field.Key)
			buffer.WriteByte('=')
			fmt.Fprintf(buffer, "%v", field.Value)
		}
		buffer.WriteByte('}')
	}

	buffer.WriteByte('\n')

	// 复制结果，归还 buffer
	result := make([]byte, buffer.Len())
	copy(result, buffer.Bytes())
	GlobalBufferPool.Put(buffer)

	return result, nil
}
