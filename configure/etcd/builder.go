package etcd

import (
	"fmt"

	"github.com/gocrud/app/core"
	"github.com/gocrud/app/logging"
)

// Builder Etcd 客户端配置构建器
type Builder struct {
	core.BaseBuilder
	configs map[string]EtcdClientOptions
	errors  []error
}

// NewBuilder 创建 Etcd 构建器
func NewBuilder(ctx *core.BuildContext) *Builder {
	return &Builder{
		BaseBuilder: core.NewBaseBuilder(ctx),
		configs:     make(map[string]EtcdClientOptions),
		errors:      make([]error, 0),
	}
}

// AddClient 添加一个 etcd 客户端配置
func (b *Builder) AddClient(name string, configure func(*EtcdClientOptions)) *Builder {
	// 检查名称冲突
	if _, exists := b.configs[name]; exists {
		b.errors = append(b.errors, fmt.Errorf("etcd client '%s' already configured", name))
		return b
	}

	// 创建默认配置
	opts := NewDefaultOptions(name)

	// 应用用户配置
	if configure != nil {
		configure(opts)
	}

	// 验证配置
	if err := opts.Validate(); err != nil {
		b.errors = append(b.errors, fmt.Errorf("invalid etcd configuration for '%s': %w", name, err))
		return b
	}

	// 保存配置
	b.configs[name] = *opts

	return b
}

// Build 构建 Etcd 客户端工厂
func (b *Builder) Build(logger logging.Logger) (*EtcdClientFactory, error) {
	// 检查是否有配置错误
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("etcd configuration errors: %v", b.errors)
	}

	if len(b.configs) == 0 {
		return nil, nil // 没有配置任何 etcd 客户端
	}

	// 创建工厂
	factory := NewEtcdClientFactory()

	// 注册所有客户端
	for _, opts := range b.configs {
		if err := factory.Register(opts); err != nil {
			return nil, fmt.Errorf("failed to register etcd client '%s': %w", opts.Name, err)
		}

		logger.Info("etcd client registered",
			logging.Field{Key: "name", Value: opts.Name},
			logging.Field{Key: "endpoints", Value: fmt.Sprintf("%v", opts.Endpoints)})
	}

	return factory, nil
}
