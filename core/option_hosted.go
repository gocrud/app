package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gocrud/app/di"
)

// WithHostedService 注册一个托管服务
// 服务必须实现 HostedService 接口。
// 框架会在 OnStart 时启动 Goroutine 调用 Start，在 OnStop 时调用 Stop。
func WithHostedService(constructor any) Option {
	return func(rt *Runtime) error {
		// 1. 注册服务
		serviceType, err := di.Provide(rt.Container, constructor)
		if err != nil {
			return fmt.Errorf("WithHostedService: failed to provide service: %w", err)
		}

		// 2. 验证接口
		hostedServiceType := reflect.TypeOf((*HostedService)(nil)).Elem()
		if !serviceType.Implements(hostedServiceType) {
			return fmt.Errorf("WithHostedService: service %v does not implement core.HostedService", serviceType)
		}

		var serviceCtx context.Context
		var serviceCancel context.CancelFunc

		// 3. 注册生命周期
		rt.Lifecycle.OnStart(func(ctx context.Context) error {
			val, err := rt.Container.Get(serviceType)
			if err != nil {
				return fmt.Errorf("failed to resolve hosted service %v: %w", serviceType, err)
			}

			// 创建服务上下文，生命周期伴随应用运行
			serviceCtx, serviceCancel = context.WithCancel(context.Background())

			// 异步调用 Start，允许 Start 方法阻塞
			go func() {
				if err := val.(HostedService).Start(serviceCtx); err != nil {
					// 记录错误
					if rt.ErrorHandler != nil {
						rt.ErrorHandler(fmt.Errorf("HostedService %v exited with error: %w", serviceType, err))
					}
					// 触发应用退出 (Fail Fast)
					rt.Shutdown()
				}
			}()
			return nil
		})

		rt.Lifecycle.OnStop(func(ctx context.Context) error {
			// 通知 Context 取消
			if serviceCancel != nil {
				serviceCancel()
			}

			val, err := rt.Container.Get(serviceType)
			if err != nil {
				return nil
			}
			return val.(HostedService).Stop(ctx)
		})

		return nil
	}
}

// WorkerFunc 定义简单的后台任务函数
// 这是一个阻塞函数，通过 ctx.Done() 判断退出。
type WorkerFunc func(ctx context.Context) error

// WithWorker 将一个阻塞的函数注册为后台服务
// 框架会自动将其适配为 HostedService (异步启动，Cancel停止)
func WithWorker(fn WorkerFunc) Option {
	return func(rt *Runtime) error {
		var workerCtx context.Context
		var workerCancel context.CancelFunc

		rt.Lifecycle.OnStart(func(ctx context.Context) error {
			// 使用 Background 确保 Worker 存活
			workerCtx, workerCancel = context.WithCancel(context.Background())

			go func() {
				if err := fn(workerCtx); err != nil {
					if rt.ErrorHandler != nil {
						rt.ErrorHandler(fmt.Errorf("Worker exited with error: %w", err))
					}
					rt.Shutdown()
				}
			}()
			return nil
		})

		rt.Lifecycle.OnStop(func(ctx context.Context) error {
			if workerCancel != nil {
				workerCancel()
			}
			return nil
		})

		return nil
	}
}
