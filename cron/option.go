package cron

import (
	"context"

	"github.com/gocrud/app/core"
)

// BuilderOption 用于配置 Cron Builder
type BuilderOption func(*Builder)

// WithSeconds 启用秒级精度
func WithSeconds() BuilderOption {
	return func(b *Builder) {
		b.WithSeconds()
	}
}

// WithLocation 设置时区
func WithLocation(location string) BuilderOption {
	return func(b *Builder) {
		b.WithLocation(location)
	}
}

// EnableCronLogger 启用 cron 库的内部调度日志
func EnableCronLogger() BuilderOption {
	return func(b *Builder) {
		b.EnableCronLogger()
	}
}

// AddJob 添加任务
func AddJob(spec, name string, handler any) BuilderOption {
	return func(b *Builder) {
		b.AddJobWithDI(spec, name, handler)
	}
}

// New 启用 Cron 能力
func New(opts ...BuilderOption) core.Option {
	return func(rt *core.Runtime) error {
		builder := NewBuilder()
		for _, opt := range opts {
			opt(builder)
		}

		// 构建 cron service (需要 DI 支持)
		// 我们在运行时通过 wrapper 使用 DI 容器，所以这里不需要在构建时传入容器
		// 只有在 job 执行时才会用到容器
		
		// 我们需要一个 Service 来持有 cron 实例
		// TODO: 注入 Logger
		svc, err := builder.build(nil) // Logger 暂时传 nil，内部会处理
		if err != nil {
			return err
		}

		// 注册为 Host Service (后台运行)
		// 使用 Runtime 的 Lifecycle
		rt.Lifecycle.OnStart(func(ctx context.Context) error {
			// 注入 DI 容器和 Logger 到 svc
			// 我们的 builder.build() 可能返回了一个未完全初始化的 svc
			// 这里需要进行一些 hack 或者重构 build 逻辑
			// 更好的方式是：builder 只是收集 Job 定义，真正的构建发生在 OnStart
			
			// 重新设计 build: 
			// cronSvc 依赖 logger 和 container (用于 DI Job)
			// 我们在 OnStart 时构建它
			
			// 由于 builder.build 在原设计中返回 HostedService，我们先重构 builder
			
			// 临时方案：builder.build 不真正创建 cron 实例，而是返回一个 Config 对象？
			// 或者让 builder 保持配置，Start 时再初始化
			
			// 让 svc 初始化
			svc.Inject(rt.Container, nil) // 需要给 svc 加一个注入方法
			
			return svc.Start(ctx)
		})

		rt.Lifecycle.OnStop(func(ctx context.Context) error {
			return svc.Stop(ctx)
		})
		
		// 注册为特性
		rt.Features.Set(svc)

		return nil
	}
}
