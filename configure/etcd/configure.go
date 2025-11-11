package etcd

import (
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Configure 返回 Etcd 配置器
// 使用示例: builder.Configure(etcd.Configure(func(b *etcd.Builder) { ... }))
func Configure(options func(*Builder)) core.Configurator {
	return func(ctx *core.BuildContext) {
		builder := NewBuilder()
		if options != nil {
			options(builder)
		}

		// 构建 etcd factory
		factory, err := builder.Build(ctx.GetLogger())
		if err != nil {
			ctx.GetLogger().Fatal("Failed to build etcd clients",
				logging.Field{Key: "error", Value: err.Error()})
		}

		// 注册 factory 到容器
		if factory != nil {
			ctx.ProvideValue(di.ValueProvider{
				Provide: di.TypeOf[*EtcdClientFactory](),
				Value:   factory,
			})

			// 如果有默认客户端，也单独注册
			if defaultClient, err := factory.Get("default"); err == nil {
				ctx.ProvideValue(di.ValueProvider{
					Provide: di.TypeOf[*clientv3.Client](),
					Value:   defaultClient,
				})
				ctx.GetLogger().Info("Default etcd client registered to DI container")
			}

			// 注册清理函数
			ctx.SetCleanup("etcd", func() {
				ctx.GetLogger().Info("Closing etcd clients")
				if err := factory.Close(); err != nil {
					ctx.GetLogger().Error("Failed to close etcd clients",
						logging.Field{Key: "error", Value: err.Error()})
				}
			})
		}
	}
}
