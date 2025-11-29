# 配置系统 (Configuration)

框架内置了强大的配置管理模块，支持多文件加载、环境变量覆盖、结构体绑定。

## 快速开始

```go
app.Run(
    // 加载配置文件
    config.Load("config.yaml"),
    
    // 绑定配置到结构体
    func(rt *core.Runtime) error {
        return config.Bind[AppConfig](rt, "app")
    },
)
```

## 加载策略

### 多文件加载

支持加载多个配置文件，后加载的配置会覆盖先加载的（Deep Merge）。

```go
config.Load("base.yaml"),
config.Load("prod.yaml"), // 覆盖 base.yaml 中的同名项
```

### 环境变量

配置模块会自动加载环境变量，并覆盖文件中的配置。

**映射规则**:
*   `_` (下划线) 映射为 `.` (层级分隔符)。
*   `__` (双下划线) 映射为 `:` (复杂键分隔符)。
*   不区分大小写。

**示例**:
假设配置文件：
```yaml
server:
  port: 8080
  db:
    host: localhost
```

*   `SERVER_PORT=9090` -> 覆盖 `server.port`
*   `SERVER_DB_HOST=10.0.0.1` -> 覆盖 `server.db.host`

## 结构体绑定 (Bind)

这是推荐的配置使用方式。

1.  **定义结构体**:
    ```go
    type RedisConfig struct {
        Host string `json:"host"`
        Port int    `json:"port"`
    }
    ```

2.  **绑定并注册**:
    ```go
    config.Bind[RedisConfig](rt, "redis")
    ```
    这会将配置文件中 `redis` 节的内容解析到 `RedisConfig` 结构体，并将 `*RedisConfig` 注册为单例服务。

3.  **注入使用**:
    ```go
    type Service struct {
        RedisCfg *RedisConfig `di:""`
    }
    ```

## 动态获取 (Get)

如果不需要强类型绑定，可以直接注入 `config.Configuration` 接口。

```go
type Service struct {
    Config config.Configuration `di:""`
}

func (s *Service) Run() {
    val := s.Config.Get("some.key")
    num, _ := s.Config.GetInt("retry.count")
}
```

## 接口定义

```go
type Configuration interface {
    Get(key string) string
    GetWithDefault(key, defaultValue string) string
    GetInt(key string) (int, error)
    GetBool(key string) (bool, error)
    GetSection(key string) Configuration
    Bind(key string, target any) error
    GetAll() map[string]any
}
```

