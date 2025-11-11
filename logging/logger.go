package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Field 日志字段
type Field struct {
	Key   string
	Value any
}

// Logger 日志接口（类似于 .NET Core ILogger）
type Logger interface {
	Trace(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	Log(level LogLevel, msg string, fields ...Field)
	WithFields(fields ...Field) Logger
	WithCategory(category string) Logger
}

// LoggerFactory 日志工厂接口
type LoggerFactory interface {
	CreateLogger(category string) Logger
	AddProvider(provider LoggerProvider)
	SetMinimumLevel(level LogLevel)
}

// LoggerProvider 日志提供者接口
// LoggerProvider 日志提供者接口
type LoggerProvider interface {
	CreateLogger(category string) Logger
	SetMinimumLevel(level LogLevel)
}

// loggerFactory 日志工厂实现
type loggerFactory struct {
	providers    []LoggerProvider
	minimumLevel LogLevel
	mu           sync.RWMutex
}

func (f *loggerFactory) CreateLogger(category string) Logger {
	f.mu.RLock()
	defer f.mu.RUnlock()

	loggers := make([]Logger, 0, len(f.providers))
	for _, provider := range f.providers {
		loggers = append(loggers, provider.CreateLogger(category))
	}

	return &compositeLogger{
		loggers:      loggers,
		minimumLevel: f.minimumLevel,
		category:     category,
	}
}

func (f *loggerFactory) AddProvider(provider LoggerProvider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	provider.SetMinimumLevel(f.minimumLevel)
	f.providers = append(f.providers, provider)
}

func (f *loggerFactory) SetMinimumLevel(level LogLevel) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.minimumLevel = level
	for _, provider := range f.providers {
		provider.SetMinimumLevel(level)
	}
}

// compositeLogger 组合日志记录器（将日志发送到多个提供者）
type compositeLogger struct {
	loggers      []Logger
	minimumLevel LogLevel
	category     string
	fields       []Field
}

// NewCompositeLogger 创建组合日志记录器（用于外部包构建）
func NewCompositeLogger(loggers []Logger, minimumLevel LogLevel, category string) Logger {
	return &compositeLogger{
		loggers:      loggers,
		minimumLevel: minimumLevel,
		category:     category,
		fields:       make([]Field, 0),
	}
}

func (l *compositeLogger) Trace(msg string, fields ...Field) {
	l.Log(LogLevelTrace, msg, fields...)
}

func (l *compositeLogger) Debug(msg string, fields ...Field) {
	l.Log(LogLevelDebug, msg, fields...)
}

func (l *compositeLogger) Info(msg string, fields ...Field) {
	l.Log(LogLevelInfo, msg, fields...)
}

func (l *compositeLogger) Warn(msg string, fields ...Field) {
	l.Log(LogLevelWarn, msg, fields...)
}

func (l *compositeLogger) Error(msg string, fields ...Field) {
	l.Log(LogLevelError, msg, fields...)
}

func (l *compositeLogger) Fatal(msg string, fields ...Field) {
	l.Log(LogLevelFatal, msg, fields...)
	os.Exit(1)
}

func (l *compositeLogger) Log(level LogLevel, msg string, fields ...Field) {
	if level < l.minimumLevel {
		return
	}

	// 合并字段
	allFields := append(l.fields, fields...)

	for _, logger := range l.loggers {
		logger.Log(level, msg, allFields...)
	}
}

func (l *compositeLogger) WithFields(fields ...Field) Logger {
	return &compositeLogger{
		loggers:      l.loggers,
		minimumLevel: l.minimumLevel,
		category:     l.category,
		fields:       append(l.fields, fields...),
	}
}

func (l *compositeLogger) WithCategory(category string) Logger {
	return &compositeLogger{
		loggers:      l.loggers,
		minimumLevel: l.minimumLevel,
		category:     category,
		fields:       l.fields,
	}
}

// ConsoleLoggerOptions 控制台日志选项
type ConsoleLoggerOptions struct {
	IncludeTimestamp bool
	TimestampFormat  string
	ColorOutput      bool
	Output           io.Writer
}

// ConsoleLoggerProvider 控制台日志提供者
type ConsoleLoggerProvider struct {
	options      ConsoleLoggerOptions
	minimumLevel LogLevel
	mu           sync.RWMutex
}

func NewConsoleLoggerProvider(options ConsoleLoggerOptions) *ConsoleLoggerProvider {
	if options.Output == nil {
		options.Output = os.Stdout
	}
	return &ConsoleLoggerProvider{
		options:      options,
		minimumLevel: LogLevelInfo,
	}
}

func (p *ConsoleLoggerProvider) CreateLogger(category string) Logger {
	return &consoleLogger{
		category:     category,
		options:      p.options,
		minimumLevel: p.minimumLevel,
	}
}

func (p *ConsoleLoggerProvider) SetMinimumLevel(level LogLevel) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.minimumLevel = level
}

// consoleLogger 控制台日志实现
type consoleLogger struct {
	category     string
	options      ConsoleLoggerOptions
	minimumLevel LogLevel
	fields       []Field
	mu           sync.Mutex
}

func (l *consoleLogger) Trace(msg string, fields ...Field) {
	l.Log(LogLevelTrace, msg, fields...)
}

func (l *consoleLogger) Debug(msg string, fields ...Field) {
	l.Log(LogLevelDebug, msg, fields...)
}

func (l *consoleLogger) Info(msg string, fields ...Field) {
	l.Log(LogLevelInfo, msg, fields...)
}

func (l *consoleLogger) Warn(msg string, fields ...Field) {
	l.Log(LogLevelWarn, msg, fields...)
}

