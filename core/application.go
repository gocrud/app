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
	Services() di.Container
	Configuration() config.Configuration
	Logger() logging.Logger
	Environment() Environment
	GetService(ptr any)
}

// ApplicationBuilder 应用程序构建器
type ApplicationBuilder struct {
	environment          string
	configBuilder        *config.ConfigurationBuilder
	loggingBuilder       *logging.LoggingBuilder
	serviceConfigurators []func(*ServiceCollection)
	configurators        []Configurator
	shutdownTimeout      time.Duration
	mu                   sync.RWMutex
}

// NewApplicationBuilder 创建应用程序构建器
func NewApplicationBuilder() *ApplicationBuilder {
	return &ApplicationBuilder{
		environment:          "development",
		configBuilder:        config.NewConfigurationBuilder(),
		loggingBuilder:       logging.NewLoggingBuilder(),
		serviceConfigurators: make([]func(*ServiceCollection), 0),
		configurators:        make([]Configurator, 0),
		shutdownTimeout:      30 * time.Second,
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
	if configure != nil {
		b.serviceConfigurators = append(b.serviceConfigurators, configure)
	}
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

// AddExtension 添加应用程序扩展
func (b *ApplicationBuilder) AddExtension(ext Extension) *ApplicationBuilder {
	validateExtension(ext)

	b.mu.Lock()
	defer b.mu.Unlock()

	// 1. 注册服务配置器
	if sc, ok := ext.(ServiceConfigurator); ok {
		b.serviceConfigurators = append(b.serviceConfigurators, sc.ConfigureServices)
	}

	// 2. 注册应用构建配置器
	if ac, ok := ext.(AppConfigurator); ok {
		b.configurators = append(b.configurators, ac.ConfigureBuilder)
	}

	return b
}

// AddOptions 注册配置选项（语法糖，简化配置选项注册）
// 使用示例: core.AddOptions[AppSetting](builder, "app")
func AddOptions[T any](b *ApplicationBuilder, section string) *ApplicationBuilder {
	return b.Configure(func(ctx *BuildContext) {
		ConfigureOptions[T](ctx, section)
	})
}

// AddTask 添加一个简单的后台任务
func (b *ApplicationBuilder) AddTask(task func(ctx context.Context) error) *ApplicationBuilder {
	b.Configure(func(ctx *BuildContext) {
		ctx.AddHostedService(&functionalService{task: task})
	})
	return b
}

// functionalService 函数式托管服务
type functionalService struct {
	task func(ctx context.Context) error
}

func (f *functionalService) Start(ctx context.Context) error {
	return f.task(ctx)
}

func (f *functionalService) Stop(ctx context.Context) error {
	return nil
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

	// 构建可重载的配置
	reloadableConfig, err := b.configBuilder.BuildReloadable()
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

	// 注册核心服务到容器
	// 1. Configuration (ReloadableConfig is the impl)
	di.Register[config.Configuration](container, di.Use[*config.ReloadableConfiguration](), di.WithValue(reloadableConfig), di.WithSingleton())
	di.Register[*config.ReloadableConfiguration](container, di.WithValue(reloadableConfig), di.WithSingleton())

	// 2. Logging
	di.Register[logging.LoggerFactory](container, di.WithValue(loggerFactory), di.WithSingleton())
	di.Register[logging.Logger](container, di.WithValue(logger), di.WithSingleton())

	// 3. Container itself
	di.Register[di.Container](container, di.WithValue(container), di.WithSingleton())

	// 创建服务集合
	services := &ServiceCollection{
		container: container,
		logger:    logger,
	}

	// 创建 BuildContext
	buildContext := &BuildContext{
		container:      container,
		configuration:  reloadableConfig,
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
	for _, configurator := range b.serviceConfigurators {
		configurator(services)
	}

	// 构建容器
	if err := container.Build(); err != nil {
		logger.Fatal("Failed to build DI container",
			logging.Field{Key: "error", Value: err.Error()})
	}

	logger.Info("DI container built successfully")

	// 合并托管服务
	// HostedService 通常有两种注册方式：
	// 1. 通过 ApplicationBuilder.AddTask 或 Configure(ctx.AddHostedService) 直接添加实例 (BuildContext)
	// 2. 通过 ConfigureServices 注册到 DI，并标记为 HostedService (ServiceCollection)
	injectedServices := make([]hosting.HostedService, 0)

	// 1. 来自 BuildContext (已经是实例)
	injectedServices = append(injectedServices, buildContext.hostedServices...)

	// 2. 来自 ServiceCollection (需要解析)
	for _, provider := range services.hostedServiceProviders {
		// 确定要解析的类型
		var serviceType reflect.Type
		providerValue := reflect.ValueOf(provider)

		if providerValue.Kind() == reflect.Func {
			// 构造函数：使用返回值类型
			funcType := providerValue.Type()
			if funcType.NumOut() > 0 {
				serviceType = funcType.Out(0)
			} else {
				logger.Warn("Constructor function has no return value, skipping hosted service")
				continue
			}
		} else {
			// 实例：使用实例的类型
			serviceType = reflect.TypeOf(provider)
		}

		logger.Debug("Retrieving hosted service from container",
			logging.Field{Key: "type", Value: serviceType.String()})

		// 解析服务
		injectedService, err := container.Get(serviceType)
		if err != nil {
			logger.Fatal("Failed to retrieve hosted service from container",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "type", Value: serviceType.String()})
		}

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
		configuration:   reloadableConfig,
		configBuilder:   b.configBuilder,
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
	container       di.Container
	configuration   *config.ReloadableConfiguration
	configBuilder   *config.ConfigurationBuilder
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

	// 启动配置源的监听（框架自动处理）
	sources := a.configBuilder.GetSources()

	for _, source := range sources {
		if err := source.StartWatch(a.runCtx, func() {
			// 配置源变更时，触发 Configuration 重载
			if err := a.configuration.Reload(); err != nil {
				a.logger.Error("Failed to reload configuration",
					logging.Field{Key: "error", Value: err.Error()})
			} else {
				a.logger.Info("Configuration reloaded successfully")
			}
		}); err != nil {
			a.logger.Warn("Failed to start config watch",
				logging.Field{Key: "source", Value: source.Name()},
				logging.Field{Key: "error", Value: err.Error()})
		}
	}

	// 创建托管服务管理器
	a.serviceManager = hosting.NewHostedServiceManager(a.logger)
	for _, service := range a.hostedServices {
		a.serviceManager.Add(service)
	}

	// 启动托管服务，使用可取消的 context
	// 获取错误通道
	errCh := a.serviceManager.StartAll(a.runCtx)

	a.logger.Info("Application started successfully")

	// 等待停止信号或错误
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	var runErr error

	select {
	case sig := <-sigCh:
		a.logger.Info("Received shutdown signal",
			logging.Field{Key: "signal", Value: sig.String()})
	case <-a.stopCh:
		a.logger.Info("Application stop requested")
	case <-ctx.Done():
		a.logger.Info("Context cancelled")
	case err := <-errCh:
		// 接收到服务启动失败的错误
		a.logger.Error("Hosted service failed, stopping application",
			logging.Field{Key: "error", Value: err.Error()})
		runErr = err
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

	// 停止配置监听
	a.logger.Info("Stopping configuration watches")
	configSources := a.configBuilder.GetSources()
	for _, source := range configSources {
		source.StopWatch()
	}

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

	return runErr
}

// Stop 停止应用程序
func (a *application) Stop(ctx context.Context) error {
	close(a.stopCh)
	return nil
}

// Services 获取服务容器
func (a *application) Services() di.Container {
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
	instance, err := a.container.Get(targetType)
	if err != nil {
		panic(fmt.Sprintf("app: failed to get service %s: %v", targetType.String(), err))
	}

	// 设置值
	elemValue.Set(reflect.ValueOf(instance))
}

// ServiceCollection 服务集合
type ServiceCollection struct {
	container              di.Container
	logger                 logging.Logger
	hostedServiceProviders []any // 存储构造函数或实例
}

// AddHostedService 添加托管服务（支持实例或构造函数）
func (s *ServiceCollection) AddHostedService(value any) {
	// Workaround: We can't fix this perfectly without a generic `AddHostedService[T]`.
	// Let's append to list and try to rely on it being registered elsewhere or fail.
	s.hostedServiceProviders = append(s.hostedServiceProviders, value)
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
