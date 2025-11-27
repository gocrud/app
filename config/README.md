# 配置系统 (Configuration)

`config` 包提供了一套灵活、分层、强类型的配置管理系统。

## 核心特性

*   **分层覆盖**: 支持多数据源，优先级从低到高：`JSON/YAML 文件` < `环境变量` < `命令行参数`。
*   **热重载 (Reloading)**: 配置文件变更时自动触发重载，支持 `IOptionsMonitor` 实时感知。
*   **强类型映射**: 将配置直接绑定到 Go 结构体。
*   **Options 模式**: 类似 .NET Core 的 Options Pattern，支持 Singleton, Snapshot, Monitor 三种生命周期。

## 快速开始

### 1. 基础用法

默认情况下，框架会自动加载运行目录下的 `config.yaml` 或 `appsettings.json`。

```go
// 1. 定义配置结构
type RedisSettings struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}

// 2. 在 ApplicationBuilder 中注册
// 将配置文件的 "redis" 节点映射到 RedisSettings
core.AddOptions[RedisSettings](builder, "redis")
```

### 2. 消费配置

建议通过构造函数注入 `config.Option[T]`。

```go
type RedisClient struct {
    settings *RedisSettings
}

// 注入 config.Option[RedisSettings]
func NewRedisClient(opts config.Option[RedisSettings]) *RedisClient {
    return &RedisClient{
        settings: opts.Value, // 获取配置值
    }
}
```

## 进阶用法

### Options 生命周期

框架提供三种 Option 包装器，适用于不同场景：

| 类型 | 描述 | 适用场景 |
| :--- | :--- | :--- |
| `Option[T]` | **单例 (Singleton)**。应用启动时读取一次，之后不再变化。 | 大多数配置，如数据库连接串（通常不希望运行时变）。 |
| `OptionSnapshot[T]` | **作用域 (Scoped)**。每个请求/作用域创建时读取最新配置。 | 希望每个请求能读到不同配置，但在请求处理过程中保持一致。 |
| `OptionMonitor[T]` | **单例 (Singleton)**。提供 `OnChange` 回调，实时感知变化。 | 需要运行时动态调整的业务开关、阈值等。 |

**示例：使用 Monitor 监听变化**

```go
type FeatureFlags struct {
    EnableNewUI bool `json:"enable_new_ui"`
}

func NewFeatureService(monitor config.OptionMonitor[FeatureFlags]) *FeatureService {
    // 注册变更回调
    monitor.OnChange(func(newConfig *FeatureFlags) {
        fmt.Println("配置已更新:", newConfig.EnableNewUI)
    })

    return &FeatureService{monitor: monitor}
}

func (s *FeatureService) DoWork() {
    // 总是获取最新值
    if s.monitor.Value().EnableNewUI {
        // ...
    }
}
```

### 自定义配置源

除了默认文件，你还可以添加自定义源：

```go
builder.ConfigureConfiguration(func(b *config.ConfigurationBuilder) {
    // 添加环境变量源，前缀为 "MYAPP_"
    // export MYAPP_REDIS__HOST=localhost
    b.AddEnvironmentVariables("MYAPP_")
    
    // 添加命令行参数源
    b.AddCommandLine(os.Args[1:])
})
```

## 配置文件格式

支持 YAML 和 JSON。

**config.yaml**
```yaml
app:
  name: "My App"
  
redis:
  host: "localhost"
  port: 6379
```

**映射结构体**
```go
type AppConfig struct {
    Name string `json:"name" yaml:"name"` // 建议同时写 json 和 yaml tag
}
```

