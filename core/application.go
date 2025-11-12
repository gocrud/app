package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	"github.com/gocrud/app/config"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/hosting"
	"github.com/gocrud/app/logging"
)

// Application 应用程序接口
type Application interface {
	Run() error
	RunAsync(ctx context.Context) error
	Stop(ctx context.Context) error
	Services() *di.Container
	Configuration() config.Configuration
	Logger() logging.Logger
	Environment() Environment
	GetService(ptr interface{})
}

// ApplicationBuilder 应用程序构建器
type ApplicationBuilder struct {
	environment         string
	configBuilder       *config.ConfigurationBuilder
	loggingBuilder      *logging.LoggingBuilder
	serviceConfigurator func(*ServiceCollection)
	configurators       []Configurator
	shutdownTimeout     time.Duration
	mu                  sync.RWMutex
}

// NewApplicationBuilder 创建应用程序构建器
func NewApplicationBuilder() *ApplicationBuilder {
	return &ApplicationBuilder{
		environment:     "development",
		configBuilder:   config.NewConfigurationBuilder(),
		loggingBuilder:  logging.NewLoggingBuilder(),
		configurators:   make([]Configurator, 0),
		shutdownTimeout: 30 * time.Second,
	}
}

// UseEnvironment 设置环境
func (b *ApplicationBuilder) UseEnvironment(env string) *ApplicationBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.environment = env
	return b
}

// ConfigureConfiguration 配置配置系统
func (b *ApplicationBuilder) ConfigureConfiguration(configure func(*config.ConfigurationBuilder)) *ApplicationBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	if configure != nil {
		configure(b.configBuilder)
	}
	return b
}

// ConfigureLogging 配置日志系统
func (b *ApplicationBuilder) ConfigureLogging(configure func(*logging.LoggingBuilder)) *ApplicationBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	if configure != nil {
		configure(b.loggingBuilder)
	}
	return b
}

// ConfigureServices 配置服务
func (b *ApplicationBuilder) ConfigureServices(configure func(*ServiceCollection)) *ApplicationBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.serviceConfigurator = configure
	return b
}

// Configure 添加配置器（支持链式调用和可变参数）
// 接受任何 func(*BuildContext) 类型的函数
func (b *ApplicationBuilder) Configure(configurators ...interface{}) *ApplicationBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, c := range configurators {
		// 尝试转换为 Configurator
		if fn, ok := c.(func(*BuildContext)); ok {
			b.configurators = append(b.configurators, fn)
		} else {
			panic(fmt.Sprintf("configurator must be func(*BuildContext), got %T", c))
		}
	}

	return b
}

// UseShutdownTimeout 设置关闭超时
func (b *ApplicationBuilder) UseShutdownTimeout(timeout time.Duration) *ApplicationBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.shutdownTimeout = timeout
	return b
}

