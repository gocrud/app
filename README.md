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
)

// CacheService - 通用缓存服务
type CacheService struct {
    redis *redisclient.Client  // 框架会自动注入
}

// 构造函数（框架会自动调用并注入依赖）
func NewCacheService(redis *redisclient.Client) *CacheService {
    return &CacheService{redis: redis}
}

// Set 设置缓存
func (s *CacheService) Set(ctx context.Context, key, value string, expiration time.Duration) error {
    return s.redis.Set(ctx, key, value, expiration).Err()
}

// Get 获取缓存
func (s *CacheService) Get(ctx context.Context, key string) (string, error) {
    return s.redis.Get(ctx, key).Result()
}

// Delete 删除缓存
func (s *CacheService) Delete(ctx context.Context, key string) error {
    return s.redis.Del(ctx, key).Err()
}
```

### 业务服务示例

```go
// User 用户模型
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// UserService - 使用缓存的用户服务
type UserService struct {
    cache *CacheService  // 依赖缓存服务
}

// 构造函数（框架会自动注入 CacheService）
func NewUserService(cache *CacheService) *UserService {
    return &UserService{cache: cache}
}

// CacheUser 缓存用户数据
func (s *UserService) CacheUser(ctx context.Context, user *User) error {
    data, err := json.Marshal(user)
    if err != nil {
        return err
    }
    
    key := fmt.Sprintf("user:%d", user.ID)
    return s.cache.Set(ctx, key, string(data), time.Hour)  // 缓存 1 小时
}

// GetCachedUser 从缓存获取用户
func (s *UserService) GetCachedUser(ctx context.Context, userID int) (*User, error) {
    key := fmt.Sprintf("user:%d", userID)
    data, err := s.cache.Get(ctx, key)
    if err != nil {
        return nil, err
    }
    
    var user User
    if err := json.Unmarshal([]byte(data), &user); err != nil {
        return nil, err
    }
    return &user, nil
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
)

func main() {
    builder := app.NewApplicationBuilder()
    
    // 1. 配置 Redis
    builder.Configure(redis.Configure(func(b *redis.Builder) {
        b.AddClient("default", func(opts *redis.RedisClientOptions) {
            opts.Addr = "localhost:6379"
        })
    }))
    
    // 2. 注册服务（框架会自动处理依赖注入）
    builder.ConfigureServices(func(services *core.ServiceCollection) {
        services.AddSingleton(NewCacheService)  // 注册缓存服务
        services.AddSingleton(NewUserService)   // 注册用户服务（依赖 CacheService）
    })
    
    application := builder.Build()
    
    // 3. 获取并使用服务
    var userService *UserService
    application.GetService(&userService)
    
    ctx := context.Background()
    
    // 缓存用户
    user := &User{ID: 1, Name: "Alice", Email: "alice@example.com"}
    userService.CacheUser(ctx, user)
    
    // 从缓存获取
    cachedUser, _ := userService.GetCachedUser(ctx, 1)
    fmt.Printf("从缓存获取: %+v\n", cachedUser)
    
    application.Run()
}
```

### 依赖注入说明

框架会自动处理依赖注入：

```
1. Redis 客户端由框架创建并注册到容器
         ↓
2. NewCacheService(redis) 被调用，框架自动注入 redis 参数
         ↓
3. NewUserService(cache) 被调用，框架自动注入 cache 参数
         ↓
4. UserService 可以直接使用 CacheService
```

**关键点：**
- ✅ 构造函数参数会被自动注入（按类型匹配）
- ✅ 注册顺序无关紧要，框架会自动解析依赖关系
- ✅ 使用 `AddSingleton` 注册单例服务（整个应用共享一个实例）

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

框架提供了强大的配置系统，支持多种配置源和三种配置模式，支持配置热更新和动态重载。

### 目录
- [配置源](#配置源)
- [三种配置模式](#三种配置模式)
- [配置监听与热更新](#配置监听与热更新)
- [配置键路径](#配置键路径)
- [配置模式选择指南](#配置模式选择指南)
- [完整配置示例](#完整配置示例)
- [最佳实践](#配置最佳实践)

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

### 配置监听与热更新

#### 动态配置更新机制

框架支持配置的动态更新，当配置源发生变化时自动重载配置。目前支持动态更新的配置源：

- ✅ **Etcd** - 通过 Watch 机制实时监听配置变更
- ❌ **JSON/YAML 文件** - 不支持文件监听（静态配置）
- ❌ **环境变量** - 不支持动态更新（静态配置）
- ❌ **内存配置** - 不支持动态更新（静态配置）

#### 配置监听开关

框架提供了全局配置监听开关，可以根据环境需求启用或禁用配置监听功能。

**方式一：代码配置（推荐）**

```go
builder := app.NewApplicationBuilder()

// 禁用配置监听（适合生产环境）
builder.UseConfigWatch(false)

// 启用配置监听（默认行为）
builder.UseConfigWatch(true)
```

**方式二：环境变量配置**

```bash
# 禁用配置监听
export APP_CONFIG_WATCH_ENABLED=false

