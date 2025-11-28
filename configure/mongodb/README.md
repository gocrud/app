# MongoDB Configuration

`configure/mongodb` 模块为应用程序提供 MongoDB 客户端的自动配置和依赖注入支持，基于 `github.com/gocrud/mgo` 库实现。

## 特性

- **多实例支持**：可以配置多个独立的 MongoDB 连接（如主库、日志库）。
- **依赖注入集成**：自动将 `*mgo.Client` 注册到 DI 容器。
- **生命周期管理**：应用退出时自动优雅关闭所有连接。
- **灵活配置**：支持 URI、认证、连接池、超时等详细配置。

## 安装

模块通常已内置在项目中，其核心依赖为：

```bash
go get github.com/gocrud/mgo
```

## 快速开始

在应用启动代码（如 `startup.go` 或 `main.go`）中配置 MongoDB：

```go
import "github.com/gocrud/app/configure/mongodb"

// ...

app.NewApplication(
    // ...
    mongodb.Configure(func(b *mongodb.Builder) {
        // 添加默认数据库（最简配置）
        b.Add("default", "mongodb://localhost:27017/mydb", nil)
    }),
)
```

在服务中使用：

```go
type UserService struct {
    // 自动注入名为 "default" 的客户端
    Client *mgo.Client `di:""`
}

func (s *UserService) FindUser(ctx context.Context, id string) (*User, error) {
    var user User
    // 使用 mgo 的流式 API
    err := s.Client.Database("mydb").Collection("users").
        Query(ctx).
        Eq("_id", id).
        One(&user)
    return &user, err
}
```

## 高级配置

### 配置选项

`Add` 方法的第三个参数允许通过 `MongoOptions` 进行详细配置：

```go
mongodb.Configure(func(b *mongodb.Builder) {
    b.Add("default", "mongodb://192.168.1.100:27017/prod_db", func(o *mongodb.MongoOptions) {
        // 认证信息
        o.Username = "admin"
        o.Password = "secret_password"
        
        // 连接池设置
        o.MaxPoolSize = 100
        o.MinPoolSize = 10
        
        // 连接超时
        o.Timeout = 5 * time.Second
    })
})
```

### 多数据库实例

你可以注册多个不同用途的数据库连接：

```go
mongodb.Configure(func(b *mongodb.Builder) {
    // 1. 主业务库 (default)
    b.Add("default", "mongodb://localhost:27017/main_db", nil)
    
    // 2. 日志归档库 (logs)
    b.Add("logs", "mongodb://log-server:27017/app_logs", nil)
})
```

在服务中注入特定实例：

```go
type LogService struct {
    // 注入名为 "logs" 的客户端
    LogDb *mgo.Client `di:"logs"`
    
    // 注入默认客户端
    MainDb *mgo.Client `di:""`
}
```

## 依赖注入说明

该模块会在 DI 容器中注册以下服务：

| 类型 | 名称 (Name) | 说明 |
|------|-------------|------|
| `*mgo.Client` | `""` (空) | **默认客户端**。只有当注册名称为 `"default"` 时才会注册此项。 |
| `*mgo.Client` | `"<name>"` | **命名客户端**。对应 `Add` 方法中指定的名称。 |
| `*mongodb.MongoFactory` | `""` | **客户端工厂**。用于高级场景，如动态获取客户端。 |

## 注意事项

1. **URI 格式**：URI 必须遵循 MongoDB Connection String 标准格式。
2. **关闭连接**：框架会在应用停止 (`Stop`) 时自动调用 `Disconnect`，无需手动关闭。
3. **mgo 版本**：本模块依赖 `github.com/gocrud/mgo`，它封装了官方驱动 `go.mongodb.org/mongo-driver/v2`，提供了更友好的流式 API。