// Build 构建应用程序
func (b *ApplicationBuilder) Build() Application {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 构建配置
	configuration, err := b.configBuilder.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to build configuration: %v", err))
	}

	// 构建日志工厂
	loggerFactory := b.loggingBuilder.Build()
	logger := loggerFactory.CreateLogger("Application")

	logger.Info("Building application",
		logging.Field{Key: "environment", Value: b.environment})

	// 创建 DI 容器
	container := di.NewContainer()

	// 注册核心服务到容器（按接口类型注册）
	container.ProvideValue(di.ValueProvider{
		Provide: di.TypeOf[config.Configuration](),
		Value:   configuration,
	})
	container.ProvideValue(di.ValueProvider{
		Provide: di.TypeOf[logging.LoggerFactory](),
		Value:   loggerFactory,
	})
	container.ProvideValue(di.ValueProvider{
		Provide: di.TypeOf[logging.Logger](),
		Value:   logger,
	})
	// 注册容器本身，以便服务可以注入容器
	container.ProvideValue(di.ValueProvider{
		Provide: di.TypeOf[di.Container](),
		Value:   container,
	})

	// 创建服务集合
	services := &ServiceCollection{
		container: container,
		logger:    logger,
	}

	// 创建 BuildContext
	buildContext := &BuildContext{
		container:      container,
		configuration:  configuration,
		logger:         logger,
		environment:    NewEnvironment(b.environment),
		hostedServices: make([]hosting.HostedService, 0),
		cleanups:       make(map[string]func()),
	}

	// 执行所有配置器
	for _, configurator := range b.configurators {
		configurator(buildContext)
	}

	// 配置用户服务
	if b.serviceConfigurator != nil {
		b.serviceConfigurator(services)
	}

	// 合并托管服务（将 HostedService 转换为 any）
	for _, hs := range buildContext.hostedServices {
		services.hostedServiceProviders = append(services.hostedServiceProviders, hs)
	}

	// 构建容器
	if err := container.Build(); err != nil {
		logger.Fatal("Failed to build DI container",
			logging.Field{Key: "error", Value: err.Error()})
	}

	logger.Info("DI container built successfully")

	// 从容器中获取所有 hosted services
	injectedServices := make([]hosting.HostedService, 0, len(services.hostedServiceProviders))

	for _, provider := range services.hostedServiceProviders {
		// 判断是构造函数还是实例
		providerValue := reflect.ValueOf(provider)
		var serviceType reflect.Type

		if providerValue.Kind() == reflect.Func {
			// 构造函数：使用返回值类型
			funcType := providerValue.Type()
			if funcType.NumOut() > 0 {
				serviceType = funcType.Out(0)
			} else {
				logger.Warn("Constructor function has no return value, skipping")
				continue
			}
		} else {
			// 实例：使用实例类型
			serviceType = reflect.TypeOf(provider)
		}

		logger.Debug("Retrieving hosted service from container",
			logging.Field{Key: "type", Value: serviceType.String()})

		injectedService, err := container.GetByType(serviceType)
		if err != nil {
			logger.Fatal("Failed to retrieve hosted service from container",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "type", Value: serviceType.String()})
		}

		logger.Debug("Successfully retrieved hosted service from container")
		hs, ok := injectedService.(hosting.HostedService)
		if !ok {
			logger.Fatal("Service does not implement HostedService interface",
				logging.Field{Key: "type", Value: serviceType.String()})
		}

		injectedServices = append(injectedServices, hs)
	}

	// 创建应用程序
	app := &application{
		container:       container,
		configuration:   configuration,
		logger:          logger,
		environment:     NewEnvironment(b.environment),
		hostedServices:  injectedServices,
		cleanups:        buildContext.cleanups,
		shutdownTimeout: b.shutdownTimeout,
		stopCh:          make(chan struct{}),
	}

	return app
}

// application 应用程序实现
type application struct {
	container       *di.Container
	configuration   config.Configuration
	logger          logging.Logger
	environment     Environment
	hostedServices  []hosting.HostedService
	serviceManager  *hosting.HostedServiceManager
	cleanups        map[string]func()
	shutdownTimeout time.Duration
	stopCh          chan struct{}
	running         bool
	runCtx          context.Context
	runCancel       context.CancelFunc
	mu              sync.RWMutex
}

// Run 运行应用程序（阻塞）
func (a *application) Run() error {
	return a.RunAsync(context.Background())
}

// RunAsync 异步运行应用程序
func (a *application) RunAsync(ctx context.Context) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return errors.New("application is already running")
	}
	a.running = true

	// 创建可取消的 context 用于运行服务
	a.runCtx, a.runCancel = context.WithCancel(ctx)
	a.mu.Unlock()

	a.logger.Info("Starting application",
		logging.Field{Key: "environment", Value: a.environment.Name()})

	// 创建托管服务管理器
	a.serviceManager = hosting.NewHostedServiceManager(a.logger)
	for _, service := range a.hostedServices {
		a.serviceManager.Add(service)
	}

	// 启动托管服务，使用可取消的 context
	if err := a.serviceManager.StartAll(a.runCtx); err != nil {
		a.logger.Error("Failed to start hosted services",
			logging.Field{Key: "error", Value: err.Error()})
		return err
	}

	a.logger.Info("Application started successfully")

	// 等待停止信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		a.logger.Info("Received shutdown signal",
			logging.Field{Key: "signal", Value: sig.String()})
	case <-a.stopCh:
		a.logger.Info("Application stop requested")
	case <-ctx.Done():
		a.logger.Info("Context cancelled")
	}

	// 优雅关闭
	a.logger.Info("Shutting down application",
		logging.Field{Key: "timeout", Value: a.shutdownTimeout.String()})

	// 取消运行 context，通知所有服务停止
	a.runCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()

	// 停止托管服务
	if err := a.serviceManager.StopAll(shutdownCtx); err != nil {
		a.logger.Error("Failed to stop hosted services",
			logging.Field{Key: "error", Value: err.Error()})
	}

	// 等待所有服务完成
	a.serviceManager.Wait()

	// 执行所有清理函数
	if len(a.cleanups) > 0 {
		a.logger.Info("Running cleanup functions",
			logging.Field{Key: "count", Value: len(a.cleanups)})
		for key, cleanup := range a.cleanups {
			a.logger.Debug("Running cleanup",
				logging.Field{Key: "key", Value: key})
			cleanup()
		}
	}

	a.logger.Info("Application stopped")

	a.mu.Lock()
	a.running = false
	a.mu.Unlock()

	return nil
}

