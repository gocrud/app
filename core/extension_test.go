package core

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gocrud/app/config"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
)

// 定义各种 Extension 实现用于测试

// EmptyExtension 未实现任何接口
type EmptyExtension struct{}

func (e *EmptyExtension) Name() string { return "Empty" }

// ServiceOnlyExtension 仅实现 ServiceConfigurator
type ServiceOnlyExtension struct{}

func (e *ServiceOnlyExtension) Name() string                           { return "ServiceOnly" }
func (e *ServiceOnlyExtension) ConfigureServices(s *ServiceCollection) {}

// AppOnlyExtension 仅实现 AppConfigurator
type AppOnlyExtension struct{}

func (e *AppOnlyExtension) Name() string                       { return "AppOnly" }
func (e *AppOnlyExtension) ConfigureBuilder(ctx *BuildContext) {}

// FullExtension 同时实现 ServiceConfigurator 和 AppConfigurator
type FullExtension struct{}

func (e *FullExtension) Name() string                           { return "Full" }
func (e *FullExtension) ConfigureServices(s *ServiceCollection) {}
func (e *FullExtension) ConfigureBuilder(ctx *BuildContext)     {}

func TestAddExtension_Panic_WhenNoInterfaceImplemented(t *testing.T) {
	builder := NewApplicationBuilder()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic as expected for EmptyExtension")
		} else {
			// 验证 panic 消息包含 Extension 名称
			errStr := r.(string)
			expectedPart := "Extension 'Empty' does not implement any supported interfaces"
			if len(errStr) < len(expectedPart) || errStr[:len(expectedPart)] != expectedPart {
				// 简单的包含检查
				importStrings := "Extension 'Empty' does not implement any supported interfaces"
				if !contains(errStr, importStrings) {
					t.Errorf("Panic message not match. Got: %v", errStr)
				}
			}
		}
	}()

	builder.AddExtension(&EmptyExtension{})
}

func TestAddExtension_Success_ServiceOnly(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&ServiceOnlyExtension{})

	if len(builder.serviceConfigurators) != 1 {
		t.Errorf("Expected 1 service configurator, got %d", len(builder.serviceConfigurators))
	}
	if len(builder.configurators) != 0 {
		t.Errorf("Expected 0 app configurators, got %d", len(builder.configurators))
	}
}

func TestAddExtension_Success_AppOnly(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&AppOnlyExtension{})

	if len(builder.serviceConfigurators) != 0 {
		t.Errorf("Expected 0 service configurators, got %d", len(builder.serviceConfigurators))
	}
	if len(builder.configurators) != 1 {
		t.Errorf("Expected 1 app configurator, got %d", len(builder.configurators))
	}
}

func TestAddExtension_Success_Full(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&FullExtension{})

	if len(builder.serviceConfigurators) != 1 {
		t.Errorf("Expected 1 service configurator, got %d", len(builder.serviceConfigurators))
	}
	if len(builder.configurators) != 1 {
		t.Errorf("Expected 1 app configurator, got %d", len(builder.configurators))
	}
}

func TestAddExtension_Multiple(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&ServiceOnlyExtension{})
	builder.AddExtension(&AppOnlyExtension{})
	builder.AddExtension(&FullExtension{})

	if len(builder.serviceConfigurators) != 2 { // ServiceOnly + Full
		t.Errorf("Expected 2 service configurators, got %d", len(builder.serviceConfigurators))
	}
	if len(builder.configurators) != 2 { // AppOnly + Full
		t.Errorf("Expected 2 app configurators, got %d", len(builder.configurators))
	}
}

// 简单的字符串包含辅助函数
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ===== 服务扩展功能测试 =====

// 测试用的服务接口和实现
type ITestService interface {
	GetValue() string
}

type TestServiceImpl struct {
	value string
}

func (s *TestServiceImpl) GetValue() string {
	return s.value
}

func NewTestService() *TestServiceImpl {
	return &TestServiceImpl{value: "test"}
}

// 带依赖注入的服务
type IRepository interface {
	Save(data string) error
}

type RepositoryImpl struct{}

func (r *RepositoryImpl) Save(data string) error {
	return nil
}

type IBusinessService interface {
	Process() string
}

type BusinessService struct {
	Repo IRepository `di:""`
}

func (s *BusinessService) Process() string {
	return "processed"
}