# 启用配置监听（默认）
export APP_CONFIG_WATCH_ENABLED=true
```

环境变量优先级高于代码配置。

#### 配置更新流程

当 Etcd 配置发生变更时：

```
1. Etcd Watch 检测到变更
         ↓
2. 触发配置重载 (ReloadableConfiguration.Reload)
         ↓
3. 更新所有 OptionsCache 缓存
         ↓
4. OptionMonitor.Value() 返回最新值
```

#### Etcd 配置示例

```go
// 在 Etcd 中存储配置（键格式：/prefix/path/to/key）
// /myapp/features/enableNewUI = true
// /myapp/features/maxConnections = 100

builder.ConfigureConfiguration(func(config *config.ConfigurationBuilder) {
    config.AddEtcd(config.EtcdOptions{
        Endpoints: []string{"localhost:2379"},
        Prefix:    "/myapp/",  // 配置前缀
    })
})

// 使用 OptionMonitor 实时获取最新配置
type FeatureSettings struct {
    EnableNewUI    bool `json:"enableNewUI"`
    MaxConnections int  `json:"maxConnections"`
}

core.AddOptions[FeatureSettings](builder, "features")

// 服务中使用
type FeatureService struct {
    features config.OptionMonitor[FeatureSettings]
}

func (s *FeatureService) Check() {
    // 总是返回最新配置，即使 Etcd 中的值已更改
    cfg := s.features.Value()
    fmt.Printf("New UI: %v, Max: %d\n", cfg.EnableNewUI, cfg.MaxConnections)
}
```

#### 配置监听注意事项

⚠️ **重要提示：**

1. **只有 `OptionMonitor[T]` 会实时更新**
   - `Option[T]` 和 `OptionSnapshot[T]` 不会自动更新
   
2. **禁用监听后的行为**
   - 应用启动时加载配置（一次性）
   - 不会监听配置变更
   - 可以手动调用 `Reload()` 方法更新（如果需要）

3. **性能考虑**
   - 启用监听会维持与 Etcd 的长连接
   - 每个配置源一个 Watch 连接
   - 配置变更时会触发全量重载

4. **线程安全**
   - 所有配置读写都使用读写锁保护
   - 多个 goroutine 可以安全并发读取
   - 配置更新时会短暂阻塞读取

### 配置键路径

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

// 获取配置节
section := config.GetSection("database")
host := section.Get("host")

// 绑定到结构体
var dbConfig DatabaseSettings
config.Bind("database", &dbConfig)
```

**路径映射规则：**

| Etcd 键 | 配置路径 | JSON 路径 |
|---------|---------|-----------|
| `/myapp/app/name` | `app:name` | `app.name` |
| `/myapp/db/host` | `db:host` | `db.host` |
| `APP_DB_HOST` (环境变量) | `db:host` | - |

### 配置模式选择指南

| 模式 | 生命周期 | 更新频率 | 适用场景 |
|------|---------|---------|---------|
| **Option[T]** | Singleton | 启动时一次 | 应用名称、端口等静态配置 |
| **OptionSnapshot[T]** | Scoped | 每个作用域 | 请求级别的配置快照 |
| **OptionMonitor[T]** | Singleton | 实时更新 | 功能开关、动态限流等 |

**选择建议：**

```go
// ✅ 使用 Option[T]：配置永不改变
type ServerConfig struct {
    Port int    `json:"port"`
    Host string `json:"host"`
}

// ✅ 使用 OptionSnapshot[T]：请求级配置隔离
type RequestConfig struct {
    Timeout  time.Duration `json:"timeout"`
    MaxRetry int           `json:"maxRetry"`
}

// ✅ 使用 OptionMonitor[T]：需要动态更新
type FeatureFlags struct {
    EnableBetaFeature bool `json:"enableBetaFeature"`
    RateLimit         int  `json:"rateLimit"`
}
```

### 配置最佳实践

#### 1. 配置分层策略

```go
builder.ConfigureConfiguration(func(cfg *config.ConfigurationBuilder) {
    // 基础配置（默认值）
    cfg.AddJsonFile("appsettings.json")
    
    // 环境特定配置（覆盖默认值）
    cfg.AddJsonFile("appsettings.dev.json", true)
    cfg.AddJsonFile("appsettings.prod.json", true)
    
    // 环境变量（最高优先级）
    cfg.AddEnvironmentVariables("APP_")
    
    // 配置中心（动态配置）
    cfg.AddEtcd(config.EtcdOptions{
        Endpoints: []string{"localhost:2379"},
        Prefix:    "/myapp/",
    })
})
```

#### 2. 敏感信息处理

```go
// ❌ 不要在代码中硬编码敏感信息
type DatabaseConfig struct {
    Password string `json:"password"` // 不要写在 JSON 文件中
}

// ✅ 使用环境变量
// export APP_DATABASE_PASSWORD=secret123

// ✅ 或使用 Etcd 加密存储
// etcdctl put /myapp/database/password "secret123"
```