// Stop 停止应用程序
func (a *application) Stop(ctx context.Context) error {
	close(a.stopCh)
	return nil
}

// Services 获取服务容器
func (a *application) Services() *di.Container {
	return a.container
}

// Configuration 获取配置
func (a *application) Configuration() config.Configuration {
	return a.configuration
}

// Logger 获取日志记录器
func (a *application) Logger() logging.Logger {
	return a.logger
}

// Environment 获取环境
func (a *application) Environment() Environment {
	return a.environment
}

// GetService 获取服务实例（通过指针参数）
//
// 使用示例：
//
//	var myService *MyService
//	app.GetService(&myService)
func (a *application) GetService(ptr any) {
	// 检查参数是否为指针
	ptrValue := reflect.ValueOf(ptr)
	if ptrValue.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("app: GetService argument must be a pointer, got %T", ptr))
	}

	// 获取指针指向的类型
	elemValue := ptrValue.Elem()
	if !elemValue.CanSet() {
		panic("app: GetService argument must be settable")
	}

	// 获取目标类型
	targetType := elemValue.Type()

	// 从容器获取服务实例
	instance, err := a.container.GetByType(targetType)
	if err != nil {
		panic(fmt.Sprintf("app: failed to get service %s: %v", targetType.String(), err))
	}

	// 设置值
	elemValue.Set(reflect.ValueOf(instance))
}

// ServiceCollection 服务集合
type ServiceCollection struct {
	container              *di.Container
	logger                 logging.Logger
	hostedServiceProviders []any // 存储构造函数或实例
}

// AddSingleton 注册单例服务
// 单例服务在整个应用程序生命周期内只创建一次实例，所有获取操作返回同一个实例
// 适用场景：无状态服务、配置、日志记录器等
func (s *ServiceCollection) AddSingleton(value any) {
	s.addWithScope(value, di.ScopeSingleton)
}

// AddScoped 注册作用域服务
// 作用域服务在同一个 Scope 内只创建一次实例，不同 Scope 之间实例相互独立
// 适用场景：HTTP 请求级别的服务、数据库连接、工作单元等
func (s *ServiceCollection) AddScoped(value any) {
	s.addWithScope(value, di.ScopeScoped)
}

// AddTransient 注册瞬态服务
// 瞬态服务每次获取都创建新实例，不缓存
// 适用场景：命令对象、事件对象等需要独立状态的对象
func (s *ServiceCollection) AddTransient(value any) {
	s.addWithScope(value, di.ScopeTransient)
}

// addWithScope 内部方法：使用指定作用域注册服务
func (s *ServiceCollection) addWithScope(value any, scope di.ScopeType) {
	// 判断是构造函数还是实例
	val := reflect.ValueOf(value)

	if val.Kind() == reflect.Func {
		// 构造函数
		funcType := val.Type()
		if funcType.NumOut() == 0 {
			s.logger.Fatal("Constructor function must return at least one value")
			return
		}

		// 使用构造函数返回值类型作为提供类型
		config := di.ProviderConfig{
			Provide:  funcType.Out(0),
			UseClass: value,
			Scope:    scope,
		}

		s.container.ProvideWithConfig(config)
	} else {
		// 实例：使用 ValueProvider
		s.container.ProvideValue(di.ValueProvider{
			Provide: reflect.TypeOf(value),
			Value:   value,
			Options: di.ProviderOptions{
				Scope: scope,
			},
		})
	}
}

// AddHostedService 添加托管服务（支持实例或构造函数）
func (s *ServiceCollection) AddHostedService(value any) {
	// 注册到容器以支持依赖注入
	s.container.Provide(value)

	// 存储提供者，稍后从容器获取实例
	s.hostedServiceProviders = append(s.hostedServiceProviders, value)
}

// Bind 绑定接口到实现
func (s *ServiceCollection) Bind(provider di.TypeProvider) {
	s.container.ProvideType(provider)
}

// Environment 环境接口
type Environment interface {
	Name() string
	IsDevelopment() bool
	IsProduction() bool
	IsStaging() bool
}

// environment 环境实现
type environment struct {
	name string
}

// NewEnvironment 创建环境
func NewEnvironment(name string) Environment {
	return &environment{name: name}
}

func (e *environment) Name() string {
	return e.name
}

func (e *environment) IsDevelopment() bool {
	return e.name == "development"
}

func (e *environment) IsProduction() bool {
	return e.name == "production"
}

func (e *environment) IsStaging() bool {
	return e.name == "staging"
}