func (l *consoleLogger) Error(msg string, fields ...Field) {
	l.Log(LogLevelError, msg, fields...)
}

func (l *consoleLogger) Fatal(msg string, fields ...Field) {
	l.Log(LogLevelFatal, msg, fields...)
	os.Exit(1)
}

func (l *consoleLogger) Log(level LogLevel, msg string, fields ...Field) {
	if level < l.minimumLevel {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 构建日志消息
	var output string

	// 时间戳
	if l.options.IncludeTimestamp {
		output += time.Now().Format(l.options.TimestampFormat) + " "
	}

	// 日志级别（带颜色）
	if l.options.ColorOutput {
		output += colorize(level, level.String())
	} else {
		output += level.String()
	}

	// 类别
	if l.category != "" {
		output += fmt.Sprintf(" [%s]", l.category)
	}

	// 消息
	output += " " + msg

	// 字段
	allFields := append(l.fields, fields...)
	if len(allFields) > 0 {
		output += " {"
		for i, field := range allFields {
			if i > 0 {
				output += ", "
			}
			output += fmt.Sprintf("%s=%v", field.Key, field.Value)
		}
		output += "}"
	}

	fmt.Fprintln(l.options.Output, output)
}

func (l *consoleLogger) WithFields(fields ...Field) Logger {
	return &consoleLogger{
		category:     l.category,
		options:      l.options,
		minimumLevel: l.minimumLevel,
		fields:       append(l.fields, fields...),
	}
}

func (l *consoleLogger) WithCategory(category string) Logger {
	return &consoleLogger{
		category:     category,
		options:      l.options,
		minimumLevel: l.minimumLevel,
		fields:       l.fields,
	}
}

// colorize 为日志级别添加颜色
func colorize(level LogLevel, text string) string {
	const (
		reset   = "\033[0m"
		gray    = "\033[90m"
		cyan    = "\033[36m"
		green   = "\033[32m"
		yellow  = "\033[33m"
		red     = "\033[31m"
		magenta = "\033[35m"
	)

	switch level {
	case LogLevelTrace:
		return gray + text + reset
	case LogLevelDebug:
		return cyan + text + reset
	case LogLevelInfo:
		return green + text + reset
	case LogLevelWarn:
		return yellow + text + reset
	case LogLevelError:
		return red + text + reset
	case LogLevelFatal:
		return magenta + text + reset
	default:
		return text
	}
}

// FileLoggerOptions 文件日志选项
type FileLoggerOptions struct {
	Path       string
	MaxSize    int64 // 字节
	MaxBackups int
	Compress   bool
}

// FileLoggerProvider 文件日志提供者
type FileLoggerProvider struct {
	options      FileLoggerOptions
	minimumLevel LogLevel
	file         *os.File
	mu           sync.RWMutex
}

func NewFileLoggerProvider(options FileLoggerOptions) *FileLoggerProvider {
	return &FileLoggerProvider{
		options:      options,
		minimumLevel: LogLevelInfo,
	}
}

func (p *FileLoggerProvider) CreateLogger(category string) Logger {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 打开或创建文件
	if p.file == nil {
		file, err := os.OpenFile(p.options.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			return &consoleLogger{category: category, options: ConsoleLoggerOptions{Output: os.Stderr}}
		}
		p.file = file
	}

	return &fileLogger{
		category:     category,
		file:         p.file,
		minimumLevel: p.minimumLevel,
	}
}

func (p *FileLoggerProvider) SetMinimumLevel(level LogLevel) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.minimumLevel = level
}

// fileLogger 文件日志实现
type fileLogger struct {
	category     string
	file         *os.File
	minimumLevel LogLevel
	fields       []Field
	mu           sync.Mutex
}

func (l *fileLogger) Trace(msg string, fields ...Field) {
	l.Log(LogLevelTrace, msg, fields...)
}

func (l *fileLogger) Debug(msg string, fields ...Field) {
	l.Log(LogLevelDebug, msg, fields...)
}

func (l *fileLogger) Info(msg string, fields ...Field) {
	l.Log(LogLevelInfo, msg, fields...)
}

func (l *fileLogger) Warn(msg string, fields ...Field) {
	l.Log(LogLevelWarn, msg, fields...)
}

func (l *fileLogger) Error(msg string, fields ...Field) {
	l.Log(LogLevelError, msg, fields...)
}

func (l *fileLogger) Fatal(msg string, fields ...Field) {
	l.Log(LogLevelFatal, msg, fields...)
	os.Exit(1)
}

func (l *fileLogger) Log(level LogLevel, msg string, fields ...Field) {
	if level < l.minimumLevel {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 构建日志消息
	output := fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05"), level.String())

	if l.category != "" {
		output += fmt.Sprintf(" [%s]", l.category)
	}

	output += " " + msg

	// 字段
	allFields := append(l.fields, fields...)
	if len(allFields) > 0 {
		output += " {"
		for i, field := range allFields {
			if i > 0 {
				output += ", "
			}
			output += fmt.Sprintf("%s=%v", field.Key, field.Value)
		}
		output += "}"
	}

	fmt.Fprintln(l.file, output)
}

func (l *fileLogger) WithFields(fields ...Field) Logger {
	return &fileLogger{
		category:     l.category,
		file:         l.file,
		minimumLevel: l.minimumLevel,
		fields:       append(l.fields, fields...),
	}
}

func (l *fileLogger) WithCategory(category string) Logger {
	return &fileLogger{
		category:     category,
		file:         l.file,
		minimumLevel: l.minimumLevel,
		fields:       l.fields,
	}
}
