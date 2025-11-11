# Redis 配置模块

Redis 配置模块已添加到 `configure/redis` 目录。

## 使用示例

### 1. 基本使用（单个 Redis 客户端）

```go
package main

import (
	"github.com/gocrud/app/configure/redis"
	"github.com/gocrud/app/core"
)

func main() {
	builder := core.NewApplicationBuilder()
	
	// 配置 Redis
	builder.Configure(redis.Configure(func(b *redis.Builder) {
		b.AddClient("default", func(opts *redis.RedisClientOptions) {
			opts.Addr = "localhost:6379"
			opts.Password = ""
			opts.DB = 0
		})
	}))
	
	app := builder.Build()
	app.Run()
}
```

### 2. 多个 Redis 客户端

```go
builder.Configure(redis.Configure(func(b *redis.Builder) {
	// 默认客户端
	b.AddClient("default", func(opts *redis.RedisClientOptions) {
		opts.Addr = "localhost:6379"
		opts.DB = 0
	})
	
	// 缓存专用客户端
	b.AddClient("cache", func(opts *redis.RedisClientOptions) {
		opts.Addr = "localhost:6379"
		opts.DB = 1
		opts.PoolSize = 20
	})
	
	// 会话专用客户端
	b.AddClient("session", func(opts *redis.RedisClientOptions) {
		opts.Addr = "localhost:6380"
		opts.Password = "secret"
		opts.DB = 0
	})
}))
```

### 3. 在服务中使用 Redis

```go
package myservice

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type MyService struct {
	redisClient *redis.Client
}

// 通过 DI 注入默认 Redis 客户端
func NewMyService(client *redis.Client) *MyService {
	return &MyService{
		redisClient: client,
	}
}

func (s *MyService) SetValue(key, value string) error {
	ctx := context.Background()
	return s.redisClient.Set(ctx, key, value, 0).Err()
}

func (s *MyService) GetValue(key string) (string, error) {
	ctx := context.Background()
	return s.redisClient.Get(ctx, key).Result()
}
```

### 4. 使用 RedisClientFactory 管理多个客户端

```go
package myservice

import (
	"context"
	"github.com/gocrud/app/configure/redis"
	redisv9 "github.com/redis/go-redis/v9"
)

type CacheService struct {
	factory *redis.RedisClientFactory
}

func NewCacheService(factory *redis.RedisClientFactory) *CacheService {
	return &CacheService{
		factory: factory,
	}
}

func (s *CacheService) SetCache(key, value string) error {
	client, err := s.factory.Get("cache")
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	return client.Set(ctx, key, value, 0).Err()
}

func (s *CacheService) SetSession(key, value string) error {
	client, err := s.factory.Get("session")
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	return client.Set(ctx, key, value, 0).Err()
}
```

## 配置选项说明

```go
type RedisClientOptions struct {
	Name         string        // 客户端名称（必填）
	Addr         string        // Redis 服务器地址，格式: "host:port"
	Password     string        // 密码（可选）
	DB           int           // 数据库编号（0-15）
	DialTimeout  time.Duration // 连接超时时间
	ReadTimeout  time.Duration // 读取超时时间
	WriteTimeout time.Duration // 写入超时时间
	PoolSize     int           // 连接池大小
	MinIdleConns int           // 最小空闲连接数
	MaxRetries   int           // 最大重试次数
}
```

## 默认值

- Addr: `"localhost:6379"`
- DB: `0`
- DialTimeout: `5s`
- ReadTimeout: `3s`
- WriteTimeout: `3s`
- PoolSize: `10`
- MinIdleConns: `5`
- MaxRetries: `3`

## 特性

1. **连接池管理**：自动管理 Redis 连接池
2. **多客户端支持**：可配置多个 Redis 客户端连接不同的实例或数据库
3. **DI 集成**：自动注册到依赖注入容器
4. **自动清理**：应用关闭时自动关闭所有 Redis 连接
5. **健康检查**：注册时自动 Ping 测试连接
