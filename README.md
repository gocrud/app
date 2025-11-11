# GoCRUD 应用框架 - 快速开始指南

这是一个基于依赖注入的 Go 应用程序框架，提供了数据库、缓存、定时任务等常用功能的快速集成。

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

## 💾 添加数据库支持（MySQL 示例）

### 第二步：配置数据库连接

```go
package main

import (
    "github.com/gocrud/app"
    "github.com/gocrud/app/configure"
    "github.com/gocrud/app/configure/gorm"
)

func main() {
    builder := app.NewApplicationBuilder()
    
    // 添加数据库配置
    builder.Configure(configure.Gorm(func(b *gorm.Builder) {
        b.AddDB("default", func(opts *gorm.DBOptions) {
            opts.Driver = "mysql"
            opts.DSN = "root:password@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local"
        })
    }))
    
    application := builder.Build()
    application.Run()
}
```

**DSN 格式说明：**
```
用户名:密码@tcp(IP地址:端口)/数据库名?charset=utf8mb4&parseTime=True&loc=Local
```

### 第三步：定义数据模型

```go
type User struct {
    ID        uint   `gorm:"primarykey"`
    Name      string `gorm:"size:100"`
    Email     string `gorm:"size:100;unique"`
    CreatedAt time.Time
}
```

### 第四步：创建服务并使用数据库

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/gocrud/app"
    "github.com/gocrud/app/configure"
    "github.com/gocrud/app/configure/gorm"
    "github.com/gocrud/app/di"
    gormdb "gorm.io/gorm"
)

// 数据模型
type User struct {
    ID        uint   `gorm:"primarykey"`
    Name      string `gorm:"size:100"`
    Email     string `gorm:"size:100;unique"`
    CreatedAt time.Time
}

// 用户服务
type UserService struct {
    db *gormdb.DB
}

func NewUserService(db *gormdb.DB) *UserService {
    // 自动创建表
    db.AutoMigrate(&User{})
    return &UserService{db: db}
}

func (s *UserService) CreateUser(name, email string) error {
    user := &User{Name: name, Email: email}
    return s.db.Create(user).Error
}

func (s *UserService) GetAllUsers() ([]User, error) {
    var users []User
    err := s.db.Find(&users).Error
    return users, err
}

func main() {
    builder := app.NewApplicationBuilder()
    
    // 配置数据库
    builder.Configure(configure.Gorm(func(b *gorm.Builder) {
        b.AddDB("default", func(opts *gorm.DBOptions) {
            opts.Driver = "mysql"
            opts.DSN = "root:password@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
        })
    }))
    
    // 注册服务到依赖注入容器
    builder.Services(func(provider *di.ServiceProvider) {
        provider.AddSingleton(di.ServiceDescriptor{
            Lifetime: di.Singleton,
            Provider: di.TypeOf[*UserService](),
            Factory: func(sp di.ServiceProvider) (any, error) {
                var db *gormdb.DB
                sp.GetRequiredService(&db)
                return NewUserService(db), nil
            },
        })
    })
    
    application := builder.Build()
    
    // 获取服务并使用
    var userService *UserService
    application.Services.GetRequiredService(&userService)
    
    // 创建用户
    userService.CreateUser("张三", "zhangsan@example.com")
    userService.CreateUser("李四", "lisi@example.com")
    
    // 查询所有用户
    users, _ := userService.GetAllUsers()
    for _, user := range users {
        fmt.Printf("ID: %d, 姓名: %s, 邮箱: %s\n", user.ID, user.Name, user.Email)
    }
    
    application.Run()
}
```

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
    b.AddJob("*/1 * * * *", "清理过期数据", func() {
        fmt.Println("执行清理任务...")
    })
    
    // 每天凌晨 2 点执行
    b.AddJob("0 2 * * *", "每日统计", func() {
        fmt.Println("执行每日统计...")
    })
}))
```

**Cron 表达式格式：**
```
分 时 日 月 周
*  *  *  *  *

示例：
*/5 * * * *    - 每 5 分钟
0 */2 * * *    - 每 2 小时
0 9 * * 1-5    - 工作日上午 9 点
0 0 1 * *      - 每月 1 日零点
```

---

## 🌐 完整的 Web 应用示例

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/gocrud/app"
    "github.com/gocrud/app/configure"
    "github.com/gocrud/app/configure/cron"
    "github.com/gocrud/app/configure/gorm"
    "github.com/gocrud/app/configure/redis"
    "github.com/gocrud/app/di"
    gormdb "gorm.io/gorm"
    redisclient "github.com/redis/go-redis/v9"
)

