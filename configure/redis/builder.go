package redis

import (
	"fmt"

	"github.com/gocrud/app/logging"
)

// Builder Redis 客户端配置构建器
type Builder struct {
	configs []RedisClientOptions
}

// NewBuilder 创建 Redis 构建器
func NewBuilder() *Builder {
	return &Builder{
		configs: make([]RedisClientOptions, 0),
	}
}

// AddClient 添加一个 Redis 客户端配置
func (b *Builder) AddClient(name string, configure func(*RedisClientOptions)) *Builder {
	// 创建默认配置
	opts := NewDefaultOptions(name)

	// 应用用户配置
	if configure != nil {
		configure(opts)
	}

	// 验证配置
	if err := opts.Validate(); err != nil {
		panic(fmt.Sprintf("Invalid redis configuration for '%s': %v", name, err))
	}

	// 保存配置
	b.configs = append(b.configs, *opts)

	return b
}

// Build 构建 Redis 客户端工厂
func (b *Builder) Build(logger logging.Logger) (*RedisClientFactory, error) {
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

		logger.Info("redis client registered",
			logging.Field{Key: "name", Value: opts.Name},
			logging.Field{Key: "addr", Value: opts.Addr},
			logging.Field{Key: "db", Value: opts.DB})
	}

	return factory, nil
}
