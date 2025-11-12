# GoCRUD 应用框架 - 快速开始指南

这是一个基于依赖注入的 Go 应用程序框架，提供了缓存、定时任务等常用功能的快速集成。

## 📦 安装

```bash
go get github.com/gocrud/app
```

## 🚀 5 分钟快速上手

### 第一步：创建最简单的应用

```go
package main

import "github.com/gocrud/app"

func main() {
    builder := app.NewApplicationBuilder()
    application := builder.Build()
    application.Run()
}
```

运行：
```bash
go run main.go
```

恭喜！你已经创建了第一个 GoCRUD 应用。

---

## 🔴 添加 Redis 缓存

```go
import (
    "github.com/gocrud/app/configure/redis"
    redisclient "github.com/redis/go-redis/v9"
)

// 在 main 函数中添加 Redis 配置
builder.Configure(redis.Configure(func(b *redis.Builder) {
    b.AddClient("default", func(opts *redis.RedisClientOptions) {
        opts.Addr = "localhost:6379"
        opts.Password = ""
        opts.DB = 0
    })
}))

// 在服务中使用 Redis
type CacheService struct {
    redis *redisclient.Client
}

func NewCacheService(redis *redisclient.Client) *CacheService {
    return &CacheService{redis: redis}
}

func (s *CacheService) Set(ctx context.Context, key, value string) error {
    return s.redis.Set(ctx, key, value, 0).Err()
}

func (s *CacheService) Get(ctx context.Context, key string) (string, error) {
    return s.redis.Get(ctx, key).Result()
}
```

---

## ⏰ 添加定时任务

```go
import (
    "github.com/gocrud/app/configure/cron"
)

builder.Configure(cron.Configure(func(b *cron.Builder) {
    // 每分钟执行一次
    b.AddJob("0 */1 * * * *", "清理过期数据", func() {
        fmt.Println("执行清理任务...")
    })
    
    // 每天凌晨 2 点执行
    b.AddJob("0 0 2 * * *", "每日统计", func() {
        fmt.Println("执行每日统计...")
    })
}))
```

**Cron 表达式格式（秒级精度 - 6 位）：**
```
秒 分 时 日 月 周
*  *  *  *  *  *

字段说明：
- 秒：0-59
- 分：0-59
- 时：0-23
- 日：1-31
- 月：1-12
- 周：0-6 (0=周日)

示例：
0 */5 * * * *      - 每 5 分钟
0 0 */2 * * *      - 每 2 小时
0 0 9 * * 1-5      - 工作日上午 9 点
0 0 0 1 * *        - 每月 1 日零点
*/10 * * * * *     - 每 10 秒
30 30 14 * * *     - 每天 14:30:30
0 0 0 * * 0        - 每周日零点
```

---

##  依赖注入与服务获取

### 获取服务实例

框架提供了两种方式来获取已注册的服务：

#### 1. 通过 Application 获取（推荐）

```go
// 在应用启动后获取服务
application := builder.Build()

var myService *MyService
application.GetService(&myService)

// 使用服务
myService.DoSomething()
```

#### 2. 通过容器直接注入

```go
// 在 ConfigureServices 或其他地方
container := application.Services()

var myService *MyService
container.Inject(&myService)
```

### 服务生命周期

```go
// Singleton - 单例，整个应用只创建一次
services.AddSingleton(NewMyService)

// Scoped - 作用域，每个作用域创建一次
services.AddScoped(NewRequestService)

// Transient - 瞬态，每次获取都创建新实例
services.AddTransient(NewTempService)
```

### 注意事项

- ⚠️ **必须传递指针的地址**：使用 `&variable`，不是 `variable`
- ⚠️ **变量必须声明为指针类型**：`var svc *MyService`，不是 `var svc MyService`
- ⚠️ **失败时会 panic**：如果服务未注册或注入失败，程序会立即 panic

### 正确示例 ✅

```go
var myService *MyService    // 声明为指针类型
application.GetService(&myService)  // 传递地址
```

### 错误示例 ❌

```go
var myService MyService     // ❌ 不是指针类型
application.GetService(&myService)

var myService *MyService    
application.GetService(myService)  // ❌ 没有传递地址
```

---

## 🎯 常见问题

### 1. Redis 连接失败？
- 确认 Redis 服务已启动
- 检查地址和端口是否正确
- 如果有密码，确保设置了 `opts.Password`

### 2. 服务注入失败？
- 确保服务已通过 `ConfigureServices` 注册
- 检查变量是否声明为指针类型
- 确保调用 `GetService` 时传递的是地址（`&variable`）

---

## 📖 详细文档

- [CRON 配置模块详细文档](configure/cron/README.md)
- [Redis 配置模块详细文档](configure/redis/README.md)
- [ETCD 配置模块详细文档](configure/etcd/README.md)

---

## 💡 下一步

- 添加 Web 路由和控制器
- 实现业务逻辑
- 添加中间件
- 配置日志
- 部署到生产环境

现在您已经掌握了基础用法，可以开始构建自己的应用了！
