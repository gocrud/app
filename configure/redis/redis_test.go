package redis_test

import (
	"testing"

	"github.com/gocrud/app/configure/redis"
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
	goredis "github.com/redis/go-redis/v9"
)

// MockRedisService 模拟依赖 Redis 客户端的服务
type MockRedisService struct {
	Cache *goredis.Client `di:"cache"`
	Queue *goredis.Client `di:"queue,?"`
}

func TestRedisConfiguration(t *testing.T) {
	builder := core.NewApplicationBuilder()

	// 配置 Redis
	configurator := redis.Configure(func(b *redis.Builder) {
		// 添加 cache 客户端
		b.AddClient("cache", func(o *redis.RedisClientOptions) {
			o.Addr = "localhost:6379"
		})
	})
	builder.Configure(func(ctx *core.BuildContext) {
		configurator(ctx)
	})

	// 注册模拟服务
	builder.Configure(func(ctx *core.BuildContext) {
		di.Register[*MockRedisService](ctx.Container())
	})

	// 构建应用
	app := builder.Build()

	// 解析服务
	var svc *MockRedisService
	app.GetService(&svc)

	// 验证注入
	if svc.Cache == nil {
		t.Error("Cache client should not be nil")
	}
	if svc.Queue != nil {
		t.Error("Queue client should be nil (optional and not configured)")
	}

	// 验证显式解析
	cache, err := di.ResolveNamed[*goredis.Client](app.Services(), "cache")
	if err != nil {
		t.Errorf("Failed to resolve named client 'cache': %v", err)
	}
	if cache == nil {
		t.Error("Resolved 'cache' client is nil")
	}
}

func TestRedisBuilder_Errors(t *testing.T) {
	logger := logging.NewLogger()
	builder := redis.NewBuilder()

	// 添加无效配置
	builder.AddClient("invalid", func(o *redis.RedisClientOptions) {
		o.Addr = "" // 必填项缺失
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
