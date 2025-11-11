# Inject 指针模式示例

本示例演示如何使用 `var svc XxxService; container.Inject(&svc)` 模式进行依赖注入。

## 使用方式

### 方式1: var + Inject (推荐，带错误处理)

```go
var userSvc *UserService
if err := container.Inject(&userSvc); err != nil {
    // 处理错误
    log.Fatal(err)
}
// 使用 userSvc
```

### 方式2: var + MustInject (简洁，失败时 panic)

```go
var userSvc *UserService
container.MustInject(&userSvc)
// 使用 userSvc
```

## 对比

| 方式 | 优点 | 缺点 |
|------|------|------|
| `container.Inject(&svc)` | 1. 语法简洁<br>2. 支持错误处理<br>3. 传统 DI 风格 | 需要先声明变量 |
| `container.MustInject(&svc)` | 1. 最简洁<br>2. 适合明确知道依赖存在的场景 | 失败时 panic |

## 运行示例

```bash
cd di/examples/inject_pointer
go run main.go
```

## 适用场景

### 使用 Inject 指针模式的场景
- 需要批量声明和注入多个依赖
- 喜欢传统 DI 容器的使用风格（类似 Java Spring、.NET Core）
- 需要细粒度的错误处理
- 使用独立容器实例

## 示例代码

查看 `main.go` 获取完整示例。
