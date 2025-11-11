package configure

import (
	"github.com/gocrud/app/configure/cron"
	"github.com/gocrud/app/configure/etcd"
	"github.com/gocrud/app/configure/redis"
	"github.com/gocrud/app/configure/web"
	"github.com/gocrud/app/core"
)

// Etcd 便捷导出 etcd 配置器
// 使用示例: builder.Configure(configure.Etcd(func(b *etcd.Builder) { ... }))
func Etcd(options func(*etcd.Builder)) core.Configurator {
	return etcd.Configure(options)
}

// Cron 便捷导出 cron 配置器
// 使用示例: builder.Configure(configure.Cron(func(b *cron.Builder) { ... }))
func Cron(options func(*cron.Builder)) core.Configurator {
	return cron.Configure(options)
}

// Web 便捷导出 web 配置器
// 使用示例: builder.Configure(configure.Web(func(b *web.Builder) { ... }))
func Web(options func(*web.Builder)) core.Configurator {
	return web.Configure(options)
}

// Redis 便捷导出 redis 配置器
// 使用示例: builder.Configure(configure.Redis(func(b *redis.Builder) { ... }))
func Redis(options func(*redis.Builder)) core.Configurator {
	return redis.Configure(options)
}
