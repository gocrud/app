# 核心概念 (Core Concepts)

框架基于微内核架构设计，核心仅由 **Runtime**、**依赖注入 (DI)** 和 **生命周期 (Lifecycle)** 组成。

## Runtime (运行时)

`Runtime` 是框架的上帝对象，贯穿应用生命周期。它作为一个容器，承载了所有运行时状态。

```go
type Runtime struct {
    // Container 核心依赖注入容器
    Container di.Container

    // Lifecycle 生命周期管理
    Lifecycle *LifecycleEvents

    // Features 存放构建时的组件特性（如 Web Host, DB Factory）
    Features FeatureCollection
}
```

在开发插件或业务模块时，你主要与 `Runtime` 交互。

## 依赖注入 (DI)

框架内置了强大的 DI 系统，基于反射实现，支持自动类型推断。

### 1. Provide (提供服务)

将服务注册到容器中。注册操作通常在 `Option` 函数（初始化阶段）中进行。

**基础注册**:

```go
// 1. 注册构造函数 (推荐)
// 框架会自动分析 NewUserService 的入参，从容器中寻找依赖并注入。
rt.Provide(NewUserService)

// 2. 注册结构体指针 (单例值)
// 框架会扫描结构体字段，如果发现 `di:""` 标签，会自动注入。
rt.Provide(&UserService{})
```

**高级注册**:

```go
// 3. 绑定接口
// 将 *RepoImpl 注册为 IRepo 接口的实现。
// 业务代码中依赖 IRepo，容器会自动注入 *RepoImpl。
di.ProvideService[IRepo](rt.Container, di.Use[*RepoImpl]())

// 4. 命名注入
// 注册时指定名称
rt.Provide(&DB{}, di.WithName("master"))

// 获取时指定名称
type Service struct {
    DB *DB `di:"name=master"`
}
```

### 2. Invoke (调用/注入)

执行函数并自动注入依赖。通常用于 `OnStart` 钩子中获取服务实例。

```go
// 这里的 svc 和 cfg 会被自动注入
rt.Invoke(func(svc *UserService, cfg *Config) {
    svc.DoSomething()
})
```

## Lifecycle (生命周期)

应用启动时，框架会按照特定顺序执行生命周期钩子。

### 阶段流程

1.  **Initialize (初始化)**: `app.Run(opts...)` 执行时。
    *   执行所有 `Option` 函数。
    *   **此时容器尚未构建**。只能进行 `Provide` 注册，**禁止** `Invoke/Get`。
2.  **DI Build (构建)**: 框架锁定 DI 容器，解析依赖图。
3.  **OnStart (启动)**: 按照注册顺序，依次执行所有 `OnStart` 钩子。
    *   **此时容器已构建**。可以安全地 `Invoke/Get` 服务。
    *   常用于：启动 HTTP Server、建立 DB 连接、启动 Cron 任务。
4.  **Running (运行)**: 应用阻塞运行，直到收到 OS 信号 (SIGINT/SIGTERM)。
5.  **OnStop (停止)**: 收到信号后，按照 **注册的相反顺序** 执行 `OnStop` 钩子。
    *   常用于：关闭 HTTP Server、关闭 DB 连接、停止 Cron。

### 钩子注册

```go
rt.Lifecycle.OnStart(func(ctx context.Context) error {
    fmt.Println("App starting...")
    return nil
})

rt.Lifecycle.OnStop(func(ctx context.Context) error {
    fmt.Println("App stopping...")
    return nil
})
```

### ⚠️ 重要：顺序与约束

1.  **Option 阶段**:
    *   ✅ **Do**: `Provide` (注册服务), `Lifecycle.OnStart/OnStop` (注册钩子)。
    *   🚫 **Don't**: `Get/Invoke` (解析服务)。此时容器为空或未构建。

2.  **OnStart 阶段**:
    *   ✅ **Do**: `Get/Invoke` (使用服务)。
    *   🚫 **Don't**: `Provide` (注册服务)。容器已锁定。

3.  **依赖顺序**:
    *   **DI Provide**: 顺序**无关**。DI 容器会自动解析依赖拓扑。
    *   **Lifecycle Hooks**: 顺序**相关**。`OnStart` 按 `app.Run` 参数顺序执行。建议将基础设施（Config, DB）放在业务模块之前。

