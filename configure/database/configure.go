package database

import (
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
	"gorm.io/gorm"
)

// Configure 返回数据库配置器
func Configure(options func(*Builder)) core.Configurator {
	return func(ctx *core.BuildContext) {
		// 注入 Context
		builder := NewBuilder(ctx)
		if options != nil {
			options(builder)
		}

		factory, err := builder.Build(ctx.GetLogger())
		if err != nil {
			ctx.GetLogger().Fatal("Failed to build databases",
				logging.Field{Key: "error", Value: err.Error()})
		}

		if factory != nil {
			// 注册工厂
			di.Register[*DatabaseFactory](ctx.Container(), di.WithValue(factory))

			// 注册所有实例
			factory.Each(func(name string, db *gorm.DB) {
				di.Register[*gorm.DB](ctx.Container(), di.WithName(name), di.WithValue(db))
				ctx.GetLogger().Info("Database client registered to DI", logging.Field{Key: "name", Value: name})

				// 默认实例兼容性
				if name == "default" {
					di.Register[*gorm.DB](ctx.Container(), di.WithValue(db))
					ctx.GetLogger().Info("Default database registered to DI (unnamed)")
				}
			})

			// 注册清理
			ctx.SetCleanup("database", func() {
				ctx.GetLogger().Info("Closing database connections")
				if err := factory.Close(); err != nil {
					ctx.GetLogger().Error("Failed to close databases",
						logging.Field{Key: "error", Value: err.Error()})
				}
			})
		}
	}
}
