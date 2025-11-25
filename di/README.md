# DI 依赖注入框架

一个基于 Go 1.18+ 泛型的轻量级依赖注入框架。

## 核心特性

- 🚀 **泛型优先**：完全类型安全的 API，无需类型断言。
- 🛠 **功能选项**：使用 Functional Options 模式配置服务。
- 🔄 **生命周期管理**：支持 Singleton（单例）、Transient（瞬态）、Scoped（作用域）。
- 💉 **自动注入**：支持构造函数自动注入和结构体字段注入（`di:""`）。
- 🔍 **循环依赖检测**：构建时自动检测并报错。

## 快速开始

### 安装

```bash
go get github.com/gocrud/app/di
```

### 基础使用

```go
package main

import (
    "fmt"
    "github.com/gocrud/app/di"
)

// 1. 定义接口
type Logger interface {
    Log(msg string)
}

// 2. 实现接口
type ConsoleLogger struct {}
func (l *ConsoleLogger) Log(msg string) { fmt.Println(msg) }

// 3. 定义依赖服务的结构体
type App struct {
    Logger Logger `di:""` // 字段自动注入
}

func main() {
    // 创建容器
    c := di.NewContainer()

    // 注册服务
    // 将接口绑定到具体实现
    di.Register[Logger](c, di.Use[*ConsoleLogger]())
    
    // 注册 App (默认单例)
    di.Register[*App](c)

    // 构建容器
    if err := c.Build(); err != nil {
        panic(err)
    }

    // 获取实例
    app, err := di.Resolve[*App](c) // 或 di.MustResolve[*App](c)
    if err != nil {
        panic(err)
    }

    app.Logger.Log("Hello DI")
}
```

## 注册方式

### 1. 绑定接口

将接口类型绑定到具体的实现类型。

```go
// 注册 Logger 接口，使用 *ConsoleLogger 作为实现
di.Register[Logger](c, di.Use[*ConsoleLogger]())
```

### 2. 注册具体值

直接注册一个现成的对象实例。

```go
// 注册 int 类型的配置值
di.Register[int](c, di.WithValue(8080))
```

### 3. 使用工厂函数

当初始化逻辑复杂时，使用工厂函数。泛型会自动推断依赖。

```go
// 使用工厂函数创建 Config
di.Register[*Config](c, di.WithFactory(func(env EnvService) *Config {
    // 容器会自动注入 env 参数
    return &Config{Port: env.Get("PORT")}
}))
```

### 4. 生命周期配置

通过 Option 配置服务的生命周期：

- `di.WithSingleton()` (默认)：全局单例，只创建一次。
- `di.WithTransient()`：每次获取都会创建一个新实例。
- `di.WithScoped()`：在每个 Scope 中保持单例。

```go
// 注册为 Transient
di.Register[*Worker](c, di.WithTransient())
```

## 获取服务 (Resolution)

### 1. Resolve (安全获取)

返回实例和错误，推荐用于可能失败的场景。

```go
svc, err := di.Resolve[*MyService](c)
if err != nil {
    // 处理错误
}
```

### 2. MustResolve (Panic 获取)

直接返回实例，如果失败则 Panic。适用于应用启动时必须成功的核心服务。

```go
// 如果解析失败会 Panic
svc := di.MustResolve[*MyService](c)
```

## 作用域 (Scopes)

适用于处理 HTTP 请求等需要隔离上下文的场景。

```go
// 注册为 Scoped
di.Register[*RequestContext](c, di.WithScoped())

// 创建作用域
scope := c.CreateScope()
defer scope.Dispose() // 确保释放资源

// 从作用域获取 (在此作用域内是单例)
// 注意：Resolve 的第一个参数可以是 Container 或 Scope
ctx := di.MustResolve[*RequestContext](scope)
```

## 字段注入

在结构体字段上添加 `di:""` 标签即可启用自动注入。支持 `optional` 标记。

```go
type Service struct {
    DB    Database `di:""`           // 必须注入，失败报错
    Cache Cache    `di:"optional"`   // 可选注入，失败忽略
}
```
