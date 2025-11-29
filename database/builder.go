package database

import (
	"fmt"

	"github.com/gocrud/app/logging"
	"gorm.io/gorm"
)

// Builder 数据库配置构建器
type Builder struct {
	configs map[string]DatabaseOptions
	errors  []error
}

// NewBuilder 创建构建器
func NewBuilder() *Builder {
	return &Builder{
		configs: make(map[string]DatabaseOptions),
		errors:  make([]error, 0),
	}
}

// Add 添加数据库配置
// name: 实例名称
// dialector: GORM 驱动 (e.g. mysql.Open(dsn))
// configure: 可选的配置函数
func (b *Builder) Add(name string, dialector gorm.Dialector, configure func(*DatabaseOptions)) *Builder {
	if _, exists := b.configs[name]; exists {
		b.errors = append(b.errors, fmt.Errorf("database '%s' already configured", name))
		return b
	}

	opts := NewDefaultOptions(name, dialector)
	if configure != nil {
		configure(opts)
	}

	if err := opts.Validate(); err != nil {
		b.errors = append(b.errors, fmt.Errorf("invalid configuration for '%s': %w", name, err))
		return b
	}

	b.configs[name] = *opts
	return b
}

// Build 构建数据库工厂
func (b *Builder) Build(logger logging.Logger) (*DatabaseFactory, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("database configuration errors: %v", b.errors)
	}

	if len(b.configs) == 0 {
		return nil, nil
	}

	factory := NewDatabaseFactory()

	for _, opts := range b.configs {
		if err := factory.Register(opts); err != nil {
			return nil, fmt.Errorf("failed to register database '%s': %w", opts.Name, err)
		}

		if logger != nil {
			logger.Info("Database registered",
				logging.Field{Key: "name", Value: opts.Name},
				logging.Field{Key: "dialector", Value: opts.Dialector.Name()})
		}
	}

	return factory, nil
}
