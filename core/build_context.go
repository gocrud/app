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

// Provide 注册服务到容器（默认行为由容器决定）
func (c *BuildContext) Provide(value any) {
	c.container.Provide(value)
}

// ProvideValue 使用 ValueProvider 注册服务
func (c *BuildContext) ProvideValue(provider di.ValueProvider) {
	c.container.ProvideValue(provider)
}

// ProvideType 使用 TypeProvider 注册服务
func (c *BuildContext) ProvideType(provider di.TypeProvider) {
	c.container.ProvideType(provider)
}

// ProvideWithConfig 使用 ProviderConfig 注册服务
func (c *BuildContext) ProvideWithConfig(config di.ProviderConfig) {
	c.container.ProvideWithConfig(config)
}

// ResolveService 从容器中解析服务（供配置器内部使用，如 Cron 的依赖注入）
// 注意：仅在必要时使用此方法，优先使用 Provide 系列方法注册服务
func (c *BuildContext) ResolveService(serviceType any) (any, error) {
	return c.container.GetByType(reflect.TypeOf(serviceType))
}

// GetContainer 获取 DI 容器（已废弃：请使用 Provide 系列方法）
// Deprecated: 直接访问容器可能导致误用，请使用 Provide/ProvideValue/ProvideType/ProvideWithConfig
func (c *BuildContext) GetContainer() di.Container {
	return c.container
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
	ctx.ProvideValue(di.ValueProvider{
		Provide: di.TypeOf[config.Option[T]](),
		Value:   config.NewOption[T](cache.Get()),
		Options: di.ProviderOptions{
			Scope: di.ScopeSingleton,
		},
	})

	// 注册 OptionMonitor[T] - Singleton（实时更新，框架自动处理）
	ctx.ProvideValue(di.ValueProvider{
		Provide: di.TypeOf[config.OptionMonitor[T]](),
		Value:   config.NewOptionMonitor[T](cache),
		Options: di.ProviderOptions{
			Scope: di.ScopeSingleton,
		},
	})

	// 注册 OptionSnapshot[T] - Scoped（每个作用域创建时的快照）
	ctx.ProvideWithConfig(di.ProviderConfig{
		Provide: di.TypeOf[config.OptionSnapshot[T]](),
		UseClass: func() config.OptionSnapshot[T] {
			return config.NewOptionSnapshot[T](cache.Snapshot())
		},
		Scope: di.ScopeScoped,
	})

	ctx.logger.Info("Configured options",
		logging.Field{Key: "type", Value: di.TypeOf[T]().String()},
		logging.Field{Key: "section", Value: section})
}
