package redis

import (
	"context"
	"fmt"

	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/redis/go-redis/v9"
)

// BuilderOption 用于配置 Redis Builder
type BuilderOption func(*Builder)

// WithClient 添加 Redis 客户端配置
func WithClient(name string, opts ...func(*RedisClientOptions)) BuilderOption {
	return func(b *Builder) {
		var configure func(*RedisClientOptions)
		if len(opts) > 0 {
			configure = func(o *RedisClientOptions) {
				for _, opt := range opts {
					opt(o)
				}
			}
		}
		b.AddClient(name, configure)
	}
}

// New 启用 Redis 能力
func New(opts ...BuilderOption) core.Option {
	return func(rt *core.Runtime) error {
		builder := NewBuilder()
		for _, opt := range opts {
			opt(builder)
		}

		// 注入 logger，如果需要的话，这里暂时传 nil
		factory, err := builder.Build(nil)
		if err != nil {
			return err
		}
		if factory == nil {
			return nil
		}

		// 注册工厂
		if err := rt.Provide(factory, di.WithValue(factory)); err != nil {
			return err
		}

		// 注册各个客户端
		var defaultRegErr error
		factory.Each(func(name string, client *redis.Client) {
			if err := rt.Provide(client, di.WithName(name), di.WithValue(client)); err != nil {
				defaultRegErr = err
			}
			// 如果是 default，也注册为默认
			if name == "default" {
				if err := rt.Provide(client, di.WithValue(client)); err != nil {
					defaultRegErr = err
				}
			}
		})

		if defaultRegErr != nil {
			return fmt.Errorf("redis: failed to register instance: %w", defaultRegErr)
		}

		// 注册清理钩子
		rt.Lifecycle.OnStop(func(ctx context.Context) error {
			fmt.Println("Closing redis clients")
			return factory.Close()
		})

		return nil
	}
}
