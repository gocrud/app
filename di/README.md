# DI 依赖注入框架

一个轻量级、高性能的 Go 语言依赖注入框架。

## 核心功能

- ✅ 类型安全的依赖注入
- ✅ 支持构造函数注入和字段注入
- ✅ 三种生命周期：Singleton、Transient、Scoped
- ✅ 循环依赖检测
- ✅ 可选依赖支持
- ✅ Token 机制支持同类型多实例
- ✅ 并发安全

## 快速开始

### 基本使用

```go
package main

import "github.com/gocrud/app/di"

type Logger interface {
    Log(msg string)
}

type ConsoleLogger struct {}

func (c *ConsoleLogger) Log(msg string) {
    println(msg)
}

type UserService struct {
    Logger Logger `di:""`
}

func main() {
    // 1. 注册依赖
    di.Bind[Logger](&ConsoleLogger{})
    di.Provide(&UserService{})
    
    // 2. 构建容器
    di.MustBuild()
    
    // 3. 获取实例
    svc := di.Inject[*UserService]()
    svc.Logger.Log("Hello DI!")
}
```

## 依赖注入方式

框架提供了两种注入方式，可以根据场景选择：

### 方式1: 泛型注入（全局容器）

```go
// 使用默认容器
svc := di.Inject[*UserService]()

// 带错误处理
svc, err := di.TryInject[*UserService]()

// 带默认值
svc := di.InjectOrDefault[*UserService](defaultSvc)
```

**优点**：
- 类型安全，编译时检查
- 语法简洁，一行完成
- Go 风格

**适用场景**：使用全局默认容器

### 方式2: 指针注入（容器实例）

```go
// 创建容器实例
container := di.NewContainer()
// ... 注册和构建 ...

// 使用 Inject（带错误处理）
var svc *UserService
if err := container.Inject(&svc); err != nil {
    log.Fatal(err)
}

// 使用 MustInject（失败时 panic）
var logger Logger
container.MustInject(&logger)
```

**优点**：
- 传统 DI 容器风格（类似 Java Spring、.NET Core）
- 适合批量声明变量
- 支持细粒度错误处理

**适用场景**：使用独立容器实例、测试、多容器隔离

### 两种方式对比

```go
// 方式1: 全局容器 - 泛型注入
svc1 := di.Inject[*UserService]()

// 方式2: 指针注入
var svc2 *UserService
container.MustInject(&svc2)
```

| 特性 | 泛型注入 | 指针注入 |
|------|----------|----------|
| 类型安全 | ✅ 编译时 | ✅ 运行时 |
| 语法简洁 | ✅ 一行完成 | ⚠️ 需要先声明 |
| 错误处理 | ⚠️ panic 或返回 | ✅ 灵活 |
| 适用场景 | 全局容器、单个注入 | 容器实例、批量注入 |
| 风格 | Go 语言风格 | 传统 DI 风格 |

## 容器实例 vs 全局容器

### 全局容器（推荐用于应用程序）

```go
// 注册
di.Bind[Logger](&ConsoleLogger{})
di.Provide(&UserService{})

// 构建
di.MustBuild()

// 注入
svc := di.Inject[*UserService]()
```

### 独立容器（推荐用于测试和隔离场景）

```go
// 创建容器
container := di.NewContainer()

// 注册（使用 With 后缀的方法）
di.BindWith[Logger](container, &ConsoleLogger{})
container.Provide(&UserService{})

// 构建
container.Build()

// 注入：使用指针方式
var svc *UserService
container.MustInject(&svc)

// 或带错误处理
var logger Logger
if err := container.Inject(&logger); err != nil {
    log.Fatal(err)
}
```

## 示例

查看 `examples/` 目录获取更多示例：

- `simple/` - 基础使用示例
- `inject_pointer/` - 指针注入示例（`var + Inject` 模式）
- `container_instance/` - 容器实例示例
- `optional/` - 可选依赖示例
- `scope_demo/` - 作用域示例

## 运行示例

```bash
# 基础示例
go run di/examples/simple/main.go

# 指针注入示例
go run di/examples/inject_pointer/main.go

# 容器实例示例
go run di/examples/container_instance/main.go
```

## 高级特性

### 作用域

```go
// Singleton（默认）- 全局唯一
di.ProvideClass(di.ClassProvider{
    Provide: di.TypeOf[*UserService](),
    UseClass: NewUserService,
    Scope: di.Singleton,
})

// Transient - 每次创建新实例
di.ProvideClass(di.ClassProvider{
    Provide: di.TypeOf[*RequestHandler](),
    UseClass: NewRequestHandler,
    Scope: di.Transient,
})

// Scoped - 作用域内唯一
scope := container.CreateScope()
defer scope.Dispose()
```

### Token 机制

```go
var PrimaryDB = di.NewToken[Database]()
var SecondaryDB = di.NewToken[Database]()

di.ProvideType(di.TypeProvider{
    Provide: PrimaryDB,
    UseType: &MySQLDatabase{Host: "primary"},
})

di.ProvideType(di.TypeProvider{
    Provide: SecondaryDB,
    UseType: &MySQLDatabase{Host: "secondary"},
})

primary := di.Inject[Database](PrimaryDB)
secondary := di.Inject[Database](SecondaryDB)
```

### 可选依赖

```go
type UserService struct {
    Logger Logger   `di:""`           // 必需
    Cache  Cache    `di:"optional"`   // 可选
}

// 或使用 InjectOrDefault
cache := di.InjectOrDefault[Cache](defaultCache)
```

## License

MIT
