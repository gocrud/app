package redis

import (
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
	"github.com/redis/go-redis/v9"
)

// Configure 返回 Redis 配置器
// 使用示例: builder.Configure(redis.Configure(func(b *redis.Builder) { ... }))
func Configure(options func(*Builder)) core.Configurator {
	return func(ctx *core.BuildContext) {
		builder := NewBuilder()
		if options != nil {
			options(builder)
		}

		// 构建 redis factory
		factory, err := builder.Build(ctx.GetLogger())
		if err != nil {
			ctx.GetLogger().Fatal("Failed to build redis clients",
				logging.Field{Key: "error", Value: err.Error()})
		}

		// 注册 factory 到容器
		if factory != nil {
			ctx.ProvideValue(di.ValueProvider{
				Provide: di.TypeOf[*RedisClientFactory](),
				Value:   factory,
			})

			// 如果有默认客户端，也单独注册
			if defaultClient, err := factory.Get("default"); err == nil {
				ctx.ProvideValue(di.ValueProvider{
					Provide: di.TypeOf[*redis.Client](),
					Value:   defaultClient,
				})
				ctx.GetLogger().Info("Default redis client registered to DI container")
			}

			// 注册清理函数
			ctx.SetCleanup("redis", func() {
				ctx.GetLogger().Info("Closing redis clients")
				if err := factory.Close(); err != nil {
					ctx.GetLogger().Error("Failed to close redis clients",
						logging.Field{Key: "error", Value: err.Error()})
				}
			})
		}
	}
}
