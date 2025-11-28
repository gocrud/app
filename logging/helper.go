package logging

// NewLogger 创建一个默认的控制台 Logger（便于测试使用）
func NewLogger() Logger {
	builder := NewLoggingBuilder()
	builder.AddConsole()
	factory := builder.Build()
	return factory.CreateLogger("default")
}
