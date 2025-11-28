package mongodb

import (
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
	"github.com/gocrud/mgo"
)

// Configure 返回 MongoDB 配置器
func Configure(options func(*Builder)) core.Configurator {
	return func(ctx *core.BuildContext) {
		builder := NewBuilder(ctx)
		if options != nil {
			options(builder)
		}

		factory, err := builder.Build(ctx.GetLogger())
		if err != nil {
			ctx.GetLogger().Fatal("Failed to build mongodb clients",
				logging.Field{Key: "error", Value: err.Error()})
		}

		if factory != nil {
			// 注册 Factory
			di.Register[*MongoFactory](ctx.Container(), di.WithValue(factory))

			// 注册 Client 实例
			factory.Each(func(name string, client *mgo.Client) {
				di.Register[*mgo.Client](ctx.Container(), di.WithName(name), di.WithValue(client))
				ctx.GetLogger().Info("Mongo client registered to DI", logging.Field{Key: "name", Value: name})

				// 默认实例兼容性
				if name == "default" {
					di.Register[*mgo.Client](ctx.Container(), di.WithValue(client))
					ctx.GetLogger().Info("Default mongo client registered to DI (unnamed)")
				}
			})

			// 注册清理
			ctx.SetCleanup("mongodb", func() {
				ctx.GetLogger().Info("Closing mongo clients")
				if err := factory.Close(); err != nil {
					ctx.GetLogger().Error("Failed to close mongo clients",
						logging.Field{Key: "error", Value: err.Error()})
				}
			})
		}
	}
}

