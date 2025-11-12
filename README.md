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

## ⚙️ 配置文件系统

框架提供了强大的配置系统，支持多种配置源和三种配置模式。

### 配置源

支持以下配置源，按加载顺序后面的会覆盖前面的：

#### 1. JSON 文件
```go
builder.ConfigureConfiguration(func(config *config.ConfigurationBuilder) {
    config.AddJsonFile("appsettings.json")         // 必需的配置文件
    config.AddJsonFile("appsettings.dev.json", true) // 可选的配置文件
})
```

**appsettings.json 示例：**
```json
{
  "app": {
    "name": "MyApp",
    "port": 8080,
    "debug": true
  },
  "database": {
    "host": "localhost",
    "port": 5432,
    "name": "mydb"
  }
}
```

#### 2. YAML 文件
```go
builder.ConfigureConfiguration(func(config *config.ConfigurationBuilder) {
    config.AddYamlFile("config.yaml")
    config.AddYamlFile("config.dev.yaml", true)
})
```

**config.yaml 示例：**
```yaml
app:
  name: MyApp
  port: 8080
  debug: true

database:
  host: localhost
  port: 5432
  name: mydb
```

#### 3. 环境变量
```go
builder.ConfigureConfiguration(func(config *config.ConfigurationBuilder) {
    // 使用前缀 APP_ 的环境变量
    // 例如: APP_DATABASE_HOST -> database:host
    config.AddEnvironmentVariables("APP_")
})
```

#### 4. 内存配置
```go
builder.ConfigureConfiguration(func(config *config.ConfigurationBuilder) {
    config.AddInMemory(map[string]any{
        "app": map[string]any{
            "name": "MyApp",
            "port": 8080,
        },
    })
})
```

#### 5. Etcd 配置中心（支持动态更新）
```go
builder.ConfigureConfiguration(func(config *config.ConfigurationBuilder) {
    config.AddEtcd(config.EtcdOptions{
        Endpoints: []string{"localhost:2379"},
        Prefix:    "/myapp/",
        Username:  "admin",    // 可选
        Password:  "password", // 可选
    })
})
```

### 三种配置模式

#### 1. Option[T] - 静态配置（应用生命周期内不变）

适用场景：应用启动时加载一次，之后不会改变的配置。

```go
// 定义配置结构
type AppSettings struct {
    Name  string `json:"name"`
    Port  int    `json:"port"`
    Debug bool   `json:"debug"`
}

// 注册配置选项
core.AddOptions[AppSettings](builder, "app")

// 在服务中使用
type MyService struct {
    settings config.Option[AppSettings]
}

func NewMyService(settings config.Option[AppSettings]) *MyService {
    return &MyService{settings: settings}
}

func (s *MyService) PrintConfig() {
    cfg := s.settings.Value()
    fmt.Printf("App: %s, Port: %d\n", cfg.Name, cfg.Port)
}
```

#### 2. OptionSnapshot[T] - 快照配置（作用域内不变）

适用场景：每个请求/作用域使用配置快照，同一作用域内保持一致。

```go
// 定义配置结构
type DatabaseSettings struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Database string `json:"database"`
}

// 注册配置选项
core.AddOptions[DatabaseSettings](builder, "database")

// 在 Scoped 服务中使用
type RequestHandler struct {
    dbConfig config.OptionSnapshot[DatabaseSettings]
}

func NewRequestHandler(dbConfig config.OptionSnapshot[DatabaseSettings]) *RequestHandler {
    return &RequestHandler{dbConfig: dbConfig}
}

func (h *RequestHandler) Process() {
    cfg := h.dbConfig.Value()
    // 同一请求中多次调用 Value() 返回相同的快照
    fmt.Printf("DB: %s:%d/%s\n", cfg.Host, cfg.Port, cfg.Database)
}
```

#### 3. OptionMonitor[T] - 监听配置（实时更新）

适用场景：配置可能动态更新，需要实时获取最新值（如从 Etcd 加载）。

```go
// 定义配置结构
type FeatureSettings struct {
    EnableNewUI    bool `json:"enableNewUI"`
    MaxConnections int  `json:"maxConnections"`
}

// 注册配置选项
core.AddOptions[FeatureSettings](builder, "features")

// 在服务中使用（通常是 Singleton）
type FeatureService struct {
    features config.OptionMonitor[FeatureSettings]
}

func NewFeatureService(features config.OptionMonitor[FeatureSettings]) *FeatureService {
    return &FeatureService{features: features}
}

func (s *FeatureService) IsNewUIEnabled() bool {
    // 总是返回最新的配置值
    return s.features.Value().EnableNewUI
}
```

### 完整配置示例

```go
package main

import (
    "github.com/gocrud/app"
    "github.com/gocrud/app/config"
    "github.com/gocrud/app/core"
)

type AppSettings struct {
    Name  string `json:"name"`
    Port  int    `json:"port"`
    Debug bool   `json:"debug"`
}

type DatabaseSettings struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}

func main() {
    builder := app.NewApplicationBuilder()
    
    // 配置多个配置源
    builder.ConfigureConfiguration(func(cfg *config.ConfigurationBuilder) {
        cfg.AddJsonFile("appsettings.json")
        cfg.AddJsonFile("appsettings.dev.json", true)
        cfg.AddEnvironmentVariables("APP_")
    })
    
    // 注册配置选项
    core.AddOptions[AppSettings](builder, "app")
    core.AddOptions[DatabaseSettings](builder, "database")
    
    // 注册服务
    builder.ConfigureServices(func(services *core.ServiceCollection) {
        services.AddSingleton(NewMyService)
    })
    
    application := builder.Build()
    application.Run()
}
```

### 配置键路径

支持 `:` 或 `.` 作为分隔符访问嵌套配置：

```go
// 直接访问配置值
config.Get("app:name")        // 或 "app.name"
config.Get("database:host")   // 或 "database.host"

// 获取整数
port, _ := config.GetInt("app:port")

// 获取布尔值
debug, _ := config.GetBool("app:debug")
```

### 配置模式选择指南

| 模式 | 生命周期 | 更新频率 | 适用场景 |
|------|---------|---------|---------|
| **Option[T]** | Singleton | 启动时一次 | 应用名称、端口等静态配置 |
| **OptionSnapshot[T]** | Scoped | 每个作用域 | 请求级别的配置快照 |
| **OptionMonitor[T]** | Singleton | 实时更新 | 功能开关、动态限流等 |

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

- [配置系统 (Configuration)](#️-配置文件系统)
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
