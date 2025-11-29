# 扩展开发 (Extension Guide)

GoCRUD 框架采用微内核设计，几乎所有功能都是通过插件（Option）扩展的。你可以轻松编写自己的插件来集成第三方库或复用通用逻辑。

## 插件原理

插件本质上是一个符合 `core.Option` 签名的函数：

```go
type Option func(rt *Runtime) error
```

插件的主要职责：
1.  初始化资源（客户端、连接等）。
2.  将资源注册到 DI 容器 (`rt.Provide`)。
3.  注册生命周期钩子 (`rt.Lifecycle.OnStart/OnStop`)。
4.  注册运行时特性 (`rt.Features.Set`)。

## 编写步骤

以集成一个假想的 `EmailClient` 为例。

### 1. 定义配置 Option

通常我们会使用 Functional Options 模式来配置插件本身。

```go
type EmailOptions struct {
    Host string
    Port int
}

type EmailOption func(*EmailOptions)

func WithHost(host string) EmailOption {
    return func(o *EmailOptions) { o.Host = host }
}
```

### 2. 编写核心 Plugin 函数

```go
func NewEmailPlugin(opts ...EmailOption) core.Option {
    return func(rt *core.Runtime) error {
        // 1. 解析配置
        options := &EmailOptions{Port: 25} // 默认值
        for _, o := range opts {
            o(options)
        }

        // 2. 构造实例 (此时可能还未连接)
        client := mylib.NewClient(options.Host, options.Port)

        // 3. 注册到 DI 容器 (单例)
        // 这样业务代码就可以通过构造函数注入 *mylib.Client
        if err := rt.Provide(client); err != nil {
            return err
        }

        // 4. 注册生命周期 (可选)
        rt.Lifecycle.OnStart(func(ctx context.Context) error {
            return client.Connect(ctx)
        })

        rt.Lifecycle.OnStop(func(ctx context.Context) error {
            return client.Close()
        })

        return nil
    }
}
```

### 3. 使用插件

```go
app.Run(
    NewEmailPlugin(WithHost("smtp.example.com")),
)
```

## 高级技巧

### 读取应用配置

如果插件需要读取 `config.yaml` 中的配置，可以使用 `core.GetFeature` 获取 Configuration 接口。

**注意**：在 Plugin 函数执行时（Initialize 阶段），DI 容器尚未构建，不能使用 `Invoke`。但 `Configuration` 模块在加载时会将自身注册到 `rt.Features`，因此可以安全获取。

```go
func NewSmartPlugin() core.Option {
    return func(rt *core.Runtime) error {
        // 获取配置接口
        cfg := core.GetFeature[config.Configuration](rt)
        
        apiKey := ""
        if cfg != nil {
            apiKey = cfg.Get("plugins.smart.api_key")
        }

        // ...
        return nil
    }
}
```

### 注册为 Feature

如果你的插件提供了一些构建时的能力（例如 Web Builder 允许添加 Controller），可以将其注册为 Feature。

```go
type MyBuilder struct { ... }

func NewMyPlugin() core.Option {
    return func(rt *core.Runtime) error {
        builder := &MyBuilder{}
        rt.Features.Set(builder) // 注册
        return nil
    }
}
```

其他插件获取：

```go
func AnotherPlugin() core.Option {
    return func(rt *core.Runtime) error {
        builder := core.GetFeature[*MyBuilder](rt)
        if builder != nil {
            builder.AddSomething(...)
        }
        return nil
    }
}
```

