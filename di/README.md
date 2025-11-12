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
    // 1. 创建容器
    container := di.NewContainer()
    
    // 2. 注册依赖
    di.BindWith[Logger](container, &ConsoleLogger{})
    container.Provide(&UserService{})
    
    // 3. 构建容器
    container.Build()
    
    // 4. 获取实例
    var svc *UserService
    container.Inject(&svc)
    svc.Logger.Log("Hello DI!")
}
```

## 依赖注入方式

框架提供了指针注入的方式：

### 指针注入（容器实例）

```go
// 创建容器实例
container := di.NewContainer()
// ... 注册和构建 ...

// 使用 Inject（失败时 panic）
var svc *UserService
container.Inject(&svc)

// Inject 不返回错误，失败时会 panic
var logger Logger
container.Inject(&logger)
```

**优点**：
- 传统 DI 容器风格（类似 Java Spring、.NET Core）
- 适合批量声明变量
- 简洁明了，失败时立即 panic

**适用场景**：使用独立容器实例、测试、多容器隔离

## 容器使用

### 创建和配置容器

```go
// 创建容器
container := di.NewContainer()

// 注册服务
di.BindWith[Logger](container, &ConsoleLogger{})
container.Provide(&UserService{})

// 构建
container.Build()

// 注入：使用 Inject
var svc *UserService
container.Inject(&svc)

// 批量注入
var logger Logger
container.Inject(&logger)
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
// 注册 Scoped 服务
container.ProvideWithConfig(di.ProviderConfig{
    Provide: di.TypeOf[*Repository](),
    UseClass: NewRepository,
    Scope: di.ScopeScoped,
})

container.Build()

// 创建作用域
scope := container.CreateScope()
defer scope.Dispose()

// 设置当前作用域
container.SetCurrentScope(scope)
defer container.ClearCurrentScope()

// 从作用域注入（与容器一致的优雅语法）
var repo *Repository
scope.Inject(&repo)

// 也支持接口注入
var logger Logger
scope.Inject(&logger)
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
