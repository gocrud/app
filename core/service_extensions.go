package core

import (
	"github.com/gocrud/app/di"
)

// AddSingleton 将接口 T 绑定到实现 impl，并注册为单例
// impl 可以是实例，也可以是构造函数
//
// 示例:
//
//	core.AddSingleton[IService](services, NewServiceImpl)
func AddSingleton[T any](s *ServiceCollection, impl any) {
	s.container.ProvideType(di.TypeProvider{
		Provide: di.TypeOf[T](),
		UseType: impl,
		Options: di.ProviderOptions{
			Scope: di.ScopeSingleton,
		},
	})
}

// AddTransient 将接口 T 绑定到实现 impl，并注册为瞬态服务
// impl 可以是实例，也可以是构造函数
//
// 示例:
//
//	core.AddTransient[IWorker](services, NewWorker)
func AddTransient[T any](s *ServiceCollection, impl any) {
	s.container.ProvideType(di.TypeProvider{
		Provide: di.TypeOf[T](),
		UseType: impl,
		Options: di.ProviderOptions{
			Scope: di.ScopeTransient,
		},
	})
}

// AddScoped 将接口 T 绑定到实现 impl，并注册为作用域服务
// impl 可以是实例，也可以是构造函数
//
// 示例:
//
//	core.AddScoped[IRequestScope](services, NewRequestScope)
func AddScoped[T any](s *ServiceCollection, impl any) {
	s.container.ProvideType(di.TypeProvider{
		Provide: di.TypeOf[T](),
		UseType: impl,
		Options: di.ProviderOptions{
			Scope: di.ScopeScoped,
		},
	})
}
