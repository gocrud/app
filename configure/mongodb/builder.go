package mongodb

import (
	"fmt"

	"github.com/gocrud/app/core"
	"github.com/gocrud/app/logging"
)

// Builder MongoDB 配置构建器
type Builder struct {
	core.BaseBuilder
	configs map[string]MongoOptions
	errors  []error
}

// NewBuilder 创建构建器
func NewBuilder(ctx *core.BuildContext) *Builder {
	return &Builder{
		BaseBuilder: core.NewBaseBuilder(ctx),
		configs:     make(map[string]MongoOptions),
		errors:      make([]error, 0),
	}
}

// Add 添加 MongoDB 客户端配置
func (b *Builder) Add(name string, uri string, configure func(*MongoOptions)) *Builder {
	if _, exists := b.configs[name]; exists {
		b.errors = append(b.errors, fmt.Errorf("mongo client '%s' already configured", name))
		return b
	}

	opts := NewDefaultOptions(name, uri)
	if configure != nil {
		configure(opts)
	}

	if err := opts.Validate(); err != nil {
		b.errors = append(b.errors, fmt.Errorf("invalid mongo configuration for '%s': %w", name, err))
		return b
	}

	b.configs[name] = *opts
	return b
}

// Build 构建 MongoDB 工厂
func (b *Builder) Build(logger logging.Logger) (*MongoFactory, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("mongo configuration errors: %v", b.errors)
	}

	if len(b.configs) == 0 {
		return nil, nil
	}

	factory := NewMongoFactory()

	for _, opts := range b.configs {
		if err := factory.Register(opts); err != nil {
			return nil, fmt.Errorf("failed to register mongo client '%s': %w", opts.Name, err)
		}

		logger.Info("Mongo client registered",
			logging.Field{Key: "name", Value: opts.Name},
			logging.Field{Key: "uri", Value: opts.Uri})
	}

	return factory, nil
}

