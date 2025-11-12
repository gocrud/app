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
//	// 1. 创建容器
//	container := di.NewContainer()
//
//	// 2. 注册服务
//	container.Provide(NewLogger)  // 注册构造函数
//	di.BindWith[UserService](container, NewUserService)  // 接口绑定
//
//	// 3. 构建容器
//	container.Build()
//
//	// 4. 获取实例
//	var service *UserService
//	container.Inject(&service)
//
// 作用域用法：
//
//	// 注册不同作用域的服务
//	container.ProvideWithConfig(di.ProviderConfig{
//		Provide: di.TypeOf[*Logger](),
//		UseClass: NewLogger,
//		Scope: di.ScopeSingleton,
//	})
//
//	// 在 HTTP 请求中使用作用域
//	scope := container.CreateScope()
//	defer scope.Dispose()
//	repo, _ := scope.GetByType(reflect.TypeOf((*Repository)(nil)))
//
// 更多文档请参见：di/SCOPE_GUIDE.md
package di
