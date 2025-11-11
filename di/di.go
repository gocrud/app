// Package di 提供了一个功能完整、高性能的依赖注入（Dependency Injection）容器。
//
// 主要特性：
//   - 支持多种注册方式：类型绑定、值注册、工厂函数、别名
//   - 三种生命周期作用域：Singleton（单例）、Transient（瞬态）、Scoped（作用域内单例）
//   - 自动依赖解析：构造函数注入和字段注入
//   - 泛型支持：类型安全的 API
//   - 并发安全：支持多线程环境
//   - 循环依赖检测：在构建阶段检测并报错
//
// 基本用法：
//
//	// 1. 注册服务
//	di.Provide(NewLogger)  // 注册构造函数
//	di.Bind[UserService](NewUserService)  // 接口绑定
//
//	// 2. 构建容器
//	di.MustBuild()
//
//	// 3. 获取实例
//	service := di.MustGet[UserService]()
//
// 作用域用法：
//
//	// 注册不同作用域的服务
//	di.Provide(NewLogger, WithScope(ScopeSingleton))     // 全局单例
//	di.Provide(NewRepository, WithScope(ScopeScoped))    // 作用域内单例
//	di.Provide(NewCommand, WithScope(ScopeTransient))    // 每次创建新实例
//
//	// 在 HTTP 请求中使用作用域
//	scope := di.GetContainer().CreateScope()
//	defer scope.Dispose()
//	repo, _ := scope.GetByType(reflect.TypeOf((*Repository)(nil)))
//
// 更多文档请参见：di/SCOPE_GUIDE.md
package di

var defaultContainer = NewContainer()

// Provide 注册值或构造函数到容器（简化语法）
func Provide(value any) {
	if err := defaultContainer.register(value); err != nil {
		panic("di.Provide failed: " + err.Error())
	}
}

// ProvideType 使用类型提供者注册（接口绑定）
func ProvideType(provider TypeProvider) {
	if err := defaultContainer.registerWithConfig(*provider.toProviderConfig()); err != nil {
		panic("di.ProvideType failed: " + err.Error())
	}
}

// ProvideValue 使用值提供者注册
func ProvideValue(provider ValueProvider) {
	if err := defaultContainer.registerWithConfig(*provider.toProviderConfig()); err != nil {
		panic("di.ProvideValue failed: " + err.Error())
	}
}

// ProvideFactory 使用工厂提供者注册
func ProvideFactory(provider FactoryProvider) {
	if err := defaultContainer.registerWithConfig(*provider.toProviderConfig()); err != nil {
		panic("di.ProvideFactory failed: " + err.Error())
	}
}

// ProvideExisting 使用别名提供者注册
func ProvideExisting(provider ExistingProvider) {
	if err := defaultContainer.registerWithConfig(*provider.toProviderConfig()); err != nil {
		panic("di.ProvideExisting failed: " + err.Error())
	}
}

// Bind 绑定接口到实现（语法糖）
func Bind[T any](impl any) {
	ProvideType(TypeProvider{
		Provide: TypeOf[T](),
		UseType: impl,
	})
}

// BindTo 创建别名（UseExisting）
func BindTo[T any](existing any) {
	ProvideExisting(ExistingProvider{
		Provide:  TypeOf[T](),
		Existing: existing,
	})
}

// Build 构建容器，完成所有依赖解析
func Build() error {
	return defaultContainer.Build()
}

// MustBuild 构建容器，失败时panic
func MustBuild() {
	if err := Build(); err != nil {
		panic("di.Build failed: " + err.Error())
	}
}

// Reset 重置全局容器（主要用于测试）
func Reset() {
	defaultContainer = NewContainer()
}

// GetContainer 获取默认容器（用于高级场景）
func GetContainer() *Container {
	return defaultContainer
}
