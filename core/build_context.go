package core

import (
	"reflect"
	"sync"

	"github.com/gocrud/app/config"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/hosting"
	"github.com/gocrud/app/logging"
)

// Configurator 配置器函数类型
// 配置器用于扩展应用程序，可以注册服务、添加托管服务等
type Configurator func(*BuildContext)

// BuildContext 构建上下文
// 提供给配置器的上下文环境，包含容器、配置、日志等核心组件
type BuildContext struct {
	// container DI 容器
	container di.Container

	// configuration 配置对象
	configuration config.Configuration

	// logger 日志记录器
	logger logging.Logger

	// environment 环境信息
	environment Environment

	// hostedServices 托管服务列表
	hostedServices []hosting.HostedService

	// cleanups 清理函数列表
	cleanups map[string]func()

	mu sync.RWMutex
}

// AddHostedService 添加托管服务
func (c *BuildContext) AddHostedService(service hosting.HostedService) {
	c.hostedServices = append(c.hostedServices, service)
}

// SetCleanup 设置资源清理函数
func (c *BuildContext) SetCleanup(key string, cleanup func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cleanups[key] = cleanup
}

// Container returns the underlying DI container.
// This allows using di.Register[T](ctx.Container(), ...) directly.
func (c *BuildContext) Container() di.Container {
	return c.container
}

// GetContainer 返回底层的 DI 容器（别名 Container）
// 用于保持 API 命名风格一致性
func (c *BuildContext) GetContainer() di.Container {
	return c.container
}

// ResolveService 从容器中解析服务
// 注意：仅在必要时使用此方法，优先使用 Register 系列方法注册服务
func (c *BuildContext) ResolveService(serviceType reflect.Type) (any, error) {
	return c.container.Get(serviceType)
}

// GetLogger 获取日志记录器
func (c *BuildContext) GetLogger() logging.Logger {
	return c.logger
}

// GetConfiguration 获取配置对象
func (c *BuildContext) GetConfiguration() config.Configuration {
	return c.configuration
}

// GetEnvironment 获取环境信息
func (c *BuildContext) GetEnvironment() Environment {
	return c.environment
}

// ConfigureOptions 配置选项模式（支持静态、快照和监听三种模式）
// T: 配置类型
// section: 配置节名称（例如 "app", "database"）
// 使用示例: ctx.ConfigureOptions[AppSetting]("app")
func ConfigureOptions[T any](ctx *BuildContext, section string) {
	// 创建 OptionsCache
	cache := config.NewOptionsCache[T](ctx.configuration, section)

	// 注册 Option[T] - Singleton（应用生命周期内不变）
	di.Register[config.Option[T]](ctx.container,
		di.WithValue(config.NewOption(cache.Get())),
		di.WithSingleton(),
	)

	// 注册 OptionMonitor[T] - Singleton（实时更新，框架自动处理）
	di.Register[config.OptionMonitor[T]](ctx.container,
		di.WithValue(config.NewOptionMonitor(cache)),
		di.WithSingleton(),
	)

	// 注册 OptionSnapshot[T] - Scoped（每个作用域创建时的快照）
	di.Register[config.OptionSnapshot[T]](ctx.container,
		di.WithFactory(func() config.OptionSnapshot[T] {
			return config.NewOptionSnapshot(cache.Snapshot())
		}),
		di.WithScoped(),
	)

	// Hack: We need to stringify T for logging, but we can't easily get type name from generic T without instance.
	// We can create zero value.
	var zero T
	typeName := reflect.TypeOf(zero).String()

	ctx.logger.Info("Configured options",
		logging.Field{Key: "type", Value: typeName},
		logging.Field{Key: "section", Value: section})
}
