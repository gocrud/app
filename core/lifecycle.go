package core

import (
	"context"

	"github.com/gocrud/app/di"
)

// LifecycleEvents 管理应用程序的生命周期
type LifecycleEvents struct {
	onStart []func(context.Context) error
	onStop  []func(context.Context) error
}

// NewLifecycle 创建新的生命周期管理器
func NewLifecycle() *LifecycleEvents {
	return &LifecycleEvents{
		onStart: make([]func(context.Context) error, 0),
		onStop:  make([]func(context.Context) error, 0),
	}
}

// OnStart 注册启动钩子
func (l *LifecycleEvents) OnStart(fn func(context.Context) error) {
	l.onStart = append(l.onStart, fn)
}

// OnStop 注册停止钩子
func (l *LifecycleEvents) OnStop(fn func(context.Context) error) {
	l.onStop = append(l.onStop, fn)
}

// Start 启动生命周期
func (l *LifecycleEvents) Start(ctx context.Context, container di.Container) error {
	// 这里未来可以加入从容器中解析 HostedService 的逻辑
	for _, fn := range l.onStart {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop 停止生命周期
func (l *LifecycleEvents) Stop(ctx context.Context) error {
	// 倒序执行停止钩子
	for i := len(l.onStop) - 1; i >= 0; i-- {
		fn := l.onStop[i]
		if err := fn(ctx); err != nil {
			// 记录错误但不中断，继续停止其他服务
			// TODO: Log error
		}
	}
	return nil
}
