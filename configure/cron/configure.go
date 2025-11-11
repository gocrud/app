package cron

import (
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/logging"
)

// Configure 返回 Cron 配置器
// 使用示例: builder.Configure(cron.Configure(func(b *cron.Builder) { ... }))
func Configure(options func(*Builder)) core.Configurator {
	return func(ctx *core.BuildContext) {
		builder := NewBuilder()
		if options != nil {
			options(builder)
		}

		// 构建 CronService（需要容器来处理依赖注入的任务）
		cronSvc, err := builder.build(ctx, ctx.GetLogger())
		if err != nil {
			ctx.GetLogger().Fatal("Failed to build cron service",
				logging.Field{Key: "error", Value: err.Error()})
		}

		// 直接添加到托管服务列表
		ctx.AddHostedService(cronSvc)

		ctx.GetLogger().Info("Cron service configured")
	}
}
