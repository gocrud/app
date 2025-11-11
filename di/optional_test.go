package di_test

import (
	"testing"

	"github.com/gocrud/app/di"
)

type TestLogger interface {
	Log(msg string)
}

type TestConsoleLogger struct {
	Name string
}

func (l *TestConsoleLogger) Log(msg string) {}

type TestCache interface {
	Get(key string) string
}

type TestMemoryCache struct {
	Host string
}

func (c *TestMemoryCache) Get(key string) string { return "" }

// 测试可选字段注入
func TestOptionalFieldInjection(t *testing.T) {
	type Service struct {
		Logger TestLogger `di:""`  // 必需
		Cache  TestCache  `di:"?"` // 可选
	}

	di.Reset()

	// 只注册 Logger，不注册 Cache
	di.Bind[TestLogger](&TestConsoleLogger{Name: "test"})
	di.Provide(&Service{})

	// 构建应该成功（Cache 是可选的）
	if err := di.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	svc := di.Inject[*Service]()
	if svc.Logger == nil {
		t.Error("Expected Logger to be injected")
	}
	if svc.Cache != nil {
		t.Error("Expected Cache to be nil (optional and not registered)")
	}
}

// 测试可选字段注入（使用 optional 标签）
func TestOptionalFieldInjection_OptionalTag(t *testing.T) {
	type Service struct {
		Logger TestLogger `di:""`         // 必需
		Cache  TestCache  `di:"optional"` // 可选
	}

	di.Reset()

	// 只注册 Logger
	di.Bind[TestLogger](&TestConsoleLogger{Name: "test"})
	di.Provide(&Service{})

	// 构建应该成功
	if err := di.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	svc := di.Inject[*Service]()
	if svc.Logger == nil {
		t.Error("Expected Logger to be injected")
	}
	if svc.Cache != nil {
		t.Error("Expected Cache to be nil")
	}
}

// 测试 InjectOrDefault - 找到时
func TestInjectOrDefault_Found(t *testing.T) {
	di.Reset()
	di.Bind[TestLogger](&TestConsoleLogger{Name: "console"})
	di.MustBuild()

	defaultLogger := &TestConsoleLogger{Name: "default"}
	logger := di.InjectOrDefault[TestLogger](defaultLogger)

	// 应该返回注册的 logger，不是默认值
	if cl, ok := logger.(*TestConsoleLogger); !ok || cl.Name != "console" {
		t.Error("Expected console logger from container")
	}
}

// 测试 InjectOrDefault - 未找到时
func TestInjectOrDefault_NotFound(t *testing.T) {
	di.Reset()
	di.MustBuild()

	defaultCache := &TestMemoryCache{Host: "default"}
	cache := di.InjectOrDefault[TestCache](defaultCache)

	// 应该返回默认值
	if rc, ok := cache.(*TestMemoryCache); !ok || rc.Host != "default" {
		t.Error("Expected default cache")
	}
}

// 测试必需字段未注册应该失败
func TestRequiredFieldInjection_Failure(t *testing.T) {
	type Service struct {
		Logger TestLogger `di:""` // 必需
		Cache  TestCache  `di:""` // 必需
	}

	di.Reset()

	// 只注册 Logger，不注册 Cache
	di.Bind[TestLogger](&TestConsoleLogger{})
	di.Provide(&Service{})

	// 构建应该失败（Cache 是必需的但未注册）
	err := di.Build()
	if err == nil {
		t.Fatal("Expected Build to fail when required dependency is missing")
	}
}
