package web

import (
	"fmt"

	"github.com/gocrud/app/core"
)

// BuilderOption 用于配置 Web Builder
type BuilderOption func(*Builder)

// WithPort 设置端口
func WithPort(port int) BuilderOption {
	return func(b *Builder) {
		b.UsePort(port)
	}
}

// WithControllers 添加控制器
func WithControllers(controllers ...any) BuilderOption {
	return func(b *Builder) {
		b.AddControllers(controllers...)
	}
}

// New 启用 Web 能力
func New(opts ...BuilderOption) core.Option {
	return func(rt *core.Runtime) error {
		// 1. 创建 WebBuilder
		// TODO: 注入 Logger
		builder := NewBuilder()

		// 应用选项
		for _, opt := range opts {
			opt(builder)
		}

		// 2. 注册为 Feature
		rt.Features.Set(builder)

		// 立即注册控制器服务到容器，因为容器很快就会被 Build
		if err := builder.RegisterServices(rt.Container); err != nil {
			return fmt.Errorf("web: failed to register services: %w", err)
		}

		// 3. 注册 Host 为 HostedService
		// 使用工厂函数延迟创建 Host，确保在 DI 容器构建后执行
		hostFactory := func() *Host {
			host := builder.Build(rt.Container)
			// 在创建实例时，顺便注册为 Feature，以便测试或其他组件获取
			rt.Features.Set(host)
			return host
		}

		// 使用 core.WithHostedService 自动管理生命周期 (Start/Stop)
		return core.WithHostedService(hostFactory)(rt)
	}
}