#### 3. 配置验证

```go
type AppSettings struct {
    Port int    `json:"port"`
    Host string `json:"host"`
}

func (s *AppSettings) Validate() error {
    if s.Port < 1 || s.Port > 65535 {
        return fmt.Errorf("invalid port: %d", s.Port)
    }
    if s.Host == "" {
        return fmt.Errorf("host is required")
    }
    return nil
}

// 在应用启动时验证
var settings config.Option[AppSettings]
application.GetService(&settings)
if err := settings.Value().Validate(); err != nil {
    panic(err)
}
```

#### 4. 配置变更监控

```go
// 自定义配置变更处理
type ConfigWatcher struct {
    features config.OptionMonitor[FeatureSettings]
    logger   logging.Logger
}

func (w *ConfigWatcher) StartMonitoring(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    lastConfig := w.features.Value()
    
    for {
        select {
        case <-ticker.C:
            currentConfig := w.features.Value()
            if currentConfig != lastConfig {
                w.logger.Info("Configuration changed",
                    logging.Field{Key: "old", Value: lastConfig},
                    logging.Field{Key: "new", Value: currentConfig})
                lastConfig = currentConfig
            }
        case <-ctx.Done():
            return
        }
    }
}
```

#### 5. 多环境配置

**方案一：文件后缀**
```
appsettings.json          # 默认配置
appsettings.dev.json      # 开发环境
appsettings.staging.json  # 预发布环境
appsettings.prod.json     # 生产环境
```

```go
env := os.Getenv("APP_ENV") // dev, staging, prod
if env == "" {
    env = "dev"
}

builder.ConfigureConfiguration(func(cfg *config.ConfigurationBuilder) {
    cfg.AddJsonFile("appsettings.json")
    cfg.AddJsonFile(fmt.Sprintf("appsettings.%s.json", env), true)
})
```

**方案二：Etcd 前缀**
```
/myapp/dev/...      # 开发环境配置
/myapp/staging/...  # 预发布环境配置
/myapp/prod/...     # 生产环境配置
```

```go
env := os.Getenv("APP_ENV")
builder.ConfigureConfiguration(func(cfg *config.ConfigurationBuilder) {
    cfg.AddEtcd(config.EtcdOptions{
        Endpoints: []string{"localhost:2379"},
        Prefix:    fmt.Sprintf("/myapp/%s/", env),
    })
})
```

#### 6. 配置性能优化

```go
// ❌ 避免在热路径频繁调用 Value()
func (h *RequestHandler) Process() {
    for i := 0; i < 1000000; i++ {
        cfg := h.monitor.Value() // 每次都重新获取，性能差
    }
}

// ✅ 在循环外获取一次
func (h *RequestHandler) Process() {
    cfg := h.monitor.Value()
    for i := 0; i < 1000000; i++ {
        // 使用 cfg
    }
}
```

#### 7. 配置调试技巧

```go
// 打印所有配置（调试用）
application := builder.Build()
config := application.Configuration()

// 获取所有配置
allConfig := config.GetAll()
fmt.Printf("All Config: %+v\n", allConfig)

// 检查特定配置是否存在
if val := config.Get("app:debug"); val == "" {
    fmt.Println("Warning: app:debug not configured")
}
```

### 配置常见问题

**Q1: 配置更新后为什么服务没有生效？**

A: 检查是否使用了正确的配置模式：
- `Option[T]` - 不会更新，只在启动时加载一次 ❌
- `OptionSnapshot[T]` - 只在作用域创建时更新 ⚠️
- `OptionMonitor[T]` - 实时更新 ✅

**Q2: 如何在不重启应用的情况下更新配置？**

A: 使用 Etcd + OptionMonitor：
```go
// 1. 启用配置监听
builder.UseConfigWatch(true)

// 2. 使用 Etcd 配置源
builder.ConfigureConfiguration(func(cfg *config.ConfigurationBuilder) {
    cfg.AddEtcd(config.EtcdOptions{...})
})

// 3. 使用 OptionMonitor
core.AddOptions[MySettings](builder, "mysettings")

// 4. 更新 Etcd 中的配置值
// etcdctl put /myapp/mysettings/key "newvalue"
```

**Q3: 配置监听会影响性能吗？**

A: 影响很小：
- 只在配置变更时触发重载
- 读取操作使用读写锁，并发读不阻塞
- 如果担心性能，可以禁用监听并使用静态配置

**Q4: 如何处理配置加载失败？**

A: 框架会在启动时 panic，建议：
```go
// 使用可选配置文件
cfg.AddJsonFile("optional.json", true) // 第二个参数表示可选

// 或提供默认值
type AppSettings struct {
    Port int `json:"port"` // 如果未配置，将使用零值
}
```

**Q5: 配置文件支持注释吗？**

A: 
- JSON 不支持注释（标准限制）
- YAML 支持 `#` 注释 ✅
- 建议使用 YAML 或在配置结构体中添加文档注释

### 完整配置示例

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
