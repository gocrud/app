package database

import (
	"context"
	"fmt"

	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"gorm.io/gorm"
)

// BuilderOption 用于配置 Database Builder
type BuilderOption func(*Builder)

// WithDatabase 添加数据库配置
func WithDatabase(name string, dialector gorm.Dialector, opts ...func(*DatabaseOptions)) BuilderOption {
	return func(b *Builder) {
		// 将变长参数转换为单个配置函数
		var configure func(*DatabaseOptions)
		if len(opts) > 0 {
			configure = func(o *DatabaseOptions) {
				for _, opt := range opts {
					opt(o)
				}
			}
		}
		b.Add(name, dialector, configure)
	}
}

// New 启用数据库能力
func New(opts ...BuilderOption) core.Option {
	return func(rt *core.Runtime) error {
		builder := NewBuilder()
		for _, opt := range opts {
			opt(builder)
		}

		// 2. 构建工厂 (此时尚未连接，连接通常是 Lazy 的或者在 Build 时发生)
		// 这里假设 Build 是安全的且不依赖 DI 容器
		// TODO: 注入 Logger
		factory, err := builder.Build(nil)
		if err != nil {
			return err
		}
		if factory == nil {
			return nil
		}

		// 3. 注册工厂到 DI
		if err := rt.Provide(factory, di.WithValue(factory)); err != nil {
			return err
		}

		// 4. 注册各个数据库实例到 DI
		var defaultRegErr error
		factory.Each(func(name string, db *gorm.DB) {
			// 注册命名实例
			if err := rt.Provide(db, di.WithName(name), di.WithValue(db)); err != nil {
				// 记录错误但不中断循环，但在最后返回
				// 实际上应该中断，这里简化处理
				defaultRegErr = err
			}

			// 如果是 default，同时也注册为默认实例
			if name == "default" {
				if err := rt.Provide(db, di.WithValue(db)); err != nil {
					defaultRegErr = err
				}
			}
		})

		if defaultRegErr != nil {
			return fmt.Errorf("database: failed to register instance: %w", defaultRegErr)
		}

		// 5. 注册清理钩子
		rt.Lifecycle.OnStop(func(ctx context.Context) error {
			// 这里简单打印日志，实际日志应该从容器获取 Logger
			fmt.Println("Closing database connections")
			return factory.Close()
		})

		return nil
	}
}
