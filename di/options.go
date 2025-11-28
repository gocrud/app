package di

import "reflect"

// Option 配置服务注册。
type Option func(*ServiceDefinition)

// WithScope 设置服务的生命周期范围。
func WithScope(scope ScopeType) Option {
	return func(s *ServiceDefinition) {
		s.Scope = scope
	}
}

// WithSingleton 将范围设置为 Singleton（默认）。
func WithSingleton() Option {
	return WithScope(ScopeSingleton)
}

// WithTransient 将范围设置为 Transient。
func WithTransient() Option {
	return WithScope(ScopeTransient)
}

// WithScoped 将范围设置为 Scoped。
func WithScoped() Option {
	return WithScope(ScopeScoped)
}

// WithValue 将具体的结构体实例注册为单例。
// 这意味着它已经创建，我们按原样使用它。
func WithValue(v any) Option {
	return func(s *ServiceDefinition) {
		s.Impl = v
		s.IsValue = true
		s.Scope = ScopeSingleton
	}
}

// WithFactory 注册一个工厂函数来创建实例。
// 工厂函数可以接受参数，这些参数将被注入。
func WithFactory(fn any) Option {
	return func(s *ServiceDefinition) {
		s.Impl = fn
		s.IsFactory = true
	}
}

// WithName 设置服务的名称，用于命名注入。
func WithName(name string) Option {
	return func(s *ServiceDefinition) {
		s.Name = name
	}
}

// Use 指定接口的实现类型。
func Use[T any]() Option {
	return func(s *ServiceDefinition) {
		s.ImplType = reflect.TypeOf((*T)(nil)).Elem()
	}
}
