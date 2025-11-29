package etcd

import (
	"context"
	"fmt"

	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// BuilderOption 用于配置 Etcd Builder
type BuilderOption func(*Builder)

// WithClient 添加 Etcd 客户端配置
func WithClient(name string, opts ...func(*EtcdClientOptions)) BuilderOption {
	return func(b *Builder) {
		var configure func(*EtcdClientOptions)
		if len(opts) > 0 {
			configure = func(o *EtcdClientOptions) {
				for _, opt := range opts {
					opt(o)
				}
			}
		}
		b.AddClient(name, configure)
	}
}

// New 启用 Etcd 能力
func New(opts ...BuilderOption) core.Option {
	return func(rt *core.Runtime) error {
		builder := NewBuilder()
		for _, opt := range opts {
			opt(builder)
		}

		// TODO: 注入 logger
		factory, err := builder.Build(nil)
		if err != nil {
			return err
		}
		if factory == nil {
			return nil
		}

		// 注册 factory 到容器
		if err := rt.Provide(factory, di.WithValue(factory)); err != nil {
			return err
		}

		// 注册各个客户端
		var defaultRegErr error
		factory.Each(func(name string, client *clientv3.Client) {
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
			return fmt.Errorf("etcd: failed to register instance: %w", defaultRegErr)
		}

		// 注册清理钩子
		rt.Lifecycle.OnStop(func(ctx context.Context) error {
			fmt.Println("Closing etcd clients")
			return factory.Close()
		})

		return nil
	}
}
