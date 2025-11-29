package redis

import (
	"fmt"

	"github.com/gocrud/app/logging"
)

// Builder Redis 客户端配置构建器
type Builder struct {
	configs map[string]RedisClientOptions
	errors  []error
}

// NewBuilder 创建 Redis 构建器
func NewBuilder() *Builder {
	return &Builder{
		configs:     make(map[string]RedisClientOptions),
		errors:      make([]error, 0),
	}
}

// AddClient 添加一个 Redis 客户端配置
func (b *Builder) AddClient(name string, configure func(*RedisClientOptions)) *Builder {
	// 检查名称冲突
	if _, exists := b.configs[name]; exists {
		b.errors = append(b.errors, fmt.Errorf("redis client '%s' already configured", name))
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
		b.errors = append(b.errors, fmt.Errorf("invalid redis configuration for '%s': %w", name, err))
		return b
	}

	// 保存配置
	b.configs[name] = *opts

	return b
}

// Build 构建 Redis 客户端工厂
func (b *Builder) Build(logger logging.Logger) (*RedisClientFactory, error) {
	// 检查是否有配置错误
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("redis configuration errors: %v", b.errors)
	}

	if len(b.configs) == 0 {
		return nil, nil // 没有配置任何 Redis 客户端
	}

	// 创建工厂
	factory := NewRedisClientFactory()

	// 注册所有客户端
	for _, opts := range b.configs {
		if err := factory.Register(opts); err != nil {
			return nil, fmt.Errorf("failed to register redis client '%s': %w", opts.Name, err)
		}

		if logger != nil {
			logger.Info("redis client registered",
				logging.Field{Key: "name", Value: opts.Name},
				logging.Field{Key: "addr", Value: opts.Addr},
				logging.Field{Key: "db", Value: opts.DB})
		}
	}

	return factory, nil
}
