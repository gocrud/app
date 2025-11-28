# 日志系统 (Logging)

`logging` 包提供了一个高性能、结构化的日志记录系统。

## 核心特性

*   **结构化**: 基于 Key-Value 对的结构化日志，方便机器解析和检索。
*   **高性能**: 异步写入，零内存分配（Zero Allocation）设计目标。
*   **级别控制**: 支持 Trace, Debug, Info, Warn, Error, Fatal 六个级别。
*   **多输出**: 支持控制台（Console）、文件（File）等多种输出目标。
*   **Scope**: 支持基于作用域的日志上下文（Logger with Context）。

## 快速开始

### 1. 获取 Logger

通常通过构造函数注入 `logging.Logger`。

```go
import "github.com/gocrud/app/logging"

type MyService struct {
    logger logging.Logger
}

func NewMyService(logger logging.Logger) *MyService {
    // 创建带有上下文的子 Logger
    // 所有通过 subLogger 打印的日志都会带上 "service": "MyService"
    subLogger := logger.With(logging.Field{Key: "service", Value: "MyService"})
    
    return &MyService{logger: subLogger}
}
```

### 2. 记录日志

```go
func (s *MyService) Process(orderId string) {
    s.logger.Info("Processing order",
        logging.Field{Key: "order_id", Value: orderId},
        logging.Field{Key: "status", Value: "pending"},
    )
    
    if err := process(orderId); err != nil {
        s.logger.Error("Failed to process order",
            logging.Field{Key: "error", Value: err.Error()},
        )
    }
}
```

## 配置日志

在 `ApplicationBuilder` 中配置日志系统。

```go
builder.ConfigureLogging(func(b *logging.LoggingBuilder) {
    // 设置最小日志级别
    b.SetMinimumLevel(logging.LogLevelInfo)
    
    // 添加控制台输出
    b.AddConsole()
    
    // 添加文件输出 (如果支持)
    // b.AddFile("logs/app.log")
})
```

## 日志级别

| 级别 | 值 | 描述 |
| :--- | :--- | :--- |
| `Trace` | 0 | 最详细的跟踪信息，通常仅用于开发调试。 |
| `Debug` | 1 | 调试信息，用于排查问题。 |
| `Info` | 2 | 一般信息，记录程序正常运行的关键事件。 |
| `Warn` | 3 | 警告信息，表明可能存在潜在问题，但不影响系统运行。 |
| `Error` | 4 | 错误信息，表明当前操作失败。 |
| `Fatal` | 5 | 致命错误，表明系统无法继续运行（通常会随后退出程序）。 |

## 最佳实践

1.  **总是使用结构化字段**: 避免使用 `fmt.Sprintf` 拼接日志消息。
    *   ❌ `logger.Info(fmt.Sprintf("User %s logged in", userId))`
    *   ✅ `logger.Info("User logged in", logging.Field{Key: "user_id", Value: userId})`
2.  **使用 With 创建上下文**: 对于某个服务或请求处理流程，使用 `logger.With(...)` 预置公共字段。
3.  **区分 Debug 和 Info**: 开发环境开 Debug，生产环境通常开 Info。

