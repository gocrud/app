# Cron 定时任务模块

Cron 模块为应用提供了强大的定时任务调度功能，基于 [robfig/cron](https://github.com/robfig/cron) 实现，并与框架的依赖注入（DI）和托管服务（HostedService）系统无缝集成。

## 特性

- ✅ **标准 Cron 表达式支持**：兼容标准的 cron 表达式语法
- ✅ **秒级精度**：可选启用秒级调度（默认为分钟级）
- ✅ **依赖注入集成**：任务函数支持自动依赖注入
- ✅ **时区支持**：可自定义时区设置
- ✅ **优雅启停**：与框架托管服务集成，支持优雅关闭
- ✅ **任务日志**：自动记录任务执行日志
- ✅ **错误恢复**：内置 panic 恢复机制

## 快速开始

### 基本用法

```go
package main

import (
    "github.com/gocrud/app/core"
    "github.com/gocrud/app/configure/cron"
)

func main() {
    builder := core.NewAppBuilder()
    
    builder.Configure(cron.Configure(func(b *cron.Builder) {
        // 添加简单任务 - 每5分钟执行一次
        b.AddJob("0 */5 * * * *", "cleanup-temp", func() {
            // 执行清理任务
            println("清理临时文件...")
        })
        
        // 每天凌晨2点执行
        b.AddJob("0 0 2 * * *", "daily-backup", func() {
            println("执行每日备份...")
        })
    }))
    
    app := builder.Build()
    app.Run()
}
```

### 启用秒级精度

默认情况下，cron 使用分钟级精度（5位表达式）。如果需要秒级调度，需要启用秒级支持：

```go
builder.Configure(cron.Configure(func(b *cron.Builder) {
    // 启用秒级精度
    b.WithSeconds()
    
    // 每30秒执行一次（6位表达式）
    b.AddJob("*/30 * * * * *", "heartbeat", func() {
        println("心跳检测...")
    })
}))
```

### 设置时区

```go
builder.Configure(cron.Configure(func(b *cron.Builder) {
    // 设置为中国时区
    b.WithLocation("Asia/Shanghai")
    
    // 每天北京时间上午9点执行
    b.AddJob("0 0 9 * * *", "morning-report", func() {
        println("生成早报...")
    })
}))
```

### 启用调度日志

```go
builder.Configure(cron.Configure(func(b *cron.Builder) {
    // 启用 cron 库的内部调度日志（用于调试）
    b.EnableCronLogger()
    
    b.AddJob("0 * * * * *", "test-job", func() {
        println("测试任务")
    })
}))
```

## 依赖注入支持

Cron 模块完全支持依赖注入，任务函数可以自动获取已注册的服务实例。

### 注入单个服务

```go
// 假设已注册 DataService
builder.Configure(cron.Configure(func(b *cron.Builder) {
    b.AddJobWithDI("0 */10 * * * *", "sync-data", 
        func(dataService *DataService) {
            dataService.Sync()
        })
}))
```

### 注入多个服务

```go
builder.Configure(cron.Configure(func(b *cron.Builder) {
    b.AddJobWithDI("0 0 * * * *", "hourly-stats", 
        func(
            statsService *StatsService,
            userService *UserService,
            logger logging.Logger,
        ) {
            users := userService.GetActiveUsers()
            statsService.Calculate(users)
            logger.Info("统计完成")
        })
}))
```

### 完整示例

```go
package main

import (
    "github.com/gocrud/app/core"
    "github.com/gocrud/app/configure/cron"
    "github.com/gocrud/app/di"
    "github.com/gocrud/app/logging"
)

type EmailService struct {
    logger logging.Logger
}

func (s *EmailService) SendDailyReport() {
    s.logger.Info("发送每日报告...")
}

type CacheService struct{}

func (s *CacheService) Clear() {
    println("清理缓存...")
}

func main() {
    builder := core.NewAppBuilder()
    
    // 注册服务
    builder.ConfigureServices(func(services *di.ServiceCollection) {
        services.AddSingleton(di.Provide(func(logger logging.Logger) *EmailService {
            return &EmailService{logger: logger}
        }))
        
        services.AddSingleton(di.Provide(func() *CacheService {
            return &CacheService{}
        }))
    })
    
    // 配置定时任务
    builder.Configure(cron.Configure(func(b *cron.Builder) {
        b.WithSeconds().
          WithLocation("Asia/Shanghai").
          EnableCronLogger()
        
        // 每天上午8点发送报告（注入 EmailService）
        b.AddJobWithDI("0 0 8 * * *", "daily-report", 
            func(email *EmailService) {
                email.SendDailyReport()
            })
        
        // 每小时清理缓存（注入多个服务）
        b.AddJobWithDI("0 0 * * * *", "cache-cleanup", 
            func(cache *CacheService, logger logging.Logger) {
                cache.Clear()
                logger.Info("缓存已清理")
            })
    }))
    
    app := builder.Build()
    app.Run()
}
```

## Cron 表达式语法

### 分钟级精度（5位表达式 - 默认）

```
┌───────────── 分钟 (0 - 59)
│ ┌─────────── 小时 (0 - 23)
│ │ ┌───────── 日期 (1 - 31)
│ │ │ ┌─────── 月份 (1 - 12)
│ │ │ │ ┌───── 星期 (0 - 6) (0 = 星期日)
│ │ │ │ │
* * * * *
```

### 秒级精度（6位表达式 - 启用 WithSeconds）

```
┌─────────────── 秒 (0 - 59)
│ ┌───────────── 分钟 (0 - 59)
│ │ ┌─────────── 小时 (0 - 23)
│ │ │ ┌───────── 日期 (1 - 31)
│ │ │ │ ┌─────── 月份 (1 - 12)
│ │ │ │ │ ┌───── 星期 (0 - 6)
│ │ │ │ │ │
* * * * * *
```

### 常用表达式示例

| 表达式 | 说明 | 精度 |
|--------|------|------|
| `0 * * * * *` | 每分钟 | 秒级 |
| `*/30 * * * * *` | 每30秒 | 秒级 |
| `0 */5 * * * *` | 每5分钟 | 秒级 |
| `0 0 * * * *` | 每小时 | 秒级 |
| `0 0 2 * * *` | 每天凌晨2点 | 秒级 |
| `0 0 9 * * 1-5` | 工作日上午9点 | 秒级 |
| `0 0 0 1 * *` | 每月1号凌晨 | 秒级 |
| `*/5 * * * *` | 每5分钟 | 分钟级 |
| `0 * * * *` | 每小时 | 分钟级 |
| `0 2 * * *` | 每天凌晨2点 | 分钟级 |
| `0 9 * * 1` | 每周一上午9点 | 分钟级 |

### 特殊字符

- `*` - 任意值
- `,` - 值列表分隔符，如 `1,3,5`
- `-` - 范围，如 `1-5`
- `/` - 步长，如 `*/10` 表示每10个单位

## 配置选项

### Builder 方法

| 方法 | 说明 | 默认值 |
|------|------|--------|
| `WithSeconds()` | 启用秒级精度 | `false` |
| `WithLocation(location)` | 设置时区 | `"UTC"` |
| `EnableCronLogger()` | 启用调度日志 | `false` |
| `AddJob(spec, name, handler)` | 添加简单任务 | - |
| `AddJobWithDI(spec, name, handler)` | 添加带 DI 的任务 | - |

### 链式调用

所有配置方法都支持链式调用：

```go
b.WithSeconds().
  WithLocation("Asia/Shanghai").
  EnableCronLogger().
  AddJob("*/30 * * * * *", "task1", func() {}).
  AddJobWithDI("0 0 * * * *", "task2", func(svc *Service) {})
```

## 任务生命周期

1. **配置阶段**：通过 `Configure` 函数配置任务
2. **注册阶段**：任务被注册到 cron 调度器
3. **启动阶段**：应用启动时，CronService 自动启动调度器
4. **执行阶段**：按 cron 表达式调度执行任务
5. **停止阶段**：应用关闭时，等待正在运行的任务完成后优雅停止

## 日志输出

### 任务日志

每个任务执行时会自动记录开始和完成日志：

```
INFO Cron job 'daily-backup' started
INFO 执行每日备份...
INFO Cron job 'daily-backup' completed
```

### 错误处理

如果任务执行过程中发生 panic，会自动捕获并记录：

```
ERROR Cron job panicked panic=runtime error: invalid memory address
```

### 调度日志（可选）

启用 `EnableCronLogger()` 后，会记录 cron 库的内部调度日志：

```
INFO schedule entries=5
INFO wake time=2025-11-11T10:00:00Z
```

## 最佳实践

### 1. 合理设置任务频率

```go
// ❌ 避免过于频繁的任务
b.AddJob("* * * * * *", "bad-job", func() {
    // 每秒执行，可能影响性能
})

// ✅ 根据实际需求设置合理频率
b.AddJob("0 */5 * * * *", "good-job", func() {
    // 每5分钟执行一次
})
```

### 2. 使用依赖注入管理服务

```go
// ❌ 使用全局变量
var globalDB *sql.DB

b.AddJob("0 0 * * * *", "bad-task", func() {
    globalDB.Query("...")
})

// ✅ 通过依赖注入获取服务
b.AddJobWithDI("0 0 * * * *", "good-task", 
    func(db *Database) {
        db.Query("...")
    })
```

### 3. 为任务命名

```go
// ✅ 使用有意义的任务名称
b.AddJob("0 2 * * *", "cleanup-old-logs", func() {})
b.AddJob("0 8 * * 1", "weekly-report", func() {})
```

### 4. 处理长时间运行的任务

```go
// ✅ 在任务内部添加超时控制
b.AddJobWithDI("0 0 * * * *", "long-task", 
    func(svc *Service, logger logging.Logger) {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
        defer cancel()
        
        if err := svc.ProcessWithContext(ctx); err != nil {
            logger.Error("任务执行失败", logging.Field{Key: "error", Value: err})
        }
    })
```

### 5. 避免任务重叠

```go
// 如果任务可能执行时间较长，确保下次调度时上一次已完成
// 可以使用互斥锁或其他机制
type TaskService struct {
    mu sync.Mutex
}

func (s *TaskService) Execute() {
    if !s.mu.TryLock() {
        // 上一次任务还在执行，跳过本次
        return
    }
    defer s.mu.Unlock()
    
    // 执行实际任务
}
```

## 故障排查

### 任务未执行

1. **检查 cron 表达式是否正确**
   ```go
   // 启用调度日志查看详细信息
   b.EnableCronLogger()
   ```

2. **确认时区设置**
   ```go
   // 确保时区与预期一致
   b.WithLocation("Asia/Shanghai")
   ```

3. **检查应用是否正常运行**
   ```go
   // 确保应用没有提前退出
   app.Run() // 阻塞主线程
   ```

### 依赖注入失败

1. **确认服务已注册**
   ```go
   builder.ConfigureServices(func(services *di.ServiceCollection) {
       services.AddSingleton(di.Provide(func() *MyService {
           return &MyService{}
       }))
   })
   ```

2. **检查日志输出**
   ```
   ERROR Failed to resolve parameter 0 (*MyService) for cron job
   ```

### 任务执行异常

查看错误日志并在任务内部添加错误处理：

```go
b.AddJobWithDI("0 * * * * *", "safe-task", 
    func(logger logging.Logger) {
        defer func() {
            if r := recover(); r != nil {
                logger.Error("任务 panic", logging.Field{Key: "panic", Value: r})
            }
        }()
        
        // 任务逻辑
    })
```

## 相关资源

- [robfig/cron 文档](https://pkg.go.dev/github.com/robfig/cron/v3)
- [Cron 表达式在线生成器](https://crontab.guru/)
- 框架依赖注入文档：`/di/README.md`
- 框架托管服务文档：`/hosting/README.md`

## API 参考

### Configure

```go
func Configure(options func(*Builder)) core.Configurator
```

创建 Cron 配置器，用于注册到应用构建器。

### Builder

```go
type Builder struct {
    // 私有字段
}
```

#### 方法

- `WithSeconds() *Builder` - 启用秒级精度
- `WithLocation(location string) *Builder` - 设置时区
- `EnableCronLogger() *Builder` - 启用调度日志
- `AddJob(spec, name string, handler func()) *Builder` - 添加简单任务
- `AddJobWithDI(spec, name string, handler any) *Builder` - 添加带 DI 的任务

---

**版本**: 1.0.0  
**更新日期**: 2025-11-11