// TestAddSingleton 测试单例服务注册
func TestAddSingleton(t *testing.T) {
	services := &ServiceCollection{
		container: di.NewContainer(),
	}

	// 测试使用工厂函数注册单例
	AddSingleton[ITestService](services, di.WithFactory(func() ITestService {
		return &TestServiceImpl{value: "singleton"}
	}))

	// 构建容器
	if err := services.container.Build(); err != nil {
		t.Fatalf("Failed to build container: %v", err)
	}

	// 解析服务
	instance1, err := services.container.Get(reflect.TypeOf((*ITestService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve singleton service: %v", err)
	}

	instance2, err := services.container.Get(reflect.TypeOf((*ITestService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve singleton service second time: %v", err)
	}

	// 验证是同一个实例（单例）
	svc1 := instance1.(ITestService)
	svc2 := instance2.(ITestService)

	if svc1.GetValue() != "singleton" {
		t.Errorf("Expected value 'singleton', got '%s'", svc1.GetValue())
	}

	// 验证指针相同（真正的单例）
	if reflect.ValueOf(svc1).Pointer() != reflect.ValueOf(svc2).Pointer() {
		t.Error("Expected same instance for singleton, got different instances")
	}
}

// TestAddTransient 测试瞬态服务注册
func TestAddTransient(t *testing.T) {
	services := &ServiceCollection{
		container: di.NewContainer(),
	}

	// 注册瞬态服务
	AddTransient[ITestService](services, di.WithFactory(func() ITestService {
		return &TestServiceImpl{value: "transient"}
	}))

	// 构建容器
	if err := services.container.Build(); err != nil {
		t.Fatalf("Failed to build container: %v", err)
	}

	// 解析服务两次
	instance1, err := services.container.Get(reflect.TypeOf((*ITestService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve transient service: %v", err)
	}

	instance2, err := services.container.Get(reflect.TypeOf((*ITestService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve transient service second time: %v", err)
	}

	svc1 := instance1.(ITestService)
	svc2 := instance2.(ITestService)

	// 验证每次都是新实例
	if reflect.ValueOf(svc1).Pointer() == reflect.ValueOf(svc2).Pointer() {
		t.Error("Expected different instances for transient, got same instance")
	}

	if svc1.GetValue() != "transient" || svc2.GetValue() != "transient" {
		t.Error("Transient services should have correct values")
	}
}

// TestAddScoped 测试作用域服务注册
func TestAddScoped(t *testing.T) {
	services := &ServiceCollection{
		container: di.NewContainer(),
	}

	// 注册作用域服务
	AddScoped[ITestService](services, di.WithFactory(func() ITestService {
		return &TestServiceImpl{value: "scoped"}
	}))

	// 构建容器
	if err := services.container.Build(); err != nil {
		t.Fatalf("Failed to build container: %v", err)
	}

	// 创建作用域1
	scope1 := services.container.CreateScope()
	instance1a, err := scope1.Get(reflect.TypeOf((*ITestService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve scoped service in scope1: %v", err)
	}
	instance1b, err := scope1.Get(reflect.TypeOf((*ITestService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve scoped service in scope1 second time: %v", err)
	}

	// 创建作用域2
	scope2 := services.container.CreateScope()
	instance2, err := scope2.Get(reflect.TypeOf((*ITestService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve scoped service in scope2: %v", err)
	}

	svc1a := instance1a.(ITestService)
	svc1b := instance1b.(ITestService)
	svc2 := instance2.(ITestService)

	// 验证同一作用域内是同一实例
	if reflect.ValueOf(svc1a).Pointer() != reflect.ValueOf(svc1b).Pointer() {
		t.Error("Expected same instance within same scope")
	}

	// 验证不同作用域是不同实例
	if reflect.ValueOf(svc1a).Pointer() == reflect.ValueOf(svc2).Pointer() {
		t.Error("Expected different instances across different scopes")
	}

	if svc1a.GetValue() != "scoped" || svc2.GetValue() != "scoped" {
		t.Error("Scoped services should have correct values")
	}
}

// TestConvertImplToOptions 测试实现转换为选项
func TestConvertImplToOptions(t *testing.T) {
	t.Run("NilImpl", func(t *testing.T) {
		opts := convertImplToOptions(nil)
		if opts != nil {
			t.Error("Expected nil options for nil impl")
		}
	})

	t.Run("FunctionImpl", func(t *testing.T) {
		factoryFunc := func() *TestServiceImpl {
			return &TestServiceImpl{value: "factory"}
		}
		opts := convertImplToOptions(factoryFunc)
		if len(opts) != 1 {
			t.Errorf("Expected 1 option for function impl, got %d", len(opts))
		}
		// 验证是工厂选项（通过应用到容器来验证）
		container := di.NewContainer()
		di.Register[*TestServiceImpl](container, opts...)
		if err := container.Build(); err != nil {
			t.Fatalf("Failed to build container: %v", err)
		}
		instance, err := container.Get(reflect.TypeOf((*TestServiceImpl)(nil)))
		if err != nil {
			t.Fatalf("Failed to resolve factory-registered service: %v", err)
		}
		svc := instance.(*TestServiceImpl)
		if svc.GetValue() != "factory" {
			t.Errorf("Expected 'factory', got '%s'", svc.GetValue())
		}
	})

	t.Run("ValueImpl", func(t *testing.T) {
		value := &TestServiceImpl{value: "value"}
		opts := convertImplToOptions(value)
		if len(opts) != 1 {
			t.Errorf("Expected 1 option for value impl, got %d", len(opts))
		}
		// 验证是值选项
		container := di.NewContainer()
		di.Register[*TestServiceImpl](container, opts...)
		if err := container.Build(); err != nil {
			t.Fatalf("Failed to build container: %v", err)
		}
		instance, err := container.Get(reflect.TypeOf((*TestServiceImpl)(nil)))
		if err != nil {
			t.Fatalf("Failed to resolve value-registered service: %v", err)
		}
		svc := instance.(*TestServiceImpl)
		if svc.GetValue() != "value" {
			t.Errorf("Expected 'value', got '%s'", svc.GetValue())
		}
	})
}

// 用于测试服务注册的扩展
type ServiceRegistrationExtension struct{}

func (e *ServiceRegistrationExtension) Name() string {
	return "ServiceRegistrationExtension"
}

func (e *ServiceRegistrationExtension) ConfigureServices(s *ServiceCollection) {
	// 注册仓储服务
	AddSingleton[IRepository](s, di.Use[*RepositoryImpl]())
	// 注册业务服务（依赖仓储）
	AddTransient[IBusinessService](s, di.Use[*BusinessService](), di.WithFields())
}

// TestExtensionWithServiceRegistration 测试扩展中的服务注册
func TestExtensionWithServiceRegistration(t *testing.T) {
	// 创建应用构建器
	builder := NewApplicationBuilder()
	builder.AddExtension(&ServiceRegistrationExtension{})

	// 构建应用
	app := builder.Build()

	// 解析并验证服务
	bizSvcInstance, err := app.Services().Get(reflect.TypeOf((*IBusinessService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve business service: %v", err)
	}

	bizSvc := bizSvcInstance.(IBusinessService)
	result := bizSvc.Process()
	if result != "processed" {
		t.Errorf("Expected 'processed', got '%s'", result)
	}

	// 验证依赖注入是否工作
	if bizSvc.(*BusinessService).Repo == nil {
		t.Error("Expected repository to be injected")
	}
}

// 扩展1：注册基础服务
type Extension1 struct{}

func (e *Extension1) Name() string { return "Extension1" }
func (e *Extension1) ConfigureServices(s *ServiceCollection) {
	AddSingleton[IRepository](s, di.Use[*RepositoryImpl]())
}

// 扩展2：注册依赖基础服务的业务服务
type Extension2 struct{}

func (e *Extension2) Name() string { return "Extension2" }
func (e *Extension2) ConfigureServices(s *ServiceCollection) {
	AddTransient[IBusinessService](s, di.Use[*BusinessService](), di.WithFields())
}

// 扩展3：配置应用
type Extension3 struct {
	configured bool
}

func (e *Extension3) Name() string { return "Extension3" }
func (e *Extension3) ConfigureBuilder(ctx *BuildContext) {
	e.configured = true
}

// TestMultipleExtensionsIntegration 测试多个扩展的集成
func TestMultipleExtensionsIntegration(t *testing.T) {
	// 创建并配置应用
	builder := NewApplicationBuilder()
	ext3 := &Extension3{}
	builder.AddExtension(&Extension1{})
	builder.AddExtension(&Extension2{})
	builder.AddExtension(ext3)

	app := builder.Build()

	// 验证服务注册成功
	_, err := app.Services().Get(reflect.TypeOf((*IRepository)(nil)).Elem())
	if err != nil {
		t.Errorf("Failed to resolve repository from Extension1: %v", err)
	}

	_, err = app.Services().Get(reflect.TypeOf((*IBusinessService)(nil)).Elem())
	if err != nil {
		t.Errorf("Failed to resolve business service from Extension2: %v", err)
	}

	// 验证应用配置器被调用
	if !ext3.configured {
		t.Error("Expected Extension3 ConfigureBuilder to be called")
	}
}

// 用于测试不同生命周期的扩展
type LifetimeExtension struct{}

func (e *LifetimeExtension) Name() string { return "LifetimeExtension" }
func (e *LifetimeExtension) ConfigureServices(s *ServiceCollection) {
	// 单例
	AddSingleton[ITestService](s,
		di.WithFactory(func() ITestService {
			return &TestServiceImpl{value: "singleton"}
		}),
		di.WithName("singleton"),
	)
	// 瞬态
	AddTransient[ITestService](s,
		di.WithFactory(func() ITestService {
			return &TestServiceImpl{value: "transient"}
		}),
		di.WithName("transient"),
	)
	// 作用域
	AddScoped[ITestService](s,
		di.WithFactory(func() ITestService {
			return &TestServiceImpl{value: "scoped"}
		}),
		di.WithName("scoped"),
	)
}

// TestExtensionServiceLifetime 测试扩展中不同生命周期的服务
func TestExtensionServiceLifetime(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&LifetimeExtension{})
	app := builder.Build()

	serviceType := reflect.TypeOf((*ITestService)(nil)).Elem()

	// 测试单例
	s1, _ := app.Services().GetNamed(serviceType, "singleton")
	s2, _ := app.Services().GetNamed(serviceType, "singleton")
	if reflect.ValueOf(s1).Pointer() != reflect.ValueOf(s2).Pointer() {
		t.Error("Singleton should return same instance")
	}

	// 测试瞬态
	t1, _ := app.Services().GetNamed(serviceType, "transient")
	t2, _ := app.Services().GetNamed(serviceType, "transient")
	if reflect.ValueOf(t1).Pointer() == reflect.ValueOf(t2).Pointer() {
		t.Error("Transient should return different instances")
	}

	// 测试作用域
	scope := app.Services().CreateScope()
	sc1, _ := scope.GetNamed(serviceType, "scoped")
	sc2, _ := scope.GetNamed(serviceType, "scoped")
	if reflect.ValueOf(sc1).Pointer() != reflect.ValueOf(sc2).Pointer() {
		t.Error("Scoped should return same instance within scope")
	}
}

// ===== 边界情况和错误处理测试 =====

// ErrorProneExtension 用于测试错误处理
type ErrorProneExtension struct {
	shouldPanic bool
}

func (e *ErrorProneExtension) Name() string { return "ErrorProneExtension" }

func (e *ErrorProneExtension) ConfigureServices(s *ServiceCollection) {
	if e.shouldPanic {
		panic("simulated error in ConfigureServices")
	}
	AddSingleton[ITestService](s, di.WithFactory(func() ITestService {
		return &TestServiceImpl{value: "error-prone"}
	}))
}

// TestExtensionErrorHandling 测试扩展中的错误处理
func TestExtensionErrorHandling(t *testing.T) {
	t.Run("PanicInConfigureServices", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic during ConfigureServices")
			}
		}()

		builder := NewApplicationBuilder()
		builder.AddExtension(&ErrorProneExtension{shouldPanic: true})
		builder.Build() // 应该在这里触发 panic
	})

	t.Run("NormalExtension", func(t *testing.T) {
		builder := NewApplicationBuilder()
		builder.AddExtension(&ErrorProneExtension{shouldPanic: false})
		app := builder.Build()

		svc, err := app.Services().Get(reflect.TypeOf((*ITestService)(nil)).Elem())
		if err != nil {
			t.Fatalf("Failed to resolve service: %v", err)
		}
		if svc.(ITestService).GetValue() != "error-prone" {
			t.Error("Service value mismatch")
		}
	})
}

// OrderTrackingExtension 用于测试执行顺序
type OrderTrackingExtension struct {
	name         string
	order        *[]string
	servicePhase bool
	builderPhase bool
}

func (e *OrderTrackingExtension) Name() string { return e.name }

func (e *OrderTrackingExtension) ConfigureServices(s *ServiceCollection) {
	if e.servicePhase {
		*e.order = append(*e.order, "service:"+e.name)
	}
}

func (e *OrderTrackingExtension) ConfigureBuilder(ctx *BuildContext) {
	if e.builderPhase {
		*e.order = append(*e.order, "builder:"+e.name)
	}
}

// TestExtensionExecutionOrder 测试扩展的执行顺序
func TestExtensionExecutionOrder(t *testing.T) {
	order := []string{}

	builder := NewApplicationBuilder()
	builder.AddExtension(&OrderTrackingExtension{
		name: "ext1", order: &order, servicePhase: true, builderPhase: true,
	})
	builder.AddExtension(&OrderTrackingExtension{
		name: "ext2", order: &order, servicePhase: true, builderPhase: true,
	})
	builder.AddExtension(&OrderTrackingExtension{
		name: "ext3", order: &order, servicePhase: true, builderPhase: true,
	})

	builder.Build()

	// 验证执行顺序：所有 ConfigureBuilder 应该先于 ConfigureServices
	// 这是框架的设计：先配置应用构建上下文，再注册服务
	expectedOrder := []string{
		"builder:ext1", "builder:ext2", "builder:ext3",
		"service:ext1", "service:ext2", "service:ext3",
	}

	if len(order) != len(expectedOrder) {
		t.Fatalf("Expected %d execution steps, got %d", len(expectedOrder), len(order))
	}

	for i, expected := range expectedOrder {
		if order[i] != expected {
			t.Errorf("Order mismatch at step %d: expected %s, got %s", i, expected, order[i])
		}
	}
}

// NamedServiceExtension 用于测试命名服务
type NamedServiceExtension struct {
	serviceName  string
	serviceValue string
}

func (e *NamedServiceExtension) Name() string { return "NamedService" }

func (e *NamedServiceExtension) ConfigureServices(s *ServiceCollection) {
	AddSingleton[ITestService](s,
		di.WithFactory(func() ITestService {
			return &TestServiceImpl{value: e.serviceValue}
		}),
		di.WithName(e.serviceName),
	)
}

// TestExtensionNamedServices 测试扩展注册命名服务
func TestExtensionNamedServices(t *testing.T) {
	builder := NewApplicationBuilder()

	// 注册多个命名服务
	builder.AddExtension(&NamedServiceExtension{serviceName: "primary", serviceValue: "primary-value"})
	builder.AddExtension(&NamedServiceExtension{serviceName: "secondary", serviceValue: "secondary-value"})

	app := builder.Build()

	serviceType := reflect.TypeOf((*ITestService)(nil)).Elem()

	// 解析不同的命名服务
	primary, err := app.Services().GetNamed(serviceType, "primary")
	if err != nil {
		t.Fatalf("Failed to resolve primary service: %v", err)
	}
	if primary.(ITestService).GetValue() != "primary-value" {
		t.Errorf("Expected 'primary-value', got '%s'", primary.(ITestService).GetValue())
	}

	secondary, err := app.Services().GetNamed(serviceType, "secondary")
	if err != nil {
		t.Fatalf("Failed to resolve secondary service: %v", err)
	}
	if secondary.(ITestService).GetValue() != "secondary-value" {
		t.Errorf("Expected 'secondary-value', got '%s'", secondary.(ITestService).GetValue())
	}
}

// ComplexDependencyExtension 测试复杂依赖关系
type ILogger interface {
	Log(msg string)
}

type Logger struct {
	prefix string
}

func (l *Logger) Log(msg string) {}

type ICache interface {
	Get(key string) any
}

type Cache struct {
	Logger ILogger `di:""`
}

func (c *Cache) Get(key string) any { return nil }

type IUserService interface {
	GetUser(id int) string
}

type UserService struct {
	Cache  ICache      `di:""`
	Logger ILogger     `di:""`
	Repo   IRepository `di:""`
}

func (s *UserService) GetUser(id int) string { return "user" }

type ComplexDependencyExtension struct{}

func (e *ComplexDependencyExtension) Name() string { return "ComplexDependency" }

func (e *ComplexDependencyExtension) ConfigureServices(s *ServiceCollection) {
	// 注册基础服务
	AddSingleton[ILogger](s, di.WithFactory(func() ILogger {
		return &Logger{prefix: "APP"}
	}))
	AddSingleton[IRepository](s, di.Use[*RepositoryImpl]())

	// 注册依赖基础服务的服务
	AddSingleton[ICache](s, di.Use[*Cache](), di.WithFields())

	// 注册依赖多个服务的复杂服务
	AddTransient[IUserService](s, di.Use[*UserService](), di.WithFields())
}

// TestComplexDependencyInjection 测试复杂的依赖注入链
func TestComplexDependencyInjection(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&ComplexDependencyExtension{})
	app := builder.Build()

	// 解析最顶层的服务
	userSvc, err := app.Services().Get(reflect.TypeOf((*IUserService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve UserService: %v", err)
	}

	// 验证所有依赖都被正确注入
	us := userSvc.(*UserService)
	if us.Logger == nil {
		t.Error("Logger should be injected")
	}
	if us.Cache == nil {
		t.Error("Cache should be injected")
	}
	if us.Repo == nil {
		t.Error("Repository should be injected")
	}

	// 验证嵌套依赖
	if us.Cache.(*Cache).Logger == nil {
		t.Error("Logger should be injected into Cache")
	}
}

// EmptyConfigurationExtension 测试空配置的扩展
type EmptyConfigurationExtension struct{}

func (e *EmptyConfigurationExtension) Name() string { return "EmptyConfiguration" }

func (e *EmptyConfigurationExtension) ConfigureServices(s *ServiceCollection) {
	// 不做任何事
}

func (e *EmptyConfigurationExtension) ConfigureBuilder(ctx *BuildContext) {
	// 不做任何事
}

// TestEmptyExtension 测试空配置的扩展不会引起问题
func TestEmptyExtension(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&EmptyConfigurationExtension{})

	// 应该能正常构建
	app := builder.Build()
	if app == nil {
		t.Error("Expected application to be built successfully")
	}
}

// NilPointerExtension 测试 nil 指针处理
type ServiceWithOptionalDep struct {
	OptionalSvc ITestService `di:"optional"`
}

type NilPointerExtension struct{}

func (e *NilPointerExtension) Name() string { return "NilPointer" }

func (e *NilPointerExtension) ConfigureServices(s *ServiceCollection) {
	// 注册一个带可选依赖的服务，但不注册依赖本身
	AddSingleton[*ServiceWithOptionalDep](s, di.Use[*ServiceWithOptionalDep](), di.WithFields())
}

// TestOptionalDependencyHandling 测试可选依赖处理
func TestOptionalDependencyHandling(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&NilPointerExtension{})
	app := builder.Build()

	// 应该能成功解析，即使可选依赖不存在
	svc, err := app.Services().Get(reflect.TypeOf((*ServiceWithOptionalDep)(nil)))
	if err != nil {
		t.Fatalf("Failed to resolve service with optional dependency: %v", err)
	}

	// 可选依赖应该为 nil
	if svc.(*ServiceWithOptionalDep).OptionalSvc != nil {
		t.Error("Optional dependency should be nil when not registered")
	}
}

// ===== ApplicationBuilder 方法测试 =====

// TestUseEnvironment 测试环境设置
func TestUseEnvironment(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.UseEnvironment("production")
	app := builder.Build()

	env := app.Environment()
	if !env.IsProduction() {
		t.Error("Expected production environment")
	}
	if env.Name() != "production" {
		t.Errorf("Expected 'production', got '%s'", env.Name())
	}
}

// TestConfigureConfiguration 测试配置系统配置
func TestConfigureConfiguration(t *testing.T) {
	builder := NewApplicationBuilder()
	configured := false

	builder.ConfigureConfiguration(func(cb *config.ConfigurationBuilder) {
		configured = true
		// 添加内存配置源
		cb.AddInMemory(map[string]any{
			"test_key": "test_value",
		})
	})

	app := builder.Build()

	if !configured {
		t.Error("Configuration callback was not called")
	}

	// 验证配置是否生效
	cfg := app.Configuration()
	value := cfg.Get("test_key")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}
}

// TestConfigureLogging 测试日志系统配置
func TestConfigureLogging(t *testing.T) {
	builder := NewApplicationBuilder()
	configured := false

	builder.ConfigureLogging(func(lb *logging.LoggingBuilder) {
		configured = true
		lb.SetMinimumLevel(logging.LogLevelInfo)
	})

	app := builder.Build()

	if !configured {
		t.Error("Logging configuration callback was not called")
	}

	logger := app.Logger()
	if logger == nil {
		t.Error("Logger should not be nil")
	}
}

// TestConfigureServices 测试服务配置
func TestConfigureServices(t *testing.T) {
	builder := NewApplicationBuilder()

	builder.ConfigureServices(func(s *ServiceCollection) {
		AddSingleton[ITestService](s, di.WithFactory(func() ITestService {
			return &TestServiceImpl{value: "configured"}
		}))
	})

	app := builder.Build()

	// 解析服务
	svc, err := app.Services().Get(reflect.TypeOf((*ITestService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve service: %v", err)
	}

	if svc.(ITestService).GetValue() != "configured" {
		t.Errorf("Expected 'configured', got '%s'", svc.(ITestService).GetValue())
	}
}

// TestConfigure 测试配置器
func TestConfigure(t *testing.T) {
	builder := NewApplicationBuilder()
	configuredCount := 0

	builder.Configure(
		func(ctx *BuildContext) {
			configuredCount++
		},
		func(ctx *BuildContext) {
			configuredCount++
		},
	)

	builder.Build()

	if configuredCount != 2 {
		t.Errorf("Expected 2 configurators to be called, got %d", configuredCount)
	}
}

// TestUseShutdownTimeout 测试关闭超时设置
func TestUseShutdownTimeout(t *testing.T) {
	builder := NewApplicationBuilder()
	timeout := 10 * time.Second
	builder.UseShutdownTimeout(timeout)
	app := builder.Build()

	// 验证应用能够构建成功
	if app == nil {
		t.Error("Expected application to be built successfully")
	}
}

// TestApplicationLogger 测试应用日志器
func TestApplicationLogger(t *testing.T) {
	builder := NewApplicationBuilder()
	app := builder.Build()

	logger := app.Logger()
	if logger == nil {
		t.Error("Logger should not be nil")
	}
}

// TestApplicationConfiguration 测试应用配置
func TestApplicationConfiguration(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.ConfigureConfiguration(func(cb *config.ConfigurationBuilder) {
		cb.AddInMemory(map[string]any{
			"app_key": "app_value",
		})
	})

	app := builder.Build()

	cfg := app.Configuration()
	if cfg == nil {
		t.Error("Configuration should not be nil")
	}

	if cfg.Get("app_key") != "app_value" {
		t.Errorf("Expected 'app_value', got '%s'", cfg.Get("app_key"))
	}
}

// TestApplicationEnvironment 测试应用环境
func TestApplicationEnvironment(t *testing.T) {
	testCases := []struct {
		name    string
		env     string
		isDev   bool
		isProd  bool
		isStage bool
	}{
		{"Development", "development", true, false, false},
		{"Production", "production", false, true, false},
		{"Staging", "staging", false, false, true},
		{"Custom", "custom", false, false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewApplicationBuilder()
			builder.UseEnvironment(tc.env)
			app := builder.Build()

			env := app.Environment()
			if env.Name() != tc.env {
				t.Errorf("Expected environment '%s', got '%s'", tc.env, env.Name())
			}
			if env.IsDevelopment() != tc.isDev {
				t.Errorf("IsDevelopment() = %v, want %v", env.IsDevelopment(), tc.isDev)
			}
			if env.IsProduction() != tc.isProd {
				t.Errorf("IsProduction() = %v, want %v", env.IsProduction(), tc.isProd)
			}
			if env.IsStaging() != tc.isStage {
				t.Errorf("IsStaging() = %v, want %v", env.IsStaging(), tc.isStage)
			}
		})
	}
}

// TestGetService 测试服务获取
func TestGetService(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.ConfigureServices(func(s *ServiceCollection) {
		AddSingleton[ITestService](s, di.WithFactory(func() ITestService {
			return &TestServiceImpl{value: "get_service"}
		}))
	})

	app := builder.Build()

	// 使用 GetService 获取服务
	var svc ITestService
	app.GetService(&svc)

	if svc == nil {
		t.Fatal("Service should not be nil")
	}

	if svc.GetValue() != "get_service" {
		t.Errorf("Expected 'get_service', got '%s'", svc.GetValue())
	}
}

// TestGetServicePanic 测试 GetService 参数错误时的 panic
func TestGetServicePanic(t *testing.T) {
	builder := NewApplicationBuilder()
	app := builder.Build()

	// 测试非指针参数
	t.Run("NonPointerArgument", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for non-pointer argument")
			}
		}()

		var svc ITestService
		app.GetService(svc) // 应该 panic
	})
}

// ===== BuildContext 方法测试 =====

type BuildContextTestExtension struct {
	ctx *BuildContext
}

func (e *BuildContextTestExtension) Name() string { return "BuildContextTest" }

func (e *BuildContextTestExtension) ConfigureBuilder(ctx *BuildContext) {
	e.ctx = ctx
}

// TestBuildContextContainer 测试 BuildContext 的容器访问
func TestBuildContextContainer(t *testing.T) {
	ext := &BuildContextTestExtension{}
	builder := NewApplicationBuilder()
	builder.AddExtension(ext)
	builder.Build()

	if ext.ctx == nil {
		t.Fatal("BuildContext should be provided to extension")
	}

	container := ext.ctx.Container()
	if container == nil {
		t.Error("Container should not be nil")
	}

	getContainer := ext.ctx.GetContainer()
	if getContainer == nil {
		t.Error("GetContainer should not be nil")
	}

	// 验证两个方法返回的是同一个容器
	if container != getContainer {
		t.Error("Container() and GetContainer() should return the same instance")
	}
}

// TestBuildContextLogger 测试 BuildContext 的日志器访问
func TestBuildContextLogger(t *testing.T) {
	ext := &BuildContextTestExtension{}
	builder := NewApplicationBuilder()
	builder.AddExtension(ext)
	builder.Build()

	logger := ext.ctx.GetLogger()
	if logger == nil {
		t.Error("Logger should not be nil")
	}
}

// TestBuildContextConfiguration 测试 BuildContext 的配置访问
func TestBuildContextConfiguration(t *testing.T) {
	ext := &BuildContextTestExtension{}
	builder := NewApplicationBuilder()
	builder.ConfigureConfiguration(func(cb *config.ConfigurationBuilder) {
		cb.AddInMemory(map[string]any{
			"build_context_key": "build_context_value",
		})
	})
	builder.AddExtension(ext)
	builder.Build()

	cfg := ext.ctx.GetConfiguration()
	if cfg == nil {
		t.Error("Configuration should not be nil")
	}

	if cfg.Get("build_context_key") != "build_context_value" {
		t.Error("Configuration value mismatch")
	}
}

// TestBuildContextEnvironment 测试 BuildContext 的环境访问
func TestBuildContextEnvironment(t *testing.T) {
	ext := &BuildContextTestExtension{}
	builder := NewApplicationBuilder()
	builder.UseEnvironment("production")
	builder.AddExtension(ext)
	builder.Build()

	env := ext.ctx.GetEnvironment()
	if env == nil {
		t.Error("Environment should not be nil")
	}

	if !env.IsProduction() {
		t.Error("Expected production environment")
	}
}

// TestBuildContextAddHostedService 测试 BuildContext 添加托管服务
func TestBuildContextAddHostedService(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.Configure(func(ctx *BuildContext) {
		ctx.AddHostedService(&testHostedService{
			onStart: func() {},
			onStop:  func() {},
		})
	})

	app := builder.Build()

	// 验证应用构建成功
	if app == nil {
		t.Error("Expected application to be built successfully")
	}

	// 注意：这里不启动应用，只验证托管服务被添加
}

// TestBuildContextSetCleanup 测试 BuildContext 设置清理函数
func TestBuildContextSetCleanup(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.Configure(func(ctx *BuildContext) {
		ctx.SetCleanup("test_cleanup", func() {
			// 清理逻辑
		})
	})

	app := builder.Build()

	// 验证应用构建成功
	if app == nil {
		t.Error("Expected application to be built successfully")
	}

	// 注意：清理函数在应用停止时才会被调用，这里只验证设置成功
}

// TestBuildContextResolveService 测试 BuildContext 解析服务
func TestBuildContextResolveService(t *testing.T) {
	ext := &BuildContextTestExtension{}
	builder := NewApplicationBuilder()
	builder.ConfigureServices(func(s *ServiceCollection) {
		AddSingleton[ITestService](s, di.WithFactory(func() ITestService {
			return &TestServiceImpl{value: "resolve_test"}
		}))
	})
	builder.AddExtension(ext)
	builder.Build()

	// 在 BuildContext 中解析服务
	svc, err := ext.ctx.ResolveService(reflect.TypeOf((*ITestService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve service: %v", err)
	}

	if svc.(ITestService).GetValue() != "resolve_test" {
		t.Errorf("Expected 'resolve_test', got '%s'", svc.(ITestService).GetValue())
	}
}

// ===== BaseBuilder 测试 =====

// TestBaseBuilderConfigContext 测试 BaseBuilder 的 ConfigContext 方法
func TestBaseBuilderConfigContext(t *testing.T) {
	builder := NewApplicationBuilder()
	var baseBuilder *BaseBuilder

	builder.Configure(func(ctx *BuildContext) {
		bb := NewBaseBuilder(ctx)
		baseBuilder = &bb

		configCtx := bb.ConfigContext()
		if configCtx == nil {
			t.Error("ConfigContext should not be nil")
		}
	})

	builder.Build()

	if baseBuilder == nil {
		t.Error("BaseBuilder should have been created")
	}
}

// TestBaseBuilderRegisterCleanup 测试 BaseBuilder 的 RegisterCleanup 方法
func TestBaseBuilderRegisterCleanup(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.Configure(func(ctx *BuildContext) {
		bb := NewBaseBuilder(ctx)
		bb.RegisterCleanup("base_builder_cleanup", func() {
			// 清理逻辑
		})
	})

	app := builder.Build()

	// 验证应用构建成功
	if app == nil {
		t.Error("Expected application to be built successfully")
	}

	// 注意：清理函数在应用停止时才会被调用
}

// ===== 辅助类型 =====

// testHostedService 测试用的托管服务
type testHostedService struct {
	onStart func()
	onStop  func()
}

func (s *testHostedService) Start(ctx context.Context) error {
	if s.onStart != nil {
		s.onStart()
	}
	return nil
}

func (s *testHostedService) Stop(ctx context.Context) error {
	if s.onStop != nil {
		s.onStop()
	}
	return nil
}

// ===== 链式调用测试 =====

// TestBuilderChaining 测试构建器的链式调用
func TestBuilderChaining(t *testing.T) {
	app := NewApplicationBuilder().
		UseEnvironment("staging").
		ConfigureConfiguration(func(cb *config.ConfigurationBuilder) {
			cb.AddInMemory(map[string]any{
				"chain_key": "chain_value",
			})
		}).
		ConfigureLogging(func(lb *logging.LoggingBuilder) {
			lb.SetMinimumLevel(logging.LogLevelDebug)
		}).
		ConfigureServices(func(s *ServiceCollection) {
			AddSingleton[ITestService](s, di.WithFactory(func() ITestService {
				return &TestServiceImpl{value: "chained"}
			}))
		}).
		UseShutdownTimeout(5 * time.Second).
		Build()

	// 验证环境
	if !app.Environment().IsStaging() {
		t.Error("Expected staging environment")
	}

	// 验证配置
	if app.Configuration().Get("chain_key") != "chain_value" {
		t.Error("Configuration value mismatch")
	}

	// 验证服务
	svc, err := app.Services().Get(reflect.TypeOf((*ITestService)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve service: %v", err)
	}
	if svc.(ITestService).GetValue() != "chained" {
		t.Error("Service value mismatch")
	}
}

// ===== 并发安全测试 =====

// TestBuilderConcurrentAccess 测试构建器的并发访问安全性
func TestBuilderConcurrentAccess(t *testing.T) {
	builder := NewApplicationBuilder()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			builder.ConfigureServices(func(s *ServiceCollection) {
				// 并发添加服务配置器
			})
		}(i)
	}

	wg.Wait()
	app := builder.Build()

	if app == nil {
		t.Error("Expected application to be built successfully after concurrent modifications")
	}
}
