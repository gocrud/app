package mongodb

import (
	"context"
	"fmt"

	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/mgo"
)

// BuilderOption 用于配置 MongoDB Builder
type BuilderOption func(*Builder)

// WithClient 添加 MongoDB 客户端配置
func WithClient(name string, uri string, opts ...func(*MongoOptions)) BuilderOption {
	return func(b *Builder) {
		var configure func(*MongoOptions)
		if len(opts) > 0 {
			configure = func(o *MongoOptions) {
				for _, opt := range opts {
					opt(o)
				}
			}
		}
		b.Add(name, uri, configure)
	}
}

// New 启用 MongoDB 能力
func New(opts ...BuilderOption) core.Option {
	return func(rt *core.Runtime) error {
		builder := NewBuilder()
		for _, opt := range opts {
			opt(builder)
		}

		// TODO: 注入 Logger
		factory, err := builder.Build(nil)
		if err != nil {
			return err
		}
		if factory == nil {
			return nil
		}

		// 注册 Factory
		if err := rt.Provide(factory, di.WithValue(factory)); err != nil {
			return err
		}

		// 注册 Client 实例
		var defaultRegErr error
		factory.Each(func(name string, client *mgo.Client) {
			if err := rt.Provide(client, di.WithName(name), di.WithValue(client)); err != nil {
				defaultRegErr = err
			}

			if name == "default" {
				if err := rt.Provide(client, di.WithValue(client)); err != nil {
					defaultRegErr = err
				}
			}
		})

		if defaultRegErr != nil {
			return fmt.Errorf("mongodb: failed to register instance: %w", defaultRegErr)
		}

		// 注册清理
		rt.Lifecycle.OnStop(func(ctx context.Context) error {
			fmt.Println("Closing mongo clients")
			return factory.Close()
		})

		return nil
	}
}
