package config

import (
	"context"
	"fmt"

	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
)

// LoadOptions 配置加载选项
type LoadOptions struct {
	Paths       []string
	HotReload   bool
	KeyDelimiter string
}

// LoadOption 配置加载选项函数
type LoadOption func(*LoadOptions)

// WithHotReload 启用热重载
func WithHotReload() LoadOption {
	return func(o *LoadOptions) {
		o.HotReload = true
	}
}

// Load 加载配置文件
// 支持 YAML, JSON (通过 YAML 解析器兼容)
func Load(path string, opts ...LoadOption) core.Option {
	return func(rt *core.Runtime) error {
		options := &LoadOptions{
			Paths:        []string{path},
			HotReload:    false,
			KeyDelimiter: ":",
		}
		for _, opt := range opts {
			opt(options)
		}

		// 创建 Configuration 实例
		cfg := NewConfiguration()
		
		// 加载文件
		for _, p := range options.Paths {
			if err := cfg.LoadFile(p); err != nil {
				// 暂时忽略文件不存在错误? 或者根据策略
				// 这里简单的打印错误
				fmt.Printf("config: failed to load %s: %v\n", p, err)
				// return err // 如果是必需的配置文件，应该返回错误
			}
		}

		// 加载环境变量
		cfg.LoadEnv()

		// 注册 Configuration 到 DI 容器
		// 同时支持 Configuration 接口和具体结构体
		di.ProvideService[Configuration](rt.Container, di.WithValue(cfg))
		// rt.Provide(cfg) // 也可以注册 *configuration，但通常接口就够了

		// 注册为 Runtime Feature
		rt.Features.Set(cfg)

		// 如果启用了热重载，启动监听
		if options.HotReload {
			rt.Lifecycle.OnStart(func(ctx context.Context) error {
				// TODO: 实现文件监听 (fsnotify)
				// 这里暂时留空
				return nil
			})
		}

		return nil
	}
}

// Bind 将配置绑定到结构体并注册到 DI 容器
func Bind[T any](rt *core.Runtime, section string) error {
	return rt.Invoke(func(cfg Configuration) error {
		var settings T
		if err := cfg.Bind(section, &settings); err != nil {
			return fmt.Errorf("config: failed to bind section '%s': %w", section, err)
		}

		// 注册为单例
		return rt.Provide(&settings)
	})
}

