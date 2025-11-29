package web

import (
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/logging"
)

// Configure 返回 Web 配置器
// 使用示例: builder.Configure(web.Configure(func(b *web.Builder) { ... }))
func Configure(options func(*Builder)) core.Configurator {
	return func(ctx *core.BuildContext) {
		builder := NewBuilder(ctx.GetLogger())
		if options != nil {
			options(builder)
		}

		// 构建 Web Host
		// 传入 DI 容器，以便 Host 启动时能解析 Controller
		webHost := builder.Build(ctx.GetContainer())

		// 直接添加到托管服务列表
		ctx.AddHostedService(webHost)

		ctx.GetLogger().Info("Web host configured",
			logging.Field{Key: "port", Value: webHost.port})
	}
}
