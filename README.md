# Gocrud App Framework

**app** 是一个现代化、模块化、高性能的 Go 语言应用程序框架，专为构建可扩展的后端服务而设计。它深受 .NET Core 架构的启发，提供了一套优雅的依赖注入（DI）、配置管理、日志记录和托管服务生命周期管理机制。

## ✨ 核心特性

*   **🏗️ 模块化架构**: 采用 `ApplicationBuilder` 模式，通过 `Extension` 机制轻松扩展功能。
*   **💉 依赖注入**: 内置强大的泛型 DI 容器，支持构造函数自动注入、属性注入，支持 `Singleton`, `Scoped`, `Transient` 生命周期。
*   **⚙️ 配置系统**: 支持 JSON, YAML, 环境变量, 命令行参数等多种配置源，支持热重载（Reloadable）和选项模式（Options Pattern）。
*   **📝 结构化日志**: 内置高性能结构化日志，支持 Log Level 控制、异步写入和多种输出格式。
*   **🔄 托管服务**: 提供 `HostedService` 接口，轻松管理后台任务（Worker）、定时任务（Cron）和 Web 服务器的生命周期（启动/优雅停止）。
*   **🔌 扩展生态**: 内置 Redis, Etcd, Cron, Web (Gin) 等常用组件的扩展支持。

## 📦 安装

```bash
go get github.com/gocrud/app
```

## 🚀 快速开始

### 1. 创建最简单的应用

```go
package main

import "github.com/gocrud/app"

func main() {
    // 1. 创建构建器
    builder := app.NewApplicationBuilder()
    
    // 2. 注册简单的后台任务
    builder.AddTask(func(ctx context.Context) error {
        println("Hello, App Framework!")
        return nil
    })

    // 3. 构建并运行
    app := builder.Build()
    app.Run() 
}
```

### 2. 模块化开发 (推荐)

使用 `Extension` 机制来组织您的业务代码。

```go
// modules/user/module.go
type UserModule struct {}

func (m *UserModule) Name() string { return "UserModule" }

// 注册服务 (DI)
func (m *UserModule) ConfigureServices(services *core.ServiceCollection) {
    core.AddScoped[IUserService](services, di.Use[*UserService]())
    core.AddSingleton[*UserRepository](services)
}

// 配置应用 (Context)
func (m *UserModule) ConfigureBuilder(ctx *core.BuildContext) {
    // 绑定配置
    core.ConfigureOptions[UserOptions](ctx, "users")
    
    // 注册后台清理任务
    ctx.AddHostedService(NewUserCleanupWorker())
}

// main.go
func main() {
    app.NewApplicationBuilder().
        AddExtension(&UserModule{}). // 注册业务模块
        Build().
        Run()
}
```

## 💡 核心功能详解

### 依赖注入 (Dependency Injection)

框架核心基于 `di` 包，支持完全的泛型操作。

```go
builder.ConfigureServices(func(s *core.ServiceCollection) {
    // 注册单例
    core.AddSingleton[*RedisCache](s)
    
    // 注册接口实现
    core.AddScoped[IUserService](s, di.Use[*UserService]())
    
    // 注册工厂方法
    core.AddTransient[*OrderService](s, di.WithFactory(func(cache *RedisCache) *OrderService {
        return NewOrderService(cache)
    }))
})
```

### 配置系统 (Configuration)

支持多层级配置覆盖：`appsettings.json` < `Environment Variables` < `Command Line Args`。

**配置文件 (config.yaml):**
```yaml
app:
  name: "MyApp"
  port: 8080
redis:
  host: "localhost"
```

**使用 Options 模式:**
```go
type AppSettings struct {
    Name string `json:"name"`
    Port int    `json:"port"`
}

// 注册
core.AddOptions[AppSettings](builder, "app")

// 使用 (注入 IOptions[T])
type Server struct {
    options config.Option[AppSettings]
}

func NewServer(opts config.Option[AppSettings]) *Server {
    fmt.Println(opts.Value.Name) // "MyApp"
    return &Server{options: opts}
}
```

### 托管服务 (Hosted Services)

实现 `HostedService` 接口来创建随应用启动和停止的后台服务。

```go
type MyWorker struct {}

func (w *MyWorker) Start(ctx context.Context) error {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                // Do work...
                time.Sleep(1 * time.Second)
            }
        }
    }()
    return nil
}

func (w *MyWorker) Stop(ctx context.Context) error {
    // Cleanup...
    return nil
}

// 注册
builder.Configure(func(ctx *core.BuildContext) {
    ctx.AddHostedService(&MyWorker{})
})
```

## 🔌 常用组件集成

框架提供了丰富的扩展包：

*   **Redis**: `github.com/gocrud/app/configure/redis`
*   **Cron**: `github.com/gocrud/app/configure/cron`
*   **Etcd**: `github.com/gocrud/app/configure/etcd`
*   **Web (Gin)**: `github.com/gocrud/app/configure/web`

**Web 服务示例:**

```go
import "github.com/gocrud/app/configure/web"

builder.Configure(web.Configure(func(b *web.Builder) {
    // 注册控制器 (支持 DI)
    b.WithControllers(NewUserController) 
    
    // 配置端口
    b.UsePort(8080)
    
    // 添加全局中间件
    b.Use(MyAuthMiddleware)
}))
```

## 📄 文档链接

*   [DI 容器文档](di/README.md)
*   [配置系统文档](config/README.md)
*   [日志系统文档](logging/README.md)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 License

MIT
