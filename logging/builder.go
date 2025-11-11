package logging

import (
	"os"
	"sync"
)

// LoggingBuilder 日志构建器
type LoggingBuilder struct {
	providers    []LoggerProvider
	minimumLevel LogLevel
	mu           sync.RWMutex
}

// NewLoggingBuilder 创建日志构建器
func NewLoggingBuilder() *LoggingBuilder {
	return &LoggingBuilder{
		providers:    make([]LoggerProvider, 0),
		minimumLevel: LogLevelInfo,
	}
}

// SetMinimumLevel 设置最小日志级别
func (b *LoggingBuilder) SetMinimumLevel(level LogLevel) *LoggingBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.minimumLevel = level
	return b
}

// AddProvider 添加日志提供者
func (b *LoggingBuilder) AddProvider(provider LoggerProvider) *LoggingBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	provider.SetMinimumLevel(b.minimumLevel)
	b.providers = append(b.providers, provider)
	return b
}

// AddConsole 添加控制台日志
func (b *LoggingBuilder) AddConsole(options ...ConsoleLoggerOptions) *LoggingBuilder {
	opts := ConsoleLoggerOptions{
		IncludeTimestamp: true,
		TimestampFormat:  "2006-01-02 15:04:05",
		ColorOutput:      true,
		Output:           os.Stdout,
	}
	if len(options) > 0 {
		opts = options[0]
	}
	return b.AddProvider(NewConsoleLoggerProvider(opts))
}

// AddFile 添加文件日志
func (b *LoggingBuilder) AddFile(path string, options ...FileLoggerOptions) *LoggingBuilder {
	opts := FileLoggerOptions{
		Path:       path,
		MaxSize:    100 * 1024 * 1024, // 100MB
		MaxBackups: 10,
	}
	if len(options) > 0 {
		opts = options[0]
	}
	return b.AddProvider(NewFileLoggerProvider(opts))
}

// Build 构建日志工厂
func (b *LoggingBuilder) Build() LoggerFactory {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// 创建日志工厂
	factory := &loggerFactory{
		providers:    make([]LoggerProvider, 0),
		minimumLevel: b.minimumLevel,
	}

	for _, provider := range b.providers {
		factory.AddProvider(provider)
	}

	return factory
}
