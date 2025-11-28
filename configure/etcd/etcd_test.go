package etcd_test

import (
	"context"
	"testing"

	"github.com/gocrud/app/configure/etcd"
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// MockService 模拟依赖 Etcd 客户端的服务
type MockService struct {
	Master *clientv3.Client `di:"master"`
	Slave  *clientv3.Client `di:"slave,?"`
}

func TestEtcdConfiguration(t *testing.T) {
	// Setup via ApplicationBuilder which is the standard way
	builder := core.NewApplicationBuilder()

	// Configure Etcd
	configurator := etcd.Configure(func(b *etcd.Builder) {
		b.AddClient("master", func(o *etcd.EtcdClientOptions) {
			o.Endpoints = []string{"localhost:2379"}
		})
	})
	builder.Configure(func(ctx *core.BuildContext) {
		configurator(ctx)
	})

	// Register MockService
	builder.Configure(func(ctx *core.BuildContext) {
		di.Register[*MockService](ctx.Container())
	})

	// Build the application
	app := builder.Build()

	// Resolve Service
	var svc *MockService
	app.GetService(&svc)

	// Verify Injection
	if svc.Master == nil {
		t.Error("Master client should not be nil")
	}
	if svc.Slave != nil {
		t.Error("Slave client should be nil")
	}

	// Verify named resolution from container directly
	container := app.Services()
	master, err := di.ResolveNamed[*clientv3.Client](container, "master")
	if err != nil {
		t.Errorf("Failed to resolve named client 'master': %v", err)
	}
	if master == nil {
		t.Error("Resolved 'master' client is nil")
	}
}

func TestEtcdBuilder_Errors(t *testing.T) {
	logger := logging.NewLogger()
	// Mock a BuildContext or pass nil if not used
	// Since we refactored NewBuilder to require *BuildContext, we can pass nil for this isolated test
	// assuming AddClient doesn't use ctx (which it doesn't currently).
	builder := etcd.NewBuilder(nil)

	// 添加无效配置
	builder.AddClient("invalid", func(o *etcd.EtcdClientOptions) {
		o.Endpoints = nil // 必填项缺失
	})

	// 添加重复配置
	builder.AddClient("duplicate", nil)
	builder.AddClient("duplicate", nil)

	_, err := builder.Build(logger)
	if err == nil {
		t.Fatal("Expected error from invalid configuration, got nil")
	}

	t.Logf("Got expected error: %v", err)
}

// Functional test for Cleanup (mocking context cancellation/app stop)
func TestEtcdCleanup(t *testing.T) {
	builder := core.NewApplicationBuilder()

	configurator := etcd.Configure(func(b *etcd.Builder) {
		b.AddClient("test-cleanup", func(o *etcd.EtcdClientOptions) {
			o.Endpoints = []string{"localhost:2379"}
		})
	})
	builder.Configure(func(ctx *core.BuildContext) {
		configurator(ctx)
	})

	app := builder.Build()

	if err := app.Stop(context.Background()); err != nil {
		t.Errorf("Failed to stop app: %v", err)
	}
}
