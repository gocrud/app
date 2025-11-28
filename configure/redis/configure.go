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
		// 使用 BuildContext 初始化 Builder
		builder := NewBuilder(ctx)
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
			// Register factory
			di.Register[*RedisClientFactory](ctx.Container(), di.WithValue(factory))

			// 遍历所有客户端并注册到 DI 容器
			factory.Each(func(name string, client *redis.Client) {
				// 使用名称注册
				di.Register[*redis.Client](ctx.Container(), di.WithName(name), di.WithValue(client))
				ctx.GetLogger().Info("Redis client registered to DI", logging.Field{Key: "name", Value: name})
			})

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
