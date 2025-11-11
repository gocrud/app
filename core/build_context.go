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
	container *di.Container

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
func (c *BuildContext) GetContainer() *di.Container {
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
