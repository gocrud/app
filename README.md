# GoCRUD 应用框架 - 快速开始指南

这是一个基于依赖注入的 Go 应用程序框架，提供了缓存、定时任务等常用功能的快速集成。

## 📦 安装

```bash
go get github.com/gocrud/app
```

## 🚀 快速上手

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

### 配置 Redis

```go
import (
    "github.com/gocrud/app/configure/redis"
    redisclient "github.com/redis/go-redis/v9"
)

// 在 main 函数中配置 Redis
builder.Configure(redis.Configure(func(b *redis.Builder) {
    b.AddClient("default", func(opts *redis.RedisClientOptions) {
        opts.Addr = "localhost:6379"
        opts.Password = ""  // 如果有密码就填写
        opts.DB = 0
    })
}))
```

### 创建缓存服务

```go
import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    redisclient "github.com/redis/go-redis/v9"
)

// CacheService - 通用缓存服务
type CacheService struct {
    redis *redisclient.Client  // 框架会自动注入
}

// 构造函数（框架会自动调用并注入依赖）
func NewCacheService(redis *redisclient.Client) *CacheService {
    return &CacheService{redis: redis}
}

// ... 实现 Set/Get 方法 ...
```

### 业务服务示例

```go
// UserService - 使用缓存的用户服务
type UserService struct {
    cache *CacheService  // 依赖缓存服务
}

// 构造函数（框架会自动注入 CacheService）
func NewUserService(cache *CacheService) *UserService {
    return &UserService{cache: cache}
}
```

### 注册和使用服务

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/gocrud/app"
    "github.com/gocrud/app/configure/redis"
    "github.com/gocrud/app/core"
    "github.com/gocrud/app/di"
)

func main() {
    builder := app.NewApplicationBuilder()
    
    // 1. 配置 Redis
    builder.Configure(redis.Configure(func(b *redis.Builder) {
        b.AddClient("default", func(opts *redis.RedisClientOptions) {
            opts.Addr = "localhost:6379"
        })
    }))
    
    // 2. 注册服务（使用泛型 API）
    builder.ConfigureServices(func(services *core.ServiceCollection) {
        // 注册具体服务 (默认单例)
        core.AddSingleton[*CacheService](services, di.WithFactory(NewCacheService))
        core.AddSingleton[*UserService](services, di.WithFactory(NewUserService))
        
        // 如果需要绑定接口:
        // core.AddSingleton[IUserService](services, di.Use[*UserService]())
    })
    
    application := builder.Build()
    
    // 3. 获取并使用服务
    var userService *UserService
    application.GetService(&userService)
    
    // 或者直接从容器获取
    // userService := di.MustResolve[*UserService](application.Services())
    
    application.Run()
}
```

### 依赖注入说明

框架会自动处理依赖注入：

1. **注册**: 使用 `core.AddSingleton[T]` 或 `di.Register[T]` 注册服务。
2. **注入**: 构造函数参数会自动从容器中解析并注入。
3. **获取**: 使用 `application.GetService(&ptr)` 或 `di.Resolve[T](container)` 获取实例。

**关键点：**
- ✅ **泛型优先**：注册和获取时使用泛型 `[T]` 指定类型。
- ✅ **自动注入**：构造函数参数按类型自动匹配。
- ✅ **生命周期**：支持 Singleton (单例)、Transient (瞬态)、Scoped (作用域)。

---

## ⏰ 添加定时任务

```go
import (
    "github.com/gocrud/app/configure/cron"
)

builder.Configure(cron.Configure(func(b *cron.Builder) {
    // 支持依赖注入的任务
    b.AddJobWithDI("0 */1 * * * *", "清理任务", func(svc *UserService) {
        svc.Cleanup()
    })
}))
```

---

## ⚙️ 配置文件系统

（此处保留原有配置文档，配置系统 API 未发生重大破坏性变更）

### 完整配置示例

（保留...）

##  依赖注入与服务获取

### 获取服务实例

框架提供了两种方式来获取已注册的服务：

#### 1. 通过 Application 获取

```go
application := builder.Build()

var myService *MyService
application.GetService(&myService) // 必须传递指针的地址
```

#### 2. 通过容器直接解析 (推荐)

使用新的泛型 API，更加安全简便：

```go
container := application.Services()

// 安全获取 (返回 error)
svc, err := di.Resolve[*MyService](container)

// 强制获取 (失败 Panic)
svc = di.MustResolve[*MyService](container)
```

### 服务生命周期注册

```go
builder.ConfigureServices(func(s *core.ServiceCollection) {
    // Singleton - 单例
    core.AddSingleton[*MyService](s) 
    
    // Scoped - 作用域
    core.AddScoped[*RequestService](s)
    
    // Transient - 瞬态
    core.AddTransient[*TempService](s)
})
```

### 注意事项

- ⚠️ **泛型类型匹配**：注册时的 `[T]` 必须与构造函数返回类型或字段类型严格匹配（包括指针 `*`）。
- ⚠️ **指针注入**：使用 `GetService` 时必须传递指针的地址 `&svc`。

---

## 📖 详细文档

- [DI 框架详细文档](di/README.md)
- [Cron 配置模块详细文档](configure/cron/README.md)
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
