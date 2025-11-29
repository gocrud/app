package core

import (
	"fmt"

	"github.com/gocrud/app/di"
)

// Runtime 是框架的上帝对象，作为状态容器
type Runtime struct {
	// Features 存放构建时特性 (WebBuilder, DbBuilder 等)
	Features FeatureCollection

	// Container 核心依赖注入容器
	Container di.Container

	// Lifecycle 生命周期管理
	Lifecycle *LifecycleEvents

	// shutdownCh 用于通知应用退出
	shutdownCh chan struct{}

	// ErrorHandler 用于记录运行时产生的严重错误
	// 外部可以通过设置此字段来接管错误日志
	ErrorHandler func(err error)
}

// NewRuntime 创建一个新的运行时实例
func NewRuntime() *Runtime {
	return &Runtime{
		Container:  di.NewContainer(),
		Lifecycle:  NewLifecycle(),
		shutdownCh: make(chan struct{}),
		ErrorHandler: func(err error) {
			// 默认输出到标准输出
			fmt.Printf("[Runtime Error] %v\n", err)
		},
	}
}

// Shutdown 请求应用退出
// 调用此方法会触发应用关闭流程
func (rt *Runtime) Shutdown() {
	select {
	case <-rt.shutdownCh:
		// 已经关闭，无需操作
	default:
		close(rt.shutdownCh)
	}
}

// Done 返回一个通道，当应用需要退出时该通道会关闭
func (rt *Runtime) Done() <-chan struct{} {
	return rt.shutdownCh
}

// Provide 注册服务提供者 (语法糖)
// 支持构造函数、结构体指针或接口绑定
func (rt *Runtime) Provide(target any, opts ...di.Option) error {
	_, err := di.Provide(rt.Container, target, opts...)
	return err
}

// Invoke 调用函数并注入依赖 (语法糖)
func (rt *Runtime) Invoke(function any) error {
	return di.Invoke(rt.Container, function)
}

// Apply 应用多个 Option
func (rt *Runtime) Apply(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(rt); err != nil {
			return err
		}
	}
	return nil
}

// As 是一个辅助函数，用于生成 di.Option，将实现绑定到接口
// 这是一个转发，为了让 core 包的使用者不需要直接引入 di 包
func As[T any]() di.Option {
	return di.Use[T]()
}