// 数据模型
type User struct {
    ID        uint      `gorm:"primarykey" json:"id"`
    Name      string    `gorm:"size:100" json:"name"`
    Email     string    `gorm:"size:100;unique" json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// 用户服务
type UserService struct {
    db    *gormdb.DB
    cache *redisclient.Client
}

func NewUserService(db *gormdb.DB, cache *redisclient.Client) *UserService {
    db.AutoMigrate(&User{})
    return &UserService{db: db, cache: cache}
}

func (s *UserService) CreateUser(name, email string) (*User, error) {
    user := &User{Name: name, Email: email}
    if err := s.db.Create(user).Error; err != nil {
        return nil, err
    }
    
    // 清除缓存
    s.cache.Del(context.Background(), "users:all")
    return user, nil
}

func (s *UserService) GetAllUsers() ([]User, error) {
    // 尝试从缓存获取
    ctx := context.Background()
    cacheKey := "users:all"
    
    var users []User
    if err := s.db.Find(&users).Error; err != nil {
        return nil, err
    }
    
    return users, nil
}

func (s *UserService) CleanupOldUsers() {
    // 删除 30 天前创建的用户
    cutoff := time.Now().AddDate(0, 0, -30)
    result := s.db.Where("created_at < ?", cutoff).Delete(&User{})
    fmt.Printf("清理了 %d 条过期用户记录\n", result.RowsAffected)
}

func main() {
    builder := app.NewApplicationBuilder()
    
    // 配置数据库
    builder.Configure(configure.Gorm(func(b *gorm.Builder) {
        b.AddDB("default", func(opts *gorm.DBOptions) {
            opts.Driver = "mysql"
            opts.DSN = "root:password@tcp(127.0.0.1:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local"
            opts.MaxIdleConns = 10
            opts.MaxOpenConns = 100
        })
    }))
    
    // 配置 Redis
    builder.Configure(redis.Configure(func(b *redis.Builder) {
        b.AddClient("default", func(opts *redis.RedisClientOptions) {
            opts.Addr = "localhost:6379"
            opts.DB = 0
        })
    }))
    
    // 注册服务
    builder.Services(func(provider *di.ServiceProvider) {
        provider.AddSingleton(di.ServiceDescriptor{
            Lifetime: di.Singleton,
            Provider: di.TypeOf[*UserService](),
            Factory: func(sp di.ServiceProvider) (any, error) {
                var db *gormdb.DB
                var cache *redisclient.Client
                sp.GetRequiredService(&db)
                sp.GetRequiredService(&cache)
                return NewUserService(db, cache), nil
            },
        })
    })
    
    // 配置定时任务
    builder.Configure(cron.Configure(func(b *cron.Builder) {
        b.AddJob("0 2 * * *", "清理过期用户", func() {
            var userService *UserService
            builder.Build().Services.GetRequiredService(&userService)
            userService.CleanupOldUsers()
        })
    }))
    
    application := builder.Build()
    
    // 使用服务
    var userService *UserService
    application.Services.GetRequiredService(&userService)
    
    // 创建测试用户
    user1, _ := userService.CreateUser("张三", "zhangsan@example.com")
    user2, _ := userService.CreateUser("李四", "lisi@example.com")
    
    fmt.Printf("创建用户: %+v\n", user1)
    fmt.Printf("创建用户: %+v\n", user2)
    
    // 查询所有用户
    users, _ := userService.GetAllUsers()
    fmt.Printf("总共有 %d 个用户\n", len(users))
    
    application.Run()
}
```

---

## 📚 更多配置选项

### 数据库驱动支持

- **MySQL**: `opts.Driver = "mysql"`
- **PostgreSQL**: `opts.Driver = "postgres"`
- **SQLite**: `opts.Driver = "sqlite"`
- **SQL Server**: `opts.Driver = "sqlserver"`

### 多数据库配置

```go
builder.Configure(configure.Gorm(func(b *gorm.Builder) {
    // 主库
    b.AddDB("default", func(opts *gorm.DBOptions) {
        opts.Driver = "mysql"
        opts.DSN = "root:password@tcp(127.0.0.1:3306)/main_db?..."
    })
    
    // 只读副本
    b.AddDB("readonly", func(opts *gorm.DBOptions) {
        opts.Driver = "mysql"
        opts.DSN = "root:password@tcp(127.0.0.1:3307)/main_db?..."
    })
}))
```

### 使用特定数据库连接

```go
import "github.com/gocrud/app/configure/gorm"

type MyService struct {
    factory *gorm.DBFactory
}

func (s *MyService) UseReadOnly() {
    readDB, _ := s.factory.Get("readonly")
    var users []User
    readDB.Find(&users)
}
```

---

## ⚙️ 依赖注入服务生命周期

```go
// Singleton - 单例，整个应用只创建一次
provider.AddSingleton(...)

// Scoped - 作用域，每个请求创建一次（适合 Web 应用）
provider.AddScoped(...)

// Transient - 瞬态，每次获取都创建新实例
provider.AddTransient(...)
```

---

## 🎯 常见问题

### 1. 数据库连接失败？
- 检查 DSN 格式是否正确
- 确认数据库服务已启动
- 检查用户名密码是否正确
- 确认数据库已创建

### 2. Redis 连接失败？
- 确认 Redis 服务已启动
- 检查地址和端口是否正确
- 如果有密码，确保设置了 `opts.Password`

### 3. 如何查看 SQL 日志？
```go
opts.LogLevel = logger.Info  // 显示所有 SQL
opts.LogLevel = logger.Warn  // 只显示警告
opts.LogLevel = logger.Error // 只显示错误
```

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
